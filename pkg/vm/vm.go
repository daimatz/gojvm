package vm

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/daimatz/gojvm/pkg/classfile"
	"github.com/daimatz/gojvm/pkg/native"
)

// VM is the virtual machine that executes Java bytecode.
type VM struct {
	ClassFile *classfile.ClassFile
	Stdout    io.Writer
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

// executeInvokevirtual handles the invokevirtual instruction.
func (vm *VM) executeInvokevirtual(frame *Frame) (Value, bool, error) {
	index := frame.ReadU16()
	pool := frame.Class.ConstantPool

	methodRef, err := classfile.ResolveMethodref(pool, index)
	if err != nil {
		return Value{}, false, fmt.Errorf("invokevirtual: %w", err)
	}

	// Handle PrintStream.println
	if methodRef.ClassName == "java/io/PrintStream" && methodRef.MethodName == "println" {
		return vm.invokePrintln(frame, methodRef.Descriptor)
	}

	return Value{}, false, fmt.Errorf("invokevirtual: unsupported method %s.%s:%s", methodRef.ClassName, methodRef.MethodName, methodRef.Descriptor)
}

// invokePrintln handles PrintStream.println with various descriptors.
func (vm *VM) invokePrintln(frame *Frame, descriptor string) (Value, bool, error) {
	switch descriptor {
	case "(I)V":
		arg := frame.Pop()
		objectRef := frame.Pop()
		ps, ok := objectRef.Ref.(*native.PrintStream)
		if !ok {
			return Value{}, false, fmt.Errorf("invokevirtual: println receiver is not a PrintStream")
		}
		ps.Println(arg.Int)
	case "(Ljava/lang/String;)V":
		arg := frame.Pop()
		objectRef := frame.Pop()
		ps, ok := objectRef.Ref.(*native.PrintStream)
		if !ok {
			return Value{}, false, fmt.Errorf("invokevirtual: println receiver is not a PrintStream")
		}
		ps.Println(arg.Ref)
	case "()V":
		objectRef := frame.Pop()
		ps, ok := objectRef.Ref.(*native.PrintStream)
		if !ok {
			return Value{}, false, fmt.Errorf("invokevirtual: println receiver is not a PrintStream")
		}
		ps.Println()
	default:
		return Value{}, false, fmt.Errorf("invokevirtual: unsupported println descriptor %s", descriptor)
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

	// Find the method in the current class
	thisClassName, _ := frame.Class.ClassName()
	if methodRef.ClassName != thisClassName {
		return Value{}, false, fmt.Errorf("invokestatic: cross-class calls not supported (calling %s.%s from %s)", methodRef.ClassName, methodRef.MethodName, thisClassName)
	}

	method := frame.Class.FindMethod(methodRef.MethodName, methodRef.Descriptor)
	if method == nil {
		return Value{}, false, fmt.Errorf("invokestatic: method %s:%s not found in class %s", methodRef.MethodName, methodRef.Descriptor, methodRef.ClassName)
	}

	// Count parameters from descriptor
	paramCount := countParams(methodRef.Descriptor)

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

// executeNew handles the new instruction (minimal for Milestone 1).
func (vm *VM) executeNew(frame *Frame) (Value, bool, error) {
	index := frame.ReadU16()
	pool := frame.Class.ConstantPool

	className, err := classfile.GetClassName(pool, index)
	if err != nil {
		return Value{}, false, fmt.Errorf("new: %w", err)
	}

	// For Milestone 1, just push a placeholder object
	frame.Push(RefValue(className))
	return Value{}, false, nil
}

// countParams counts the number of parameters in a method descriptor.
func countParams(descriptor string) int {
	// Parse between ( and )
	start := strings.Index(descriptor, "(")
	end := strings.Index(descriptor, ")")
	if start == -1 || end == -1 {
		return 0
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
			i++
		}
	}
	return count
}

// isVoidReturn checks if a method descriptor has void return type.
func isVoidReturn(descriptor string) bool {
	return strings.HasSuffix(descriptor, ")V")
}
