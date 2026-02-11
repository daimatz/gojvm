package vm

import (
	"fmt"
	"math"

	"github.com/daimatz/gojvm/pkg/classfile"
)

// Opcodes
const (
	OpNop        = 0x00
	OpAconstNull = 0x01
	OpIconstM1   = 0x02
	OpIconst0    = 0x03
	OpIconst1    = 0x04
	OpIconst2    = 0x05
	OpIconst3    = 0x06
	OpIconst4    = 0x07
	OpIconst5    = 0x08
	OpLconst0    = 0x09
	OpLconst1    = 0x0A
	OpFconst0    = 0x0B
	OpFconst1    = 0x0C
	OpFconst2    = 0x0D
	OpBipush     = 0x10
	OpSipush     = 0x11
	OpLdc        = 0x12
	OpLdcW       = 0x13
	OpLdc2W      = 0x14
	OpIload      = 0x15
	OpLload      = 0x16
	OpFload      = 0x17
	OpAload      = 0x19
	OpIload0     = 0x1A
	OpIload1     = 0x1B
	OpIload2     = 0x1C
	OpIload3     = 0x1D
	OpLload0     = 0x1E
	OpLload1     = 0x1F
	OpLload2     = 0x20
	OpLload3     = 0x21
	OpFload0     = 0x22
	OpFload1     = 0x23
	OpFload2     = 0x24
	OpFload3     = 0x25
	OpAload0     = 0x2A
	OpAload1     = 0x2B
	OpAload2     = 0x2C
	OpAload3     = 0x2D
	OpIaload     = 0x2E
	OpAaload     = 0x32
	OpBaload     = 0x33
	OpCaload     = 0x34
	OpIstore     = 0x36
	OpLstore     = 0x37
	OpFstore     = 0x38
	OpAstore     = 0x3A
	OpIstore0    = 0x3B
	OpIstore1    = 0x3C
	OpIstore2    = 0x3D
	OpIstore3    = 0x3E
	OpLstore0    = 0x3F
	OpLstore1    = 0x40
	OpLstore2    = 0x41
	OpLstore3    = 0x42
	OpFstore0    = 0x43
	OpFstore1    = 0x44
	OpFstore2    = 0x45
	OpFstore3    = 0x46
	OpAstore0    = 0x4B
	OpAstore1    = 0x4C
	OpAstore2    = 0x4D
	OpAstore3    = 0x4E
	OpIastore    = 0x4F
	OpAastore    = 0x53
	OpBastore    = 0x54
	OpCastore    = 0x55
	OpPop        = 0x57
	OpDup        = 0x59
	OpDupX1      = 0x5A
	OpDupX2      = 0x5B
	OpSwap       = 0x5F
	OpIadd       = 0x60
	OpLadd       = 0x61
	OpIsub       = 0x64
	OpLsub       = 0x65
	OpImul       = 0x68
	OpFmul       = 0x6A
	OpIdiv       = 0x6C
	OpIrem       = 0x70
	OpIneg       = 0x74
	OpIshl       = 0x78
	OpLshl       = 0x79
	OpIshr       = 0x7A
	OpLshr       = 0x7B
	OpIushr      = 0x7C
	OpLushr      = 0x7D
	OpIand       = 0x7E
	OpLand       = 0x7F
	OpIor        = 0x80
	OpLor        = 0x81
	OpIxor       = 0x82
	OpLxor       = 0x83
	OpIinc       = 0x84
	OpI2l        = 0x85
	OpI2f        = 0x86
	OpL2i        = 0x88
	OpF2i        = 0x8B
	OpI2b        = 0x91
	OpI2s        = 0x93
	OpLcmp       = 0x94
	OpFcmpl      = 0x95
	OpFcmpg      = 0x96
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
	OpIfAcmpeq   = 0xA5
	OpIfAcmpne   = 0xA6
	OpGoto       = 0xA7
	OpTableswitch  = 0xAA
	OpLookupswitch = 0xAB
	OpIreturn    = 0xAC
	OpLreturn    = 0xAD
	OpFreturn    = 0xAE
	OpAreturn    = 0xB0
	OpReturn     = 0xB1
	OpGetstatic  = 0xB2
	OpPutstatic     = 0xB3
	OpGetfield      = 0xB4
	OpPutfield      = 0xB5
	OpInvokevirtual = 0xB6
	OpInvokespecial = 0xB7
	OpInvokestatic  = 0xB8
	OpInvokeinterface = 0xB9
	OpNew           = 0xBB
	OpNewarray      = 0xBC
	OpAnewarray     = 0xBD
	OpArraylength   = 0xBE
	OpAthrow        = 0xBF
	OpCheckcast     = 0xC0
	OpInstanceof    = 0xC1
	OpIfnull        = 0xC6
	OpIfnonnull     = 0xC7
	OpGotoW         = 0xC8
)

// executeInstruction executes a single bytecode instruction.
// Returns (returnValue, hasReturn, error).
func (vm *VM) executeInstruction(frame *Frame, opcode byte) (Value, bool, error) {
	switch opcode {
	case OpNop:
		// do nothing

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

	case OpLconst0:
		frame.Push(LongValue(0))
	case OpLconst1:
		frame.Push(LongValue(1))

	case OpFconst0:
		frame.Push(FloatValue(0.0))
	case OpFconst1:
		frame.Push(FloatValue(1.0))
	case OpFconst2:
		frame.Push(FloatValue(2.0))

	case OpBipush:
		val := frame.ReadI8()
		frame.Push(IntValue(int32(val)))

	case OpSipush:
		val := frame.ReadI16()
		frame.Push(IntValue(int32(val)))

	case OpLdc:
		index := frame.ReadU8()
		return vm.executeLdc(frame, uint16(index))

	case OpLdcW:
		index := frame.ReadU16()
		return vm.executeLdc(frame, index)

	case OpLdc2W:
		index := frame.ReadU16()
		pool := frame.Class.ConstantPool
		if int(index) >= len(pool) || pool[index] == nil {
			return Value{}, false, fmt.Errorf("ldc2_w: invalid constant pool index %d", index)
		}
		switch c := pool[index].(type) {
		case *classfile.ConstantLong:
			frame.Push(LongValue(c.Value))
		case *classfile.ConstantDouble:
			frame.Push(FloatValue(float32(c.Value))) // simplified: treat double as float
		default:
			return Value{}, false, fmt.Errorf("ldc2_w: unsupported type at index %d", index)
		}

	// --- Local variable load instructions ---
	case OpIload:
		index := frame.ReadU8()
		frame.Push(frame.GetLocal(int(index)))
	case OpIload0:
		frame.Push(frame.GetLocal(0))
	case OpIload1:
		frame.Push(frame.GetLocal(1))
	case OpIload2:
		frame.Push(frame.GetLocal(2))
	case OpIload3:
		frame.Push(frame.GetLocal(3))

	case OpLload:
		index := frame.ReadU8()
		frame.Push(frame.GetLocal(int(index)))
	case OpLload0:
		frame.Push(frame.GetLocal(0))
	case OpLload1:
		frame.Push(frame.GetLocal(1))
	case OpLload2:
		frame.Push(frame.GetLocal(2))
	case OpLload3:
		frame.Push(frame.GetLocal(3))

	case OpFload:
		index := frame.ReadU8()
		frame.Push(frame.GetLocal(int(index)))
	case OpFload0:
		frame.Push(frame.GetLocal(0))
	case OpFload1:
		frame.Push(frame.GetLocal(1))
	case OpFload2:
		frame.Push(frame.GetLocal(2))
	case OpFload3:
		frame.Push(frame.GetLocal(3))

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

	// --- Array load ---
	case OpIaload, OpBaload, OpCaload:
		index := frame.Pop().Int
		arrRef := frame.Pop()
		if arrRef.Type == TypeNull || arrRef.Ref == nil {
			return Value{}, false, NewJavaException("java/lang/NullPointerException")
		}
		arr, ok := arrRef.Ref.(*JArray)
		if !ok {
			return Value{}, false, fmt.Errorf("xaload: reference is not an array")
		}
		if index < 0 || int(index) >= len(arr.Elements) {
			return Value{}, false, NewJavaException("java/lang/ArrayIndexOutOfBoundsException")
		}
		frame.Push(arr.Elements[index])

	case OpAaload:
		index := frame.Pop().Int
		arrRef := frame.Pop()
		if arrRef.Type == TypeNull || arrRef.Ref == nil {
			return Value{}, false, NewJavaException("java/lang/NullPointerException")
		}
		arr, ok := arrRef.Ref.(*JArray)
		if !ok {
			return Value{}, false, fmt.Errorf("aaload: reference is not an array")
		}
		if index < 0 || int(index) >= len(arr.Elements) {
			return Value{}, false, NewJavaException("java/lang/ArrayIndexOutOfBoundsException")
		}
		frame.Push(arr.Elements[index])

	// --- Local variable store instructions ---
	case OpIstore:
		index := frame.ReadU8()
		frame.SetLocal(int(index), frame.Pop())
	case OpIstore0:
		frame.SetLocal(0, frame.Pop())
	case OpIstore1:
		frame.SetLocal(1, frame.Pop())
	case OpIstore2:
		frame.SetLocal(2, frame.Pop())
	case OpIstore3:
		frame.SetLocal(3, frame.Pop())

	case OpLstore:
		index := frame.ReadU8()
		frame.SetLocal(int(index), frame.Pop())
	case OpLstore0:
		frame.SetLocal(0, frame.Pop())
	case OpLstore1:
		frame.SetLocal(1, frame.Pop())
	case OpLstore2:
		frame.SetLocal(2, frame.Pop())
	case OpLstore3:
		frame.SetLocal(3, frame.Pop())

	case OpFstore:
		index := frame.ReadU8()
		frame.SetLocal(int(index), frame.Pop())
	case OpFstore0:
		frame.SetLocal(0, frame.Pop())
	case OpFstore1:
		frame.SetLocal(1, frame.Pop())
	case OpFstore2:
		frame.SetLocal(2, frame.Pop())
	case OpFstore3:
		frame.SetLocal(3, frame.Pop())

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

	// --- Array store ---
	case OpIastore, OpBastore, OpCastore:
		value := frame.Pop()
		index := frame.Pop().Int
		arrRef := frame.Pop()
		if arrRef.Type == TypeNull || arrRef.Ref == nil {
			return Value{}, false, NewJavaException("java/lang/NullPointerException")
		}
		arr, ok := arrRef.Ref.(*JArray)
		if !ok {
			return Value{}, false, fmt.Errorf("xastore: reference is not an array")
		}
		if index < 0 || int(index) >= len(arr.Elements) {
			return Value{}, false, NewJavaException("java/lang/ArrayIndexOutOfBoundsException")
		}
		arr.Elements[index] = value

	case OpAastore:
		value := frame.Pop()
		index := frame.Pop().Int
		arrRef := frame.Pop()
		if arrRef.Type == TypeNull || arrRef.Ref == nil {
			return Value{}, false, NewJavaException("java/lang/NullPointerException")
		}
		arr, ok := arrRef.Ref.(*JArray)
		if !ok {
			return Value{}, false, fmt.Errorf("aastore: reference is not an array")
		}
		if index < 0 || int(index) >= len(arr.Elements) {
			return Value{}, false, NewJavaException("java/lang/ArrayIndexOutOfBoundsException")
		}
		arr.Elements[index] = value

	// --- Stack manipulation ---
	case OpPop:
		frame.Pop()

	case OpDup:
		v := frame.Pop()
		frame.Push(v)
		frame.Push(v)

	case OpDupX1:
		v1 := frame.Pop()
		v2 := frame.Pop()
		frame.Push(v1)
		frame.Push(v2)
		frame.Push(v1)

	case OpDupX2:
		v1 := frame.Pop()
		v2 := frame.Pop()
		v3 := frame.Pop()
		frame.Push(v1)
		frame.Push(v3)
		frame.Push(v2)
		frame.Push(v1)

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

	case OpLadd:
		v2 := frame.Pop()
		v1 := frame.Pop()
		frame.Push(LongValue(v1.Long + v2.Long))

	case OpIsub:
		v2 := frame.Pop()
		v1 := frame.Pop()
		frame.Push(IntValue(v1.Int - v2.Int))

	case OpLsub:
		v2 := frame.Pop()
		v1 := frame.Pop()
		frame.Push(LongValue(v1.Long - v2.Long))

	case OpImul:
		v2 := frame.Pop()
		v1 := frame.Pop()
		frame.Push(IntValue(v1.Int * v2.Int))

	case OpFmul:
		v2 := frame.Pop()
		v1 := frame.Pop()
		frame.Push(FloatValue(v1.Float * v2.Float))

	case OpIdiv:
		v2 := frame.Pop()
		v1 := frame.Pop()
		if v2.Int == 0 {
			return Value{}, false, NewJavaException("java/lang/ArithmeticException")
		}
		frame.Push(IntValue(v1.Int / v2.Int))

	case OpIrem:
		v2 := frame.Pop()
		v1 := frame.Pop()
		if v2.Int == 0 {
			return Value{}, false, NewJavaException("java/lang/ArithmeticException")
		}
		frame.Push(IntValue(v1.Int % v2.Int))

	case OpIneg:
		v := frame.Pop()
		frame.Push(IntValue(-v.Int))

	// --- Bit operations ---
	case OpIshl:
		v2 := frame.Pop()
		v1 := frame.Pop()
		frame.Push(IntValue(v1.Int << (uint(v2.Int) & 0x1f)))

	case OpLshl:
		v2 := frame.Pop()
		v1 := frame.Pop()
		frame.Push(LongValue(v1.Long << (uint(v2.Int) & 0x3f)))

	case OpIshr:
		v2 := frame.Pop()
		v1 := frame.Pop()
		frame.Push(IntValue(v1.Int >> (uint(v2.Int) & 0x1f)))

	case OpLshr:
		v2 := frame.Pop()
		v1 := frame.Pop()
		frame.Push(LongValue(v1.Long >> (uint(v2.Int) & 0x3f)))

	case OpIushr:
		v2 := frame.Pop()
		v1 := frame.Pop()
		frame.Push(IntValue(int32(uint32(v1.Int) >> (uint(v2.Int) & 0x1f))))

	case OpLushr:
		v2 := frame.Pop()
		v1 := frame.Pop()
		frame.Push(LongValue(int64(uint64(v1.Long) >> (uint(v2.Int) & 0x3f))))

	case OpIand:
		v2 := frame.Pop()
		v1 := frame.Pop()
		frame.Push(IntValue(v1.Int & v2.Int))

	case OpLand:
		v2 := frame.Pop()
		v1 := frame.Pop()
		frame.Push(LongValue(v1.Long & v2.Long))

	case OpIor:
		v2 := frame.Pop()
		v1 := frame.Pop()
		frame.Push(IntValue(v1.Int | v2.Int))

	case OpLor:
		v2 := frame.Pop()
		v1 := frame.Pop()
		frame.Push(LongValue(v1.Long | v2.Long))

	case OpIxor:
		v2 := frame.Pop()
		v1 := frame.Pop()
		frame.Push(IntValue(v1.Int ^ v2.Int))

	case OpLxor:
		v2 := frame.Pop()
		v1 := frame.Pop()
		frame.Push(LongValue(v1.Long ^ v2.Long))

	case OpIinc:
		index := frame.ReadU8()
		constVal := frame.ReadI8()
		local := frame.GetLocal(int(index))
		frame.SetLocal(int(index), IntValue(local.Int+int32(constVal)))

	// --- Type conversions ---
	case OpI2l:
		v := frame.Pop()
		frame.Push(LongValue(int64(v.Int)))

	case OpI2f:
		v := frame.Pop()
		frame.Push(FloatValue(float32(v.Int)))

	case OpL2i:
		v := frame.Pop()
		frame.Push(IntValue(int32(v.Long)))

	case OpF2i:
		v := frame.Pop()
		if math.IsNaN(float64(v.Float)) {
			frame.Push(IntValue(0))
		} else {
			frame.Push(IntValue(int32(v.Float)))
		}

	case OpI2b:
		v := frame.Pop()
		frame.Push(IntValue(int32(int8(v.Int))))

	case OpI2s:
		v := frame.Pop()
		frame.Push(IntValue(int32(int16(v.Int))))

	// --- Comparisons ---
	case OpLcmp:
		v2 := frame.Pop()
		v1 := frame.Pop()
		if v1.Long > v2.Long {
			frame.Push(IntValue(1))
		} else if v1.Long < v2.Long {
			frame.Push(IntValue(-1))
		} else {
			frame.Push(IntValue(0))
		}

	case OpFcmpl:
		v2 := frame.Pop()
		v1 := frame.Pop()
		if math.IsNaN(float64(v1.Float)) || math.IsNaN(float64(v2.Float)) {
			frame.Push(IntValue(-1))
		} else if v1.Float > v2.Float {
			frame.Push(IntValue(1))
		} else if v1.Float < v2.Float {
			frame.Push(IntValue(-1))
		} else {
			frame.Push(IntValue(0))
		}

	case OpFcmpg:
		v2 := frame.Pop()
		v1 := frame.Pop()
		if math.IsNaN(float64(v1.Float)) || math.IsNaN(float64(v2.Float)) {
			frame.Push(IntValue(1))
		} else if v1.Float > v2.Float {
			frame.Push(IntValue(1))
		} else if v1.Float < v2.Float {
			frame.Push(IntValue(-1))
		} else {
			frame.Push(IntValue(0))
		}

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

	case OpIfAcmpeq:
		branchPC := frame.PC - 1
		offset := frame.ReadI16()
		v2 := frame.Pop()
		v1 := frame.Pop()
		eq := (v1.Type == TypeNull && v2.Type == TypeNull) ||
			(v1.Type == v2.Type && v1.Ref == v2.Ref)
		if eq {
			frame.PC = branchPC + int(offset)
		}

	case OpIfAcmpne:
		branchPC := frame.PC - 1
		offset := frame.ReadI16()
		v2 := frame.Pop()
		v1 := frame.Pop()
		eq := (v1.Type == TypeNull && v2.Type == TypeNull) ||
			(v1.Type == v2.Type && v1.Ref == v2.Ref)
		if !eq {
			frame.PC = branchPC + int(offset)
		}

	case OpGoto:
		branchPC := frame.PC - 1
		offset := frame.ReadI16()
		frame.PC = branchPC + int(offset)

	case OpGotoW:
		branchPC := frame.PC - 1
		offset := frame.ReadI32()
		frame.PC = branchPC + int(offset)

	case OpTableswitch:
		// PC of the tableswitch opcode
		opcodePC := frame.PC - 1
		// Padding to align to 4-byte boundary
		for frame.PC%4 != 0 {
			frame.PC++
		}
		defaultOffset := frame.ReadI32()
		low := frame.ReadI32()
		high := frame.ReadI32()
		numOffsets := int(high - low + 1)
		offsets := make([]int32, numOffsets)
		for i := 0; i < numOffsets; i++ {
			offsets[i] = frame.ReadI32()
		}
		index := frame.Pop().Int
		if index >= low && index <= high {
			frame.PC = opcodePC + int(offsets[index-low])
		} else {
			frame.PC = opcodePC + int(defaultOffset)
		}

	case OpLookupswitch:
		opcodePC := frame.PC - 1
		for frame.PC%4 != 0 {
			frame.PC++
		}
		defaultOffset := frame.ReadI32()
		npairs := frame.ReadI32()
		key := frame.Pop().Int
		matched := false
		for i := int32(0); i < npairs; i++ {
			matchVal := frame.ReadI32()
			offset := frame.ReadI32()
			if key == matchVal {
				frame.PC = opcodePC + int(offset)
				matched = true
				// skip remaining pairs
				for j := i + 1; j < npairs; j++ {
					frame.ReadI32()
					frame.ReadI32()
				}
				break
			}
		}
		if !matched {
			frame.PC = opcodePC + int(defaultOffset)
		}

	// --- Return ---
	case OpIreturn, OpFreturn, OpAreturn, OpLreturn:
		return frame.Pop(), true, nil

	case OpReturn:
		return Value{}, true, nil

	// --- Method invocation and field access ---
	case OpGetstatic:
		return vm.executeGetstatic(frame)

	case OpPutstatic:
		return vm.executePutstatic(frame)

	case OpGetfield:
		return vm.executeGetfield(frame)

	case OpPutfield:
		return vm.executePutfield(frame)

	case OpInvokevirtual:
		return vm.executeInvokevirtual(frame)

	case OpInvokespecial:
		return vm.executeInvokespecial(frame)

	case OpInvokestatic:
		return vm.executeInvokestatic(frame)

	case OpInvokeinterface:
		return vm.executeInvokeinterface(frame)

	case OpNew:
		return vm.executeNew(frame)

	case OpNewarray:
		atype := frame.ReadU8()
		count := frame.Pop().Int
		if count < 0 {
			return Value{}, false, NewJavaException("java/lang/NegativeArraySizeException")
		}
		elements := make([]Value, count)
		_ = atype // type doesn't matter for initialization, all zero
		for i := range elements {
			elements[i] = IntValue(0)
		}
		arr := &JArray{Elements: elements}
		frame.Push(RefValue(arr))

	case OpAnewarray:
		_ = frame.ReadU16() // CP index for element type
		count := frame.Pop().Int
		if count < 0 {
			return Value{}, false, NewJavaException("java/lang/NegativeArraySizeException")
		}
		elements := make([]Value, count)
		for i := range elements {
			elements[i] = NullValue()
		}
		arr := &JArray{Elements: elements}
		frame.Push(RefValue(arr))

	case OpArraylength:
		arrRef := frame.Pop()
		if arrRef.Type == TypeNull || arrRef.Ref == nil {
			return Value{}, false, NewJavaException("java/lang/NullPointerException")
		}
		arr, ok := arrRef.Ref.(*JArray)
		if !ok {
			return Value{}, false, fmt.Errorf("arraylength: reference is not an array")
		}
		frame.Push(IntValue(int32(len(arr.Elements))))

	case OpAthrow:
		excRef := frame.Pop()
		if excRef.Type == TypeNull {
			return Value{}, false, NewJavaException("java/lang/NullPointerException")
		}
		if obj, ok := excRef.Ref.(*JObject); ok {
			return Value{}, false, &JavaException{Object: obj}
		}
		return Value{}, false, fmt.Errorf("athrow: non-object on stack")

	case OpCheckcast:
		index := frame.ReadU16()
		pool := frame.Class.ConstantPool
		className, err := classfile.GetClassName(pool, index)
		if err != nil {
			return Value{}, false, fmt.Errorf("checkcast: %w", err)
		}
		val := frame.Peek()
		if val.Type != TypeNull {
			if obj, ok := val.Ref.(*JObject); ok {
				if !vm.isInstanceOf(obj.ClassName, className) {
					return Value{}, false, NewJavaException("java/lang/ClassCastException")
				}
			}
		}

	case OpInstanceof:
		index := frame.ReadU16()
		pool := frame.Class.ConstantPool
		className, err := classfile.GetClassName(pool, index)
		if err != nil {
			return Value{}, false, fmt.Errorf("instanceof: %w", err)
		}
		ref := frame.Pop()
		if ref.Type == TypeNull {
			frame.Push(IntValue(0))
		} else if obj, ok := ref.Ref.(*JObject); ok && vm.isInstanceOf(obj.ClassName, className) {
			frame.Push(IntValue(1))
		} else {
			frame.Push(IntValue(0))
		}

	case OpIfnull:
		branchPC := frame.PC - 1
		offset := frame.ReadI16()
		val := frame.Pop()
		if val.Type == TypeNull || (val.Type == TypeRef && val.Ref == nil) {
			frame.PC = branchPC + int(offset)
		}

	case OpIfnonnull:
		branchPC := frame.PC - 1
		offset := frame.ReadI16()
		val := frame.Pop()
		isNull := val.Type == TypeNull || (val.Type == TypeRef && val.Ref == nil)
		if !isNull {
			frame.PC = branchPC + int(offset)
		}

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
