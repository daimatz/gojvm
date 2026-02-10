package vm

import "fmt"

// Opcodes
const (
	OpAconstNull = 0x01
	OpIconstM1   = 0x02
	OpIconst0    = 0x03
	OpIconst1    = 0x04
	OpIconst2    = 0x05
	OpIconst3    = 0x06
	OpIconst4    = 0x07
	OpIconst5    = 0x08
	OpBipush     = 0x10
	OpSipush     = 0x11
	OpLdc        = 0x12
	OpIload      = 0x15
	OpAload      = 0x19
	OpIload0     = 0x1A
	OpIload1     = 0x1B
	OpIload2     = 0x1C
	OpIload3     = 0x1D
	OpAload0     = 0x2A
	OpAload1     = 0x2B
	OpAload2     = 0x2C
	OpAload3     = 0x2D
	OpIstore     = 0x36
	OpAstore     = 0x3A
	OpIstore0    = 0x3B
	OpIstore1    = 0x3C
	OpIstore2    = 0x3D
	OpIstore3    = 0x3E
	OpAstore0    = 0x4B
	OpAstore1    = 0x4C
	OpAstore2    = 0x4D
	OpAstore3    = 0x4E
	OpPop        = 0x57
	OpDup        = 0x59
	OpSwap       = 0x5F
	OpIadd       = 0x60
	OpIsub       = 0x64
	OpImul       = 0x68
	OpIdiv       = 0x6C
	OpIrem       = 0x70
	OpIneg       = 0x74
	OpIfeq       = 0x99
	OpIfne       = 0x9A
	OpIflt       = 0x9B
	OpIfge       = 0x9C
	OpIfgt       = 0x9D
	OpIfle       = 0x9E
	OpIfIcmpeq   = 0x9F
	OpIfIcmpne   = 0xA0
	OpIfIcmplt   = 0xA1
	OpIfIcmpge   = 0xA2
	OpIfIcmpgt   = 0xA3
	OpIfIcmple   = 0xA4
	OpGoto       = 0xA7
	OpIreturn    = 0xAC
	OpReturn     = 0xB1
	OpGetstatic  = 0xB2
	OpInvokevirtual = 0xB6
	OpInvokestatic  = 0xB8
	OpNew        = 0xBB
)

// executeInstruction executes a single bytecode instruction.
// Returns (returnValue, hasReturn, error).
func (vm *VM) executeInstruction(frame *Frame, opcode byte) (Value, bool, error) {
	switch opcode {
	// --- Constant load instructions ---
	case OpAconstNull:
		frame.Push(NullValue())

	case OpIconstM1:
		frame.Push(IntValue(-1))
	case OpIconst0:
		frame.Push(IntValue(0))
	case OpIconst1:
		frame.Push(IntValue(1))
	case OpIconst2:
		frame.Push(IntValue(2))
	case OpIconst3:
		frame.Push(IntValue(3))
	case OpIconst4:
		frame.Push(IntValue(4))
	case OpIconst5:
		frame.Push(IntValue(5))

	case OpBipush:
		val := frame.ReadI8()
		frame.Push(IntValue(int32(val)))

	case OpSipush:
		val := frame.ReadI16()
		frame.Push(IntValue(int32(val)))

	case OpLdc:
		index := frame.ReadU8()
		return vm.executeLdc(frame, uint16(index))

	// --- Local variable load instructions ---
	case OpIload:
		index := frame.ReadU8()
		frame.Push(IntValue(frame.GetLocal(int(index)).Int))
	case OpIload0:
		frame.Push(IntValue(frame.GetLocal(0).Int))
	case OpIload1:
		frame.Push(IntValue(frame.GetLocal(1).Int))
	case OpIload2:
		frame.Push(IntValue(frame.GetLocal(2).Int))
	case OpIload3:
		frame.Push(IntValue(frame.GetLocal(3).Int))

	case OpAload:
		index := frame.ReadU8()
		frame.Push(frame.GetLocal(int(index)))
	case OpAload0:
		frame.Push(frame.GetLocal(0))
	case OpAload1:
		frame.Push(frame.GetLocal(1))
	case OpAload2:
		frame.Push(frame.GetLocal(2))
	case OpAload3:
		frame.Push(frame.GetLocal(3))

	// --- Local variable store instructions ---
	case OpIstore:
		index := frame.ReadU8()
		val := frame.Pop()
		frame.SetLocal(int(index), IntValue(val.Int))
	case OpIstore0:
		frame.SetLocal(0, IntValue(frame.Pop().Int))
	case OpIstore1:
		frame.SetLocal(1, IntValue(frame.Pop().Int))
	case OpIstore2:
		frame.SetLocal(2, IntValue(frame.Pop().Int))
	case OpIstore3:
		frame.SetLocal(3, IntValue(frame.Pop().Int))

	case OpAstore:
		index := frame.ReadU8()
		frame.SetLocal(int(index), frame.Pop())
	case OpAstore0:
		frame.SetLocal(0, frame.Pop())
	case OpAstore1:
		frame.SetLocal(1, frame.Pop())
	case OpAstore2:
		frame.SetLocal(2, frame.Pop())
	case OpAstore3:
		frame.SetLocal(3, frame.Pop())

	// --- Stack manipulation ---
	case OpPop:
		frame.Pop()

	case OpDup:
		v := frame.Pop()
		frame.Push(v)
		frame.Push(v)

	case OpSwap:
		v2 := frame.Pop()
		v1 := frame.Pop()
		frame.Push(v2)
		frame.Push(v1)

	// --- Arithmetic ---
	case OpIadd:
		v2 := frame.Pop()
		v1 := frame.Pop()
		frame.Push(IntValue(v1.Int + v2.Int))

	case OpIsub:
		v2 := frame.Pop()
		v1 := frame.Pop()
		frame.Push(IntValue(v1.Int - v2.Int))

	case OpImul:
		v2 := frame.Pop()
		v1 := frame.Pop()
		frame.Push(IntValue(v1.Int * v2.Int))

	case OpIdiv:
		v2 := frame.Pop()
		v1 := frame.Pop()
		if v2.Int == 0 {
			return Value{}, false, fmt.Errorf("ArithmeticException: / by zero")
		}
		frame.Push(IntValue(v1.Int / v2.Int))

	case OpIrem:
		v2 := frame.Pop()
		v1 := frame.Pop()
		if v2.Int == 0 {
			return Value{}, false, fmt.Errorf("ArithmeticException: / by zero")
		}
		frame.Push(IntValue(v1.Int % v2.Int))

	case OpIneg:
		v := frame.Pop()
		frame.Push(IntValue(-v.Int))

	// --- Comparison and branch ---
	case OpIfeq:
		return vm.executeBranchUnary(frame, func(v int32) bool { return v == 0 })
	case OpIfne:
		return vm.executeBranchUnary(frame, func(v int32) bool { return v != 0 })
	case OpIflt:
		return vm.executeBranchUnary(frame, func(v int32) bool { return v < 0 })
	case OpIfge:
		return vm.executeBranchUnary(frame, func(v int32) bool { return v >= 0 })
	case OpIfgt:
		return vm.executeBranchUnary(frame, func(v int32) bool { return v > 0 })
	case OpIfle:
		return vm.executeBranchUnary(frame, func(v int32) bool { return v <= 0 })

	case OpIfIcmpeq:
		return vm.executeBranchBinary(frame, func(v1, v2 int32) bool { return v1 == v2 })
	case OpIfIcmpne:
		return vm.executeBranchBinary(frame, func(v1, v2 int32) bool { return v1 != v2 })
	case OpIfIcmplt:
		return vm.executeBranchBinary(frame, func(v1, v2 int32) bool { return v1 < v2 })
	case OpIfIcmpge:
		return vm.executeBranchBinary(frame, func(v1, v2 int32) bool { return v1 >= v2 })
	case OpIfIcmpgt:
		return vm.executeBranchBinary(frame, func(v1, v2 int32) bool { return v1 > v2 })
	case OpIfIcmple:
		return vm.executeBranchBinary(frame, func(v1, v2 int32) bool { return v1 <= v2 })

	case OpGoto:
		// PC is at the opcode position + 1 (we already consumed the opcode)
		// branchPC = PC of the goto instruction = frame.PC - 1
		branchPC := frame.PC - 1
		offset := frame.ReadI16()
		frame.PC = branchPC + int(offset)

	// --- Return ---
	case OpIreturn:
		return frame.Pop(), true, nil

	case OpReturn:
		return Value{}, true, nil

	// --- Method invocation and field access ---
	case OpGetstatic:
		return vm.executeGetstatic(frame)

	case OpInvokevirtual:
		return vm.executeInvokevirtual(frame)

	case OpInvokestatic:
		return vm.executeInvokestatic(frame)

	case OpNew:
		return vm.executeNew(frame)

	default:
		return Value{}, false, fmt.Errorf("unknown opcode: 0x%02X at PC=%d", opcode, frame.PC-1)
	}

	return Value{}, false, nil
}

// executeBranchUnary handles unary branch instructions (ifeq, ifne, etc.)
func (vm *VM) executeBranchUnary(frame *Frame, cond func(int32) bool) (Value, bool, error) {
	branchPC := frame.PC - 1 // PC of the branch instruction
	offset := frame.ReadI16()
	val := frame.Pop()
	if cond(val.Int) {
		frame.PC = branchPC + int(offset)
	}
	return Value{}, false, nil
}

// executeBranchBinary handles binary branch instructions (if_icmpeq, etc.)
func (vm *VM) executeBranchBinary(frame *Frame, cond func(int32, int32) bool) (Value, bool, error) {
	branchPC := frame.PC - 1 // PC of the branch instruction
	offset := frame.ReadI16()
	v2 := frame.Pop()
	v1 := frame.Pop()
	if cond(v1.Int, v2.Int) {
		frame.PC = branchPC + int(offset)
	}
	return Value{}, false, nil
}
