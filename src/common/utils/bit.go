package utils

type BitNumberType interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// GetInt32AtBit num按照二进制，获取第bit位数值
func GetInt32AtBit(num, bit int32) int32 {
	return num >> (bit - 1) & 1
}
