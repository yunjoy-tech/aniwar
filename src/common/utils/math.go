package utils

import "math"

type NumberType interface {
	int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64 | float32 | float64
}

// Min 获取a, b 中的最小值
func Min[T NumberType](a, b T) T {
	if a <= b {
		return a
	}
	return b
}

// Max 获取a, b 中的最大值
func Max[T NumberType](a, b T) T {
	if a >= b {
		return a
	}
	return b
}

// Abs 绝对值
func Abs[T NumberType](x T) T {
	if x < 0 {
		return -x
	}
	return x
}

// Percent returns a values percent of the total
func Percent(val, total int) float64 {
	if total == 0 {
		return float64(0)
	}
	return (float64(val) / float64(total)) * 100
}

// CalTenThousand 计算给定val的万分比的值, 返回向上取整的结果
func CalTenThousand(val, percent int32) int32 {
	return int32(math.Ceil(float64(percent) / 10000 * float64(val)))
}
