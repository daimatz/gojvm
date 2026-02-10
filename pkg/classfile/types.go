package classfile

// Access flags
const (
	AccPublic = 0x0001
	AccStatic = 0x0008
	AccSuper  = 0x0020
)

// ClassFile represents a parsed .class file.
type ClassFile struct {
	MinorVersion uint16
	MajorVersion uint16
	ConstantPool []ConstantPoolEntry
	AccessFlags  uint16
	ThisClass    uint16
	SuperClass   uint16
	Interfaces   []uint16
	Fields       []FieldInfo
	Methods      []MethodInfo
}

// ConstantPoolEntry is an interface implemented by all constant pool types.
type ConstantPoolEntry interface {
	Tag() uint8
}

type ConstantUtf8 struct {
	Value string
}

func (c *ConstantUtf8) Tag() uint8 { return TagUtf8 }

type ConstantInteger struct {
	Value int32
}

func (c *ConstantInteger) Tag() uint8 { return TagInteger }

type ConstantClass struct {
	NameIndex uint16
}

func (c *ConstantClass) Tag() uint8 { return TagClass }

type ConstantString struct {
	StringIndex uint16
}

func (c *ConstantString) Tag() uint8 { return TagString }

type ConstantFieldref struct {
	ClassIndex       uint16
	NameAndTypeIndex uint16
}

func (c *ConstantFieldref) Tag() uint8 { return TagFieldref }

type ConstantMethodref struct {
	ClassIndex       uint16
	NameAndTypeIndex uint16
}

func (c *ConstantMethodref) Tag() uint8 { return TagMethodref }

type ConstantNameAndType struct {
	NameIndex       uint16
	DescriptorIndex uint16
}

func (c *ConstantNameAndType) Tag() uint8 { return TagNameAndType }

// MethodInfo represents a method in a class file.
type MethodInfo struct {
	AccessFlags uint16
	Name        string
	Descriptor  string
	Attributes  []AttributeInfo
	Code        *CodeAttribute
}

// FieldInfo represents a field in a class file.
type FieldInfo struct {
	AccessFlags uint16
	Name        string
	Descriptor  string
	Attributes  []AttributeInfo
}

// AttributeInfo represents a raw attribute.
type AttributeInfo struct {
	Name string
	Data []byte
}

// CodeAttribute represents the Code attribute of a method.
type CodeAttribute struct {
	MaxStack  uint16
	MaxLocals uint16
	Code      []byte
}
