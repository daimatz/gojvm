package vm

import (
	"io"
	"testing"

	"github.com/daimatz/gojvm/pkg/classfile"
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

func TestSipush(t *testing.T) {
	tests := []struct {
		name string
		hi   byte
		lo   byte
		want int32
	}{
		{"positive", 0x01, 0x00, 256},
		{"negative", 0xFF, 0x00, -256},
		{"zero", 0x00, 0x00, 0},
		{"max_short", 0x7F, 0xFF, 32767},
		{"min_short", 0x80, 0x00, -32768},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := []byte{0x11, tt.hi, tt.lo, 0xAC} // sipush hi lo, ireturn
			got := executeAndGetInt(t, code)
			if got != tt.want {
				t.Errorf("sipush %d: got %d, want %d", tt.want, got, tt.want)
			}
		})
	}
}

func TestDivisionByZero(t *testing.T) {
	v := &VM{Stdout: io.Discard}

	t.Run("idiv by zero", func(t *testing.T) {
		// iconst_5, iconst_0, idiv, ireturn
		code := []byte{0x08, 0x03, 0x6C, 0xAC}
		frame := NewFrame(4, 10, code, nil)

		var err error
		for frame.PC < len(frame.Code) {
			opcode := frame.Code[frame.PC]
			frame.PC++
			_, hasReturn, e := v.executeInstruction(frame, opcode)
			if e != nil {
				err = e
				break
			}
			if hasReturn {
				break
			}
		}

		if err == nil {
			t.Fatal("expected ArithmeticException for idiv by zero, got nil")
		}
		if got := err.Error(); got != "ArithmeticException: / by zero" {
			t.Errorf("error message: got %q, want %q", got, "ArithmeticException: / by zero")
		}
	})

	t.Run("irem by zero", func(t *testing.T) {
		// iconst_5, iconst_0, irem, ireturn
		code := []byte{0x08, 0x03, 0x70, 0xAC}
		frame := NewFrame(4, 10, code, nil)

		var err error
		for frame.PC < len(frame.Code) {
			opcode := frame.Code[frame.PC]
			frame.PC++
			_, hasReturn, e := v.executeInstruction(frame, opcode)
			if e != nil {
				err = e
				break
			}
			if hasReturn {
				break
			}
		}

		if err == nil {
			t.Fatal("expected ArithmeticException for irem by zero, got nil")
		}
	})
}

func TestOverflow(t *testing.T) {
	tests := []struct {
		name   string
		code   []byte
		want   int32
		locals []int32
	}{
		{
			name: "iadd overflow wraps",
			// bipush 127 is max we can push with bipush; use locals for larger values
			// iload_0, iload_1, iadd, ireturn
			code:   []byte{0x1A, 0x1B, 0x60, 0xAC},
			locals: []int32{2147483647, 1}, // MaxInt32 + 1
			want:   -2147483648,            // wraps to MinInt32
		},
		{
			name:   "isub underflow wraps",
			code:   []byte{0x1A, 0x1B, 0x64, 0xAC},
			locals: []int32{-2147483648, 1}, // MinInt32 - 1
			want:   2147483647,              // wraps to MaxInt32
		},
		{
			name:   "imul overflow wraps",
			code:   []byte{0x1A, 0x1B, 0x68, 0xAC},
			locals: []int32{2147483647, 2},
			want:   -2,
		},
		{
			name:   "ineg MinInt32 stays MinInt32",
			code:   []byte{0x1A, 0x74, 0xAC},
			locals: []int32{-2147483648},
			want:   -2147483648, // -MinInt32 overflows back to MinInt32
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := executeAndGetInt(t, tt.code, tt.locals...)
			if got != tt.want {
				t.Errorf("%s: got %d, want %d", tt.name, got, tt.want)
			}
		})
	}
}

func TestIfIcmp(t *testing.T) {
	// Helper: builds bytecode for if_icmpXX where the two values come from locals
	// iload_0, iload_1, if_icmpXX(offset=5, target=7), iconst_0, ireturn, iconst_1, ireturn
	buildCode := func(opcode byte) []byte {
		return []byte{0x1A, 0x1B, opcode, 0x00, 0x05, 0x03, 0xAC, 0x04, 0xAC}
	}

	tests := []struct {
		name   string
		opcode byte
		a, b   int32
		want   int32 // 1=taken, 0=not taken
	}{
		{"if_icmpeq taken", 0x9F, 5, 5, 1},
		{"if_icmpeq not taken", 0x9F, 5, 3, 0},
		{"if_icmpne taken", 0xA0, 5, 3, 1},
		{"if_icmpne not taken", 0xA0, 5, 5, 0},
		{"if_icmplt taken", 0xA1, 3, 5, 1},
		{"if_icmplt not taken", 0xA1, 5, 3, 0},
		{"if_icmpge taken (>)", 0xA2, 5, 3, 1},
		{"if_icmpge taken (=)", 0xA2, 5, 5, 1},
		{"if_icmpge not taken", 0xA2, 3, 5, 0},
		{"if_icmpgt taken", 0xA3, 5, 3, 1},
		{"if_icmpgt not taken (=)", 0xA3, 5, 5, 0},
		{"if_icmple taken (<)", 0xA4, 3, 5, 1},
		{"if_icmple taken (=)", 0xA4, 5, 5, 1},
		{"if_icmple not taken", 0xA4, 5, 3, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := buildCode(tt.opcode)
			got := executeAndGetInt(t, code, tt.a, tt.b)
			if got != tt.want {
				t.Errorf("%s (%d vs %d): got %d, want %d", tt.name, tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestRemainingBranches(t *testing.T) {
	// Tests for ifge, ifgt, ifle that aren't covered by TestBranch
	// Bytecode: iload_0, ifXX(offset=5, target=5), iconst_0, ireturn, iconst_1, ireturn
	buildCode := func(opcode byte) []byte {
		return []byte{0x1A, opcode, 0x00, 0x05, 0x03, 0xAC, 0x04, 0xAC}
	}

	tests := []struct {
		name   string
		opcode byte
		val    int32
		want   int32 // 1=taken, 0=not taken
	}{
		{"ifge taken (positive)", 0x9C, 5, 1},
		{"ifge taken (zero)", 0x9C, 0, 1},
		{"ifge not taken (negative)", 0x9C, -1, 0},
		{"ifgt taken", 0x9D, 5, 1},
		{"ifgt not taken (zero)", 0x9D, 0, 0},
		{"ifgt not taken (negative)", 0x9D, -1, 0},
		{"ifle taken (negative)", 0x9E, -1, 1},
		{"ifle taken (zero)", 0x9E, 0, 1},
		{"ifle not taken (positive)", 0x9E, 5, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := executeAndGetInt(t, buildCode(tt.opcode), tt.val)
			if got != tt.want {
				t.Errorf("%s (val=%d): got %d, want %d", tt.name, tt.val, got, tt.want)
			}
		})
	}
}

func TestIfnull(t *testing.T) {
	v := &VM{Stdout: io.Discard}

	// Bytecode layout:
	// Byte 0: aload_0       (0x2A)
	// Byte 1: ifnull        (0xC6) branchPC=1, offset=5, target=6
	// Byte 2: 0x00
	// Byte 3: 0x05
	// Byte 4: iconst_1      (0x04)  -- not taken (non-null)
	// Byte 5: ireturn       (0xAC)
	// Byte 6: iconst_2      (0x05)  -- taken (null)
	// Byte 7: ireturn       (0xAC)

	t.Run("taken (null value)", func(t *testing.T) {
		code := []byte{0x2A, 0xC6, 0x00, 0x05, 0x04, 0xAC, 0x05, 0xAC}
		frame := NewFrame(4, 10, code, nil)
		frame.SetLocal(0, NullValue())

		for frame.PC < len(frame.Code) {
			opcode := frame.Code[frame.PC]
			frame.PC++
			retVal, hasReturn, err := v.executeInstruction(frame, opcode)
			if err != nil {
				t.Fatalf("execution error at PC=%d: %v", frame.PC-1, err)
			}
			if hasReturn {
				if retVal.Int != 2 {
					t.Errorf("ifnull taken: got %d, want 2", retVal.Int)
				}
				return
			}
		}
		t.Fatal("bytecode did not return a value")
	})

	t.Run("not taken (non-null value)", func(t *testing.T) {
		code := []byte{0x2A, 0xC6, 0x00, 0x05, 0x04, 0xAC, 0x05, 0xAC}
		frame := NewFrame(4, 10, code, nil)
		frame.SetLocal(0, RefValue("some object"))

		for frame.PC < len(frame.Code) {
			opcode := frame.Code[frame.PC]
			frame.PC++
			retVal, hasReturn, err := v.executeInstruction(frame, opcode)
			if err != nil {
				t.Fatalf("execution error at PC=%d: %v", frame.PC-1, err)
			}
			if hasReturn {
				if retVal.Int != 1 {
					t.Errorf("ifnull not taken: got %d, want 1", retVal.Int)
				}
				return
			}
		}
		t.Fatal("bytecode did not return a value")
	})
}

func TestAreturn(t *testing.T) {
	v := &VM{Stdout: io.Discard}

	t.Run("return object reference", func(t *testing.T) {
		// aload_0, areturn (0xB0)
		code := []byte{0x2A, 0xB0}
		frame := NewFrame(4, 10, code, nil)
		obj := "test-object"
		frame.SetLocal(0, RefValue(obj))

		for frame.PC < len(frame.Code) {
			opcode := frame.Code[frame.PC]
			frame.PC++
			retVal, hasReturn, err := v.executeInstruction(frame, opcode)
			if err != nil {
				t.Fatalf("execution error at PC=%d: %v", frame.PC-1, err)
			}
			if hasReturn {
				if retVal.Type != TypeRef {
					t.Errorf("areturn: got type %v, want TypeRef", retVal.Type)
				}
				if retVal.Ref != obj {
					t.Errorf("areturn: got ref %v, want %v", retVal.Ref, obj)
				}
				return
			}
		}
		t.Fatal("bytecode did not return a value")
	})

	t.Run("return null reference", func(t *testing.T) {
		// aconst_null, areturn
		code := []byte{0x01, 0xB0}
		frame := NewFrame(4, 10, code, nil)

		for frame.PC < len(frame.Code) {
			opcode := frame.Code[frame.PC]
			frame.PC++
			retVal, hasReturn, err := v.executeInstruction(frame, opcode)
			if err != nil {
				t.Fatalf("execution error at PC=%d: %v", frame.PC-1, err)
			}
			if hasReturn {
				if retVal.Type != TypeNull {
					t.Errorf("areturn null: got type %v, want TypeNull", retVal.Type)
				}
				return
			}
		}
		t.Fatal("bytecode did not return a value")
	})
}

func TestGetfieldPutfield(t *testing.T) {
	// Constant pool: Fieldref for "TestClass.x:I"
	pool := make([]classfile.ConstantPoolEntry, 7)
	pool[1] = &classfile.ConstantFieldref{ClassIndex: 2, NameAndTypeIndex: 3}
	pool[2] = &classfile.ConstantClass{NameIndex: 4}
	pool[3] = &classfile.ConstantNameAndType{NameIndex: 5, DescriptorIndex: 6}
	pool[4] = &classfile.ConstantUtf8{Value: "TestClass"}
	pool[5] = &classfile.ConstantUtf8{Value: "x"}
	pool[6] = &classfile.ConstantUtf8{Value: "I"}
	cf := &classfile.ClassFile{ConstantPool: pool}

	t.Run("putfield then getfield returns stored value", func(t *testing.T) {
		code := []byte{
			0x2A,             // aload_0
			0x10, 0x37,       // bipush 55
			0xB5, 0x00, 0x01, // putfield #1
			0x2A,             // aload_0
			0xB4, 0x00, 0x01, // getfield #1
			0xAC,             // ireturn
		}

		frame := NewFrame(4, 10, code, cf)
		obj := &JObject{ClassName: "TestClass", Fields: make(map[string]Value)}
		frame.SetLocal(0, RefValue(obj))

		v := &VM{Stdout: io.Discard}
		for frame.PC < len(frame.Code) {
			opcode := frame.Code[frame.PC]
			frame.PC++
			retVal, hasReturn, err := v.executeInstruction(frame, opcode)
			if err != nil {
				t.Fatalf("execution error at PC=%d: %v", frame.PC-1, err)
			}
			if hasReturn {
				if retVal.Int != 55 {
					t.Errorf("getfield after putfield: got %d, want 55", retVal.Int)
				}
				return
			}
		}
		t.Fatal("bytecode did not return a value")
	})

	t.Run("getfield on unset int field returns zero", func(t *testing.T) {
		code := []byte{
			0x2A,             // aload_0
			0xB4, 0x00, 0x01, // getfield #1 (descriptor "I")
			0xAC,             // ireturn
		}

		frame := NewFrame(4, 10, code, cf)
		obj := &JObject{ClassName: "TestClass", Fields: make(map[string]Value)}
		frame.SetLocal(0, RefValue(obj))

		v := &VM{Stdout: io.Discard}
		for frame.PC < len(frame.Code) {
			opcode := frame.Code[frame.PC]
			frame.PC++
			retVal, hasReturn, err := v.executeInstruction(frame, opcode)
			if err != nil {
				t.Fatalf("execution error at PC=%d: %v", frame.PC-1, err)
			}
			if hasReturn {
				if retVal.Type != TypeInt || retVal.Int != 0 {
					t.Errorf("getfield on unset int field: got type=%v val=%d, want TypeInt val=0", retVal.Type, retVal.Int)
				}
				return
			}
		}
		t.Fatal("bytecode did not return a value")
	})
}

func TestInvokespecialObjectInit(t *testing.T) {
	// Constant pool: Methodref for java/lang/Object.<init>:()V
	pool := make([]classfile.ConstantPoolEntry, 7)
	pool[1] = &classfile.ConstantMethodref{ClassIndex: 2, NameAndTypeIndex: 3}
	pool[2] = &classfile.ConstantClass{NameIndex: 4}
	pool[3] = &classfile.ConstantNameAndType{NameIndex: 5, DescriptorIndex: 6}
	pool[4] = &classfile.ConstantUtf8{Value: "java/lang/Object"}
	pool[5] = &classfile.ConstantUtf8{Value: "<init>"}
	pool[6] = &classfile.ConstantUtf8{Value: "()V"}
	cf := &classfile.ClassFile{ConstantPool: pool}

	t.Run("Object init is no-op", func(t *testing.T) {
		code := []byte{
			0x08,             // iconst_5 (marker)
			0x2A,             // aload_0 (push object for invokespecial)
			0xB7, 0x00, 0x01, // invokespecial #1 (Object.<init>)
			0xAC,             // ireturn (returns marker 5)
		}

		frame := NewFrame(4, 10, code, cf)
		obj := &JObject{ClassName: "TestClass", Fields: make(map[string]Value)}
		frame.SetLocal(0, RefValue(obj))

		v := &VM{Stdout: io.Discard}
		for frame.PC < len(frame.Code) {
			opcode := frame.Code[frame.PC]
			frame.PC++
			retVal, hasReturn, err := v.executeInstruction(frame, opcode)
			if err != nil {
				t.Fatalf("execution error at PC=%d: %v", frame.PC-1, err)
			}
			if hasReturn {
				if retVal.Int != 5 {
					t.Errorf("invokespecial Object.<init>: got %d, want 5 (marker)", retVal.Int)
				}
				return
			}
		}
		t.Fatal("bytecode did not return a value")
	})
}

func TestCheckcast(t *testing.T) {
	// Constant pool with a Class entry (checkcast reads and discards it)
	pool := make([]classfile.ConstantPoolEntry, 3)
	pool[1] = &classfile.ConstantClass{NameIndex: 2}
	pool[2] = &classfile.ConstantUtf8{Value: "SomeClass"}
	cf := &classfile.ClassFile{ConstantPool: pool}

	t.Run("checkcast passes through reference", func(t *testing.T) {
		code := []byte{
			0x2A,             // aload_0
			0xC0, 0x00, 0x01, // checkcast #1
			0xB0,             // areturn
		}

		frame := NewFrame(4, 10, code, cf)
		obj := &JObject{ClassName: "SomeClass", Fields: make(map[string]Value)}
		frame.SetLocal(0, RefValue(obj))

		v := &VM{Stdout: io.Discard}
		for frame.PC < len(frame.Code) {
			opcode := frame.Code[frame.PC]
			frame.PC++
			retVal, hasReturn, err := v.executeInstruction(frame, opcode)
			if err != nil {
				t.Fatalf("execution error at PC=%d: %v", frame.PC-1, err)
			}
			if hasReturn {
				if retVal.Type != TypeRef {
					t.Errorf("checkcast: got type %v, want TypeRef", retVal.Type)
				}
				if retVal.Ref != obj {
					t.Errorf("checkcast: reference not preserved")
				}
				return
			}
		}
		t.Fatal("bytecode did not return a value")
	})
}

func TestIfnullNonNullJObject(t *testing.T) {
	v := &VM{Stdout: io.Discard}

	// aload_0, ifnull(offset=5, target=6), iconst_1, ireturn, iconst_2, ireturn
	code := []byte{0x2A, 0xC6, 0x00, 0x05, 0x04, 0xAC, 0x05, 0xAC}

	t.Run("JObject is non-null", func(t *testing.T) {
		frame := NewFrame(4, 10, code, nil)
		obj := &JObject{ClassName: "java/lang/Integer", Fields: make(map[string]Value)}
		frame.SetLocal(0, RefValue(obj))

		for frame.PC < len(frame.Code) {
			opcode := frame.Code[frame.PC]
			frame.PC++
			retVal, hasReturn, err := v.executeInstruction(frame, opcode)
			if err != nil {
				t.Fatalf("execution error at PC=%d: %v", frame.PC-1, err)
			}
			if hasReturn {
				if retVal.Int != 1 {
					t.Errorf("ifnull with JObject: got %d, want 1 (not taken)", retVal.Int)
				}
				return
			}
		}
		t.Fatal("bytecode did not return a value")
	})
}

func TestIinc(t *testing.T) {
	tests := []struct {
		name    string
		initial int32
		inc     int8
		want    int32
	}{
		{"positive increment", 10, 5, 15},
		{"negative increment", 10, -3, 7},
		{"zero increment", 42, 0, 42},
		{"increment from zero", 0, 1, 1},
		{"large negative", 100, -128, -28},
		{"large positive", 0, 127, 127},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// iload_0, iinc 0 <const>, iload_0, ireturn
			code := []byte{0x1A, OpIinc, 0x00, byte(tt.inc), 0x1A, 0xAC}
			got := executeAndGetInt(t, code, tt.initial)
			if got != tt.want {
				t.Errorf("iinc(%d, %d): got %d, want %d", tt.initial, tt.inc, got, tt.want)
			}
		})
	}
}

func TestAnewarray(t *testing.T) {
	v := &VM{Stdout: io.Discard}

	// Constant pool with a Class entry for the element type
	pool := make([]classfile.ConstantPoolEntry, 3)
	pool[1] = &classfile.ConstantClass{NameIndex: 2}
	pool[2] = &classfile.ConstantUtf8{Value: "java/lang/Object"}
	cf := &classfile.ClassFile{ConstantPool: pool}

	t.Run("create array of size 5", func(t *testing.T) {
		code := []byte{
			0x08,             // iconst_5
			OpAnewarray, 0x00, 0x01, // anewarray #1
			0xB0,             // areturn
		}
		frame := NewFrame(4, 10, code, cf)

		for frame.PC < len(frame.Code) {
			opcode := frame.Code[frame.PC]
			frame.PC++
			retVal, hasReturn, err := v.executeInstruction(frame, opcode)
			if err != nil {
				t.Fatalf("execution error at PC=%d: %v", frame.PC-1, err)
			}
			if hasReturn {
				arr, ok := retVal.Ref.(*JArray)
				if !ok {
					t.Fatalf("expected *JArray, got %T", retVal.Ref)
				}
				if len(arr.Elements) != 5 {
					t.Errorf("array length: got %d, want 5", len(arr.Elements))
				}
				return
			}
		}
		t.Fatal("bytecode did not return a value")
	})

	t.Run("create array of size 0", func(t *testing.T) {
		code := []byte{
			0x03,             // iconst_0
			OpAnewarray, 0x00, 0x01, // anewarray #1
			0xB0,             // areturn
		}
		frame := NewFrame(4, 10, code, cf)

		for frame.PC < len(frame.Code) {
			opcode := frame.Code[frame.PC]
			frame.PC++
			retVal, hasReturn, err := v.executeInstruction(frame, opcode)
			if err != nil {
				t.Fatalf("execution error at PC=%d: %v", frame.PC-1, err)
			}
			if hasReturn {
				arr, ok := retVal.Ref.(*JArray)
				if !ok {
					t.Fatalf("expected *JArray, got %T", retVal.Ref)
				}
				if len(arr.Elements) != 0 {
					t.Errorf("array length: got %d, want 0", len(arr.Elements))
				}
				return
			}
		}
		t.Fatal("bytecode did not return a value")
	})
}

func TestAaloadAastore(t *testing.T) {
	v := &VM{Stdout: io.Discard}

	t.Run("store and load element", func(t *testing.T) {
		// Create an array, store a value, load it back
		arr := &JArray{Elements: make([]Value, 3)}

		// aload_0, iconst_1, bipush 99, aastore, aload_0, iconst_1, aaload, ireturn
		code := []byte{
			0x2A,       // aload_0 (array ref)
			0x04,       // iconst_1 (index)
			0x10, 0x63, // bipush 99 (value)
			OpAastore,  // aastore
			0x2A,       // aload_0 (array ref)
			0x04,       // iconst_1 (index)
			OpAaload,   // aaload
			0xAC,       // ireturn
		}
		frame := NewFrame(4, 10, code, nil)
		frame.SetLocal(0, RefValue(arr))

		for frame.PC < len(frame.Code) {
			opcode := frame.Code[frame.PC]
			frame.PC++
			retVal, hasReturn, err := v.executeInstruction(frame, opcode)
			if err != nil {
				t.Fatalf("execution error at PC=%d: %v", frame.PC-1, err)
			}
			if hasReturn {
				if retVal.Int != 99 {
					t.Errorf("aaload after aastore: got %d, want 99", retVal.Int)
				}
				return
			}
		}
		t.Fatal("bytecode did not return a value")
	})

	t.Run("store at different indices", func(t *testing.T) {
		arr := &JArray{Elements: make([]Value, 3)}
		arr.Elements[0] = IntValue(10)
		arr.Elements[1] = IntValue(20)
		arr.Elements[2] = IntValue(30)

		// aload_0, iconst_2, aaload, ireturn
		code := []byte{0x2A, 0x05, OpAaload, 0xAC}
		frame := NewFrame(4, 10, code, nil)
		frame.SetLocal(0, RefValue(arr))

		for frame.PC < len(frame.Code) {
			opcode := frame.Code[frame.PC]
			frame.PC++
			retVal, hasReturn, err := v.executeInstruction(frame, opcode)
			if err != nil {
				t.Fatalf("execution error at PC=%d: %v", frame.PC-1, err)
			}
			if hasReturn {
				if retVal.Int != 30 {
					t.Errorf("aaload index 2: got %d, want 30", retVal.Int)
				}
				return
			}
		}
		t.Fatal("bytecode did not return a value")
	})
}

func TestIfAcmpne(t *testing.T) {
	v := &VM{Stdout: io.Discard}

	t.Run("same reference - not taken", func(t *testing.T) {
		obj := &JObject{ClassName: "Test", Fields: make(map[string]Value)}
		// aload_0, aload_0, if_acmpne(offset=5, target=7), iconst_0, ireturn, iconst_1, ireturn
		code := []byte{0x2A, 0x2A, OpIfAcmpne, 0x00, 0x05, 0x03, 0xAC, 0x04, 0xAC}
		frame := NewFrame(4, 10, code, nil)
		frame.SetLocal(0, RefValue(obj))

		for frame.PC < len(frame.Code) {
			opcode := frame.Code[frame.PC]
			frame.PC++
			retVal, hasReturn, err := v.executeInstruction(frame, opcode)
			if err != nil {
				t.Fatalf("execution error at PC=%d: %v", frame.PC-1, err)
			}
			if hasReturn {
				if retVal.Int != 0 {
					t.Errorf("if_acmpne same ref: got %d, want 0 (not taken)", retVal.Int)
				}
				return
			}
		}
		t.Fatal("bytecode did not return a value")
	})

	t.Run("different references - taken", func(t *testing.T) {
		obj1 := &JObject{ClassName: "Test", Fields: make(map[string]Value)}
		obj2 := &JObject{ClassName: "Test", Fields: make(map[string]Value)}
		// aload_0, aload_1, if_acmpne(offset=5, target=7), iconst_0, ireturn, iconst_1, ireturn
		code := []byte{0x2A, 0x2B, OpIfAcmpne, 0x00, 0x05, 0x03, 0xAC, 0x04, 0xAC}
		frame := NewFrame(4, 10, code, nil)
		frame.SetLocal(0, RefValue(obj1))
		frame.SetLocal(1, RefValue(obj2))

		for frame.PC < len(frame.Code) {
			opcode := frame.Code[frame.PC]
			frame.PC++
			retVal, hasReturn, err := v.executeInstruction(frame, opcode)
			if err != nil {
				t.Fatalf("execution error at PC=%d: %v", frame.PC-1, err)
			}
			if hasReturn {
				if retVal.Int != 1 {
					t.Errorf("if_acmpne diff ref: got %d, want 1 (taken)", retVal.Int)
				}
				return
			}
		}
		t.Fatal("bytecode did not return a value")
	})

	t.Run("both null - not taken", func(t *testing.T) {
		// aload_0, aload_1, if_acmpne(offset=5, target=7), iconst_0, ireturn, iconst_1, ireturn
		code := []byte{0x2A, 0x2B, OpIfAcmpne, 0x00, 0x05, 0x03, 0xAC, 0x04, 0xAC}
		frame := NewFrame(4, 10, code, nil)
		frame.SetLocal(0, NullValue())
		frame.SetLocal(1, NullValue())

		for frame.PC < len(frame.Code) {
			opcode := frame.Code[frame.PC]
			frame.PC++
			retVal, hasReturn, err := v.executeInstruction(frame, opcode)
			if err != nil {
				t.Fatalf("execution error at PC=%d: %v", frame.PC-1, err)
			}
			if hasReturn {
				if retVal.Int != 0 {
					t.Errorf("if_acmpne both null: got %d, want 0 (not taken)", retVal.Int)
				}
				return
			}
		}
		t.Fatal("bytecode did not return a value")
	})
}

func TestInstanceof(t *testing.T) {
	v := &VM{Stdout: io.Discard}

	pool := make([]classfile.ConstantPoolEntry, 3)
	pool[1] = &classfile.ConstantClass{NameIndex: 2}
	pool[2] = &classfile.ConstantUtf8{Value: "java/lang/Integer"}
	cf := &classfile.ClassFile{ConstantPool: pool}

	t.Run("matching class", func(t *testing.T) {
		obj := &JObject{ClassName: "java/lang/Integer", Fields: make(map[string]Value)}
		// aload_0, instanceof #1, ireturn
		code := []byte{0x2A, OpInstanceof, 0x00, 0x01, 0xAC}
		frame := NewFrame(4, 10, code, cf)
		frame.SetLocal(0, RefValue(obj))

		for frame.PC < len(frame.Code) {
			opcode := frame.Code[frame.PC]
			frame.PC++
			retVal, hasReturn, err := v.executeInstruction(frame, opcode)
			if err != nil {
				t.Fatalf("execution error at PC=%d: %v", frame.PC-1, err)
			}
			if hasReturn {
				if retVal.Int != 1 {
					t.Errorf("instanceof matching: got %d, want 1", retVal.Int)
				}
				return
			}
		}
		t.Fatal("bytecode did not return a value")
	})

	t.Run("non-matching class", func(t *testing.T) {
		obj := &JObject{ClassName: "java/lang/String", Fields: make(map[string]Value)}
		code := []byte{0x2A, OpInstanceof, 0x00, 0x01, 0xAC}
		frame := NewFrame(4, 10, code, cf)
		frame.SetLocal(0, RefValue(obj))

		for frame.PC < len(frame.Code) {
			opcode := frame.Code[frame.PC]
			frame.PC++
			retVal, hasReturn, err := v.executeInstruction(frame, opcode)
			if err != nil {
				t.Fatalf("execution error at PC=%d: %v", frame.PC-1, err)
			}
			if hasReturn {
				if retVal.Int != 0 {
					t.Errorf("instanceof non-matching: got %d, want 0", retVal.Int)
				}
				return
			}
		}
		t.Fatal("bytecode did not return a value")
	})

	t.Run("null reference", func(t *testing.T) {
		code := []byte{0x2A, OpInstanceof, 0x00, 0x01, 0xAC}
		frame := NewFrame(4, 10, code, cf)
		frame.SetLocal(0, NullValue())

		for frame.PC < len(frame.Code) {
			opcode := frame.Code[frame.PC]
			frame.PC++
			retVal, hasReturn, err := v.executeInstruction(frame, opcode)
			if err != nil {
				t.Fatalf("execution error at PC=%d: %v", frame.PC-1, err)
			}
			if hasReturn {
				if retVal.Int != 0 {
					t.Errorf("instanceof null: got %d, want 0", retVal.Int)
				}
				return
			}
		}
		t.Fatal("bytecode did not return a value")
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
