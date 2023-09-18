package utils

// ThreeWayOperator 三元运算，此函数无法执行前后关联的嵌套
// 如：ThreeWayOperator(p == nil, "未知", Any(p.gender == 1, "男", "女")) 会有panic错误
func ThreeWayOperator[T any](condition bool, trueVal, falseVal T) T {
	if condition {
		return trueVal
	}
	return falseVal
}
