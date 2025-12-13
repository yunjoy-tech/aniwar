package datahelper

import (
	"gitee.com/bychannel/aniwar/src/excel/data"
)

func ConvertItem2ByTpl(items []*data.ItemReward) map[int32]int32 {
	costs := make(map[int32]int32)
	for _, v := range items {
		costs[v.ItemId] += v.Num
	}
	return costs
}

func ConvertItem3(items []*data.KeyVal) map[int32]int32 {
	costs := make(map[int32]int32)
	for _, v := range items {
		costs[v.Key] += v.Val
	}
	return costs
}

func MergeKeyVal(source map[int32]int32, add []*data.KeyVal) {
	for _, eachKV := range add {
		if eachKV == nil {
			continue
		}

		key := eachKV.Key
		value := eachKV.Val

		if v, ok := source[key]; ok {
			source[key] = v + value
		} else {
			source[key] = value
		}
	}
}
