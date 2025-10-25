package chunkedit

import "testing"

func TestGetBlockEntity(t *testing.T) {
	chunk := map[string]any{
		"block_entities": []any{
			map[string]any{"x": 1, "y": 64, "z": -3, "id": "minecraft:chest"},
		},
	}

	ent, ok := GetBlockEntity(chunk, 1, 64, -3)
	if !ok {
		t.Fatalf("expected block entity to be found")
	}
	if ent["id"] != "minecraft:chest" {
		t.Fatalf("id: got %v, want minecraft:chest", ent["id"])
	}
}

func TestDeleteBlockEntity(t *testing.T) {
	chunk := map[string]any{
		"BlockEntities": []any{
			map[string]any{"x": 5, "y": 70, "z": 9},
		},
	}

	if !DeleteBlockEntity(chunk, 5, 70, 9) {
		t.Fatalf("expected delete to succeed")
	}
	arr, _ := getArray(chunk, "BlockEntities")
	if len(arr) != 0 {
		t.Fatalf("expected array to be empty, got %d entries", len(arr))
	}
	if DeleteBlockEntity(chunk, 5, 70, 9) {
		t.Fatalf("expected delete on missing entity to return false")
	}
}

func TestCreateOrUpdateBlockEntity(t *testing.T) {
	chunk := map[string]any{}

	CreateOrUpdateBlockEntity(chunk, 3, 60, -2, "minecraft:lectern", map[string]any{"Book": "foo"})
	arr, _ := getArray(chunk, "block_entities")
	if len(arr) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(arr))
	}
	ent, ok := arr[0].(map[string]any)
	if !ok {
		t.Fatalf("expected element to be a map, got %T", arr[0])
	}
	if ent["id"] != "minecraft:lectern" {
		t.Fatalf("id: got %v, want minecraft:lectern", ent["id"])
	}
	if ent["Book"] != "foo" {
		t.Fatalf("Book: got %v, want foo", ent["Book"])
	}

	CreateOrUpdateBlockEntity(chunk, 3, 60, -2, "minecraft:lectern", map[string]any{"Book": "bar"})
	ent, ok = arr[0].(map[string]any)
	if !ok {
		t.Fatalf("expected element to remain a map, got %T", arr[0])
	}
	if ent["Book"] != "bar" {
		t.Fatalf("Book: got %v, want bar", ent["Book"])
	}
}

func TestMergeJSONData(t *testing.T) {
	dst := map[string]any{"foo": "bar"}
	if err := MergeJSONData(dst, "{\"baz\": 1}"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dst["baz"] != float64(1) {
		t.Fatalf("baz: got %v, want 1", dst["baz"])
	}
	if err := MergeJSONData(dst, "not-json"); err == nil {
		t.Fatalf("expected error for invalid json")
	}
}

