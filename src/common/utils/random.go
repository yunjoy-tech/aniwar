package utils

import (
	"errors"
	"fmt"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"math/rand"
	"sort"
)

type Pair struct {
	Key   interface{}
	Value int32
}

type PairList []*Pair

func (p PairList) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p PairList) Len() int {
	return len(p)
}

func (p PairList) Less(i, j int) bool {
	// 抽卡特殊排序处理
	if cfg1, ok := p[i].Key.(*excel.PoolContentCfg); ok {
		cfg2, ok2 := p[j].Key.(*excel.PoolContentCfg)
		if ok2 {
			return p[i].Value < p[j].Value || cfg1.Id < cfg2.Id
		}
	}
	return p[i].Value < p[j].Value
}

type Pair2 struct {
	Key   int32
	Value int32
}

type Temp_PairList2 []*Pair2

func (p Temp_PairList2) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p Temp_PairList2) Len() int           { return len(p) }
func (p Temp_PairList2) Less(i, j int) bool { return p[i].Key < p[j].Key }

// RandomInt 随机获取[min, max)中的值
func RandomInt[T int | int32 | int64](min, max T) (t T, err error) {
	if min > max {
		return t, errors.New("min more than max")
	}

	var ti interface{} = &min
	switch ti.(type) {
	case *int:
		t = T(rand.Intn(int(max-min)) + int(min))
	case *int32:
		t = T(rand.Int31n(int32(int(max-min))) + int32(min))
	case *int64:
		t = T(rand.Int63n(int64(int(max-min))) + int64(min))
	default:
		return t, errors.New("unsupported type")
	}

	return
}

// RandomList 给定元素中返回一个元素
func RandomList[T any](source []T) T {
	var t T
	if source == nil || len(source) == 0 {
		return t
	}
	index, err := RandomInt(0, len(source))
	if err != nil {
		return t
	}
	return source[index]
}

// RandomListN 给定集合中随机N个元素
func RandomListN[T any](source []T, num int32) []T {
	var result []T
	// 没有源或要取的数小于1个就直接返回空列表
	if source == nil || num < 1 {
		return result
	}

	// 数量刚刚好
	if int32(len(source)) <= num {
		return source
	}

	// 随机位
	sourceX := make([]T, len(source))
	copy(sourceX, source)
	for i := int32(0); i < num; i++ {
		index, err := RandomInt(0, int32(len(sourceX)))
		if err != nil {
			return []T{}
		}
		result = append(result, sourceX[index])
		sourceX = append(sourceX[:index], sourceX[index+1:]...)
	}

	return result
}

// 随机一个map元素
func RandMapAnyKey[K, V MapKeyNumberTypeExtra](m map[K]V) interface{} {
	// 数组默认长度为map长度,后面append时,不需要重新申请内存和拷贝,效率很高
	j := 0
	keys := make([]interface{}, len(m))
	for k := range m {
		keys[j] = k
		j++
	}

	// 从切片中随机一个元素出来
	key := keys[rand.Intn(len(keys))]

	return m[key.(K)]
}

// RandomMap
//
//	@Description: 按权重随机返回一个key
//	@param source key为随机对象,value为权重值
//	@param isSort 是否需要排序，排序后排除对map的range二次随机影响，完全等价于系统随机数
//	@return interface{}
func RandomMap(source map[interface{}]int32, isSort ...bool) interface{} {
	// 权重和
	var sum int32 = 0
	for _, w := range source {
		if w <= 0 {
			continue
		}
		sum += w
	}

	if sum <= 0 {
		return nil
	}

	// 取随机数
	randValue := rand.Int31n(sum)

	sortFlag := false
	if len(isSort) > 0 {
		sortFlag = isSort[0]
	}
	if sortFlag {
		temp := make(PairList, 0)
		for k, v := range source {
			temp = append(temp, &Pair{k, v})
		}
		sort.Sort(temp)
		for _, pair := range temp {
			if randValue < pair.Value {
				return pair.Key
			}
			randValue -= pair.Value
		}
	} else {
		for k, w := range source {
			if randValue < w {
				return k
			}
			randValue -= w
		}
	}
	return nil
}

func Temp_RandomMap2(source map[int32]int32, isSort ...bool) int32 {
	// 权重和
	var sum int32 = 0
	for _, w := range source {
		if w <= 0 {
			continue
		}
		sum += w
	}

	if sum <= 0 {
		return 0
	}

	// 取随机数
	randValue := rand.Int31n(sum)

	sortFlag := false
	if len(isSort) > 0 {
		sortFlag = isSort[0]
	}
	if sortFlag {
		temp := make(Temp_PairList2, len(source))
		i := 0
		for k, v := range source {
			temp[i] = &Pair2{k, v}
			i++
		}
		sort.Sort(temp)
		for _, pair := range temp {
			if randValue < pair.Value {
				return pair.Key
			}
			randValue -= pair.Value
		}
	} else {
		for k, w := range source {
			if randValue < w {
				return k
			}
			randValue -= w
		}
	}
	return 0
}

func Temp_RandomMap4(source map[int32]int32, isSort ...bool) int32 {
	// 权重和
	var sum int32 = 0
	for _, w := range source {
		if w <= 0 {
			continue
		}
		sum += w
	}

	if sum <= 0 {
		return 0
	}

	// 取随机数
	randValue := rand.Int31n(sum)

	sortFlag := false
	if len(isSort) > 0 {
		sortFlag = isSort[0]
	}
	if sortFlag {
		temp := make(Temp_PairList2, len(source))
		i := 0
		for k, v := range source {
			temp[i] = &Pair2{k, v}
			i++
		}
		sort.Sort(temp)

		total := int32(0)
		for _, pair := range temp {
			total += pair.Value
			if total > randValue {
				return pair.Key
			}
			//if randValue < pair.Value {
			//	return pair.Key
			//}
			//randValue -= pair.Value
		}
	} else {
		total := int32(0)
		for k, w := range source {
			//if randValue < w {
			//	return k
			//}
			//randValue -= w
			total += w
			if total > randValue {
				return k
			}
		}
	}
	return 0
}

// IsSuccessByPercentage 判断一次百分比的随机事件是否成功
func IsSuccessByPercentage(rate int32) bool {
	return IsSuccess(float64(rate) / 100)
}

// IsSuccess 判断一次随机事件是否成功
func IsSuccess(rate float64) bool {
	return rand.Float64() < rate
}

// RandomByWeights 根据权重随机
func RandomByWeights[T any](vals []T, weights []int32) (val T, err error) {
	if len(vals) != len(weights) {
		return val, errors.New(fmt.Sprintf("randomByWeights, len(vals)=%d, len(weights)=%d",
			len(vals), len(weights)))
	}

	// 权重和
	totalWeight := int32(0)
	for _, each := range weights {
		totalWeight += each
	}

	if totalWeight <= 0 {
		return val, errors.New("total weight less than or equal to zero(0)")
	}

	// 取随机数
	randWeight, err := RandomInt(0, totalWeight)
	if err != nil {
		return val, err
	}

	tempWeight := int32(0)
	for i, each := range weights {
		tempWeight += each
		if tempWeight >= randWeight {
			val = vals[i]
			return val, nil
		}
	}

	return val, errors.New("no value is returned")
}
