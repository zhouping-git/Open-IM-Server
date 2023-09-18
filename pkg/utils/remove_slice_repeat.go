package utils

func RemoveRepeatForLoop[T comparable](t []T) []T {
	var result []T
	for _, d1 := range t {
		flag := true
		for _, d2 := range result {
			if d1 == d2 {
				flag = false
				break
			}
		}
		if flag {
			result = append(result, d1)
		}
	}
	return result
}

func RemoveRepeatForMap[T comparable](t []T) []T {
	var result []T
	tMap := map[T]byte{}
	for _, d := range t {
		tLen := len(tMap)
		tMap[d] = 0
		if len(tMap) != tLen {
			result = append(result, d)
		}
	}
	return result
}

// RemoveSliceRepeat 去除集合或切片重复数据
func RemoveSliceRepeat[T comparable](t []T) []T {
	// 数据量小时循环性能比较好
	if len(t) <= 999 {
		return RemoveRepeatForLoop(t)
	}
	return RemoveRepeatForMap(t)
}
