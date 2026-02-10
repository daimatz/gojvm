package native

import (
	"fmt"
	"io"
)

// SystemOut is a placeholder object for java/lang/System.out.
type SystemOut struct{}

// PrintStream represents a java.io.PrintStream.
type PrintStream struct {
	Writer io.Writer
}

// Println prints a value followed by a newline.
func (ps *PrintStream) Println(args ...interface{}) {
	if len(args) == 0 {
		fmt.Fprintln(ps.Writer)
		return
	}
	fmt.Fprintln(ps.Writer, args[0])
}
