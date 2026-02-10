package vm

import (
	"bytes"
	"os"
	"testing"

	"github.com/daimatz/gojvm/pkg/classfile"
)

// runClass parses a .class file, loads it into a VM, executes it,
// and returns the captured stdout output.
func runClass(t *testing.T, classFilePath string) string {
	t.Helper()

	f, err := os.Open(classFilePath)
	if err != nil {
		t.Fatalf("failed to open %s: %v", classFilePath, err)
	}
	defer f.Close()

	cf, err := classfile.Parse(f)
	if err != nil {
		t.Fatalf("failed to parse %s: %v", classFilePath, err)
	}

	var buf bytes.Buffer
	v := NewVM(cf)
	v.Stdout = &buf

	err = v.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	return buf.String()
}

func TestHello(t *testing.T) {
	got := runClass(t, "../../testdata/Hello.class")
	want := "42\n"
	if got != want {
		t.Errorf("Hello output:\ngot  %q\nwant %q", got, want)
	}
}

func TestAdd(t *testing.T) {
	got := runClass(t, "../../testdata/Add.class")
	want := "7\n"
	if got != want {
		t.Errorf("Add output:\ngot  %q\nwant %q", got, want)
	}
}

func TestArithmetic(t *testing.T) {
	got := runClass(t, "../../testdata/Arithmetic.class")
	want := "13\n7\n30\n3\n1\n-10\n"
	if got != want {
		t.Errorf("Arithmetic output:\ngot  %q\nwant %q", got, want)
	}
}

func TestControlFlow(t *testing.T) {
	got := runClass(t, "../../testdata/ControlFlow.class")
	want := "5\n3\n120\n"
	if got != want {
		t.Errorf("ControlFlow output:\ngot  %q\nwant %q", got, want)
	}
}

func TestPrintString(t *testing.T) {
	got := runClass(t, "../../testdata/PrintString.class")
	want := "Hello, World!\n"
	if got != want {
		t.Errorf("PrintString output:\ngot  %q\nwant %q", got, want)
	}
}

func TestFib(t *testing.T) {
	got := runClass(t, "../../testdata/Fib.class")
	want := "89\n"
	if got != want {
		t.Errorf("Fib output:\ngot  %q\nwant %q", got, want)
	}
}
