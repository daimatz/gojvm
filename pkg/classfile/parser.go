package classfile

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

const classMagic = 0xCAFEBABE

// ParseFile opens and parses a .class file from the given path.
func ParseFile(path string) (*ClassFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

// Parse reads a .class file from the given reader and returns a ClassFile.
func Parse(r io.Reader) (*ClassFile, error) {
	cf := &ClassFile{}

	// Magic number
	var magic uint32
	if err := binary.Read(r, binary.BigEndian, &magic); err != nil {
		return nil, fmt.Errorf("reading magic number: %w", err)
	}
	if magic != classMagic {
		return nil, fmt.Errorf("invalid magic number: 0x%X (expected 0xCAFEBABE)", magic)
	}

	// Version
	if err := binary.Read(r, binary.BigEndian, &cf.MinorVersion); err != nil {
		return nil, fmt.Errorf("reading minor version: %w", err)
	}
	if err := binary.Read(r, binary.BigEndian, &cf.MajorVersion); err != nil {
		return nil, fmt.Errorf("reading major version: %w", err)
	}

	// Constant pool
	var cpCount uint16
	if err := binary.Read(r, binary.BigEndian, &cpCount); err != nil {
		return nil, fmt.Errorf("reading constant pool count: %w", err)
	}
	pool, err := parseConstantPool(r, cpCount)
	if err != nil {
		return nil, fmt.Errorf("parsing constant pool: %w", err)
	}
	cf.ConstantPool = pool

	// Access flags, this_class, super_class
	if err := binary.Read(r, binary.BigEndian, &cf.AccessFlags); err != nil {
		return nil, fmt.Errorf("reading access flags: %w", err)
	}
	if err := binary.Read(r, binary.BigEndian, &cf.ThisClass); err != nil {
		return nil, fmt.Errorf("reading this_class: %w", err)
	}
	if err := binary.Read(r, binary.BigEndian, &cf.SuperClass); err != nil {
		return nil, fmt.Errorf("reading super_class: %w", err)
	}

	// Interfaces
	var interfacesCount uint16
	if err := binary.Read(r, binary.BigEndian, &interfacesCount); err != nil {
		return nil, fmt.Errorf("reading interfaces count: %w", err)
	}
	cf.Interfaces = make([]uint16, interfacesCount)
	for i := uint16(0); i < interfacesCount; i++ {
		if err := binary.Read(r, binary.BigEndian, &cf.Interfaces[i]); err != nil {
			return nil, fmt.Errorf("reading interface %d: %w", i, err)
		}
	}

	// Fields
	var fieldsCount uint16
	if err := binary.Read(r, binary.BigEndian, &fieldsCount); err != nil {
		return nil, fmt.Errorf("reading fields count: %w", err)
	}
	cf.Fields, err = parseFields(r, cf.ConstantPool, fieldsCount)
	if err != nil {
		return nil, fmt.Errorf("parsing fields: %w", err)
	}

	// Methods
	var methodsCount uint16
	if err := binary.Read(r, binary.BigEndian, &methodsCount); err != nil {
		return nil, fmt.Errorf("reading methods count: %w", err)
	}
	cf.Methods, err = parseMethods(r, cf.ConstantPool, methodsCount)
	if err != nil {
		return nil, fmt.Errorf("parsing methods: %w", err)
	}

	// Class-level attributes (parse BootstrapMethods, skip others)
	if err := cf.parseClassAttributes(r); err != nil {
		return nil, fmt.Errorf("parsing class attributes: %w", err)
	}

	return cf, nil
}

func parseFields(r io.Reader, pool []ConstantPoolEntry, count uint16) ([]FieldInfo, error) {
	fields := make([]FieldInfo, count)
	for i := uint16(0); i < count; i++ {
		var accessFlags, nameIndex, descIndex, attrCount uint16
		if err := binary.Read(r, binary.BigEndian, &accessFlags); err != nil {
			return nil, fmt.Errorf("reading field %d access flags: %w", i, err)
		}
		if err := binary.Read(r, binary.BigEndian, &nameIndex); err != nil {
			return nil, fmt.Errorf("reading field %d name index: %w", i, err)
		}
		if err := binary.Read(r, binary.BigEndian, &descIndex); err != nil {
			return nil, fmt.Errorf("reading field %d descriptor index: %w", i, err)
		}
		if err := binary.Read(r, binary.BigEndian, &attrCount); err != nil {
			return nil, fmt.Errorf("reading field %d attributes count: %w", i, err)
		}

		name, err := GetUtf8(pool, nameIndex)
		if err != nil {
			return nil, fmt.Errorf("resolving field %d name: %w", i, err)
		}
		desc, err := GetUtf8(pool, descIndex)
		if err != nil {
			return nil, fmt.Errorf("resolving field %d descriptor: %w", i, err)
		}

		attrs, err := parseAttributeInfos(r, pool, attrCount)
		if err != nil {
			return nil, fmt.Errorf("parsing field %d attributes: %w", i, err)
		}

		fields[i] = FieldInfo{
			AccessFlags: accessFlags,
			Name:        name,
			Descriptor:  desc,
			Attributes:  attrs,
		}
	}
	return fields, nil
}

func parseMethods(r io.Reader, pool []ConstantPoolEntry, count uint16) ([]MethodInfo, error) {
	methods := make([]MethodInfo, count)
	for i := uint16(0); i < count; i++ {
		var accessFlags, nameIndex, descIndex, attrCount uint16
		if err := binary.Read(r, binary.BigEndian, &accessFlags); err != nil {
			return nil, fmt.Errorf("reading method %d access flags: %w", i, err)
		}
		if err := binary.Read(r, binary.BigEndian, &nameIndex); err != nil {
			return nil, fmt.Errorf("reading method %d name index: %w", i, err)
		}
		if err := binary.Read(r, binary.BigEndian, &descIndex); err != nil {
			return nil, fmt.Errorf("reading method %d descriptor index: %w", i, err)
		}
		if err := binary.Read(r, binary.BigEndian, &attrCount); err != nil {
			return nil, fmt.Errorf("reading method %d attributes count: %w", i, err)
		}

		name, err := GetUtf8(pool, nameIndex)
		if err != nil {
			return nil, fmt.Errorf("resolving method %d name: %w", i, err)
		}
		desc, err := GetUtf8(pool, descIndex)
		if err != nil {
			return nil, fmt.Errorf("resolving method %d descriptor: %w", i, err)
		}

		attrs, err := parseAttributeInfos(r, pool, attrCount)
		if err != nil {
			return nil, fmt.Errorf("parsing method %d attributes: %w", i, err)
		}

		m := MethodInfo{
			AccessFlags: accessFlags,
			Name:        name,
			Descriptor:  desc,
			Attributes:  attrs,
		}

		// Extract Code attribute
		for _, attr := range attrs {
			if attr.Name == "Code" {
				code, err := parseCodeAttribute(attr.Data)
				if err != nil {
					return nil, fmt.Errorf("parsing Code attribute for method %s: %w", name, err)
				}
				m.Code = code
				break
			}
		}

		methods[i] = m
	}
	return methods, nil
}

func parseAttributeInfos(r io.Reader, pool []ConstantPoolEntry, count uint16) ([]AttributeInfo, error) {
	attrs := make([]AttributeInfo, count)
	for i := uint16(0); i < count; i++ {
		var nameIndex uint16
		if err := binary.Read(r, binary.BigEndian, &nameIndex); err != nil {
			return nil, fmt.Errorf("reading attribute %d name index: %w", i, err)
		}
		var length uint32
		if err := binary.Read(r, binary.BigEndian, &length); err != nil {
			return nil, fmt.Errorf("reading attribute %d length: %w", i, err)
		}
		data := make([]byte, length)
		if _, err := io.ReadFull(r, data); err != nil {
			return nil, fmt.Errorf("reading attribute %d data: %w", i, err)
		}

		name, err := GetUtf8(pool, nameIndex)
		if err != nil {
			return nil, fmt.Errorf("resolving attribute %d name: %w", i, err)
		}

		attrs[i] = AttributeInfo{Name: name, Data: data}
	}
	return attrs, nil
}

func parseCodeAttribute(data []byte) (*CodeAttribute, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("Code attribute too short: %d bytes", len(data))
	}

	maxStack := binary.BigEndian.Uint16(data[0:2])
	maxLocals := binary.BigEndian.Uint16(data[2:4])
	codeLength := binary.BigEndian.Uint32(data[4:8])

	if len(data) < 8+int(codeLength) {
		return nil, fmt.Errorf("Code attribute data too short for code_length %d", codeLength)
	}

	code := make([]byte, codeLength)
	copy(code, data[8:8+codeLength])

	// Parse exception table
	offset := 8 + int(codeLength)
	var handlers []ExceptionHandler
	if offset+2 <= len(data) {
		exTableLen := binary.BigEndian.Uint16(data[offset : offset+2])
		offset += 2
		handlers = make([]ExceptionHandler, exTableLen)
		for i := uint16(0); i < exTableLen; i++ {
			if offset+8 > len(data) {
				break
			}
			handlers[i] = ExceptionHandler{
				StartPC:   binary.BigEndian.Uint16(data[offset : offset+2]),
				EndPC:     binary.BigEndian.Uint16(data[offset+2 : offset+4]),
				HandlerPC: binary.BigEndian.Uint16(data[offset+4 : offset+6]),
				CatchType: binary.BigEndian.Uint16(data[offset+6 : offset+8]),
			}
			offset += 8
		}
	}

	return &CodeAttribute{
		MaxStack:          maxStack,
		MaxLocals:         maxLocals,
		Code:              code,
		ExceptionHandlers: handlers,
	}, nil
}

func (cf *ClassFile) parseClassAttributes(r io.Reader) error {
	var count uint16
	if err := binary.Read(r, binary.BigEndian, &count); err != nil {
		return err
	}
	for i := uint16(0); i < count; i++ {
		var nameIndex uint16
		if err := binary.Read(r, binary.BigEndian, &nameIndex); err != nil {
			return err
		}
		var length uint32
		if err := binary.Read(r, binary.BigEndian, &length); err != nil {
			return err
		}
		data := make([]byte, length)
		if _, err := io.ReadFull(r, data); err != nil {
			return err
		}
		name, err := GetUtf8(cf.ConstantPool, nameIndex)
		if err != nil {
			continue // skip unknown attributes
		}
		if name == "BootstrapMethods" {
			cf.BootstrapMethods, err = parseBootstrapMethods(data)
			if err != nil {
				return fmt.Errorf("parsing BootstrapMethods: %w", err)
			}
		}
	}
	return nil
}

func parseBootstrapMethods(data []byte) ([]BootstrapMethod, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("BootstrapMethods data too short")
	}
	numMethods := binary.BigEndian.Uint16(data[0:2])
	offset := 2
	methods := make([]BootstrapMethod, numMethods)
	for i := uint16(0); i < numMethods; i++ {
		if offset+4 > len(data) {
			return nil, fmt.Errorf("BootstrapMethods truncated at method %d", i)
		}
		methodRef := binary.BigEndian.Uint16(data[offset : offset+2])
		numArgs := binary.BigEndian.Uint16(data[offset+2 : offset+4])
		offset += 4
		args := make([]uint16, numArgs)
		for j := uint16(0); j < numArgs; j++ {
			if offset+2 > len(data) {
				return nil, fmt.Errorf("BootstrapMethods truncated at arg %d of method %d", j, i)
			}
			args[j] = binary.BigEndian.Uint16(data[offset : offset+2])
			offset += 2
		}
		methods[i] = BootstrapMethod{MethodRef: methodRef, BootstrapArguments: args}
	}
	return methods, nil
}

// ClassName returns the fully qualified name of this class.
func (cf *ClassFile) ClassName() (string, error) {
	return GetClassName(cf.ConstantPool, cf.ThisClass)
}

// FindMethod finds a method by name and descriptor.
func (cf *ClassFile) FindMethod(name, descriptor string) *MethodInfo {
	for i := range cf.Methods {
		if cf.Methods[i].Name == name && cf.Methods[i].Descriptor == descriptor {
			return &cf.Methods[i]
		}
	}
	return nil
}

// FindMethodByName finds a method by name only (first match).
func (cf *ClassFile) FindMethodByName(name string) *MethodInfo {
	for i := range cf.Methods {
		if cf.Methods[i].Name == name {
			return &cf.Methods[i]
		}
	}
	return nil
}
