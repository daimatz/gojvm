package vm

import "fmt"

// JavaException represents a JVM exception being thrown.
type JavaException struct {
	Object *JObject
}

func (e *JavaException) Error() string {
	return fmt.Sprintf("JavaException: %s", e.Object.ClassName)
}

func NewJavaException(className string) *JavaException {
	return &JavaException{
		Object: &JObject{
			ClassName: className,
			Fields:    make(map[string]Value),
		},
	}
}
