package chunkedit

import (
	"encoding/json"
)

func asMap(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	return m, ok
}

func getArray(chunk map[string]any, keys ...string) ([]any, string) {
	for _, k := range keys {
		if v, ok := chunk[k]; ok {
			if arr, ok := v.([]any); ok {
				return arr, k
			}
		}
	}
	return nil, ""
}

func intsEqual(a any, b int) bool {
	switch t := a.(type) {
	case int8:
		return int(t) == b
	case int16:
		return int(t) == b
	case int32:
		return int(t) == b
	case int64:
		return int(t) == b
	case int:
		return t == b
	case float64:
		return int(t) == b
	default:
		return false
	}
}

func findBlockEntityIndex(chunk map[string]any, x, y, z int) (int, string, map[string]any) {
	arr, key := getArray(chunk, "block_entities", "BlockEntities", "TileEntities")
	if arr == nil {
		return -1, "", nil
	}
	for i, v := range arr {
		m, ok := asMap(v)
		if !ok {
			continue
		}
		xv, xok := m["x"]
		yv, yok := m["y"]
		zv, zok := m["z"]
		if xok && yok && zok && intsEqual(xv, x) && intsEqual(yv, y) && intsEqual(zv, z) {
			return i, key, m
		}
	}
	return -1, key, nil
}

func GetBlockEntity(chunk map[string]any, x, y, z int) (map[string]any, bool) {
	_, key, ent := findBlockEntityIndex(chunk, x, y, z)
	if key == "" || ent == nil {
		return nil, false
	}
	return ent, true
}

func DeleteBlockEntity(chunk map[string]any, x, y, z int) bool {
	idx, key, _ := findBlockEntityIndex(chunk, x, y, z)
	if key == "" || idx < 0 {
		return false
	}
	arr, _ := getArray(chunk, key)
	arr = append(arr[:idx], arr[idx+1:]...)
	chunk[key] = arr
	return true
}

func CreateOrUpdateBlockEntity(chunk map[string]any, x, y, z int, id string, data map[string]any) {
	idx, key, ent := findBlockEntityIndex(chunk, x, y, z)
	if key == "" {
		key = "block_entities"
	}
	if idx < 0 || ent == nil {
		ent = map[string]any{"x": x, "y": y, "z": z}
		if id != "" {
			ent["id"] = id
		}
		for k, v := range data {
			ent[k] = v
		}
		arr, _ := getArray(chunk, key)
		arr = append(arr, ent)
		chunk[key] = arr
		return
	}
	if id != "" {
		ent["id"] = id
	}
	for k, v := range data {
		ent[k] = v
	}
}

func MergeJSONData(dst map[string]any, raw string) error {
	if raw == "" {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return err
	}
	for k, v := range m {
		dst[k] = v
	}
	return nil
}


