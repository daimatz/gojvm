package classfile

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

// Constant pool tags
const (
	TagUtf8               = 1
	TagInteger            = 3
	TagFloat              = 4
	TagLong               = 5
	TagDouble             = 6
	TagClass              = 7
	TagString             = 8
	TagFieldref           = 9
	TagMethodref          = 10
	TagInterfaceMethodref = 11
	TagNameAndType        = 12
	TagMethodHandle       = 15
	TagMethodType         = 16
	TagDynamic            = 17
	TagInvokeDynamic      = 18
)

// parseConstantPool reads constant_pool_count-1 entries from the reader.
// The returned slice is 1-indexed: index 0 is nil.
func parseConstantPool(r io.Reader, count uint16) ([]ConstantPoolEntry, error) {
	pool := make([]ConstantPoolEntry, count)
	// pool[0] is unused (constant pool is 1-indexed)

	for i := uint16(1); i < count; i++ {
		var tag uint8
		if err := binary.Read(r, binary.BigEndian, &tag); err != nil {
			return nil, fmt.Errorf("reading constant pool tag at index %d: %w", i, err)
		}

		switch tag {
		case TagUtf8:
			var length uint16
			if err := binary.Read(r, binary.BigEndian, &length); err != nil {
				return nil, fmt.Errorf("reading Utf8 length at index %d: %w", i, err)
			}
			bytes := make([]byte, length)
			if _, err := io.ReadFull(r, bytes); err != nil {
				return nil, fmt.Errorf("reading Utf8 bytes at index %d: %w", i, err)
			}
			pool[i] = &ConstantUtf8{Value: string(bytes)}

		case TagInteger:
			var val int32
			if err := binary.Read(r, binary.BigEndian, &val); err != nil {
				return nil, fmt.Errorf("reading Integer at index %d: %w", i, err)
			}
			pool[i] = &ConstantInteger{Value: val}

		case TagFloat:
			var bits uint32
			if err := binary.Read(r, binary.BigEndian, &bits); err != nil {
				return nil, fmt.Errorf("reading Float at index %d: %w", i, err)
			}
			pool[i] = &ConstantFloat{Value: math.Float32frombits(bits)}

		case TagLong:
			var val int64
			if err := binary.Read(r, binary.BigEndian, &val); err != nil {
				return nil, fmt.Errorf("reading Long at index %d: %w", i, err)
			}
			pool[i] = &ConstantLong{Value: val}
			i++ // long takes 2 slots

		case TagDouble:
			var bits uint64
			if err := binary.Read(r, binary.BigEndian, &bits); err != nil {
				return nil, fmt.Errorf("reading Double at index %d: %w", i, err)
			}
			pool[i] = &ConstantDouble{Value: math.Float64frombits(bits)}
			i++ // double takes 2 slots

		case TagClass:
			var nameIndex uint16
			if err := binary.Read(r, binary.BigEndian, &nameIndex); err != nil {
				return nil, fmt.Errorf("reading Class at index %d: %w", i, err)
			}
			pool[i] = &ConstantClass{NameIndex: nameIndex}

		case TagString:
			var stringIndex uint16
			if err := binary.Read(r, binary.BigEndian, &stringIndex); err != nil {
				return nil, fmt.Errorf("reading String at index %d: %w", i, err)
			}
			pool[i] = &ConstantString{StringIndex: stringIndex}

		case TagFieldref:
			var classIndex, natIndex uint16
			if err := binary.Read(r, binary.BigEndian, &classIndex); err != nil {
				return nil, fmt.Errorf("reading Fieldref class_index at index %d: %w", i, err)
			}
			if err := binary.Read(r, binary.BigEndian, &natIndex); err != nil {
				return nil, fmt.Errorf("reading Fieldref name_and_type_index at index %d: %w", i, err)
			}
			pool[i] = &ConstantFieldref{ClassIndex: classIndex, NameAndTypeIndex: natIndex}

		case TagMethodref:
			var classIndex, natIndex uint16
			if err := binary.Read(r, binary.BigEndian, &classIndex); err != nil {
				return nil, fmt.Errorf("reading Methodref class_index at index %d: %w", i, err)
			}
			if err := binary.Read(r, binary.BigEndian, &natIndex); err != nil {
				return nil, fmt.Errorf("reading Methodref name_and_type_index at index %d: %w", i, err)
			}
			pool[i] = &ConstantMethodref{ClassIndex: classIndex, NameAndTypeIndex: natIndex}

		case TagInterfaceMethodref:
			var classIndex, natIndex uint16
			if err := binary.Read(r, binary.BigEndian, &classIndex); err != nil {
				return nil, fmt.Errorf("reading InterfaceMethodref class_index at index %d: %w", i, err)
			}
			if err := binary.Read(r, binary.BigEndian, &natIndex); err != nil {
				return nil, fmt.Errorf("reading InterfaceMethodref name_and_type_index at index %d: %w", i, err)
			}
			pool[i] = &ConstantInterfaceMethodref{ClassIndex: classIndex, NameAndTypeIndex: natIndex}

		case TagNameAndType:
			var nameIndex, descIndex uint16
			if err := binary.Read(r, binary.BigEndian, &nameIndex); err != nil {
				return nil, fmt.Errorf("reading NameAndType name_index at index %d: %w", i, err)
			}
			if err := binary.Read(r, binary.BigEndian, &descIndex); err != nil {
				return nil, fmt.Errorf("reading NameAndType descriptor_index at index %d: %w", i, err)
			}
			pool[i] = &ConstantNameAndType{NameIndex: nameIndex, DescriptorIndex: descIndex}

		case TagMethodHandle:
			// reference_kind (u1) + reference_index (u2) = 3 bytes
			skip := make([]byte, 3)
			if _, err := io.ReadFull(r, skip); err != nil {
				return nil, fmt.Errorf("reading MethodHandle at index %d: %w", i, err)
			}
			pool[i] = &constantPlaceholder{tag: tag}

		case TagMethodType:
			// descriptor_index (u2) = 2 bytes
			skip := make([]byte, 2)
			if _, err := io.ReadFull(r, skip); err != nil {
				return nil, fmt.Errorf("reading MethodType at index %d: %w", i, err)
			}
			pool[i] = &constantPlaceholder{tag: tag}

		case TagDynamic, TagInvokeDynamic:
			// bootstrap_method_attr_index (u2) + name_and_type_index (u2) = 4 bytes
			skip := make([]byte, 4)
			if _, err := io.ReadFull(r, skip); err != nil {
				return nil, fmt.Errorf("reading Dynamic/InvokeDynamic at index %d: %w", i, err)
			}
			pool[i] = &constantPlaceholder{tag: tag}

		default:
			return nil, fmt.Errorf("unknown constant pool tag %d at index %d", tag, i)
		}
	}

	return pool, nil
}

// constantPlaceholder is used for constant pool entries we don't fully parse.
type constantPlaceholder struct {
	tag uint8
}

func (c *constantPlaceholder) Tag() uint8 { return c.tag }

// GetUtf8 returns the Utf8 string at the given constant pool index.
func GetUtf8(pool []ConstantPoolEntry, index uint16) (string, error) {
	if int(index) >= len(pool) || pool[index] == nil {
		return "", fmt.Errorf("invalid constant pool index %d", index)
	}
	utf8, ok := pool[index].(*ConstantUtf8)
	if !ok {
		return "", fmt.Errorf("constant pool index %d is not Utf8 (tag=%d)", index, pool[index].Tag())
	}
	return utf8.Value, nil
}

// GetClassName returns the class name referenced by a CONSTANT_Class entry.
func GetClassName(pool []ConstantPoolEntry, classIndex uint16) (string, error) {
	if int(classIndex) >= len(pool) || pool[classIndex] == nil {
		return "", fmt.Errorf("invalid constant pool index %d", classIndex)
	}
	class, ok := pool[classIndex].(*ConstantClass)
	if !ok {
		return "", fmt.Errorf("constant pool index %d is not Class", classIndex)
	}
	return GetUtf8(pool, class.NameIndex)
}

// MethodRefInfo holds resolved method reference info.
type MethodRefInfo struct {
	ClassName  string
	MethodName string
	Descriptor string
}

// ResolveMethodref resolves a CONSTANT_Methodref entry.
func ResolveMethodref(pool []ConstantPoolEntry, index uint16) (*MethodRefInfo, error) {
	if int(index) >= len(pool) || pool[index] == nil {
		return nil, fmt.Errorf("invalid constant pool index %d", index)
	}
	mref, ok := pool[index].(*ConstantMethodref)
	if !ok {
		return nil, fmt.Errorf("constant pool index %d is not Methodref", index)
	}

	className, err := GetClassName(pool, mref.ClassIndex)
	if err != nil {
		return nil, fmt.Errorf("resolving Methodref class: %w", err)
	}

	if int(mref.NameAndTypeIndex) >= len(pool) || pool[mref.NameAndTypeIndex] == nil {
		return nil, fmt.Errorf("invalid NameAndType index %d", mref.NameAndTypeIndex)
	}
	nat, ok := pool[mref.NameAndTypeIndex].(*ConstantNameAndType)
	if !ok {
		return nil, fmt.Errorf("constant pool index %d is not NameAndType", mref.NameAndTypeIndex)
	}

	methodName, err := GetUtf8(pool, nat.NameIndex)
	if err != nil {
		return nil, fmt.Errorf("resolving method name: %w", err)
	}

	descriptor, err := GetUtf8(pool, nat.DescriptorIndex)
	if err != nil {
		return nil, fmt.Errorf("resolving method descriptor: %w", err)
	}

	return &MethodRefInfo{
		ClassName:  className,
		MethodName: methodName,
		Descriptor: descriptor,
	}, nil
}

// ResolveInterfaceMethodref resolves a CONSTANT_InterfaceMethodref entry.
func ResolveInterfaceMethodref(pool []ConstantPoolEntry, index uint16) (*MethodRefInfo, error) {
	if int(index) >= len(pool) || pool[index] == nil {
		return nil, fmt.Errorf("invalid constant pool index %d", index)
	}
	mref, ok := pool[index].(*ConstantInterfaceMethodref)
	if !ok {
		return nil, fmt.Errorf("constant pool index %d is not InterfaceMethodref", index)
	}

	className, err := GetClassName(pool, mref.ClassIndex)
	if err != nil {
		return nil, fmt.Errorf("resolving InterfaceMethodref class: %w", err)
	}

	if int(mref.NameAndTypeIndex) >= len(pool) || pool[mref.NameAndTypeIndex] == nil {
		return nil, fmt.Errorf("invalid NameAndType index %d", mref.NameAndTypeIndex)
	}
	nat, ok := pool[mref.NameAndTypeIndex].(*ConstantNameAndType)
	if !ok {
		return nil, fmt.Errorf("constant pool index %d is not NameAndType", mref.NameAndTypeIndex)
	}

	methodName, err := GetUtf8(pool, nat.NameIndex)
	if err != nil {
		return nil, fmt.Errorf("resolving method name: %w", err)
	}

	descriptor, err := GetUtf8(pool, nat.DescriptorIndex)
	if err != nil {
		return nil, fmt.Errorf("resolving method descriptor: %w", err)
	}

	return &MethodRefInfo{
		ClassName:  className,
		MethodName: methodName,
		Descriptor: descriptor,
	}, nil
}

// FieldRefInfo holds resolved field reference info.
type FieldRefInfo struct {
	ClassName  string
	FieldName  string
	Descriptor string
}

// ResolveFieldref resolves a CONSTANT_Fieldref entry.
func ResolveFieldref(pool []ConstantPoolEntry, index uint16) (*FieldRefInfo, error) {
	if int(index) >= len(pool) || pool[index] == nil {
		return nil, fmt.Errorf("invalid constant pool index %d", index)
	}
	fref, ok := pool[index].(*ConstantFieldref)
	if !ok {
		return nil, fmt.Errorf("constant pool index %d is not Fieldref", index)
	}

	className, err := GetClassName(pool, fref.ClassIndex)
	if err != nil {
		return nil, fmt.Errorf("resolving Fieldref class: %w", err)
	}

	if int(fref.NameAndTypeIndex) >= len(pool) || pool[fref.NameAndTypeIndex] == nil {
		return nil, fmt.Errorf("invalid NameAndType index %d", fref.NameAndTypeIndex)
	}
	nat, ok := pool[fref.NameAndTypeIndex].(*ConstantNameAndType)
	if !ok {
		return nil, fmt.Errorf("constant pool index %d is not NameAndType", fref.NameAndTypeIndex)
	}

	fieldName, err := GetUtf8(pool, nat.NameIndex)
	if err != nil {
		return nil, fmt.Errorf("resolving field name: %w", err)
	}

	descriptor, err := GetUtf8(pool, nat.DescriptorIndex)
	if err != nil {
		return nil, fmt.Errorf("resolving field descriptor: %w", err)
	}

	return &FieldRefInfo{
		ClassName:  className,
		FieldName:  fieldName,
		Descriptor: descriptor,
	}, nil
}
