package datahelper

import (
	"errors"
	"math/rand"

	"fmt"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
)

// RandomByWeightVo 根据WeightVo随机权重
func RandomByWeightVo(vos []*data.WeightVo) (*data.WeightVo, error) {
	if len(vos) <= 0 {
		return nil, errors.New("param is nil or empty")
	}

	// 权重和
	totalWeight := int32(0)
	for _, each := range vos {
		totalWeight += each.Weight
	}

	if totalWeight <= 0 {
		return nil, errors.New("total weight less than or equal to zero(0)")
	}

	// 取随机数
	randWeight := rand.Int31n(totalWeight)

	tempWeight := int32(0)
	for _, each := range vos {
		tempWeight += each.Weight
		if tempWeight >= randWeight {
			return each, nil
		}
	}

	return nil, errors.New("no value is returned")
}

// RandomByWeightVo2 根据WeightVo2, 以权重随机奖励
func RandomByWeightVo2(vos []*data.WeightVo2) (*data.WeightVo2, error) {
	if len(vos) <= 0 {
		return nil, errors.New("param is nil or empty")
	}

	// 权重和
	totalWeight := int32(0)
	for _, each := range vos {
		totalWeight += each.Weight
	}

	if totalWeight <= 0 {
		return nil, errors.New("total weight less than or equal to zero(0)")
	}

	// 取随机数
	randWeight := rand.Int31n(totalWeight)

	tempWeight := int32(0)
	for _, each := range vos {
		tempWeight += each.Weight
		if tempWeight >= randWeight {
			return each, nil
		}
	}

	return nil, fmt.Errorf("no value is returned")
}
