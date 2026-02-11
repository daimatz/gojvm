package vm

import (
	"fmt"
	"io"
	"math"
	"os"
	"reflect"
	"sort"
	"strconv"
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

	case "jdk/internal/misc/CDS.getRandomSeedForDumping:()J":
		return LongValue(0), nil

	case "java/lang/Float.floatToRawIntBits:(F)I":
		return IntValue(int32(math.Float32bits(args[0].Float))), nil

	case "java/lang/Double.doubleToRawLongBits:(D)J":
		return LongValue(int64(math.Float64bits(args[0].Double))), nil

	case "java/lang/Double.longBitsToDouble:(J)D":
		return DoubleValue(math.Float64frombits(uint64(args[0].Long))), nil

	case "java/lang/Math.sqrt:(D)D":
		return DoubleValue(math.Sqrt(args[0].Double)), nil

	case "java/lang/Math.pow:(DD)D":
		return DoubleValue(math.Pow(args[0].Double, args[1].Double)), nil

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

	case "java/lang/System.nanoTime:()J":
		return LongValue(0), nil

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

	case "jdk/internal/reflect/Reflection.getCallerClass:()Ljava/lang/Class;":
		classObj := &JObject{
			ClassName: "java/lang/Class",
			Fields:    map[string]Value{"name": RefValue("java/lang/Object")},
		}
		return RefValue(classObj), nil

	case "java/lang/reflect/Array.newArray:(Ljava/lang/Class;I)Ljava/lang/Object;":
		length := int(args[1].Int)
		arr := &JArray{Elements: make([]Value, length)}
		return RefValue(arr), nil
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
	case 'D':
		return DoubleValue(0)
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

	// Handle array clone
	if arr, ok := objectRef.Ref.(*JArray); ok {
		if methodRef.MethodName == "clone" {
			newElements := make([]Value, len(arr.Elements))
			copy(newElements, arr.Elements)
			newArr := &JArray{Elements: newElements}
			frame.Push(RefValue(newArr))
			return Value{}, false, nil
		}
	}

	// Handle String methods natively (strings are stored as Go string, not JObject)
	if str, ok := objectRef.Ref.(string); ok {
		retVal, err := vm.handleStringMethod(str, methodRef.MethodName, methodRef.Descriptor, args)
		if err != nil {
			return Value{}, false, err
		}
		if !isVoidReturn(methodRef.Descriptor) {
			frame.Push(retVal)
		}
		return Value{}, false, nil
	}

	if objectRef.Type == TypeNull || objectRef.Ref == nil {
		return Value{}, false, NewJavaException("java/lang/NullPointerException")
	}

	// StringBuilder native handling
	if obj, ok := objectRef.Ref.(*JObject); ok && obj.ClassName == "java/lang/StringBuilder" {
		retVal, _, err := vm.handleStringBuilder(objectRef, methodRef.MethodName, methodRef.Descriptor, args)
		if err != nil {
			return Value{}, false, err
		}
		if !isVoidReturn(methodRef.Descriptor) {
			frame.Push(retVal)
		}
		return Value{}, false, nil
	}

	// JObject method resolution via ClassLoader
	obj, ok := objectRef.Ref.(*JObject)
	if !ok {
		return Value{}, false, fmt.Errorf("invokevirtual: receiver is not a JObject for method %s.%s", methodRef.ClassName, methodRef.MethodName)
	}

	// Handle ArrayList.sort natively (avoids deep JDK internals)
	if obj.ClassName == "java/util/ArrayList" && methodRef.MethodName == "sort" {
		return vm.handleArrayListSort(frame, obj, args)
	}

	// Handle Integer/Long/Double native methods
	if retVal, handled, err := vm.handleBoxedType(frame, obj, methodRef.MethodName, methodRef.Descriptor, args); handled {
		return retVal, false, err
	}

	// Lambda proxy dispatch
	if obj.LambdaTarget != nil && methodRef.MethodName == obj.LambdaTarget.MethodName {
		lt := obj.LambdaTarget
		cf, method, err := vm.resolveMethod(lt.TargetClass, lt.TargetMethod, lt.TargetDesc)
		if err != nil {
			return Value{}, false, err
		}
		fullArgs := make([]Value, 0, len(lt.CapturedArgs)+len(args))
		fullArgs = append(fullArgs, lt.CapturedArgs...)
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
		case "(J)V":
			ps.Println(args[0].Long)
		case "(D)V":
			d := args[0].Double
			if d == float64(int64(d)) && !math.IsInf(d, 0) {
				fmt.Fprintf(ps.Writer, "%s\n", strconv.FormatFloat(d, 'f', 1, 64))
			} else {
				fmt.Fprintf(ps.Writer, "%s\n", strconv.FormatFloat(d, 'f', -1, 64))
			}
		case "(F)V":
			fmt.Fprintf(ps.Writer, "%v\n", args[0].Float)
		case "(Z)V":
			if args[0].Int != 0 {
				ps.Println("true")
			} else {
				ps.Println("false")
			}
		case "(C)V":
			fmt.Fprintf(ps.Writer, "%c\n", rune(args[0].Int))
		case "(Ljava/lang/String;)V":
			ps.Println(args[0].Ref)
		case "(Ljava/lang/Object;)V":
			if args[0].Type == TypeNull {
				ps.Println("null")
			} else if obj, ok := args[0].Ref.(*JObject); ok {
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
		case "(J)V":
			fmt.Fprintf(ps.Writer, "%d", args[0].Long)
		case "(D)V":
			d := args[0].Double
			if d == float64(int64(d)) && !math.IsInf(d, 0) {
				fmt.Fprintf(ps.Writer, "%s", strconv.FormatFloat(d, 'f', 1, 64))
			} else {
				fmt.Fprintf(ps.Writer, "%s", strconv.FormatFloat(d, 'f', -1, 64))
			}
		case "(F)V":
			fmt.Fprintf(ps.Writer, "%v", args[0].Float)
		case "(C)V":
			fmt.Fprintf(ps.Writer, "%c", rune(args[0].Int))
		case "(Z)V":
			if args[0].Int != 0 {
				fmt.Fprintf(ps.Writer, "true")
			} else {
				fmt.Fprintf(ps.Writer, "false")
			}
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

	// StringBuilder native handling
	if methodRef.ClassName == "java/lang/StringBuilder" ||
		(objectRef.Ref != nil && func() bool {
			if o, ok := objectRef.Ref.(*JObject); ok {
				return o.ClassName == "java/lang/StringBuilder"
			}
			return false
		}()) {
		_, _, err := vm.handleStringBuilder(objectRef, methodRef.MethodName, methodRef.Descriptor, args)
		if err != nil {
			return Value{}, false, err
		}
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
		// Some JDK classes use InterfaceMethodref for invokestatic
		methodRef, err = classfile.ResolveInterfaceMethodref(pool, index)
		if err != nil {
			return Value{}, false, fmt.Errorf("invokestatic: %w", err)
		}
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

	// Handle AccessController.doPrivileged natively — just call action.run()
	if methodRef.ClassName == "java/security/AccessController" && methodRef.MethodName == "doPrivileged" {
		// args[0] is the PrivilegedAction; call its run() method
		action := args[0]
		if action.Type == TypeNull || action.Ref == nil {
			return Value{}, false, NewJavaException("java/lang/NullPointerException")
		}
		if obj, ok := action.Ref.(*JObject); ok && obj.LambdaTarget != nil {
			lt := obj.LambdaTarget
			cf2, method2, err2 := vm.resolveMethod(lt.TargetClass, lt.TargetMethod, lt.TargetDesc)
			if err2 != nil {
				return Value{}, false, err2
			}
			fullArgs := make([]Value, 0, len(lt.CapturedArgs))
			fullArgs = append(fullArgs, lt.CapturedArgs...)
			retVal, err2 := vm.executeMethod(cf2, method2, fullArgs)
			if err2 != nil {
				return Value{}, false, err2
			}
			if !isVoidReturn(methodRef.Descriptor) {
				frame.Push(retVal)
			}
			return Value{}, false, nil
		}
		// Non-lambda PrivilegedAction: call run() via interface dispatch
		if obj, ok := action.Ref.(*JObject); ok {
			cf2, method2, err2 := vm.resolveMethod(obj.ClassName, "run", "()Ljava/lang/Object;")
			if err2 != nil {
				return Value{}, false, err2
			}
			retVal, err2 := vm.executeMethod(cf2, method2, []Value{action})
			if err2 != nil {
				return Value{}, false, err2
			}
			if !isVoidReturn(methodRef.Descriptor) {
				frame.Push(retVal)
			}
			return Value{}, false, nil
		}
		// Fallback: return null
		if !isVoidReturn(methodRef.Descriptor) {
			frame.Push(NullValue())
		}
		return Value{}, false, nil
	}

	// Handle Collections.sort natively (avoids deep JDK internals)
	if methodRef.ClassName == "java/util/Collections" && methodRef.MethodName == "sort" {
		return vm.handleCollectionsSort(frame, methodRef.Descriptor, args)
	}

	// Handle Integer.compare natively
	if methodRef.ClassName == "java/lang/Integer" && methodRef.MethodName == "compare" {
		frame.Push(IntValue(vm.compareInt32(args[0].Int, args[1].Int)))
		return Value{}, false, nil
	}

	// Handle String.valueOf natively
	if methodRef.ClassName == "java/lang/String" && methodRef.MethodName == "valueOf" {
		retVal, err := vm.handleStringValueOf(methodRef.Descriptor, args)
		if err != nil {
			return Value{}, false, err
		}
		frame.Push(retVal)
		return Value{}, false, nil
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

	// Handle String methods natively via interface dispatch
	if str, isStr := objectRef.Ref.(string); isStr {
		retVal, err := vm.handleStringMethod(str, methodRef.MethodName, methodRef.Descriptor, args)
		if err != nil {
			return Value{}, false, err
		}
		if !isVoidReturn(methodRef.Descriptor) {
			frame.Push(retVal)
		}
		return Value{}, false, nil
	}

	obj, ok := objectRef.Ref.(*JObject)
	if !ok {
		return Value{}, false, fmt.Errorf("invokeinterface: receiver is not a JObject for %s.%s", methodRef.ClassName, methodRef.MethodName)
	}

	// Lambda proxy dispatch
	if obj.LambdaTarget != nil && methodRef.MethodName == obj.LambdaTarget.MethodName {
		lt := obj.LambdaTarget
		cf, method, err := vm.resolveMethod(lt.TargetClass, lt.TargetMethod, lt.TargetDesc)
		if err != nil {
			return Value{}, false, err
		}
		fullArgs := make([]Value, 0, len(lt.CapturedArgs)+len(args))
		fullArgs = append(fullArgs, lt.CapturedArgs...)
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
	if className == "java/lang/StringBuilder" {
		obj.Fields["_buffer"] = RefValue("")
	}
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

// formatDouble formats a double value matching Java's Double.toString behavior.
func formatDouble(d float64) string {
	if d == float64(int64(d)) && !math.IsInf(d, 0) {
		return strconv.FormatFloat(d, 'f', 1, 64)
	}
	return strconv.FormatFloat(d, 'f', -1, 64)
}

// executeInvokedynamic handles the invokedynamic instruction.
func (vm *VM) executeInvokedynamic(frame *Frame) (Value, bool, error) {
	index := frame.ReadU16()
	_ = frame.ReadU8() // must be 0
	_ = frame.ReadU8() // must be 0

	pool := frame.Class.ConstantPool

	// Resolve ConstantInvokeDynamic
	invDyn, ok := pool[index].(*classfile.ConstantInvokeDynamic)
	if !ok {
		return Value{}, false, fmt.Errorf("invokedynamic: CP index %d is not InvokeDynamic", index)
	}

	// Get name and type
	nat, ok := pool[invDyn.NameAndTypeIndex].(*classfile.ConstantNameAndType)
	if !ok {
		return Value{}, false, fmt.Errorf("invokedynamic: invalid NameAndType index")
	}
	methodName, _ := classfile.GetUtf8(pool, nat.NameIndex)
	descriptor, _ := classfile.GetUtf8(pool, nat.DescriptorIndex)

	// Get bootstrap method
	bsmIdx := invDyn.BootstrapMethodAttrIndex
	if int(bsmIdx) >= len(frame.Class.BootstrapMethods) {
		return Value{}, false, fmt.Errorf("invokedynamic: bootstrap method index %d out of range", bsmIdx)
	}
	bsm := frame.Class.BootstrapMethods[bsmIdx]

	// Get bootstrap method handle
	mh, ok := pool[bsm.MethodRef].(*classfile.ConstantMethodHandle)
	if !ok {
		return Value{}, false, fmt.Errorf("invokedynamic: bootstrap method is not MethodHandle")
	}

	// Resolve bootstrap method class and name
	var bsmClassName, bsmMethodName string
	switch mh.ReferenceKind {
	case 6: // REF_invokeStatic
		mref, ok := pool[mh.ReferenceIndex].(*classfile.ConstantMethodref)
		if !ok {
			// Try InterfaceMethodref
			imref, ok2 := pool[mh.ReferenceIndex].(*classfile.ConstantInterfaceMethodref)
			if !ok2 {
				return Value{}, false, fmt.Errorf("invokedynamic: cannot resolve bootstrap method ref")
			}
			bsmClassName, _ = classfile.GetClassName(pool, imref.ClassIndex)
			bsmNat, _ := pool[imref.NameAndTypeIndex].(*classfile.ConstantNameAndType)
			bsmMethodName, _ = classfile.GetUtf8(pool, bsmNat.NameIndex)
		} else {
			bsmClassName, _ = classfile.GetClassName(pool, mref.ClassIndex)
			bsmNat, _ := pool[mref.NameAndTypeIndex].(*classfile.ConstantNameAndType)
			bsmMethodName, _ = classfile.GetUtf8(pool, bsmNat.NameIndex)
		}
	default:
		return Value{}, false, fmt.Errorf("invokedynamic: unsupported bootstrap method reference kind %d", mh.ReferenceKind)
	}

	bsmKey := bsmClassName + "." + bsmMethodName

	switch bsmKey {
	case "java/lang/invoke/LambdaMetafactory.metafactory":
		return vm.handleLambdaMetafactory(frame, pool, bsm, methodName, descriptor)
	case "java/lang/invoke/StringConcatFactory.makeConcatWithConstants":
		return vm.handleStringConcatFactory(frame, pool, bsm, methodName, descriptor)
	default:
		return Value{}, false, fmt.Errorf("invokedynamic: unsupported bootstrap method %s", bsmKey)
	}
}

// handleLambdaMetafactory handles LambdaMetafactory.metafactory bootstrap calls.
func (vm *VM) handleLambdaMetafactory(frame *Frame, pool []classfile.ConstantPoolEntry, bsm classfile.BootstrapMethod, methodName, descriptor string) (Value, bool, error) {
	if len(bsm.BootstrapArguments) < 3 {
		return Value{}, false, fmt.Errorf("LambdaMetafactory: expected 3+ bootstrap args, got %d", len(bsm.BootstrapArguments))
	}

	// Get implementation method handle (arg[1])
	implHandle, ok := pool[bsm.BootstrapArguments[1]].(*classfile.ConstantMethodHandle)
	if !ok {
		return Value{}, false, fmt.Errorf("LambdaMetafactory: arg[1] is not MethodHandle")
	}

	// Resolve implementation method
	var targetClass, targetMethod, targetDesc string
	switch implHandle.ReferenceKind {
	case 5, 6, 7: // invokevirtual, invokestatic, invokespecial
		mref, ok := pool[implHandle.ReferenceIndex].(*classfile.ConstantMethodref)
		if ok {
			targetClass, _ = classfile.GetClassName(pool, mref.ClassIndex)
			implNat := pool[mref.NameAndTypeIndex].(*classfile.ConstantNameAndType)
			targetMethod, _ = classfile.GetUtf8(pool, implNat.NameIndex)
			targetDesc, _ = classfile.GetUtf8(pool, implNat.DescriptorIndex)
		}
	default:
		return Value{}, false, fmt.Errorf("LambdaMetafactory: unsupported impl reference kind %d", implHandle.ReferenceKind)
	}

	// Get interface name from return type of factory descriptor
	retStart := strings.Index(descriptor, ")L")
	interfaceName := ""
	if retStart != -1 {
		retType := descriptor[retStart+2:]
		interfaceName = strings.TrimSuffix(retType, ";")
	}

	// Pop captured args from stack (parameters of factory type)
	capturedCount, _ := countParams(descriptor)
	capturedArgs := make([]Value, capturedCount)
	for i := capturedCount - 1; i >= 0; i-- {
		capturedArgs[i] = frame.Pop()
	}

	// Create lambda proxy object
	obj := &JObject{
		ClassName: interfaceName,
		Fields:    make(map[string]Value),
		LambdaTarget: &LambdaTarget{
			InterfaceName: interfaceName,
			MethodName:    methodName,
			TargetClass:   targetClass,
			TargetMethod:  targetMethod,
			TargetDesc:    targetDesc,
			CapturedArgs:  capturedArgs,
			ReferenceKind: implHandle.ReferenceKind,
		},
	}

	frame.Push(RefValue(obj))
	return Value{}, false, nil
}

// handleStringConcatFactory handles StringConcatFactory.makeConcatWithConstants bootstrap calls.
func (vm *VM) handleStringConcatFactory(frame *Frame, pool []classfile.ConstantPoolEntry, bsm classfile.BootstrapMethod, methodName, descriptor string) (Value, bool, error) {
	// Get recipe from bootstrap args[0] (ConstantString)
	recipe := ""
	if len(bsm.BootstrapArguments) > 0 {
		argEntry := pool[bsm.BootstrapArguments[0]]
		switch c := argEntry.(type) {
		case *classfile.ConstantString:
			recipe, _ = classfile.GetUtf8(pool, c.StringIndex)
		}
	}

	// Count parameters from descriptor
	paramCount, _ := countParams(descriptor)
	args := make([]Value, paramCount)
	for i := paramCount - 1; i >= 0; i-- {
		args[i] = frame.Pop()
	}

	// Constants from bootstrap args [1:]
	constants := make([]string, 0)
	for i := 1; i < len(bsm.BootstrapArguments); i++ {
		argEntry := pool[bsm.BootstrapArguments[i]]
		switch c := argEntry.(type) {
		case *classfile.ConstantString:
			s, _ := classfile.GetUtf8(pool, c.StringIndex)
			constants = append(constants, s)
		case *classfile.ConstantInteger:
			constants = append(constants, fmt.Sprintf("%d", c.Value))
		default:
			constants = append(constants, "")
		}
	}

	// Build result string from recipe
	// \x01 = argument placeholder, \x02 = constant placeholder
	var result strings.Builder
	argIdx := 0
	constIdx := 0
	for i := 0; i < len(recipe); i++ {
		ch := recipe[i]
		if ch == '\x01' {
			if argIdx < len(args) {
				result.WriteString(vm.valueToString(args[argIdx]))
				argIdx++
			}
		} else if ch == '\x02' {
			if constIdx < len(constants) {
				result.WriteString(constants[constIdx])
				constIdx++
			}
		} else {
			result.WriteByte(ch)
		}
	}

	frame.Push(RefValue(result.String()))
	return Value{}, false, nil
}

// valueToString converts a Value to its string representation.
func (vm *VM) valueToString(v Value) string {
	switch v.Type {
	case TypeInt:
		return fmt.Sprintf("%d", v.Int)
	case TypeLong:
		return fmt.Sprintf("%d", v.Long)
	case TypeFloat:
		return fmt.Sprintf("%v", v.Float)
	case TypeDouble:
		return fmt.Sprintf("%v", v.Double)
	case TypeNull:
		return "null"
	case TypeRef:
		if s, ok := v.Ref.(string); ok {
			return s
		}
		if obj, ok := v.Ref.(*JObject); ok {
			if val, exists := obj.Fields["value"]; exists {
				switch obj.ClassName {
				case "java/lang/Integer", "java/lang/Short", "java/lang/Byte":
					return fmt.Sprintf("%d", val.Int)
				case "java/lang/Long":
					return fmt.Sprintf("%d", val.Long)
				case "java/lang/Float":
					return fmt.Sprintf("%v", val.Float)
				case "java/lang/Double":
					return formatDouble(val.Double)
				case "java/lang/Boolean":
					if val.Int != 0 {
						return "true"
					}
					return "false"
				case "java/lang/Character":
					return string(rune(val.Int))
				}
			}
			return obj.ClassName
		}
		return fmt.Sprintf("%v", v.Ref)
	}
	return ""
}

// handleStringBuilder handles StringBuilder method calls natively.
func (vm *VM) handleStringBuilder(objectRef Value, methodName, descriptor string, args []Value) (Value, bool, error) {
	obj := objectRef.Ref.(*JObject)
	buf, _ := obj.Fields["_buffer"].Ref.(string)

	switch methodName {
	case "<init>":
		switch descriptor {
		case "()V":
			// already initialized
		case "(Ljava/lang/String;)V":
			if s, ok := args[0].Ref.(string); ok {
				obj.Fields["_buffer"] = RefValue(s)
			}
		case "(I)V":
			// capacity hint, ignore
		}
		return Value{}, false, nil

	case "append":
		var appendStr string
		switch descriptor {
		case "(Ljava/lang/String;)Ljava/lang/StringBuilder;":
			if args[0].Type == TypeNull {
				appendStr = "null"
			} else if s, ok := args[0].Ref.(string); ok {
				appendStr = s
			} else {
				appendStr = fmt.Sprintf("%v", args[0].Ref)
			}
		case "(I)Ljava/lang/StringBuilder;":
			appendStr = fmt.Sprintf("%d", args[0].Int)
		case "(J)Ljava/lang/StringBuilder;":
			appendStr = fmt.Sprintf("%d", args[0].Long)
		case "(D)Ljava/lang/StringBuilder;":
			appendStr = formatDouble(args[0].Double)
		case "(F)Ljava/lang/StringBuilder;":
			appendStr = fmt.Sprintf("%v", args[0].Float)
		case "(C)Ljava/lang/StringBuilder;":
			appendStr = string(rune(args[0].Int))
		case "(Z)Ljava/lang/StringBuilder;":
			if args[0].Int != 0 {
				appendStr = "true"
			} else {
				appendStr = "false"
			}
		case "(Ljava/lang/Object;)Ljava/lang/StringBuilder;":
			appendStr = vm.valueToString(args[0])
		}
		obj.Fields["_buffer"] = RefValue(buf + appendStr)
		return objectRef, false, nil

	case "toString":
		return RefValue(buf), false, nil

	case "length":
		return IntValue(int32(len(buf))), false, nil
	}

	return Value{}, false, fmt.Errorf("StringBuilder: unsupported method %s:%s", methodName, descriptor)
}

// handleStringMethod handles String instance method calls natively.
func (vm *VM) handleStringMethod(str, methodName, descriptor string, args []Value) (Value, error) {
	switch methodName {
	case "length":
		return IntValue(int32(len(str))), nil
	case "charAt":
		idx := int(args[0].Int)
		if idx < 0 || idx >= len(str) {
			return Value{}, NewJavaException("java/lang/StringIndexOutOfBoundsException")
		}
		return IntValue(int32(str[idx])), nil
	case "substring":
		if descriptor == "(I)Ljava/lang/String;" {
			begin := int(args[0].Int)
			if begin < 0 || begin > len(str) {
				return Value{}, NewJavaException("java/lang/StringIndexOutOfBoundsException")
			}
			return RefValue(str[begin:]), nil
		}
		// (II)Ljava/lang/String;
		begin := int(args[0].Int)
		end := int(args[1].Int)
		if begin < 0 || end > len(str) || begin > end {
			return Value{}, NewJavaException("java/lang/StringIndexOutOfBoundsException")
		}
		return RefValue(str[begin:end]), nil
	case "indexOf":
		if descriptor == "(Ljava/lang/String;)I" {
			target, _ := args[0].Ref.(string)
			return IntValue(int32(strings.Index(str, target))), nil
		}
		if descriptor == "(I)I" {
			ch := rune(args[0].Int)
			return IntValue(int32(strings.IndexRune(str, ch))), nil
		}
		return IntValue(-1), nil
	case "contains":
		target, _ := args[0].Ref.(string)
		if strings.Contains(str, target) {
			return IntValue(1), nil
		}
		return IntValue(0), nil
	case "equals":
		if args[0].Type == TypeNull {
			return IntValue(0), nil
		}
		other, ok := args[0].Ref.(string)
		if ok && str == other {
			return IntValue(1), nil
		}
		return IntValue(0), nil
	case "toUpperCase":
		return RefValue(strings.ToUpper(str)), nil
	case "toLowerCase":
		return RefValue(strings.ToLower(str)), nil
	case "trim":
		return RefValue(strings.TrimSpace(str)), nil
	case "replace":
		if descriptor == "(CC)Ljava/lang/String;" {
			oldCh := rune(args[0].Int)
			newCh := rune(args[1].Int)
			return RefValue(strings.ReplaceAll(str, string(oldCh), string(newCh))), nil
		}
		// (Ljava/lang/CharSequence;Ljava/lang/CharSequence;)Ljava/lang/String;
		if len(args) >= 2 {
			oldStr, _ := args[0].Ref.(string)
			newStr, _ := args[1].Ref.(string)
			return RefValue(strings.ReplaceAll(str, oldStr, newStr)), nil
		}
		return RefValue(str), nil
	case "isEmpty":
		if len(str) == 0 {
			return IntValue(1), nil
		}
		return IntValue(0), nil
	case "hashCode":
		h := int32(0)
		for _, c := range str {
			h = 31*h + int32(c)
		}
		return IntValue(h), nil
	case "toString":
		return RefValue(str), nil
	case "startsWith":
		prefix, _ := args[0].Ref.(string)
		if strings.HasPrefix(str, prefix) {
			return IntValue(1), nil
		}
		return IntValue(0), nil
	case "endsWith":
		suffix, _ := args[0].Ref.(string)
		if strings.HasSuffix(str, suffix) {
			return IntValue(1), nil
		}
		return IntValue(0), nil
	case "toCharArray":
		chars := make([]Value, len(str))
		for i, c := range str {
			chars[i] = IntValue(int32(c))
		}
		return RefValue(&JArray{Elements: chars}), nil
	case "getBytes":
		bytes := make([]Value, len(str))
		for i := 0; i < len(str); i++ {
			bytes[i] = IntValue(int32(str[i]))
		}
		return RefValue(&JArray{Elements: bytes}), nil
	case "compareTo":
		other, _ := args[0].Ref.(string)
		return IntValue(int32(strings.Compare(str, other))), nil
	case "intern":
		return RefValue(str), nil
	}
	return Value{}, fmt.Errorf("String method not implemented: %s:%s", methodName, descriptor)
}

// handleStringValueOf handles String.valueOf static method calls natively.
func (vm *VM) handleStringValueOf(descriptor string, args []Value) (Value, error) {
	switch descriptor {
	case "(I)Ljava/lang/String;":
		return RefValue(fmt.Sprintf("%d", args[0].Int)), nil
	case "(J)Ljava/lang/String;":
		return RefValue(fmt.Sprintf("%d", args[0].Long)), nil
	case "(D)Ljava/lang/String;":
		return RefValue(fmt.Sprintf("%v", args[0].Double)), nil
	case "(F)Ljava/lang/String;":
		return RefValue(fmt.Sprintf("%v", args[0].Float)), nil
	case "(Z)Ljava/lang/String;":
		if args[0].Int != 0 {
			return RefValue("true"), nil
		}
		return RefValue("false"), nil
	case "(C)Ljava/lang/String;":
		return RefValue(string(rune(args[0].Int))), nil
	case "(Ljava/lang/Object;)Ljava/lang/String;":
		if args[0].Type == TypeNull {
			return RefValue("null"), nil
		}
		if s, ok := args[0].Ref.(string); ok {
			return RefValue(s), nil
		}
		return RefValue(vm.valueToString(args[0])), nil
	}
	return Value{}, fmt.Errorf("String.valueOf not implemented for %s", descriptor)
}

// handleCollectionsSort handles Collections.sort natively.
func (vm *VM) handleCollectionsSort(frame *Frame, descriptor string, args []Value) (Value, bool, error) {
	list := args[0]
	obj, ok := list.Ref.(*JObject)
	if !ok {
		return Value{}, false, fmt.Errorf("Collections.sort: list is not a JObject")
	}
	elemData, ok := obj.Fields["elementData"]
	if !ok {
		return Value{}, false, fmt.Errorf("Collections.sort: no elementData field")
	}
	arr, ok := elemData.Ref.(*JArray)
	if !ok {
		return Value{}, false, fmt.Errorf("Collections.sort: elementData is not a JArray")
	}
	size := int(obj.Fields["size"].Int)

	if descriptor == "(Ljava/util/List;)V" {
		// Natural ordering
		sort.SliceStable(arr.Elements[:size], func(i, j int) bool {
			return vm.compareNatural(arr.Elements[i], arr.Elements[j]) < 0
		})
	} else {
		// With Comparator — use lambda if available
		comparator := args[1]
		if comparator.Type != TypeNull {
			sort.SliceStable(arr.Elements[:size], func(i, j int) bool {
				result, err := vm.invokeComparator(comparator, arr.Elements[i], arr.Elements[j])
				if err != nil {
					return false
				}
				return result < 0
			})
		} else {
			sort.SliceStable(arr.Elements[:size], func(i, j int) bool {
				return vm.compareNatural(arr.Elements[i], arr.Elements[j]) < 0
			})
		}
	}
	return Value{}, false, nil
}

// handleArrayListSort handles ArrayList.sort(Comparator) natively.
func (vm *VM) handleArrayListSort(frame *Frame, obj *JObject, args []Value) (Value, bool, error) {
	elemData, ok := obj.Fields["elementData"]
	if !ok {
		return Value{}, false, fmt.Errorf("ArrayList.sort: no elementData field")
	}
	arr, ok := elemData.Ref.(*JArray)
	if !ok {
		return Value{}, false, fmt.Errorf("ArrayList.sort: elementData is not a JArray")
	}
	size := int(obj.Fields["size"].Int)

	if len(args) > 0 && args[0].Type != TypeNull {
		comparator := args[0]
		sort.SliceStable(arr.Elements[:size], func(i, j int) bool {
			result, err := vm.invokeComparator(comparator, arr.Elements[i], arr.Elements[j])
			if err != nil {
				return false
			}
			return result < 0
		})
	} else {
		sort.SliceStable(arr.Elements[:size], func(i, j int) bool {
			return vm.compareNatural(arr.Elements[i], arr.Elements[j]) < 0
		})
	}
	// Increment modCount
	if mc, ok := obj.Fields["modCount"]; ok {
		obj.Fields["modCount"] = IntValue(mc.Int + 1)
	}
	return Value{}, false, nil
}

// compareNatural compares two Values using natural ordering (Comparable).
func (vm *VM) compareNatural(a, b Value) int {
	// String comparison
	if aStr, ok := a.Ref.(string); ok {
		if bStr, ok := b.Ref.(string); ok {
			return strings.Compare(aStr, bStr)
		}
	}
	// Boxed Integer comparison
	if aObj, ok := a.Ref.(*JObject); ok {
		if bObj, ok := b.Ref.(*JObject); ok {
			aVal, aHas := aObj.Fields["value"]
			bVal, bHas := bObj.Fields["value"]
			if aHas && bHas {
				switch {
				case aVal.Type == TypeInt && bVal.Type == TypeInt:
					return int(vm.compareInt32(aVal.Int, bVal.Int))
				case aVal.Type == TypeLong && bVal.Type == TypeLong:
					if aVal.Long < bVal.Long {
						return -1
					} else if aVal.Long > bVal.Long {
						return 1
					}
					return 0
				case aVal.Type == TypeDouble && bVal.Type == TypeDouble:
					if aVal.Double < bVal.Double {
						return -1
					} else if aVal.Double > bVal.Double {
						return 1
					}
					return 0
				}
			}
		}
	}
	return 0
}

// invokeComparator calls a Comparator's compare(Object, Object) method.
func (vm *VM) invokeComparator(comparator Value, a, b Value) (int32, error) {
	if obj, ok := comparator.Ref.(*JObject); ok && obj.LambdaTarget != nil {
		lt := obj.LambdaTarget
		cf, method, err := vm.resolveMethod(lt.TargetClass, lt.TargetMethod, lt.TargetDesc)
		if err != nil {
			return 0, err
		}
		fullArgs := make([]Value, 0, len(lt.CapturedArgs)+2)
		fullArgs = append(fullArgs, lt.CapturedArgs...)
		fullArgs = append(fullArgs, a, b)
		retVal, err := vm.executeMethod(cf, method, fullArgs)
		if err != nil {
			return 0, err
		}
		return retVal.Int, nil
	}
	if obj, ok := comparator.Ref.(*JObject); ok {
		cf, method, err := vm.resolveMethod(obj.ClassName, "compare", "(Ljava/lang/Object;Ljava/lang/Object;)I")
		if err != nil {
			return 0, err
		}
		retVal, err := vm.executeMethod(cf, method, []Value{comparator, a, b})
		if err != nil {
			return 0, err
		}
		return retVal.Int, nil
	}
	return 0, nil
}

// compareInt32 compares two int32 values.
func (vm *VM) compareInt32(a, b int32) int32 {
	if a < b {
		return -1
	} else if a > b {
		return 1
	}
	return 0
}

// handleBoxedType handles methods on boxed types (Integer, Long, Double, etc.)
func (vm *VM) handleBoxedType(frame *Frame, obj *JObject, methodName, descriptor string, args []Value) (Value, bool, error) {
	val, hasValue := obj.Fields["value"]
	if !hasValue {
		return Value{}, false, nil // not handled
	}
	switch methodName {
	case "intValue":
		if val.Type == TypeInt {
			frame.Push(IntValue(val.Int))
			return Value{}, true, nil
		}
	case "longValue":
		if val.Type == TypeLong {
			frame.Push(LongValue(val.Long))
			return Value{}, true, nil
		}
		if val.Type == TypeInt {
			frame.Push(LongValue(int64(val.Int)))
			return Value{}, true, nil
		}
	case "doubleValue":
		if val.Type == TypeDouble {
			frame.Push(DoubleValue(val.Double))
			return Value{}, true, nil
		}
		if val.Type == TypeInt {
			frame.Push(DoubleValue(float64(val.Int)))
			return Value{}, true, nil
		}
	case "floatValue":
		if val.Type == TypeFloat {
			frame.Push(FloatValue(val.Float))
			return Value{}, true, nil
		}
	case "compareTo":
		if len(args) > 0 {
			if other, ok := args[0].Ref.(*JObject); ok {
				if otherVal, ok := other.Fields["value"]; ok {
					frame.Push(IntValue(vm.compareInt32(val.Int, otherVal.Int)))
					return Value{}, true, nil
				}
			}
		}
	case "equals":
		if len(args) > 0 {
			if args[0].Type == TypeNull {
				frame.Push(IntValue(0))
				return Value{}, true, nil
			}
			if other, ok := args[0].Ref.(*JObject); ok {
				if otherVal, ok := other.Fields["value"]; ok && val.Type == otherVal.Type && val.Int == otherVal.Int {
					frame.Push(IntValue(1))
					return Value{}, true, nil
				}
			}
			frame.Push(IntValue(0))
			return Value{}, true, nil
		}
	case "hashCode":
		frame.Push(IntValue(val.Int))
		return Value{}, true, nil
	case "toString":
		frame.Push(RefValue(fmt.Sprintf("%d", val.Int)))
		return Value{}, true, nil
	}
	return Value{}, false, nil // not handled
}
