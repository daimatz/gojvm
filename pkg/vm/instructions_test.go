package vm

import (
	"io"
	"testing"
)

// executeAndGetInt creates a Frame with the given bytecodes, runs the execution
// loop, and returns the int result. The bytecodes must end with ireturn (0xAC).
// Optional locals are set as int32 values starting at index 0.
func executeAndGetInt(t *testing.T, code []byte, locals ...int32) int32 {
	t.Helper()

	maxLocals := uint16(len(locals))
	if maxLocals < 4 {
		maxLocals = 4
	}

	frame := NewFrame(maxLocals, 10, code, nil)
	for i, val := range locals {
		frame.SetLocal(i, IntValue(val))
	}

	v := &VM{Stdout: io.Discard}

	for frame.PC < len(frame.Code) {
		opcode := frame.Code[frame.PC]
		frame.PC++
		retVal, hasReturn, err := v.executeInstruction(frame, opcode)
		if err != nil {
			t.Fatalf("execution error at PC=%d: %v", frame.PC-1, err)
		}
		if hasReturn {
			return retVal.Int
		}
	}

	t.Fatal("bytecode did not return a value (missing ireturn?)")
	return 0
}

func TestIconst(t *testing.T) {
	tests := []struct {
		name   string
		opcode byte
		want   int32
	}{
		{"iconst_m1", 0x02, -1},
		{"iconst_0", 0x03, 0},
		{"iconst_1", 0x04, 1},
		{"iconst_2", 0x05, 2},
		{"iconst_3", 0x06, 3},
		{"iconst_4", 0x07, 4},
		{"iconst_5", 0x08, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := []byte{tt.opcode, 0xAC} // iconst_N, ireturn
			got := executeAndGetInt(t, code)
			if got != tt.want {
				t.Errorf("%s: got %d, want %d", tt.name, got, tt.want)
			}
		})
	}
}

func TestBipush(t *testing.T) {
	tests := []struct {
		name string
		val  int8
		want int32
	}{
		{"positive", 42, 42},
		{"negative", -5, -5},
		{"zero", 0, 0},
		{"max_byte", 127, 127},
		{"min_byte", -128, -128},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := []byte{0x10, byte(tt.val), 0xAC} // bipush N, ireturn
			got := executeAndGetInt(t, code)
			if got != tt.want {
				t.Errorf("bipush %d: got %d, want %d", tt.val, got, tt.want)
			}
		})
	}
}

func TestArithmeticInstructions(t *testing.T) {
	tests := []struct {
		name string
		code []byte
		want int32
	}{
		{
			name: "iadd: 3+4=7",
			code: []byte{0x06, 0x07, 0x60, 0xAC}, // iconst_3, iconst_4, iadd, ireturn
			want: 7,
		},
		{
			name: "isub: 5-3=2",
			code: []byte{0x08, 0x06, 0x64, 0xAC}, // iconst_5, iconst_3, isub, ireturn
			want: 2,
		},
		{
			name: "imul: 3*4=12",
			code: []byte{0x06, 0x07, 0x68, 0xAC}, // iconst_3, iconst_4, imul, ireturn
			want: 12,
		},
		{
			name: "idiv: 5/2=2",
			code: []byte{0x08, 0x05, 0x6C, 0xAC}, // iconst_5, iconst_2, idiv, ireturn
			want: 2,
		},
		{
			name: "irem: 5%3=2",
			code: []byte{0x08, 0x06, 0x70, 0xAC}, // iconst_5, iconst_3, irem, ireturn
			want: 2,
		},
		{
			name: "ineg: -(5)=-5",
			code: []byte{0x08, 0x74, 0xAC}, // iconst_5, ineg, ireturn
			want: -5,
		},
		{
			name: "ineg double: -(-(3))=3",
			code: []byte{0x06, 0x74, 0x74, 0xAC}, // iconst_3, ineg, ineg, ireturn
			want: 3,
		},
		{
			name: "compound: (2+3)*4=20",
			code: []byte{0x05, 0x06, 0x60, 0x07, 0x68, 0xAC}, // iconst_2, iconst_3, iadd, iconst_4, imul, ireturn
			want: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := executeAndGetInt(t, tt.code)
			if got != tt.want {
				t.Errorf("%s: got %d, want %d", tt.name, got, tt.want)
			}
		})
	}
}

func TestBranch(t *testing.T) {
	t.Run("ifeq: taken (value == 0)", func(t *testing.T) {
		// Byte 0: iconst_0    (0x03)
		// Byte 1: ifeq        (0x99) branchPC=1, offset=5, target=6
		// Byte 2: 0x00
		// Byte 3: 0x05
		// Byte 4: iconst_1    (0x04)  -- not taken path
		// Byte 5: ireturn     (0xAC)
		// Byte 6: iconst_2    (0x05)  -- taken path
		// Byte 7: ireturn     (0xAC)
		code := []byte{0x03, 0x99, 0x00, 0x05, 0x04, 0xAC, 0x05, 0xAC}
		got := executeAndGetInt(t, code)
		if got != 2 {
			t.Errorf("ifeq taken: got %d, want 2", got)
		}
	})

	t.Run("ifeq: not taken (value != 0)", func(t *testing.T) {
		// iconst_1, ifeq(offset=5, target=6), iconst_3, ireturn, iconst_4, ireturn
		code := []byte{0x04, 0x99, 0x00, 0x05, 0x06, 0xAC, 0x07, 0xAC}
		got := executeAndGetInt(t, code)
		if got != 3 {
			t.Errorf("ifeq not taken: got %d, want 3", got)
		}
	})

	t.Run("ifne: taken (value != 0)", func(t *testing.T) {
		// iconst_1, ifne(offset=5, target=6), iconst_3, ireturn, iconst_4, ireturn
		code := []byte{0x04, 0x9A, 0x00, 0x05, 0x06, 0xAC, 0x07, 0xAC}
		got := executeAndGetInt(t, code)
		if got != 4 {
			t.Errorf("ifne taken: got %d, want 4", got)
		}
	})

	t.Run("ifne: not taken (value == 0)", func(t *testing.T) {
		// iconst_0, ifne(offset=5, target=6), iconst_3, ireturn, iconst_4, ireturn
		code := []byte{0x03, 0x9A, 0x00, 0x05, 0x06, 0xAC, 0x07, 0xAC}
		got := executeAndGetInt(t, code)
		if got != 3 {
			t.Errorf("ifne not taken: got %d, want 3", got)
		}
	})

	t.Run("goto: unconditional jump", func(t *testing.T) {
		// Byte 0: goto        (0xA7) branchPC=0, offset=5, target=5
		// Byte 1: 0x00
		// Byte 2: 0x05
		// Byte 3: iconst_1    (0x04)  -- skipped
		// Byte 4: ireturn     (0xAC)  -- skipped
		// Byte 5: iconst_2    (0x05)  -- jumped to here
		// Byte 6: ireturn     (0xAC)
		code := []byte{0xA7, 0x00, 0x05, 0x04, 0xAC, 0x05, 0xAC}
		got := executeAndGetInt(t, code)
		if got != 2 {
			t.Errorf("goto: got %d, want 2", got)
		}
	})

	t.Run("iflt: taken (value < 0)", func(t *testing.T) {
		// Byte 0: bipush      (0x10)
		// Byte 1: 0xFF        (-1 as signed byte)
		// Byte 2: iflt        (0x9B) branchPC=2, offset=5, target=7
		// Byte 3: 0x00
		// Byte 4: 0x05
		// Byte 5: iconst_0    (0x03)  -- not taken
		// Byte 6: ireturn     (0xAC)
		// Byte 7: iconst_1    (0x04)  -- taken
		// Byte 8: ireturn     (0xAC)
		code := []byte{0x10, 0xFF, 0x9B, 0x00, 0x05, 0x03, 0xAC, 0x04, 0xAC}
		got := executeAndGetInt(t, code)
		if got != 1 {
			t.Errorf("iflt taken: got %d, want 1", got)
		}
	})
}

func TestStackOps(t *testing.T) {
	t.Run("dup: duplicate top of stack", func(t *testing.T) {
		// iconst_3, dup, iadd, ireturn -> 3+3=6
		code := []byte{0x06, 0x59, 0x60, 0xAC}
		got := executeAndGetInt(t, code)
		if got != 6 {
			t.Errorf("dup + iadd: got %d, want 6", got)
		}
	})

	t.Run("pop: discard top of stack", func(t *testing.T) {
		// iconst_3, iconst_4, pop, ireturn -> pop 4, return 3
		code := []byte{0x06, 0x07, 0x57, 0xAC}
		got := executeAndGetInt(t, code)
		if got != 3 {
			t.Errorf("pop: got %d, want 3", got)
		}
	})

	t.Run("swap: exchange top two values", func(t *testing.T) {
		// iconst_5, iconst_2, swap, isub, ireturn
		// stack after push: [5, 2] (bottom to top)
		// after swap: [2, 5]
		// isub: value1=2, value2=5 -> 2-5 = -3
		code := []byte{0x08, 0x05, 0x5F, 0x64, 0xAC}
		got := executeAndGetInt(t, code)
		if got != -3 {
			t.Errorf("swap + isub: got %d, want -3", got)
		}
	})
}

func TestLocalVarInstructions(t *testing.T) {
	t.Run("istore and iload", func(t *testing.T) {
		// iconst_5, istore_0, iload_0, ireturn -> store 5, load 5, return 5
		code := []byte{0x08, 0x3B, 0x1A, 0xAC}
		got := executeAndGetInt(t, code)
		if got != 5 {
			t.Errorf("istore_0/iload_0: got %d, want 5", got)
		}
	})

	t.Run("istore/iload with index", func(t *testing.T) {
		// bipush 42, istore 2, iload 2, ireturn
		code := []byte{0x10, 0x2A, 0x36, 0x02, 0x15, 0x02, 0xAC}
		got := executeAndGetInt(t, code)
		if got != 42 {
			t.Errorf("istore/iload index 2: got %d, want 42", got)
		}
	})

	t.Run("iload with pre-set locals", func(t *testing.T) {
		// iload_0, iload_1, iadd, ireturn -> locals[0]+locals[1]
		code := []byte{0x1A, 0x1B, 0x60, 0xAC}
		got := executeAndGetInt(t, code, 10, 20)
		if got != 30 {
			t.Errorf("iload from preset locals: got %d, want 30", got)
		}
	})
}
