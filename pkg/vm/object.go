package vm

// LambdaTarget holds the implementation target for a lambda proxy object.
type LambdaTarget struct {
	InterfaceName string
	MethodName    string
	TargetClass   string
	TargetMethod  string
	TargetDesc    string
	CapturedArgs  []Value
	ReferenceKind uint8
}

// JObject represents a JVM object instance.
type JObject struct {
	ClassName    string
	Fields       map[string]Value
	LambdaTarget *LambdaTarget
}

// JArray represents a JVM reference array.
type JArray struct {
	Elements []Value
}
