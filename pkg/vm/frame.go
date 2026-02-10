package vm

import (
	"fmt"

	"github.com/daimatz/gojvm/pkg/classfile"
)

// ValueType represents the type of a Value on the stack or in local variables.
type ValueType int

const (
	TypeInt  ValueType = iota
	TypeRef
	TypeNull
)

// Value represents a value on the operand stack or in local variables.
type Value struct {
	Type ValueType
	Int  int32
	Ref  interface{}
}

// IntValue creates an integer Value.
func IntValue(v int32) Value {
	return Value{Type: TypeInt, Int: v}
}

// RefValue creates a reference Value.
func RefValue(ref interface{}) Value {
	return Value{Type: TypeRef, Ref: ref}
}

// NullValue creates a null reference Value.
func NullValue() Value {
	return Value{Type: TypeNull}
}

// Frame represents a stack frame for method execution.
type Frame struct {
	LocalVars    []Value
	OperandStack []Value
	SP           int
	Code         []byte
	PC           int
	Class        *classfile.ClassFile
}

// NewFrame creates a new Frame with the given parameters.
func NewFrame(maxLocals, maxStack uint16, code []byte, class *classfile.ClassFile) *Frame {
	return &Frame{
		LocalVars:    make([]Value, maxLocals),
		OperandStack: make([]Value, maxStack),
		SP:           0,
		Code:         code,
		PC:           0,
		Class:        class,
	}
}

// Push pushes a value onto the operand stack.
func (f *Frame) Push(v Value) {
	if f.SP >= len(f.OperandStack) {
		panic(fmt.Sprintf("operand stack overflow: SP=%d, max=%d", f.SP, len(f.OperandStack)))
	}
	f.OperandStack[f.SP] = v
	f.SP++
}

// Pop pops a value from the operand stack.
func (f *Frame) Pop() Value {
	if f.SP <= 0 {
		panic("operand stack underflow: SP=0")
	}
	f.SP--
	return f.OperandStack[f.SP]
}

// GetLocal returns the value at the given local variable index.
func (f *Frame) GetLocal(index int) Value {
	if index < 0 || index >= len(f.LocalVars) {
		panic(fmt.Sprintf("local variable index out of range: index=%d, max=%d", index, len(f.LocalVars)))
	}
	return f.LocalVars[index]
}

// SetLocal sets the value at the given local variable index.
func (f *Frame) SetLocal(index int, v Value) {
	if index < 0 || index >= len(f.LocalVars) {
		panic(fmt.Sprintf("local variable index out of range: index=%d, max=%d", index, len(f.LocalVars)))
	}
	f.LocalVars[index] = v
}

// ReadU8 reads a uint8 operand and advances PC.
func (f *Frame) ReadU8() uint8 {
	val := f.Code[f.PC]
	f.PC++
	return val
}

// ReadI8 reads an int8 operand and advances PC.
func (f *Frame) ReadI8() int8 {
	val := int8(f.Code[f.PC])
	f.PC++
	return val
}

// ReadU16 reads a uint16 operand (big-endian) and advances PC by 2.
func (f *Frame) ReadU16() uint16 {
	val := uint16(f.Code[f.PC])<<8 | uint16(f.Code[f.PC+1])
	f.PC += 2
	return val
}

// ReadI16 reads an int16 operand (big-endian) and advances PC by 2.
func (f *Frame) ReadI16() int16 {
	val := int16(f.Code[f.PC])<<8 | int16(f.Code[f.PC+1])
	f.PC += 2
	return val
}
