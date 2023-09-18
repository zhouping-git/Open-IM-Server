package utils

// ElementInSlice 属性是否在切片中存在，如需频繁判断使用GenericsSet实现
func ElementInSlice[T comparable](t []T, st T) bool {
	// 使用空结构体 `struct{}` 作为 value 的类型，因为 `struct{}` 不占用任何内存空间
	set := make(map[T]struct{}, len(t))
	for _, d := range t {
		set[d] = struct{}{}
	}

	_, ok := set[st]
	return ok
}
