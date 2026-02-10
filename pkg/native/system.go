package native

import (
	"fmt"
	"io"
)

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
