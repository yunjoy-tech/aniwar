package utils

import (
	"fmt"
	"testing"
)

func Test_del_array_by_element(t *testing.T) {
	arr := make([]int, 0)
	arr = append(arr, 11)
	arr = append(arr, 100)
	arr = append(arr, 12)
	arr = append(arr, 12)
	arr = append(arr, 13)
	arr = append(arr, 100)
	arr = append(arr, 14)
	arr = append(arr, 15)
	arr = append(arr, 100)
	fmt.Printf("初始: arr:%p, arr:%+v\n", arr, arr)

	var v1 = 11
	arrAfterDel := DeleteOneByElement(arr, v1)
	fmt.Printf("删除一个[%d], arr:%p, arrAfterDel:%p\n arr:%+v, arrAfterDel:%+v\n", v1, arr, arrAfterDel, arr, arrAfterDel)

	var v2 = 12
	arrAfterDel = DeleteAllByElement(arr, v2)
	fmt.Printf("删除所有[%d], arr:%p, arrAfterDel:%p\n arr:%+v, arrAfterDel:%+v\n", v2, arr, arrAfterDel, arr, arrAfterDel)

	var v3 = 100
	arrAfterDel = DeleteFirstByElement(arr, v3)
	fmt.Printf("删除第一个[%d], arr:%p, arrAfterDel:%p\n arr:%+v, arrAfterDel:%+v\n", v3, arr, arrAfterDel, arr, arrAfterDel)

	var v4 = 100
	arrAfterDel = DeleteLastByElement(arr, v4)
	fmt.Printf("删除最后一个[%d], arr:%p, arrAfterDel:%p\n arr:%+v, arrAfterDel:%+v\n", v4, arr, arrAfterDel, arr, arrAfterDel)

}

// 循环中删除元素
func Test_del_array_by_element1(t *testing.T) {
	arr := []int{11, 12, 13, 14, 15}

	for i, each := range arr {
		if each%2 == 0 {
			arr = append(arr[:i], arr[i+1:]...)
		}
	}
	fmt.Printf(fmt.Sprintf("%+v", arr))
}

// 循环中删除元素
func Test_WithinArray(t *testing.T) {
	arr1 := []int32{1, 2, 3}
	arr2 := []int32{2, 3, 4, 5, 2}

	fmt.Printf("数组1:%v, 是否有交集:%v\n", arr1, WithinArray(arr1))
	fmt.Printf("数组1:%v, 是否有交集:%v\n", arr2, WithinArray(arr2))
}

// 循环中删除元素
func Test_WithinArray2(t *testing.T) {
	arr1 := []int32{1, 2, 3}
	arr2 := []int32{2, 3, 4, 5}

	//fmt.Printf("数组1:%v, 数组2:%v, 是否有交集:%v\n", arr1, arr1, WithinArray(arr1, arr1))
	//fmt.Printf("数组1:%v, 数组2:%v, 是否有交集:%v\n", arr2, arr2, WithinArray(arr2, arr2))
	fmt.Printf("数组1:%v, 数组2:%v, 是否有交集:%v\n", arr1, arr2, WithinArray2(arr1, arr2))
}

// 循环中删除元素
func Test_WithinArray3(t *testing.T) {
	arr1 := []string{"101", "102"}
	arr2 := []string{"102"}

	//fmt.Printf("数组1:%v, 数组2:%v, 是否有交集:%v\n", arr1, arr1, WithinArray(arr1, arr1))
	//fmt.Printf("数组1:%v, 数组2:%v, 是否有交集:%v\n", arr2, arr2, WithinArray(arr2, arr2))
	fmt.Printf("数组1:%v, 数组2:%v, 是否有交集:%v\n", arr1, arr2, WithinArray2(arr1, arr2))
}
