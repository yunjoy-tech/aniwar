package utils

import (
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"sort"
)

type SORT_ORDER int

const (
	SORT_ORDER_DESC = 1 // 降序
	SORT_ORDER_ASC  = 2 // 升序
)

type Sorter[K MapKeyType, V MapValueNumberType] struct {
	Key   K
	Value V
}

type MapKeyType interface {
	MapKeyNumberType | string | bool
}

type MapKeyNumberTypeExtra interface {
	int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64 | float32 | float64 | *cmd.PCommonCardInfo
}

type MapKeyNumberType interface {
	int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64 | float32 | float64
}

type MapValueNumberType interface {
	int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64 | float32 | float64
}

func MergeItems(source, add map[int32]int32) {
	for key, value := range add {
		//if value <= 0 {
		//	continue
		//}
		if v, ok := source[key]; ok {
			source[key] = v + value
		} else {
			source[key] = value
		}
	}
}

// SortMapKeys
//
//	@Description: 对map的key做排序
//	@param m
//	@param sortOrder 升序/降序
//	@return []K 排序后端key列表
func SortMapKeys[K MapKeyNumberType, T any](m map[K]T, sortOrder SORT_ORDER) []K {
	list := make([]K, 0)
	for key := range m {
		list = append(list, key)
	}
	sort.Slice(list, func(i, j int) bool {
		if sortOrder == SORT_ORDER_ASC {
			if list[i] < list[j] {
				return false
			}
			if list[i] > list[j] {
				return true
			}
			return false
		} else if sortOrder == SORT_ORDER_DESC {
			if list[i] < list[j] {
				return true
			}
			if list[i] > list[j] {
				return false
			}
			return false
		}
		return false
	})

	return list
}

// SortMapValByKeys
//
//	@Description: 根据Key对map的值做排序
//	@param m
//	@param sortOrder 升序/降序
//	@return []T 排序后端map的值列表
func SortMapValByKeys[K MapKeyNumberType, T any](m map[K]T, sortOrder SORT_ORDER) []T {
	sortedKeys := SortMapKeys(m, sortOrder)

	list := make([]T, 0)
	for i := 0; i < len(sortedKeys); i++ {
		eachKey := sortedKeys[i]
		list = append(list, m[eachKey])
	}

	return list
}

// SortMapVal[K MapKeyType, V MapValueNumberType]
//
//	@Description: 对map的val做排序
//	@param m 源map数据
//	@param sortOrder 升序/降序
//	@return []*Sorter[K,V] 排序后排序器列表
func SortMapVal[K MapKeyType, V MapValueNumberType](m map[K]V, sortOrder SORT_ORDER) []*Sorter[K, V] {
	var sorter []*Sorter[K, V]
	for k, v := range m {
		sorter = append(sorter, &Sorter[K, V]{k, v})
	}

	if sortOrder == SORT_ORDER_ASC {
		sort.Slice(sorter, func(i, j int) bool {
			return sorter[i].Value < sorter[j].Value // 升序
		})
	} else {
		sort.Slice(sorter, func(i, j int) bool {
			return sorter[i].Value > sorter[j].Value // 降序
		})
	}
	return sorter
}

// SortMapKeyByVal[K MapKeyType, V MapValueNumberType]
//
//	@Description: 根据val对map的key做排序
//	@param m 源map数据
//	@param sortOrder 升序/降序
//	@return []K 排序后key的列表
func SortMapKeyByVal[K MapKeyType, V MapValueNumberType](m map[K]V, sortOrder SORT_ORDER) []K {
	vals := SortMapVal(m, sortOrder)

	ret := make([]K, 0)
	for i := 0; i < len(vals); i++ {
		ret = append(ret, vals[i].Key)
	}

	return ret
}

// Map2List
//
//	@Description: 将map的值转为列表
//	@param m
//	@return []T
func Map2List[K MapKeyType, T any](m map[K]T) []T {
	list := make([]T, 0)

	for _, val := range m {
		list = append(list, val)
	}

	return list
}

// CompareSameMap
//
//	@Description: 比较两个map的值是否相同, 当m1,m2均为nil时返回true, 需要根据业务判定
//	@param m1
//	@param m2
//	@return bool
func CompareSameMap(m1, m2 map[int32]int32) bool {
	//return reflect.DeepEqual(m1, m2)
	if len(m1) != len(m2) {
		return false
	}

	if (m1 == nil && m2 != nil) || (m1 != nil && m2 == nil) {
		return false
	}

	for k, v := range m1 {
		if m2[k] != v {
			return false
		}
	}

	return true
}

func MapToKeyValueItem(source map[int32]int32) []*cmd.KeyValueItem {
	item := make([]*cmd.KeyValueItem, 0, len(source))
	for key, value := range source {
		item = append(item, &cmd.KeyValueItem{
			Key:   key,
			Value: value,
		})
	}
	return item
}

func KeyValueItem2Map(source []*cmd.KeyValueItem) map[int32]int32 {
	item := make(map[int32]int32, len(source))
	for _, value := range source {
		item[value.Key] = value.Value
	}
	return item
}
