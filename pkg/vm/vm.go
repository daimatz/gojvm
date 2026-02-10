package vm

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/daimatz/gojvm/pkg/classfile"
	"github.com/daimatz/gojvm/pkg/native"
)

// maxFrameDepth is the maximum number of nested method calls.
const maxFrameDepth = 1024

// VM is the virtual machine that executes Java bytecode.
type VM struct {
	ClassFile  *classfile.ClassFile
	Stdout     io.Writer
	frameDepth int
}

// NewVM creates a new VM with the given class file.
func NewVM(cf *classfile.ClassFile) *VM {
	return &VM{
		ClassFile: cf,
		Stdout:    os.Stdout,
	}
}

// Execute finds and executes the main method of the class.
func (vm *VM) Execute() error {
	method := vm.ClassFile.FindMethod("main", "([Ljava/lang/String;)V")
	if method == nil {
		return fmt.Errorf("main method not found")
	}
	if method.Code == nil {
		return fmt.Errorf("main method has no Code attribute")
	}

	// main(String[] args) — pass null for args
	args := []Value{NullValue()}
	_, err := vm.executeMethod(method, args)
	return err
}

// executeMethod executes a method with the given arguments and returns its return value.
func (vm *VM) executeMethod(method *classfile.MethodInfo, args []Value) (Value, error) {
	if method.Code == nil {
		return Value{}, fmt.Errorf("method %s has no Code attribute", method.Name)
	}

	vm.frameDepth++
	if vm.frameDepth > maxFrameDepth {
		return Value{}, fmt.Errorf("stack overflow: frame depth exceeded %d", maxFrameDepth)
	}
	defer func() { vm.frameDepth-- }()

	frame := NewFrame(method.Code.MaxLocals, method.Code.MaxStack, method.Code.Code, vm.ClassFile)

	// Set arguments into local variables
	for i, arg := range args {
		frame.SetLocal(i, arg)
	}

	// Execution loop
	for frame.PC < len(frame.Code) {
		opcode := frame.Code[frame.PC]
		frame.PC++

		retVal, hasReturn, err := vm.executeInstruction(frame, opcode)
		if err != nil {
			return Value{}, err
		}
		if hasReturn {
			return retVal, nil
		}
	}

	// Fell off the end of the method (implicit return for void methods)
	return Value{}, nil
}

// executeLdc handles the ldc instruction.
func (vm *VM) executeLdc(frame *Frame, index uint16) (Value, bool, error) {
	pool := frame.Class.ConstantPool
	if int(index) >= len(pool) || pool[index] == nil {
		return Value{}, false, fmt.Errorf("ldc: invalid constant pool index %d", index)
	}

	entry := pool[index]
	switch c := entry.(type) {
	case *classfile.ConstantInteger:
		frame.Push(IntValue(c.Value))
	case *classfile.ConstantString:
		str, err := classfile.GetUtf8(pool, c.StringIndex)
		if err != nil {
			return Value{}, false, fmt.Errorf("ldc: resolving string: %w", err)
		}
		frame.Push(RefValue(str))
	default:
		return Value{}, false, fmt.Errorf("ldc: unsupported constant pool entry type at index %d (tag=%d)", index, entry.Tag())
	}

	return Value{}, false, nil
}

// executeGetstatic handles the getstatic instruction.
func (vm *VM) executeGetstatic(frame *Frame) (Value, bool, error) {
	index := frame.ReadU16()
	pool := frame.Class.ConstantPool

	fieldRef, err := classfile.ResolveFieldref(pool, index)
	if err != nil {
		return Value{}, false, fmt.Errorf("getstatic: %w", err)
	}

	// Handle java/lang/System.out
	if fieldRef.ClassName == "java/lang/System" && fieldRef.FieldName == "out" {
		frame.Push(RefValue(&native.PrintStream{Writer: vm.Stdout}))
		return Value{}, false, nil
	}

	return Value{}, false, fmt.Errorf("getstatic: unsupported field %s.%s:%s", fieldRef.ClassName, fieldRef.FieldName, fieldRef.Descriptor)
}

// executeGetfield handles the getfield instruction.
func (vm *VM) executeGetfield(frame *Frame) (Value, bool, error) {
	index := frame.ReadU16()
	pool := frame.Class.ConstantPool

	fieldRef, err := classfile.ResolveFieldref(pool, index)
	if err != nil {
		return Value{}, false, fmt.Errorf("getfield: %w", err)
	}

	objectRef := frame.Pop()
	if objectRef.Type == TypeNull || objectRef.Ref == nil {
		return Value{}, false, fmt.Errorf("getfield: NullPointerException")
	}
	obj, ok := objectRef.Ref.(*JObject)
	if !ok {
		return Value{}, false, fmt.Errorf("getfield: receiver is not a JObject")
	}

	val, exists := obj.Fields[fieldRef.FieldName]
	if !exists {
		frame.Push(NullValue())
	} else {
		frame.Push(val)
	}
	return Value{}, false, nil
}

// executePutfield handles the putfield instruction.
func (vm *VM) executePutfield(frame *Frame) (Value, bool, error) {
	index := frame.ReadU16()
	pool := frame.Class.ConstantPool

	fieldRef, err := classfile.ResolveFieldref(pool, index)
	if err != nil {
		return Value{}, false, fmt.Errorf("putfield: %w", err)
	}

	value := frame.Pop()
	objectRef := frame.Pop()
	if objectRef.Type == TypeNull || objectRef.Ref == nil {
		return Value{}, false, fmt.Errorf("putfield: NullPointerException")
	}
	obj, ok := objectRef.Ref.(*JObject)
	if !ok {
		return Value{}, false, fmt.Errorf("putfield: receiver is not a JObject")
	}

	obj.Fields[fieldRef.FieldName] = value
	return Value{}, false, nil
}

// executeInvokevirtual handles the invokevirtual instruction.
func (vm *VM) executeInvokevirtual(frame *Frame) (Value, bool, error) {
	index := frame.ReadU16()
	pool := frame.Class.ConstantPool

	methodRef, err := classfile.ResolveMethodref(pool, index)
	if err != nil {
		return Value{}, false, fmt.Errorf("invokevirtual: %w", err)
	}

	paramCount, err := countParams(methodRef.Descriptor)
	if err != nil {
		return Value{}, false, fmt.Errorf("invokevirtual: %w", err)
	}

	args := make([]Value, paramCount)
	for i := paramCount - 1; i >= 0; i-- {
		args[i] = frame.Pop()
	}
	objectRef := frame.Pop()

	// PrintStream.println
	if methodRef.ClassName == "java/io/PrintStream" && methodRef.MethodName == "println" {
		ps, ok := objectRef.Ref.(*native.PrintStream)
		if !ok {
			return Value{}, false, fmt.Errorf("invokevirtual: println receiver is not a PrintStream")
		}
		switch methodRef.Descriptor {
		case "(I)V":
			ps.Println(args[0].Int)
		case "(Ljava/lang/String;)V":
			ps.Println(args[0].Ref)
		case "()V":
			ps.Println()
		default:
			return Value{}, false, fmt.Errorf("invokevirtual: unsupported println descriptor %s", methodRef.Descriptor)
		}
		return Value{}, false, nil
	}

	// HashMap.get
	if methodRef.ClassName == "java/util/HashMap" && methodRef.MethodName == "get" {
		hm, ok := objectRef.Ref.(*native.NativeHashMap)
		if !ok {
			return Value{}, false, fmt.Errorf("invokevirtual: HashMap.get receiver is not a NativeHashMap")
		}
		result := hm.Get(args[0].Ref)
		if result == nil {
			frame.Push(NullValue())
		} else {
			frame.Push(RefValue(result))
		}
		return Value{}, false, nil
	}

	// HashMap.put
	if methodRef.ClassName == "java/util/HashMap" && methodRef.MethodName == "put" {
		hm, ok := objectRef.Ref.(*native.NativeHashMap)
		if !ok {
			return Value{}, false, fmt.Errorf("invokevirtual: HashMap.put receiver is not a NativeHashMap")
		}
		old := hm.Put(args[0].Ref, args[1].Ref)
		if old == nil {
			frame.Push(NullValue())
		} else {
			frame.Push(RefValue(old))
		}
		return Value{}, false, nil
	}

	// Integer.intValue
	if methodRef.ClassName == "java/lang/Integer" && methodRef.MethodName == "intValue" {
		ni, ok := objectRef.Ref.(*native.NativeInteger)
		if !ok {
			return Value{}, false, fmt.Errorf("invokevirtual: Integer.intValue receiver is not a NativeInteger")
		}
		frame.Push(IntValue(ni.Value))
		return Value{}, false, nil
	}

	// User-defined method (e.g., Fib.fib)
	method := frame.Class.FindMethod(methodRef.MethodName, methodRef.Descriptor)
	if method != nil {
		fullArgs := make([]Value, 0, len(args)+1)
		fullArgs = append(fullArgs, objectRef)
		fullArgs = append(fullArgs, args...)
		retVal, err := vm.executeMethod(method, fullArgs)
		if err != nil {
			return Value{}, false, err
		}
		if !isVoidReturn(methodRef.Descriptor) {
			frame.Push(retVal)
		}
		return Value{}, false, nil
	}

	return Value{}, false, fmt.Errorf("invokevirtual: unsupported method %s.%s:%s", methodRef.ClassName, methodRef.MethodName, methodRef.Descriptor)
}

// executeInvokespecial handles the invokespecial instruction.
func (vm *VM) executeInvokespecial(frame *Frame) (Value, bool, error) {
	index := frame.ReadU16()
	pool := frame.Class.ConstantPool

	methodRef, err := classfile.ResolveMethodref(pool, index)
	if err != nil {
		return Value{}, false, fmt.Errorf("invokespecial: %w", err)
	}

	paramCount, err := countParams(methodRef.Descriptor)
	if err != nil {
		return Value{}, false, fmt.Errorf("invokespecial: %w", err)
	}

	args := make([]Value, paramCount)
	for i := paramCount - 1; i >= 0; i-- {
		args[i] = frame.Pop()
	}
	objectRef := frame.Pop() // this

	switch {
	case methodRef.ClassName == "java/lang/Object" && methodRef.MethodName == "<init>":
		// no-op
	case methodRef.ClassName == "java/util/HashMap" && methodRef.MethodName == "<init>":
		// HashMap already initialized in new, no-op
	default:
		// User class constructor
		method := frame.Class.FindMethod(methodRef.MethodName, methodRef.Descriptor)
		if method == nil {
			return Value{}, false, fmt.Errorf("invokespecial: method %s:%s not found", methodRef.MethodName, methodRef.Descriptor)
		}
		fullArgs := make([]Value, 0, len(args)+1)
		fullArgs = append(fullArgs, objectRef)
		fullArgs = append(fullArgs, args...)
		_, err = vm.executeMethod(method, fullArgs)
		if err != nil {
			return Value{}, false, err
		}
	}
	return Value{}, false, nil
}

// executeInvokestatic handles the invokestatic instruction.
func (vm *VM) executeInvokestatic(frame *Frame) (Value, bool, error) {
	index := frame.ReadU16()
	pool := frame.Class.ConstantPool

	methodRef, err := classfile.ResolveMethodref(pool, index)
	if err != nil {
		return Value{}, false, fmt.Errorf("invokestatic: %w", err)
	}

	// Native static methods
	if methodRef.ClassName == "java/lang/Integer" && methodRef.MethodName == "valueOf" {
		intVal := frame.Pop()
		frame.Push(RefValue(native.IntegerValueOf(intVal.Int)))
		return Value{}, false, nil
	}

	// Find the method in the current class
	method := frame.Class.FindMethod(methodRef.MethodName, methodRef.Descriptor)
	if method == nil {
		return Value{}, false, fmt.Errorf("invokestatic: method %s:%s not found in class %s", methodRef.MethodName, methodRef.Descriptor, methodRef.ClassName)
	}

	// Count parameters from descriptor
	paramCount, err := countParams(methodRef.Descriptor)
	if err != nil {
		return Value{}, false, fmt.Errorf("invokestatic: %w", err)
	}

	// Pop arguments from stack (in reverse order)
	args := make([]Value, paramCount)
	for i := paramCount - 1; i >= 0; i-- {
		args[i] = frame.Pop()
	}

	// Execute the method
	retVal, err := vm.executeMethod(method, args)
	if err != nil {
		return Value{}, false, err
	}

	// Push return value if the method returns something
	if !isVoidReturn(methodRef.Descriptor) {
		frame.Push(retVal)
	}

	return Value{}, false, nil
}

// executeNew handles the new instruction.
func (vm *VM) executeNew(frame *Frame) (Value, bool, error) {
	index := frame.ReadU16()
	pool := frame.Class.ConstantPool

	className, err := classfile.GetClassName(pool, index)
	if err != nil {
		return Value{}, false, fmt.Errorf("new: %w", err)
	}

	switch className {
	case "java/util/HashMap":
		frame.Push(RefValue(native.NewNativeHashMap()))
	default:
		obj := &JObject{ClassName: className, Fields: make(map[string]Value)}
		frame.Push(RefValue(obj))
	}
	return Value{}, false, nil
}

// countParams counts the number of parameters in a method descriptor.
func countParams(descriptor string) (int, error) {
	// Parse between ( and )
	start := strings.Index(descriptor, "(")
	end := strings.Index(descriptor, ")")
	if start == -1 || end == -1 {
		return 0, fmt.Errorf("invalid method descriptor: %s", descriptor)
	}

	params := descriptor[start+1 : end]
	count := 0
	i := 0
	for i < len(params) {
		switch params[i] {
		case 'B', 'C', 'D', 'F', 'I', 'J', 'S', 'Z':
			count++
			i++
		case 'L':
			count++
			// Skip until ';'
			for i < len(params) && params[i] != ';' {
				i++
			}
			i++ // skip ';'
		case '[':
			// Array — skip dimensions, then count the element type
			for i < len(params) && params[i] == '[' {
				i++
			}
			if i < len(params) && params[i] == 'L' {
				for i < len(params) && params[i] != ';' {
					i++
				}
				i++ // skip ';'
			} else if i < len(params) {
				i++ // primitive type
			}
			count++
		default:
			return 0, fmt.Errorf("invalid type descriptor char '%c' in %s", params[i], descriptor)
		}
	}
	return count, nil
}

// isVoidReturn checks if a method descriptor has void return type.
func isVoidReturn(descriptor string) bool {
	return strings.HasSuffix(descriptor, ")V")
}
