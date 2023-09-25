package utils

// SliceIntersect 获取切片交集，只能适配基本数据类型
func SliceIntersect[T comparable](slice1 []T, slice2 []T) []T {
	resp := make([]T, 0)
	mp := make(map[T]struct{})

	for _, item := range slice1 {
		if _, ok := mp[item]; !ok {
			mp[item] = struct{}{}
		}
	}
	for _, item := range slice2 {
		if _, ok := mp[item]; ok {
			resp = append(resp, item)
		}
	}
	return resp
}
