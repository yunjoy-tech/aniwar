package utils

// GetIntWithIndexes 根据int32各位编号，获取各位上的数
// @param origin 源数
// @param indexes 指定获取数位的编号
// @result 指定数位的数字拼成的新数, 指定数位的数字组成的数组
// example: 1234, [1,0] ===> 针对"1234"数字, 获取其第1、0位上的数 ===> 结果: 34, [4, 3]
func GetIntWithIndexes[T int | int32](origin T, indexes []T) (T, []T) {
	multipliers := []T{1, 10, 100, 1000, 10000, 100000, 1000000, 10000000, 100000000, 1000000000} // 避免使用int32(math.Pow(10, float64(pos)))
	digits := make([]T, len(indexes))
	var result T

	for i, pos := range indexes {
		multiplier := multipliers[pos]
		digit := (origin / multiplier) % 10 // 这里要使用T类型
		digits[i] = digit
		result = result*10 + digit // 这里也要使用T类型
	}

	return result, digits
}
