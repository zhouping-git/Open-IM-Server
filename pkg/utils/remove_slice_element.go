package utils

// RemoveSliceElement 根据内容删除切片指定元素
func RemoveSliceElement[T comparable](t []T, rt T) []T {
	for i, d := range t {
		if d == rt {
			t = append(t[:i], t[i+1:]...)
		}
	}
	return t
}

// RemoveSliceElementForIndex 根据索引删除切片指定元素
func RemoveSliceElementForIndex[T comparable](t []T, index int32) []T {
	return append(t[:index], t[index+1:]...)
}
