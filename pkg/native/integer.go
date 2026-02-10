package native

// NativeInteger represents a java.lang.Integer.
type NativeInteger struct {
	Value int32
}

// IntegerValueOf creates a NativeInteger (boxing).
func IntegerValueOf(v int32) *NativeInteger {
	return &NativeInteger{Value: v}
}

// IntegerIntValue returns the int32 value of a NativeInteger (unboxing).
func IntegerIntValue(ni *NativeInteger) int32 {
	return ni.Value
}
