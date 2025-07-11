package utils

import "reflect"

// MapEquals 深度比较两个 map[string]any 是否内容一致
func MapEquals(a, b map[string]any) bool {
	return deepEqual(a, b)
}

func deepEqual(a, b any) bool {
	if a == nil || b == nil {
		return a == b
	}
	ta, tb := reflect.TypeOf(a), reflect.TypeOf(b)
	if ta != tb {
		return false
	}
	switch va := a.(type) {
	case map[string]any:
		vb := b.(map[string]any)
		if len(va) != len(vb) {
			return false
		}
		for k, vaVal := range va {
			vbVal, ok := vb[k]
			if !ok {
				return false
			}
			if !deepEqual(vaVal, vbVal) {
				return false
			}
		}
		return true
	case []any:
		vb := b.([]any)
		if len(va) != len(vb) {
			return false
		}
		for i := range va {
			if !deepEqual(va[i], vb[i]) {
				return false
			}
		}
		return true
	default:
		return reflect.DeepEqual(a, b)
	}
}
