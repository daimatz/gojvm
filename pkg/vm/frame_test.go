package vm

import (
	"testing"
)

func TestFramePushPop(t *testing.T) {
	t.Run("LIFO order", func(t *testing.T) {
		frame := NewFrame(0, 10, nil, nil)

		frame.Push(IntValue(10))
		frame.Push(IntValue(20))
		frame.Push(IntValue(30))

		v := frame.Pop()
		if v.Int != 30 {
			t.Errorf("first Pop: got %d, want 30", v.Int)
		}

		v = frame.Pop()
		if v.Int != 20 {
			t.Errorf("second Pop: got %d, want 20", v.Int)
		}

		v = frame.Pop()
		if v.Int != 10 {
			t.Errorf("third Pop: got %d, want 10", v.Int)
		}
	})

	t.Run("push after pop reuses space", func(t *testing.T) {
		frame := NewFrame(0, 10, nil, nil)

		frame.Push(IntValue(1))
		frame.Push(IntValue(2))
		frame.Pop() // remove 2

		frame.Push(IntValue(3))
		v := frame.Pop()
		if v.Int != 3 {
			t.Errorf("got %d, want 3", v.Int)
		}

		v = frame.Pop()
		if v.Int != 1 {
			t.Errorf("got %d, want 1", v.Int)
		}
	})

	t.Run("single push pop", func(t *testing.T) {
		frame := NewFrame(0, 10, nil, nil)

		frame.Push(IntValue(42))
		v := frame.Pop()
		if v.Int != 42 {
			t.Errorf("got %d, want 42", v.Int)
		}
	})

	t.Run("negative values", func(t *testing.T) {
		frame := NewFrame(0, 10, nil, nil)

		frame.Push(IntValue(-100))
		v := frame.Pop()
		if v.Int != -100 {
			t.Errorf("got %d, want -100", v.Int)
		}
	})
}

func TestFrameLocalVars(t *testing.T) {
	t.Run("basic set and get", func(t *testing.T) {
		frame := NewFrame(4, 10, nil, nil)

		frame.SetLocal(0, IntValue(10))
		frame.SetLocal(1, IntValue(20))
		frame.SetLocal(2, IntValue(30))
		frame.SetLocal(3, IntValue(40))

		if v := frame.GetLocal(0); v.Int != 10 {
			t.Errorf("GetLocal(0): got %d, want 10", v.Int)
		}
		if v := frame.GetLocal(1); v.Int != 20 {
			t.Errorf("GetLocal(1): got %d, want 20", v.Int)
		}
		if v := frame.GetLocal(2); v.Int != 30 {
			t.Errorf("GetLocal(2): got %d, want 30", v.Int)
		}
		if v := frame.GetLocal(3); v.Int != 40 {
			t.Errorf("GetLocal(3): got %d, want 40", v.Int)
		}
	})

	t.Run("overwrite local variable", func(t *testing.T) {
		frame := NewFrame(4, 10, nil, nil)

		frame.SetLocal(0, IntValue(10))
		frame.SetLocal(0, IntValue(99))

		if v := frame.GetLocal(0); v.Int != 99 {
			t.Errorf("GetLocal(0) after overwrite: got %d, want 99", v.Int)
		}
	})

	t.Run("non-contiguous set", func(t *testing.T) {
		frame := NewFrame(4, 10, nil, nil)

		frame.SetLocal(0, IntValue(100))
		frame.SetLocal(3, IntValue(300))

		if v := frame.GetLocal(0); v.Int != 100 {
			t.Errorf("GetLocal(0): got %d, want 100", v.Int)
		}
		if v := frame.GetLocal(3); v.Int != 300 {
			t.Errorf("GetLocal(3): got %d, want 300", v.Int)
		}
	})

	t.Run("local vars independent from stack", func(t *testing.T) {
		frame := NewFrame(4, 10, nil, nil)

		frame.SetLocal(0, IntValue(10))
		frame.Push(IntValue(99))

		if v := frame.GetLocal(0); v.Int != 10 {
			t.Errorf("GetLocal(0) after push: got %d, want 10", v.Int)
		}

		v := frame.Pop()
		if v.Int != 99 {
			t.Errorf("Pop after SetLocal: got %d, want 99", v.Int)
		}
	})
}
