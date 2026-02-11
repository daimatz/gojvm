package vm

// JObject represents a JVM object instance.
type JObject struct {
	ClassName string
	Fields    map[string]Value
}

// JArray represents a JVM reference array.
type JArray struct {
	Elements []Value
}
