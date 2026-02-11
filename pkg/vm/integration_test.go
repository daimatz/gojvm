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

func TestForEach(t *testing.T) {
	got := runClass(t, "../../testdata/ForEach.class")
	want := "60\nHello\nWorld\n"
	if got != want {
		t.Errorf("ForEach output:\ngot  %q\nwant %q", got, want)
	}
}

func TestStringMethods(t *testing.T) {
	got := runClass(t, "../../testdata/StringMethods.class")
	want := "13\nH\nWorld!\nHello\n7\ntrue\ntrue\nfalse\nHELLO, WORLD!\nhello, world!\nHello, World!\nhi\nHello; World!\n42\nfalse\ntrue\n"
	if got != want {
		t.Errorf("StringMethods output:\ngot  %q\nwant %q", got, want)
	}
}

func TestArrayListTest(t *testing.T) {
	got := runClass(t, "../../testdata/ArrayListTest.class")
	want := "3\nAlice\nCharlie\nBeth\nAlice\nBeth\nCharlie\n"
	if got != want {
		t.Errorf("ArrayListTest output:\ngot  %q\nwant %q", got, want)
	}
}

func TestStaticInit(t *testing.T) {
	got := runClass(t, "../../testdata/StaticInit.class")
	want := "10\nhello\n11\n12\n12\n"
	if got != want {
		t.Errorf("StaticInit output:\ngot  %q\nwant %q", got, want)
	}
}

func TestMultiArray(t *testing.T) {
	got := runClass(t, "../../testdata/MultiArray.class")
	want := "1\n5\n9\n3\n4\n20\n50\n"
	if got != want {
		t.Errorf("MultiArray output:\ngot  %q\nwant %q", got, want)
	}
}

func TestTypeCasting(t *testing.T) {
	got := runClass(t, "../../testdata/TypeCasting.class")
	want := "Woof\ntrue\nfalse\ntrue\nfetching\nString: hello\nDog: Rex\nCat: Whiskers\n"
	if got != want {
		t.Errorf("TypeCasting output:\ngot  %q\nwant %q", got, want)
	}
}

func TestHashMapIteration(t *testing.T) {
	got := runClass(t, "../../testdata/HashMapIteration.class")
	want := "3\n85\ntrue\nfalse\nAlice=90\nBob=85\nCharlie=95\n"
	if got != want {
		t.Errorf("HashMapIteration output:\ngot  %q\nwant %q", got, want)
	}
}

func TestCollectionsSort(t *testing.T) {
	got := runClass(t, "../../testdata/CollectionsSortTest.class")
	want := "Alice\nBob\nCharlie\n10\n20\n30\n"
	if got != want {
		t.Errorf("CollectionsSort output:\ngot  %q\nwant %q", got, want)
	}
}

func TestVarargs(t *testing.T) {
	got := runClass(t, "../../testdata/Varargs.class")
	want := "6\n100\na, b, c\n"
	if got != want {
		t.Errorf("Varargs output:\ngot  %q\nwant %q", got, want)
	}
}

func TestCustomIterator(t *testing.T) {
	got := runClass(t, "../../testdata/CustomIterator.class")
	want := "X\nY\nZ\n"
	if got != want {
		t.Errorf("CustomIterator output:\ngot  %q\nwant %q", got, want)
	}
}

func TestAbstractClass(t *testing.T) {
	got := runClass(t, "../../testdata/AbstractClass.class")
	want := "Circle: 78.53975\nRect: 12.0\ntrue\ntrue\n"
	if got != want {
		t.Errorf("AbstractClass output:\ngot  %q\nwant %q", got, want)
	}
}

func TestStackTrace(t *testing.T) {
	got := runClass(t, "../../testdata/StackTrace.class")
	want := "120\nnegative: -1\n"
	if got != want {
		t.Errorf("StackTrace output:\ngot  %q\nwant %q", got, want)
	}
}

func TestGenericClass(t *testing.T) {
	got := runClass(t, "../../testdata/GenericClass.class")
	want := "hello\n42\n(hello, 42)\ndefault\n30\n"
	if got != want {
		t.Errorf("GenericClass output:\ngot  %q\nwant %q", got, want)
	}
}

func TestTryWithResources(t *testing.T) {
	got := runClass(t, "../../testdata/TryWithResources.class")
	want := "open A\nuse A\nclose A\ndone\n"
	if got != want {
		t.Errorf("TryWithResources output:\ngot  %q\nwant %q", got, want)
	}
}

func TestStringFormat(t *testing.T) {
	got := runClass(t, "../../testdata/StringFormat.class")
	want := "x=42\npi=3.14\nHello World!\ntrue\nA\n100\n123\n456\n2147483647\n"
	if got != want {
		t.Errorf("StringFormat output:\ngot  %q\nwant %q", got, want)
	}
}

func TestNestedLoop(t *testing.T) {
	got := runClass(t, "../../testdata/NestedLoop.class")
	want := "19\n22\n43\n50\napple:3\nbanana:2\ncherry:1\n"
	if got != want {
		t.Errorf("NestedLoop output:\ngot  %q\nwant %q", got, want)
	}
}

func TestBitwiseOps(t *testing.T) {
	got := runClass(t, "../../testdata/BitwiseOps.class")
	want := "8\n14\n6\n-1\n1024\n-32\n1073741792\n1099511627776\n1048576\n"
	if got != want {
		t.Errorf("BitwiseOps output:\ngot  %q\nwant %q", got, want)
	}
}

func TestFloatArith(t *testing.T) {
	got := runClass(t, "../../testdata/FloatArith.class")
	want := "5\n1\n7\n1\n100\n1\n15\n"
	if got != want {
		t.Errorf("FloatArith output:\ngot  %q\nwant %q", got, want)
	}
}

func TestRecursiveDS(t *testing.T) {
	got := runClass(t, "../../testdata/RecursiveDS.class")
	want := "6\n3\n2\n1\n3628800\n"
	if got != want {
		t.Errorf("RecursiveDS output:\ngot  %q\nwant %q", got, want)
	}
}

func TestComparableTest(t *testing.T) {
	got := runClass(t, "../../testdata/ComparableTest.class")
	want := "Alice:95\nDiana:90\nBob:85\nCharlie:85\n"
	if got != want {
		t.Errorf("ComparableTest output:\ngot  %q\nwant %q", got, want)
	}
}

func TestStringBuilderTest(t *testing.T) {
	got := runClass(t, "../../testdata/StringBuilderTest.class")
	want := "Hello World\nn=42,pi=3.14\n11\nH\nabc\n"
	if got != want {
		t.Errorf("StringBuilderTest output:\ngot  %q\nwant %q", got, want)
	}
}

func TestMathTest(t *testing.T) {
	got := runClass(t, "../../testdata/MathTest.class")
	want := "42\n10\n7\n3\n1024\n5\n3\n4\n"
	if got != want {
		t.Errorf("MathTest output:\ngot  %q\nwant %q", got, want)
	}
}

func TestArrayCopyTest(t *testing.T) {
	got := runClass(t, "../../testdata/ArrayCopyTest.class")
	want := "1\n2\n3\n4\n5\n20\n50\nBob\n"
	if got != want {
		t.Errorf("ArrayCopyTest output:\ngot  %q\nwant %q", got, want)
	}
}

func TestExceptionChain(t *testing.T) {
	got := runClass(t, "../../testdata/ExceptionChain.class")
	want := "ok:20\nillegal:zero\nruntime:negative\nfinally1\ninner\ncaught\n"
	if got != want {
		t.Errorf("ExceptionChain output:\ngot  %q\nwant %q", got, want)
	}
}
