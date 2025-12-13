package utils

import (
	"bufio"
	"fmt"
	"gitee.com/bychannel/aniwar/src/proto/pb"
	"gitee.com/bychannel/musae/framework/logger"
	"os"
	"reflect"
	"strconv"
	"strings"
)

// ScanfUID
//
//	@Description: 解析uid信息
//	@param uid: 游戏内uid string
//	@return channel: sdk 渠道信息
//	@return appid： sdk appid
//	@return accountId: sdk 账号id
//	@return err
func ScanfUID(uid string) (channel string, appid int64, accountId string, err error) {
	s := strings.Split(uid, "_")
	if err != nil {
		return "", 0, "", err
	}
	if len(s) != 3 {
		return "", 0, "", fmt.Errorf("uid fromat error")
	}
	channel = s[0]
	appid, err = strconv.ParseInt(s[1], 10, 64)
	if err != nil {
		return "", 0, "", err
	}
	accountId = s[2]
	return channel, appid, accountId, nil
}

func ScanfUAID(uaid string) (channel string, appid int64, accountId string, playerId uint64, err error) {
	s := strings.Split(uaid, "_")
	if err != nil {
		return "", 0, "", 0, err
	}
	if len(s) != 4 {
		return "", 0, "", 0, fmt.Errorf("uaid fromat error")
	}
	channel = s[0]
	appid, err = strconv.ParseInt(s[1], 10, 64)
	if err != nil {
		return "", 0, "", 0, err
	}
	accountId = s[2]
	playerId, err = strconv.ParseUint(s[3], 10, 64)
	if err != nil {
		return "", 0, "", 0, err
	}
	return channel, appid, accountId, playerId, nil
}

// 解析bool类型不定参数
func GotBoolParam(params ...bool) bool {
	ret := false // 默认为false
	if len(params) > 0 {
		ret = params[0]
	}
	return ret
}

func ConvertItem(items []*pb.KeyValueItem) map[int32]int32 {
	costs := make(map[int32]int32)
	for _, v := range items {
		costs[v.Key] += v.Value
	}
	return costs
}

// PPlayerCampFunctionBuildingFormula
func ConvertCampFormula(items []*pb.PPlayerCampFunctionBuildingFormula) map[int32]int32 {
	costs := make(map[int32]int32)

	for _, v := range items {
		costs[v.FormulaId] += v.Num
	}
	return costs
}

func ConvertItem2(items []*pb.ItemReward) map[int32]int32 {
	costs := make(map[int32]int32)
	for _, v := range items {
		costs[int32(v.ItemId)] += int32(v.Num)
	}
	return costs
}

func ConvertItem4(items []*pb.CostItem) map[uint64]uint32 {
	costs := make(map[uint64]uint32)
	for _, v := range items {
		costs[v.UniqueId] += v.Num
	}
	return costs
}

func ConvertToKVItem(source map[int32]int32) []*pb.KeyValueItem {
	var ret []*pb.KeyValueItem
	for k, v := range source {
		ret = append(ret, &pb.KeyValueItem{
			Key:   k,
			Value: v,
		})
	}
	return ret
}

func IsNull(i interface{}) bool {
	vi := reflect.ValueOf(i)
	if vi.Kind() == reflect.Ptr {
		return vi.IsNil()
	}
	return false
}

func ReflectNew(target interface{}) {
	if target == nil {
		logger.Debug("ReflectNew --- return")
		return
	}

	t := reflect.TypeOf(target)
	logger.Debugf("ReflectNew %+v", t.Kind())
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		logger.Debugf("ReflectNew %+v", t.Kind())
	}

	ttt := reflect.New(t)
	logger.Debugf("ReflectNew ttt %+v, %+v, %+v", ttt.Type(), ttt.Kind(), ttt.Kind())
	// logger.Debugf("ReflectNew --- ttt, %T - %p, %T - %p", target, target, ttt, ttt)
	logger.Debugf("ReflectNew --- ttt1, %+v, %+v", reflect.ValueOf(target), reflect.TypeOf(target))
	// logger.Debugf("ReflectNew --- ttt2, %+v, %+v", reflect.ValueOf(ttt), reflect.TypeOf(ttt))
	// logger.Debugf("ReflectNew --- ttt2, %+v, %+v", reflect.ValueOf(ttt.Elem()), reflect.TypeOf(ttt.Elem()))
	//
	logger.Debugf("ReflectNew --- ttt4, %+v, %+v", reflect.ValueOf(ttt.Interface()), reflect.TypeOf(ttt.Interface()))
	logger.Debugf("ReflectNew --- ttt3, %+v, %+v", reflect.ValueOf(ttt.Elem().Interface()), reflect.TypeOf(ttt.Elem().Interface()))

	target = ttt
}

func SaveLogToFile(filePath string, data string) {
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		fmt.Printf("打开文件错误= %v \n", err)
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	writer.WriteString(fmt.Sprintf("#auto generate:\n"))
	writer.WriteString(data)
	writer.Flush()
}
