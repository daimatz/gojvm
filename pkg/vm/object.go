package vm

// JObject represents a JVM object instance.
type JObject struct {
	ClassName string
	Fields    map[string]Value
}
