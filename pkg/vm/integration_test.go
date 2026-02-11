package vm

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

const testJmodPath = "/usr/lib/jvm/java-17-openjdk-arm64/jmods/java.base.jmod"

// runClass parses a .class file, loads it into a VM with class loader, executes it,
// and returns the captured stdout output.
func runClass(t *testing.T, classFilePath string) string {
	t.Helper()

	classPath := filepath.Dir(classFilePath)
	className := strings.TrimSuffix(filepath.Base(classFilePath), ".class")

	bootstrap := NewJmodClassLoader(testJmodPath)
	userCL := NewUserClassLoader(classPath, bootstrap)

	var buf bytes.Buffer
	v := NewVM(userCL)
	v.Stdout = &buf

	err := v.Execute(className)
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

func TestSwitch(t *testing.T) {
	got := runClass(t, "../../testdata/Switch.class")
	want := "5\n4\n1\n"
	if got != want {
		t.Errorf("Switch output:\ngot  %q\nwant %q", got, want)
	}
}

func TestSort(t *testing.T) {
	got := runClass(t, "../../testdata/Sort.class")
	want := "1\n2\n3\n4\n5\n"
	if got != want {
		t.Errorf("Sort output:\ngot  %q\nwant %q", got, want)
	}
}

func TestInheritance(t *testing.T) {
	got := runClass(t, "../../testdata/Inheritance.class")
	want := "1\n2\n"
	if got != want {
		t.Errorf("Inheritance output:\ngot  %q\nwant %q", got, want)
	}
}

func TestTryCatch(t *testing.T) {
	got := runClass(t, "../../testdata/TryCatch.class")
	want := "5\n-1\n0\n"
	if got != want {
		t.Errorf("TryCatch output:\ngot  %q\nwant %q", got, want)
	}
}

func TestInterface(t *testing.T) {
	got := runClass(t, "../../testdata/Interface.class")
	want := "7\n12\n"
	if got != want {
		t.Errorf("Interface output:\ngot  %q\nwant %q", got, want)
	}
}

func TestDoubleArith(t *testing.T) {
	got := runClass(t, "../../testdata/DoubleArith.class")
	want := "5\n1\n7\n1\n9\n42\n"
	if got != want {
		t.Errorf("DoubleArith output:\ngot  %q\nwant %q", got, want)
	}
}

func TestLongArith(t *testing.T) {
	got := runClass(t, "../../testdata/LongArith.class")
	want := "3000000000\n3000000000\n2000000000\n"
	if got != want {
		t.Errorf("LongArith output:\ngot  %q\nwant %q", got, want)
	}
}

func TestStringConcat(t *testing.T) {
	got := runClass(t, "../../testdata/StringConcat.class")
	want := "Hello World\nx=42\n3+4=7\n"
	if got != want {
		t.Errorf("StringConcat output:\ngot  %q\nwant %q", got, want)
	}
}

func TestLambda(t *testing.T) {
	got := runClass(t, "../../testdata/Lambda.class")
	want := "7\n12\n"
	if got != want {
		t.Errorf("Lambda output:\ngot  %q\nwant %q", got, want)
	}
}

func TestStringConcat2(t *testing.T) {
	got := runClass(t, "../../testdata/StringConcat2.class")
	want := "Hello World\nn=42\n"
	if got != want {
		t.Errorf("StringConcat2 output:\ngot  %q\nwant %q", got, want)
	}
}

func TestFinally(t *testing.T) {
	got := runClass(t, "../../testdata/Finally.class")
	want := "1\n2\n3\n4\n5\n10\n"
	if got != want {
		t.Errorf("Finally output:\ngot  %q\nwant %q", got, want)
	}
}

func TestEnumTest(t *testing.T) {
	got := runClass(t, "../../testdata/EnumTest.class")
	want := "1\nGREEN\n3\n"
	if got != want {
		t.Errorf("EnumTest output:\ngot  %q\nwant %q", got, want)
	}
}
