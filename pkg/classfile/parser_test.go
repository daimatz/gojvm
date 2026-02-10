package classfile

import (
	"os"
	"testing"
)

func TestParseClassFile(t *testing.T) {
	f, err := os.Open("../../testdata/Hello.class")
	if err != nil {
		t.Fatalf("failed to open Hello.class: %v", err)
	}
	defer f.Close()

	cf, err := Parse(f)
	if err != nil {
		t.Fatalf("failed to parse Hello.class: %v", err)
	}

	// マジックナンバーは Parse 内で検証済み (不一致ならエラー)
	// ここでは正常にパースできたことが検証になる

	// メジャーバージョンの検証 (Java 8 = 52, Java 17 = 61)
	if cf.MajorVersion < 52 {
		t.Errorf("major version: got %d, want >= 52", cf.MajorVersion)
	}

	// this_class が "Hello" を指すこと
	className, err := GetClassName(cf.ConstantPool, cf.ThisClass)
	if err != nil {
		t.Fatalf("resolving this_class: %v", err)
	}
	if className != "Hello" {
		t.Errorf("this_class: got %q, want %q", className, "Hello")
	}

	// main メソッドが存在すること
	mainMethod := cf.FindMethod("main", "([Ljava/lang/String;)V")
	if mainMethod == nil {
		t.Fatal("main method not found")
	}

	// main メソッドのディスクリプタが正しいこと
	if mainMethod.Descriptor != "([Ljava/lang/String;)V" {
		t.Errorf("main descriptor: got %q, want %q", mainMethod.Descriptor, "([Ljava/lang/String;)V")
	}

	// main メソッドの Code 属性が存在すること
	if mainMethod.Code == nil {
		t.Fatal("main method has no Code attribute")
	}

	if len(mainMethod.Code.Code) == 0 {
		t.Error("Code attribute has empty bytecode")
	}

	if mainMethod.Code.MaxStack == 0 {
		t.Error("Code attribute has MaxStack == 0")
	}

	if mainMethod.Code.MaxLocals == 0 {
		t.Error("Code attribute has MaxLocals == 0")
	}
}

func TestParseAddClassFile(t *testing.T) {
	f, err := os.Open("../../testdata/Add.class")
	if err != nil {
		t.Fatalf("failed to open Add.class: %v", err)
	}
	defer f.Close()

	cf, err := Parse(f)
	if err != nil {
		t.Fatalf("failed to parse Add.class: %v", err)
	}

	className, err := GetClassName(cf.ConstantPool, cf.ThisClass)
	if err != nil {
		t.Fatalf("resolving this_class: %v", err)
	}
	if className != "Add" {
		t.Errorf("this_class: got %q, want %q", className, "Add")
	}

	// main メソッドが存在すること
	if cf.FindMethod("main", "([Ljava/lang/String;)V") == nil {
		t.Error("main method not found")
	}

	// add メソッドが (II)I ディスクリプタで存在すること
	addMethod := cf.FindMethod("add", "(II)I")
	if addMethod == nil {
		t.Error("add(II)I method not found")
	}

	// add メソッドの Code 属性が存在すること
	if addMethod != nil && addMethod.Code == nil {
		t.Error("add method has no Code attribute")
	}
}

func TestParseInvalidMagic(t *testing.T) {
	// 不正なバイト列を Parse に渡してエラーになることを確認
	f, err := os.CreateTemp("", "invalid*.class")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	defer os.Remove(f.Name())

	// 不正なマジックナンバーを書き込む
	f.Write([]byte{0xDE, 0xAD, 0xBE, 0xEF})
	f.Close()

	r, err := os.Open(f.Name())
	if err != nil {
		t.Fatalf("opening temp file: %v", err)
	}
	defer r.Close()

	_, err = Parse(r)
	if err == nil {
		t.Error("expected error for invalid magic number, got nil")
	}
}
