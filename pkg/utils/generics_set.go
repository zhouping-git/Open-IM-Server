package utils

type GenericsSetInterface[T comparable] interface {
	Contains(i ...T) bool
	Replace(oldKey T, newKey T)
	ToSlice() []T
}

type GenericsSet[T comparable] struct {
	Set map[T]struct{}
}

// NewGenericsSet 自定义泛型Set
func NewGenericsSet[T comparable](t []T) *GenericsSet[T] {
	// 使用空结构体 `struct{}` 作为 value 的类型，因为 `struct{}` 不占用任何内存空间
	set := make(map[T]struct{}, len(t))
	for _, d := range t {
		set[d] = struct{}{}
	}
	return &GenericsSet[T]{
		Set: set,
	}
}

// NewSet 创建一个空泛型Set
func NewSet[T comparable]() *GenericsSet[T] {
	return &GenericsSet[T]{
		Set: make(map[T]struct{}),
	}
}

// Contains 是否存在属性
func (o *GenericsSet[T]) Contains(i ...T) bool {
	for _, val := range i {
		if _, ok := o.Set[val]; !ok {
			return false
		}
	}
	return true
}

// Replace 替换集合值
func (o *GenericsSet[T]) Replace(oldKey T, newKey T) {
	delete(o.Set, oldKey)
	o.Set[newKey] = struct{}{}
}

// ToSlice 集合转换为切片
func (o *GenericsSet[T]) ToSlice() []T {
	result := make([]T, len(o.Set))
	for key, _ := range o.Set {
		result = append(result, key)
	}
	return result
}
