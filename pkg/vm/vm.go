package vm

import (
	"fmt"
	"io"
	"math"
	"os"
	"reflect"
	"strings"

	"github.com/daimatz/gojvm/pkg/classfile"
	"github.com/daimatz/gojvm/pkg/native"
)

// maxFrameDepth is the maximum number of nested method calls.
const maxFrameDepth = 1024

// AccNative is the access flag for native methods.
const AccNative = 0x0100

// AccAbstract is the access flag for abstract methods.
const AccAbstract = 0x0400

// VM is the virtual machine that executes Java bytecode.
type VM struct {
	ClassLoader        ClassLoader
	Stdout             io.Writer
	frameDepth         int
	staticFields       map[string]map[string]Value // className -> fieldName -> Value
	initializedClasses map[string]bool             // <clinit> done
}

// NewVM creates a new VM with the given class loader.
func NewVM(cl ClassLoader) *VM {
	return &VM{
		ClassLoader:        cl,
		Stdout:             os.Stdout,
		staticFields:       make(map[string]map[string]Value),
		initializedClasses: make(map[string]bool),
	}
}

// Execute finds and executes the main method of the given class.
func (vm *VM) Execute(mainClassName string) error {
	cf, err := vm.ClassLoader.LoadClass(mainClassName)
	if err != nil {
		return err
	}

	method := cf.FindMethod("main", "([Ljava/lang/String;)V")
	if method == nil {
		return fmt.Errorf("main method not found")
	}
	if method.Code == nil {
		return fmt.Errorf("main method has no Code attribute")
	}

	// main(String[] args) — pass null for args
	args := []Value{NullValue()}
	_, err = vm.executeMethod(cf, method, args)
	return err
}

// executeMethod executes a method with the given arguments and returns its return value.
func (vm *VM) executeMethod(cf *classfile.ClassFile, method *classfile.MethodInfo, args []Value) (Value, error) {
	// Check for native method
	if method.AccessFlags&AccNative != 0 {
		className, _ := cf.ClassName()
		return vm.executeNativeMethod(className, method.Name, method.Descriptor, args)
	}

	// Check for abstract method
	if method.AccessFlags&AccAbstract != 0 {
		className, _ := cf.ClassName()
		return Value{}, fmt.Errorf("AbstractMethodError: %s.%s:%s", className, method.Name, method.Descriptor)
	}

	if method.Code == nil {
		return Value{}, fmt.Errorf("method %s has no Code attribute", method.Name)
	}

	vm.frameDepth++
	if vm.frameDepth > maxFrameDepth {
		return Value{}, fmt.Errorf("stack overflow: frame depth exceeded %d", maxFrameDepth)
	}
	defer func() { vm.frameDepth-- }()

	frame := NewFrame(method.Code.MaxLocals, method.Code.MaxStack, method.Code.Code, cf)

	// Set arguments into local variables
	for i, arg := range args {
		frame.SetLocal(i, arg)
	}

	className, _ := cf.ClassName()
	// Execution loop
	for frame.PC < len(frame.Code) {
		opcode := frame.Code[frame.PC]
		instructionPC := frame.PC
		frame.PC++

		retVal, hasReturn, err := vm.executeInstruction(frame, opcode)
		if err != nil {
			javaExc, isJavaExc := err.(*JavaException)
			if !isJavaExc {
				return Value{}, fmt.Errorf("in %s.%s:%s at PC=%d: %w", className, method.Name, method.Descriptor, instructionPC, err)
			}
			// Search exception table for matching handler
			handler := vm.findExceptionHandler(method.Code, instructionPC, javaExc, cf)
			if handler != nil {
				frame.SP = 0
				frame.Push(RefValue(javaExc.Object))
				frame.PC = int(handler.HandlerPC)
				continue
			}
			// No handler found, propagate
			return Value{}, javaExc
		}
		if hasReturn {
			return retVal, nil
		}
	}

	// Fell off the end of the method (implicit return for void methods)
	return Value{}, nil
}

// executeNativeMethod dispatches native method calls.
func (vm *VM) executeNativeMethod(className, methodName, descriptor string, args []Value) (Value, error) {
	key := className + "." + methodName + ":" + descriptor

	switch key {
	case "java/lang/Object.hashCode:()I":
		obj, ok := args[0].Ref.(*JObject)
		if !ok {
			return Value{}, fmt.Errorf("Object.hashCode: receiver is not a JObject")
		}
		hash := int32(reflect.ValueOf(obj).Pointer() & 0x7FFFFFFF)
		return IntValue(hash), nil

	case "java/lang/Object.getClass:()Ljava/lang/Class;":
		obj, ok := args[0].Ref.(*JObject)
		if !ok {
			return Value{}, fmt.Errorf("Object.getClass: receiver is not a JObject")
		}
		classObj := &JObject{
			ClassName: "java/lang/Class",
			Fields:    map[string]Value{"name": RefValue(obj.ClassName)},
		}
		return RefValue(classObj), nil

	case "java/lang/Class.getPrimitiveClass:(Ljava/lang/String;)Ljava/lang/Class;":
		name := ""
		if args[0].Ref != nil {
			name = fmt.Sprintf("%v", args[0].Ref)
		}
		classObj := &JObject{
			ClassName: "java/lang/Class",
			Fields:    map[string]Value{"name": RefValue(name)},
		}
		return RefValue(classObj), nil

	case "java/lang/Class.desiredAssertionStatus0:(Ljava/lang/Class;)Z",
		"java/lang/Class.desiredAssertionStatus:()Z":
		return IntValue(0), nil

	case "jdk/internal/misc/VM.getSavedProperty:(Ljava/lang/String;)Ljava/lang/String;":
		return NullValue(), nil

	case "jdk/internal/misc/CDS.initializeFromArchive:(Ljava/lang/Class;)V":
		return Value{}, nil

	case "jdk/internal/misc/CDS.isDumpingClassList0:()Z",
		"jdk/internal/misc/CDS.isDumpingArchive0:()Z",
		"jdk/internal/misc/CDS.isSharingEnabled0:()Z":
		return IntValue(0), nil

	case "java/lang/Float.floatToRawIntBits:(F)I":
		return IntValue(int32(math.Float32bits(args[0].Float))), nil

	case "java/lang/System.registerNatives:()V",
		"java/lang/Object.registerNatives:()V",
		"java/lang/Class.registerNatives:()V":
		return Value{}, nil

	case "java/lang/Float.isNaN:(F)Z":
		return IntValue(0), nil

	case "java/lang/String.intern:()Ljava/lang/String;":
		return args[0], nil

	case "jdk/internal/misc/Unsafe.getUnsafe:()Ljdk/internal/misc/Unsafe;":
		obj := &JObject{ClassName: "jdk/internal/misc/Unsafe", Fields: make(map[string]Value)}
		return RefValue(obj), nil

	case "jdk/internal/misc/Unsafe.storeFence:()V":
		return Value{}, nil

	case "java/lang/Class.isArray:()Z":
		return IntValue(0), nil

	case "java/lang/Class.isPrimitive:()Z":
		return IntValue(0), nil

	case "jdk/internal/misc/Unsafe.arrayBaseOffset:(Ljava/lang/Class;)I":
		return IntValue(0), nil

	case "jdk/internal/misc/Unsafe.arrayIndexScale:(Ljava/lang/Class;)I":
		return IntValue(1), nil

	case "jdk/internal/misc/Unsafe.objectFieldOffset1:(Ljava/lang/Class;Ljava/lang/String;)J":
		return LongValue(0), nil

	case "jdk/internal/misc/VM.initialize:()V":
		// Set savedProps to an empty HashMap so getSavedProperty doesn't NPE
		propsObj := &JObject{ClassName: "java/util/HashMap", Fields: make(map[string]Value)}
		vm.setStaticField("jdk/internal/misc/VM", "savedProps", RefValue(propsObj))
		return Value{}, nil

	case "java/lang/StringUTF16.isBigEndian:()Z":
		return IntValue(0), nil

	case "java/lang/System.arraycopy:(Ljava/lang/Object;ILjava/lang/Object;II)V":
		return vm.nativeArraycopy(args)

	case "java/lang/Class.forName0:(Ljava/lang/String;ZLjava/lang/ClassLoader;Ljava/lang/Class;)Ljava/lang/Class;":
		name := ""
		if args[0].Ref != nil {
			name = fmt.Sprintf("%v", args[0].Ref)
		}
		classObj := &JObject{
			ClassName: "java/lang/Class",
			Fields:    map[string]Value{"name": RefValue(name)},
		}
		return RefValue(classObj), nil

	case "java/lang/Object.notifyAll:()V",
		"java/lang/Object.notify:()V":
		return Value{}, nil

	case "java/lang/Thread.currentThread:()Ljava/lang/Thread;":
		obj := &JObject{ClassName: "java/lang/Thread", Fields: make(map[string]Value)}
		return RefValue(obj), nil

	case "java/lang/Thread.setPriority:(I)V":
		return Value{}, nil

	case "java/lang/Runtime.maxMemory:()J":
		return LongValue(256 * 1024 * 1024), nil

	case "jdk/internal/misc/Unsafe.compareAndSetInt:(Ljava/lang/Object;JII)Z":
		return IntValue(1), nil // pretend CAS always succeeds

	case "jdk/internal/misc/Unsafe.compareAndSetLong:(Ljava/lang/Object;JJJ)Z":
		return IntValue(1), nil

	case "jdk/internal/misc/Unsafe.compareAndSetReference:(Ljava/lang/Object;JLjava/lang/Object;Ljava/lang/Object;)Z":
		return IntValue(1), nil

	case "jdk/internal/misc/Unsafe.getIntVolatile:(Ljava/lang/Object;J)I":
		return IntValue(0), nil

	case "jdk/internal/misc/Unsafe.getReferenceVolatile:(Ljava/lang/Object;J)Ljava/lang/Object;":
		return NullValue(), nil

	case "jdk/internal/misc/Unsafe.putReferenceVolatile:(Ljava/lang/Object;JLjava/lang/Object;)V":
		return Value{}, nil

	case "jdk/internal/misc/Unsafe.getObjectSize:(Ljava/lang/Object;)J":
		return LongValue(16), nil

	case "java/lang/Class.getComponentType:()Ljava/lang/Class;":
		return NullValue(), nil

	case "java/lang/Class.isAssignableFrom:(Ljava/lang/Class;)Z":
		return IntValue(1), nil
	}

	// registerNatives pattern
	if methodName == "registerNatives" && descriptor == "()V" {
		return Value{}, nil
	}
	// initIDs pattern (used by many JDK classes)
	if methodName == "initIDs" && descriptor == "()V" {
		return Value{}, nil
	}

	return Value{}, fmt.Errorf("native method not implemented: %s.%s:%s", className, methodName, descriptor)
}

// ensureInitialized runs <clinit> for a class if it hasn't been run yet.
func (vm *VM) ensureInitialized(className string) error {
	if vm.initializedClasses[className] {
		return nil
	}
	vm.initializedClasses[className] = true // set before to prevent recursion

	cf, err := vm.ClassLoader.LoadClass(className)
	if err != nil {
		vm.initializedClasses[className] = false
		return nil // class not found is OK for initialization
	}

	// Initialize superclass first
	superName := cf.SuperClassName()
	if superName != "" {
		if err := vm.ensureInitialized(superName); err != nil {
			return err
		}
	}

	// Run <clinit> if present
	clinit := cf.FindMethod("<clinit>", "()V")
	if clinit != nil {
		_, err := vm.executeMethod(cf, clinit, nil)
		if err != nil {
			if _, ok := err.(*JavaException); ok {
				return err // JavaException はそのまま伝播
			}
			return fmt.Errorf("error in <clinit> of %s: %w", className, err)
		}
	}
	return nil
}

// getStaticField returns the value of a static field.
func (vm *VM) getStaticField(className, fieldName string) Value {
	if fields, ok := vm.staticFields[className]; ok {
		if val, ok := fields[fieldName]; ok {
			return val
		}
	}
	return Value{} // default zero value
}

// getStaticFieldOk returns the value and whether it was set.
func (vm *VM) getStaticFieldOk(className, fieldName string) (Value, bool) {
	if fields, ok := vm.staticFields[className]; ok {
		if val, ok := fields[fieldName]; ok {
			return val, true
		}
	}
	return Value{}, false
}

// defaultValueForDescriptor returns the default value for a field based on its descriptor.
func defaultValueForDescriptor(descriptor string) Value {
	if len(descriptor) == 0 {
		return NullValue()
	}
	switch descriptor[0] {
	case 'L', '[':
		return NullValue()
	case 'F':
		return FloatValue(0)
	case 'J':
		return LongValue(0)
	default:
		return IntValue(0)
	}
}

// setStaticField sets the value of a static field.
func (vm *VM) setStaticField(className, fieldName string, val Value) {
	if _, ok := vm.staticFields[className]; !ok {
		vm.staticFields[className] = make(map[string]Value)
	}
	vm.staticFields[className][fieldName] = val
}

// isInstanceOf checks whether objectClassName is an instance of targetClassName,
// walking the superclass chain and recursively checking interfaces.
func (vm *VM) isInstanceOf(objectClassName, targetClassName string) bool {
	return vm.isInstanceOfWithVisited(objectClassName, targetClassName, make(map[string]bool))
}

func (vm *VM) isInstanceOfWithVisited(objectClassName, targetClassName string, visited map[string]bool) bool {
	if objectClassName == targetClassName {
		return true
	}
	if visited[objectClassName] {
		return false
	}
	visited[objectClassName] = true
	if vm.ClassLoader == nil {
		return false
	}
	current := objectClassName
	for current != "" {
		cf, err := vm.ClassLoader.LoadClass(current)
		if err != nil {
			return false
		}
		// Check interfaces
		for _, ifIdx := range cf.Interfaces {
			ifName, err := classfile.GetClassName(cf.ConstantPool, ifIdx)
			if err == nil && (ifName == targetClassName || vm.isInstanceOfWithVisited(ifName, targetClassName, visited)) {
				return true
			}
		}
		// Move to superclass
		current = cf.SuperClassName()
		if current == targetClassName {
			return true
		}
	}
	return false
}

// findExceptionHandler searches the exception table for a matching handler.
func (vm *VM) findExceptionHandler(code *classfile.CodeAttribute, pc int, exc *JavaException, cf *classfile.ClassFile) *classfile.ExceptionHandler {
	for i := range code.ExceptionHandlers {
		h := &code.ExceptionHandlers[i]
		if pc < int(h.StartPC) || pc >= int(h.EndPC) {
			continue
		}
		if h.CatchType == 0 {
			return h // catch-all (finally)
		}
		catchClassName, err := classfile.GetClassName(cf.ConstantPool, h.CatchType)
		if err != nil {
			continue
		}
		if vm.isInstanceOf(exc.Object.ClassName, catchClassName) {
			return h
		}
	}
	return nil
}

// resolveMethod resolves a method from its class name, walking up the class hierarchy.
func (vm *VM) resolveMethod(className, methodName, descriptor string) (*classfile.ClassFile, *classfile.MethodInfo, error) {
	// Walk superclass chain
	current := className
	for current != "" {
		cf, err := vm.ClassLoader.LoadClass(current)
		if err != nil {
			return nil, nil, err
		}
		method := cf.FindMethod(methodName, descriptor)
		if method != nil {
			return cf, method, nil
		}
		current = cf.SuperClassName()
	}
	// Walk superclass chain again, searching interfaces for default methods
	current = className
	for current != "" {
		cf, err := vm.ClassLoader.LoadClass(current)
		if err != nil {
			break
		}
		for _, ifIdx := range cf.Interfaces {
			ifName, err := classfile.GetClassName(cf.ConstantPool, ifIdx)
			if err != nil {
				continue
			}
			ifCf, ifMethod, err := vm.resolveMethod(ifName, methodName, descriptor)
			if err == nil {
				return ifCf, ifMethod, nil
			}
		}
		current = cf.SuperClassName()
	}
	return nil, nil, fmt.Errorf("method %s.%s:%s not found", className, methodName, descriptor)
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
	case *classfile.ConstantFloat:
		frame.Push(FloatValue(c.Value))
	case *classfile.ConstantString:
		str, err := classfile.GetUtf8(pool, c.StringIndex)
		if err != nil {
			return Value{}, false, fmt.Errorf("ldc: resolving string: %w", err)
		}
		frame.Push(RefValue(str))
	case *classfile.ConstantClass:
		name, err := classfile.GetUtf8(pool, c.NameIndex)
		if err != nil {
			return Value{}, false, fmt.Errorf("ldc: resolving class name: %w", err)
		}
		classObj := &JObject{
			ClassName: "java/lang/Class",
			Fields:    map[string]Value{"name": RefValue(name)},
		}
		frame.Push(RefValue(classObj))
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

	if err := vm.ensureInitialized(fieldRef.ClassName); err != nil {
		return Value{}, false, fmt.Errorf("getstatic: initializing %s: %w", fieldRef.ClassName, err)
	}

	// Handle java/lang/System.out
	if fieldRef.ClassName == "java/lang/System" && fieldRef.FieldName == "out" {
		frame.Push(RefValue(&native.PrintStream{Writer: vm.Stdout}))
		return Value{}, false, nil
	}

	val, ok := vm.getStaticFieldOk(fieldRef.ClassName, fieldRef.FieldName)
	if !ok {
		// Field never set: return type-appropriate default
		val = defaultValueForDescriptor(fieldRef.Descriptor)
	}
	frame.Push(val)
	return Value{}, false, nil
}

// executePutstatic handles the putstatic instruction.
func (vm *VM) executePutstatic(frame *Frame) (Value, bool, error) {
	index := frame.ReadU16()
	pool := frame.Class.ConstantPool

	fieldRef, err := classfile.ResolveFieldref(pool, index)
	if err != nil {
		return Value{}, false, fmt.Errorf("putstatic: %w", err)
	}

	if err := vm.ensureInitialized(fieldRef.ClassName); err != nil {
		return Value{}, false, fmt.Errorf("putstatic: initializing %s: %w", fieldRef.ClassName, err)
	}

	value := frame.Pop()
	vm.setStaticField(fieldRef.ClassName, fieldRef.FieldName, value)
	return Value{}, false, nil
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
		return Value{}, false, NewJavaException("java/lang/NullPointerException")
	}
	obj, ok := objectRef.Ref.(*JObject)
	if !ok {
		return Value{}, false, fmt.Errorf("getfield: receiver is not a JObject")
	}

	val, exists := obj.Fields[fieldRef.FieldName]
	if !exists {
		frame.Push(defaultValueForDescriptor(fieldRef.Descriptor))
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
		return Value{}, false, NewJavaException("java/lang/NullPointerException")
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

	// PrintStream.println / print (native I/O)
	if methodRef.ClassName == "java/io/PrintStream" {
		ps, ok := objectRef.Ref.(*native.PrintStream)
		if ok {
			return vm.handlePrintStream(frame, ps, methodRef.MethodName, methodRef.Descriptor, args)
		}
	}

	if objectRef.Type == TypeNull || objectRef.Ref == nil {
		return Value{}, false, NewJavaException("java/lang/NullPointerException")
	}

	// JObject method resolution via ClassLoader
	obj, ok := objectRef.Ref.(*JObject)
	if !ok {
		return Value{}, false, fmt.Errorf("invokevirtual: receiver is not a JObject for method %s.%s", methodRef.ClassName, methodRef.MethodName)
	}

	cf, method, err := vm.resolveMethod(obj.ClassName, methodRef.MethodName, methodRef.Descriptor)
	if err != nil {
		return Value{}, false, err
	}

	fullArgs := make([]Value, 0, len(args)+1)
	fullArgs = append(fullArgs, objectRef)
	fullArgs = append(fullArgs, args...)
	retVal, err := vm.executeMethod(cf, method, fullArgs)
	if err != nil {
		return Value{}, false, err
	}
	if !isVoidReturn(methodRef.Descriptor) {
		frame.Push(retVal)
	}
	return Value{}, false, nil
}

// handlePrintStream handles PrintStream method calls.
func (vm *VM) handlePrintStream(frame *Frame, ps *native.PrintStream, methodName, descriptor string, args []Value) (Value, bool, error) {
	if methodName == "println" {
		switch descriptor {
		case "(I)V":
			ps.Println(args[0].Int)
		case "(Ljava/lang/String;)V":
			ps.Println(args[0].Ref)
		case "(Ljava/lang/Object;)V":
			if args[0].Type == TypeNull {
				ps.Println("null")
			} else if obj, ok := args[0].Ref.(*JObject); ok {
				// Try to call toString or use a reasonable default
				val, exists := obj.Fields["value"]
				if exists {
					ps.Println(val.Int)
				} else {
					ps.Println(obj.ClassName)
				}
			} else {
				ps.Println(args[0].Ref)
			}
		case "()V":
			ps.Println()
		default:
			return Value{}, false, fmt.Errorf("invokevirtual: unsupported println descriptor %s", descriptor)
		}
		return Value{}, false, nil
	}
	if methodName == "print" {
		switch descriptor {
		case "(I)V":
			fmt.Fprintf(ps.Writer, "%d", args[0].Int)
		case "(Ljava/lang/String;)V":
			fmt.Fprintf(ps.Writer, "%v", args[0].Ref)
		default:
			return Value{}, false, fmt.Errorf("invokevirtual: unsupported print descriptor %s", descriptor)
		}
		return Value{}, false, nil
	}
	return Value{}, false, fmt.Errorf("invokevirtual: unsupported PrintStream method %s:%s", methodName, descriptor)
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

	// Object.<init> is a no-op
	if methodRef.ClassName == "java/lang/Object" && methodRef.MethodName == "<init>" {
		return Value{}, false, nil
	}

	// Resolve method from class loader
	cf, method, err := vm.resolveMethod(methodRef.ClassName, methodRef.MethodName, methodRef.Descriptor)
	if err != nil {
		return Value{}, false, err
	}

	fullArgs := make([]Value, 0, len(args)+1)
	fullArgs = append(fullArgs, objectRef)
	fullArgs = append(fullArgs, args...)
	retVal, err := vm.executeMethod(cf, method, fullArgs)
	if err != nil {
		return Value{}, false, err
	}
	if !isVoidReturn(methodRef.Descriptor) {
		frame.Push(retVal)
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

	if err := vm.ensureInitialized(methodRef.ClassName); err != nil {
		return Value{}, false, fmt.Errorf("invokestatic: initializing %s: %w", methodRef.ClassName, err)
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

	// Resolve method from class loader
	cf, method, err := vm.resolveMethod(methodRef.ClassName, methodRef.MethodName, methodRef.Descriptor)
	if err != nil {
		return Value{}, false, err
	}

	// Execute the method
	retVal, err := vm.executeMethod(cf, method, args)
	if err != nil {
		return Value{}, false, err
	}

	// Push return value if the method returns something
	if !isVoidReturn(methodRef.Descriptor) {
		frame.Push(retVal)
	}

	return Value{}, false, nil
}

// executeInvokeinterface handles the invokeinterface instruction.
func (vm *VM) executeInvokeinterface(frame *Frame) (Value, bool, error) {
	index := frame.ReadU16()
	_ = frame.ReadU8() // count (unused)
	_ = frame.ReadU8() // reserved (0)

	pool := frame.Class.ConstantPool

	methodRef, err := classfile.ResolveInterfaceMethodref(pool, index)
	if err != nil {
		return Value{}, false, fmt.Errorf("invokeinterface: %w", err)
	}

	paramCount, err := countParams(methodRef.Descriptor)
	if err != nil {
		return Value{}, false, fmt.Errorf("invokeinterface: %w", err)
	}

	args := make([]Value, paramCount)
	for i := paramCount - 1; i >= 0; i-- {
		args[i] = frame.Pop()
	}
	objectRef := frame.Pop()

	if objectRef.Type == TypeNull || objectRef.Ref == nil {
		return Value{}, false, NewJavaException("java/lang/NullPointerException")
	}

	obj, ok := objectRef.Ref.(*JObject)
	if !ok {
		// Handle string receiver for methods like equals
		if _, isStr := objectRef.Ref.(string); isStr && methodRef.MethodName == "equals" {
			if len(args) == 1 {
				otherStr, ok2 := args[0].Ref.(string)
				if ok2 && objectRef.Ref.(string) == otherStr {
					frame.Push(IntValue(1))
				} else {
					frame.Push(IntValue(0))
				}
				return Value{}, false, nil
			}
		}
		return Value{}, false, fmt.Errorf("invokeinterface: receiver is not a JObject for %s.%s", methodRef.ClassName, methodRef.MethodName)
	}

	cf, method, err := vm.resolveMethod(obj.ClassName, methodRef.MethodName, methodRef.Descriptor)
	if err != nil {
		return Value{}, false, err
	}

	fullArgs := make([]Value, 0, len(args)+1)
	fullArgs = append(fullArgs, objectRef)
	fullArgs = append(fullArgs, args...)
	retVal, err := vm.executeMethod(cf, method, fullArgs)
	if err != nil {
		return Value{}, false, err
	}
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

	if err := vm.ensureInitialized(className); err != nil {
		return Value{}, false, fmt.Errorf("new: initializing %s: %w", className, err)
	}

	obj := &JObject{ClassName: className, Fields: make(map[string]Value)}
	frame.Push(RefValue(obj))
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

// nativeArraycopy implements System.arraycopy.
func (vm *VM) nativeArraycopy(args []Value) (Value, error) {
	srcRef := args[0]
	srcPos := int(args[1].Int)
	destRef := args[2]
	destPos := int(args[3].Int)
	length := int(args[4].Int)

	if srcRef.Type == TypeNull || destRef.Type == TypeNull {
		return Value{}, NewJavaException("java/lang/NullPointerException")
	}

	srcArr, ok1 := srcRef.Ref.(*JArray)
	destArr, ok2 := destRef.Ref.(*JArray)
	if !ok1 || !ok2 {
		return Value{}, NewJavaException("java/lang/ArrayStoreException")
	}

	if srcPos < 0 || destPos < 0 || length < 0 ||
		srcPos+length > len(srcArr.Elements) ||
		destPos+length > len(destArr.Elements) {
		return Value{}, NewJavaException("java/lang/ArrayIndexOutOfBoundsException")
	}

	for i := 0; i < length; i++ {
		destArr.Elements[destPos+i] = srcArr.Elements[srcPos+i]
	}
	return Value{}, nil
}

// isVoidReturn checks if a method descriptor has void return type.
func isVoidReturn(descriptor string) bool {
	return strings.HasSuffix(descriptor, ")V")
}
