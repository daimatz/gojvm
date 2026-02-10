package native

// NativeHashMap represents a java.util.HashMap.
type NativeHashMap struct {
	Data map[interface{}]interface{}
}

// NewNativeHashMap creates a new NativeHashMap.
func NewNativeHashMap() *NativeHashMap {
	return &NativeHashMap{Data: make(map[interface{}]interface{})}
}

// NewHashMap is an alias for NewNativeHashMap (used by tests).
func NewHashMap() *NativeHashMap {
	return NewNativeHashMap()
}

// Get returns the value for the given key.
// If key is a *NativeInteger, its Value (int32) is used as the map key.
func (m *NativeHashMap) Get(key interface{}) interface{} {
	if ni, ok := key.(*NativeInteger); ok {
		return m.Data[ni.Value]
	}
	return m.Data[key]
}

// Put stores a key-value pair and returns the previous value.
// If key is a *NativeInteger, its Value (int32) is used as the map key.
func (m *NativeHashMap) Put(key, value interface{}) interface{} {
	var mapKey interface{} = key
	if ni, ok := key.(*NativeInteger); ok {
		mapKey = ni.Value
	}
	old := m.Data[mapKey]
	m.Data[mapKey] = value
	return old
}
