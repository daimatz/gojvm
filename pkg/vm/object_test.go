package vm

import "testing"

func TestJArrayBasic(t *testing.T) {
	t.Run("create and access elements", func(t *testing.T) {
		arr := &JArray{Elements: make([]Value, 3)}
		arr.Elements[0] = IntValue(10)
		arr.Elements[1] = IntValue(20)
		arr.Elements[2] = IntValue(30)

		if arr.Elements[0].Int != 10 {
			t.Errorf("element 0: got %d, want 10", arr.Elements[0].Int)
		}
		if arr.Elements[1].Int != 20 {
			t.Errorf("element 1: got %d, want 20", arr.Elements[1].Int)
		}
		if arr.Elements[2].Int != 30 {
			t.Errorf("element 2: got %d, want 30", arr.Elements[2].Int)
		}
	})

	t.Run("overwrite element", func(t *testing.T) {
		arr := &JArray{Elements: make([]Value, 2)}
		arr.Elements[0] = IntValue(1)
		arr.Elements[0] = IntValue(99)

		if arr.Elements[0].Int != 99 {
			t.Errorf("overwritten element 0: got %d, want 99", arr.Elements[0].Int)
		}
	})

	t.Run("reference elements", func(t *testing.T) {
		arr := &JArray{Elements: make([]Value, 2)}
		obj := &JObject{ClassName: "Test", Fields: make(map[string]Value)}
		arr.Elements[0] = RefValue(obj)
		arr.Elements[1] = NullValue()

		if arr.Elements[0].Type != TypeRef || arr.Elements[0].Ref != obj {
			t.Error("element 0: expected matching reference")
		}
		if arr.Elements[1].Type != TypeNull {
			t.Errorf("element 1: got type %v, want TypeNull", arr.Elements[1].Type)
		}
	})

	t.Run("empty array", func(t *testing.T) {
		arr := &JArray{Elements: make([]Value, 0)}
		if len(arr.Elements) != 0 {
			t.Errorf("empty array length: got %d, want 0", len(arr.Elements))
		}
	})
}

func TestJObjectFields(t *testing.T) {
	t.Run("set and get field", func(t *testing.T) {
		obj := &JObject{ClassName: "TestClass", Fields: make(map[string]Value)}
		obj.Fields["x"] = IntValue(42)

		got := obj.Fields["x"]
		if got.Type != TypeInt || got.Int != 42 {
			t.Errorf("field x: got %+v, want IntValue(42)", got)
		}
	})

	t.Run("multiple fields", func(t *testing.T) {
		obj := &JObject{ClassName: "Point", Fields: make(map[string]Value)}
		obj.Fields["x"] = IntValue(10)
		obj.Fields["y"] = IntValue(20)

		if obj.Fields["x"].Int != 10 {
			t.Errorf("field x: got %d, want 10", obj.Fields["x"].Int)
		}
		if obj.Fields["y"].Int != 20 {
			t.Errorf("field y: got %d, want 20", obj.Fields["y"].Int)
		}
	})

	t.Run("overwrite field", func(t *testing.T) {
		obj := &JObject{ClassName: "TestClass", Fields: make(map[string]Value)}
		obj.Fields["x"] = IntValue(1)
		obj.Fields["x"] = IntValue(99)

		if obj.Fields["x"].Int != 99 {
			t.Errorf("overwritten field x: got %d, want 99", obj.Fields["x"].Int)
		}
	})

	t.Run("reference field", func(t *testing.T) {
		obj := &JObject{ClassName: "Container", Fields: make(map[string]Value)}
		inner := &JObject{ClassName: "Inner", Fields: make(map[string]Value)}
		obj.Fields["child"] = RefValue(inner)

		got := obj.Fields["child"]
		if got.Type != TypeRef {
			t.Errorf("field child: got type %v, want TypeRef", got.Type)
		}
		if got.Ref != inner {
			t.Errorf("field child: reference mismatch")
		}
	})

	t.Run("null field", func(t *testing.T) {
		obj := &JObject{ClassName: "TestClass", Fields: make(map[string]Value)}
		obj.Fields["ref"] = NullValue()

		got := obj.Fields["ref"]
		if got.Type != TypeNull {
			t.Errorf("null field: got type %v, want TypeNull", got.Type)
		}
	})

	t.Run("class name preserved", func(t *testing.T) {
		obj := &JObject{ClassName: "java/util/HashMap", Fields: make(map[string]Value)}
		if obj.ClassName != "java/util/HashMap" {
			t.Errorf("class name: got %q, want %q", obj.ClassName, "java/util/HashMap")
		}
	})
}
