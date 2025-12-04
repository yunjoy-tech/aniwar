package taptap

import (
	"encoding/json"
	"fmt"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"reflect"
	"strings"
)

type ArrayNumberType interface {
	int | int32 | int64 | uint | uint32 | uint64 | string
}
type MapKeyNumberType interface {
	int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64 | float32 | float64 | string
}
type MyStructType interface {
	any | *cmd.ItemReward | *cmd.PPlayerCampFunctionBuildingFormula | *cmd.KeyValueItem | *cmd.ShopGoodsInfo | *cmd.GeneralTeamTemp | *PPlayerCampTraderListTemp |
		*cmd.PPlayerBattleCard
}

// ConvertList2Str 莉莉丝埋点业务专用
//
//	@Description: 将列表元素拼接成字符串输出
//	@param arr 给定列表源数据
//	@return string 输出字符串
func ConvertList2Str[T ArrayNumberType](arr []T) string {
	var sb strings.Builder
	for i := 0; i < len(arr); i++ {
		sb.WriteString(fmt.Sprint(arr[i]))
		sb.WriteString(";")
	}
	return strings.TrimSuffix(sb.String(), ";")
}

// ConvertMap2Str 莉莉丝埋点业务专用
//
//	@Description: 将map元素拼接成字符串输出
//	@param m 给定map元数据
//	@return string 输出字符串
func ConvertMap2Str[K, V MapKeyNumberType](m map[K]V) string {
	var sb strings.Builder
	for k, v := range m {
		sb.WriteString(fmt.Sprint(k))
		sb.WriteString(",")
		sb.WriteString(fmt.Sprint(v))
		sb.WriteString(";")
	}

	return strings.TrimSuffix(sb.String(), ";")
}

// MapToJson 莉莉丝埋点业务专用
//
//	@Description: 将map元素拼接成字符串输出
//	@param m 给定map元数据
//	@return string 输出字符串
func MapToJson(param map[string]interface{}) string {
	dataType, _ := json.Marshal(param)
	dataString := string(dataType)
	return dataString
}

// ConvertListStruct2Str 莉莉丝埋点业务专用
//
//	@Description: 将结构体列表转换成字符串输出
//	@param arr 给定结构体列表
//	@return string 返回字符串
func ConvertListStruct2Str[T MyStructType](arr []T) string {
	var sb strings.Builder
	for _, v := range arr {
		sb.WriteString(convertStruct2Str(v, ","))
		sb.WriteString(";")
	}
	return strings.TrimSuffix(sb.String(), ";")
}

// ConvertStruct2Str 莉莉丝埋点业务专用
//
//	@Description: 转换结构体为字符串
//	@param ins 给定结构体，其他类型将返回空串
//	@return string 返回字符串
func ConvertStruct2Str(ins any) string {
	return convertStruct2Str(ins, ";")
}

func convertStruct2Str(ins any, sep string) string {
	typeOfIns := reflect.TypeOf(ins)
	if typeOfIns == nil || (typeOfIns.Kind() != reflect.Struct && typeOfIns.Kind() != reflect.Ptr) {
		return ""
	}

	valueOfIns := reflect.ValueOf(ins)
	if valueOfIns.Kind() == reflect.Ptr {
		valueOfIns = reflect.ValueOf(ins).Elem()
		typeOfIns = reflect.TypeOf(ins).Elem()
	}
	var sb strings.Builder
	for i := 0; i < valueOfIns.NumField(); i++ {
		fieldType := typeOfIns.Field(i)
		if _, ok := fieldType.Tag.Lookup("json"); !ok {
			continue
		}
		sb.WriteString(fmt.Sprint(valueOfIns.Field(i)))
		sb.WriteString(sep)
	}

	return strings.TrimSuffix(sb.String(), sep)
}
