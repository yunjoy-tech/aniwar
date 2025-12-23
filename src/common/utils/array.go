package utils

type ArrayNumberType interface {
	int | int32 | int64 | uint | uint32 | uint64
}

// DeleteAllByElement 删除所有匹配的元素
func DeleteAllByElement[T ArrayNumberType](arr []T, delVal T) []T {
	return deleteByElement(arr, delVal, true)
}

// DeleteOneByElement 删除一个匹配的元素
func DeleteOneByElement[T ArrayNumberType](arr []T, delVal T) []T {
	return deleteByElement(arr, delVal, false)
}

// DeleteFirstByElement 删除第一个匹配的元素
func DeleteFirstByElement[T ArrayNumberType](arr []T, delVal T) []T {
	for i := 0; i < len(arr); i++ {
		val := arr[i]

		if val == delVal {
			arr = append(arr[:i], arr[i+1:]...)
			break
		}
	}
	return arr
}

// DeleteLastByElement 删除最后一个匹配的元素
func DeleteLastByElement[T ArrayNumberType](arr []T, delVal T) []T {
	for i := len(arr) - 1; i >= 0; i-- {
		val := arr[i]

		if val == delVal {
			arr = append(arr[:i], arr[i+1:]...)
			break
		}
	}
	return arr
}

func deleteByElement[T ArrayNumberType](arr []T, delVal T, deleteAll bool) []T {
	// for idx, val := range arr {
	for i := len(arr) - 1; i >= 0; i-- {
		val := arr[i]

		if val == delVal {
			arr = append(arr[:i], arr[i+1:]...)
			if !deleteAll {
				break
			}
		}
	}

	return arr
}

// WithinArray 集合中是否有重复元素
// @return true:有重复元素; false:没有重复元素
func WithinArray[T ArrayNumberType](array []T) bool {
	arrayMap := make(map[T]int)
	for _, eachB := range array {
		arrayMap[eachB] = 1
	}

	return len(arrayMap) != len(array) // 长度不同: 则表示有重复元素
}

// WithinArray2 两个集合的交集是否有值
// @return true:有重复元素; false:没有重复元素
func WithinArray2[T comparable](array1, array2 []T) bool {
	arrayA := array1
	arrayB := array2

	if len(array1) < len(array2) {
		arrayA = array2 // 元素多的数组
		arrayB = array1 // 元素少的数组
	}

	arrayBMap := make(map[T]int)
	for _, eachB := range arrayB {
		arrayBMap[eachB] = 1
	}

	// 遍历元素多的数组
	for _, eachA := range arrayA {
		if _, ok := arrayBMap[eachA]; ok {
			return true
		}
	}

	return false
}

func ArrayContain[T comparable](arr []T, val T) bool {
	for _, each := range arr {
		if each == val {
			return true
		}
	}

	return false
}
