package utils

import (
	"fmt"
	"testing"
)

func TestMergeMap(t *testing.T) {
	//source := map[int32]int32{1: 1, 2: 2, 3: 3}
	//add1 := map[int32]int32{1: 0, 3: 2}
	//add2 := map[int32]int32{4: 1, 5: 0}
	//add3 := map[int32]int32{3: 1, 4: 2}

	//fmt.Println(MergeItems(source, add1))
	//fmt.Println(MergeItems(source, add2))
	//fmt.Println(MergeItems(source, add3))
}

func TestSortMapKeys(t *testing.T) {
	m := make(map[int32]string)
	m[0] = "a"
	m[3] = "c"
	m[2] = "b"

	keys1 := SortMapKeys(m, SORT_ORDER_ASC)
	fmt.Println("key排序:", keys1)

	keys2 := SortMapKeys(m, SORT_ORDER_DESC)
	fmt.Println("key排序:", keys2)

	vals1 := SortMapValByKeys(m, SORT_ORDER_ASC)
	fmt.Println("值排序:", vals1)

	vals2 := SortMapValByKeys(m, SORT_ORDER_DESC)
	fmt.Println("值排序:", vals2)
}

func TestMap2List(t *testing.T) {
	m := make(map[int32]string)
	m[0] = "a"
	m[3] = "c"
	m[2] = "b"

	list := Map2List(m)
	fmt.Println(list)
}

func TestCompareSameMap(t *testing.T) {
	var m1 map[int32]int32
	var m2 map[int32]int32
	fmt.Println("m1=nil, m2=nil, result:", CompareSameMap(m1, m2))

	m1 = make(map[int32]int32)
	fmt.Println("m1=[], m2=nil, result:", CompareSameMap(m1, m2))

	m2 = make(map[int32]int32)
	fmt.Println("m1=[], m2=[], result:", CompareSameMap(m1, m2))

	m1[1] = 1
	m2[2] = 2
	fmt.Println("m1=[1:1], m2=[2:2], result:", CompareSameMap(m1, m2))

	m2[1] = 1
	fmt.Println("m1=[1:1], m2=[2:2,1:1], result:", CompareSameMap(m1, m2))

	delete(m2, 2)
	fmt.Println("m1=[1:1], m2=[1:1], result:", CompareSameMap(m1, m2))
}
