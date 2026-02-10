package native

import "testing"

func TestNativeHashMap(t *testing.T) {
	t.Run("put and get", func(t *testing.T) {
		hm := NewHashMap()
		hm.Put("key1", "value1")

		got := hm.Get("key1")
		if got != "value1" {
			t.Errorf("Get(key1): got %v, want %q", got, "value1")
		}
	})

	t.Run("get missing key returns nil", func(t *testing.T) {
		hm := NewHashMap()

		got := hm.Get("nonexistent")
		if got != nil {
			t.Errorf("Get(nonexistent): got %v, want nil", got)
		}
	})

	t.Run("overwrite value", func(t *testing.T) {
		hm := NewHashMap()
		hm.Put("key", "old")
		hm.Put("key", "new")

		got := hm.Get("key")
		if got != "new" {
			t.Errorf("Get(key) after overwrite: got %v, want %q", got, "new")
		}
	})

	t.Run("multiple keys", func(t *testing.T) {
		hm := NewHashMap()
		hm.Put("a", "1")
		hm.Put("b", "2")
		hm.Put("c", "3")

		if hm.Get("a") != "1" {
			t.Errorf("Get(a): got %v, want %q", hm.Get("a"), "1")
		}
		if hm.Get("b") != "2" {
			t.Errorf("Get(b): got %v, want %q", hm.Get("b"), "2")
		}
		if hm.Get("c") != "3" {
			t.Errorf("Get(c): got %v, want %q", hm.Get("c"), "3")
		}
	})

	t.Run("integer keys", func(t *testing.T) {
		hm := NewHashMap()
		hm.Put(int32(0), int32(1))
		hm.Put(int32(1), int32(1))

		got := hm.Get(int32(0))
		if got != int32(1) {
			t.Errorf("Get(0): got %v, want 1", got)
		}
	})
}

func TestNativeInteger(t *testing.T) {
	t.Run("valueOf and intValue roundtrip", func(t *testing.T) {
		boxed := IntegerValueOf(42)
		got := IntegerIntValue(boxed)
		if got != 42 {
			t.Errorf("intValue(valueOf(42)): got %d, want 42", got)
		}
	})

	t.Run("valueOf preserves value", func(t *testing.T) {
		boxed := IntegerValueOf(-100)
		got := IntegerIntValue(boxed)
		if got != -100 {
			t.Errorf("intValue(valueOf(-100)): got %d, want -100", got)
		}
	})

	t.Run("valueOf zero", func(t *testing.T) {
		boxed := IntegerValueOf(0)
		got := IntegerIntValue(boxed)
		if got != 0 {
			t.Errorf("intValue(valueOf(0)): got %d, want 0", got)
		}
	})

	t.Run("different values are distinct", func(t *testing.T) {
		a := IntegerValueOf(10)
		b := IntegerValueOf(20)

		if IntegerIntValue(a) == IntegerIntValue(b) {
			t.Errorf("valueOf(10) and valueOf(20) should be different")
		}
	})
}
