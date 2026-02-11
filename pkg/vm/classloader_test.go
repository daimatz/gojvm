package vm

import (
	"testing"
)

func TestJmodClassLoader(t *testing.T) {
	cl := NewJmodClassLoader(testJmodPath)

	t.Run("load Integer class", func(t *testing.T) {
		cf, err := cl.LoadClass("java/lang/Integer")
		if err != nil {
			t.Fatalf("failed to load java/lang/Integer: %v", err)
		}
		name, err := cf.ClassName()
		if err != nil {
			t.Fatalf("failed to get class name: %v", err)
		}
		if name != "java/lang/Integer" {
			t.Errorf("class name: got %q, want %q", name, "java/lang/Integer")
		}
	})

	t.Run("load Object class", func(t *testing.T) {
		cf, err := cl.LoadClass("java/lang/Object")
		if err != nil {
			t.Fatalf("failed to load java/lang/Object: %v", err)
		}
		name, err := cf.ClassName()
		if err != nil {
			t.Fatalf("failed to get class name: %v", err)
		}
		if name != "java/lang/Object" {
			t.Errorf("class name: got %q, want %q", name, "java/lang/Object")
		}
	})
}

func TestUserClassLoader(t *testing.T) {
	bootstrap := NewJmodClassLoader(testJmodPath)
	userCL := NewUserClassLoader("../../testdata", bootstrap)

	t.Run("load Hello class", func(t *testing.T) {
		cf, err := userCL.LoadClass("Hello")
		if err != nil {
			t.Fatalf("failed to load Hello: %v", err)
		}
		name, err := cf.ClassName()
		if err != nil {
			t.Fatalf("failed to get class name: %v", err)
		}
		if name != "Hello" {
			t.Errorf("class name: got %q, want %q", name, "Hello")
		}
	})

	t.Run("delegates to parent for stdlib classes", func(t *testing.T) {
		cf, err := userCL.LoadClass("java/lang/Integer")
		if err != nil {
			t.Fatalf("failed to load java/lang/Integer via user class loader: %v", err)
		}
		name, err := cf.ClassName()
		if err != nil {
			t.Fatalf("failed to get class name: %v", err)
		}
		if name != "java/lang/Integer" {
			t.Errorf("class name: got %q, want %q", name, "java/lang/Integer")
		}
	})
}

func TestClassLoaderCache(t *testing.T) {
	cl := NewJmodClassLoader(testJmodPath)

	cf1, err := cl.LoadClass("java/lang/Integer")
	if err != nil {
		t.Fatalf("first load failed: %v", err)
	}

	cf2, err := cl.LoadClass("java/lang/Integer")
	if err != nil {
		t.Fatalf("second load failed: %v", err)
	}

	if cf1 != cf2 {
		t.Error("expected same ClassFile instance for cached load, got different pointers")
	}
}

func TestClassNotFound(t *testing.T) {
	t.Run("jmod class not found", func(t *testing.T) {
		cl := NewJmodClassLoader(testJmodPath)
		_, err := cl.LoadClass("com/nonexistent/Foo")
		if err == nil {
			t.Error("expected error for nonexistent class, got nil")
		}
	})

	t.Run("user class not found", func(t *testing.T) {
		bootstrap := NewJmodClassLoader(testJmodPath)
		userCL := NewUserClassLoader("../../testdata", bootstrap)
		_, err := userCL.LoadClass("NonExistentClass")
		if err == nil {
			t.Error("expected error for nonexistent class, got nil")
		}
	})
}
