package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"gitee.com/aniwar2/musae/framework/logger"
	"strings"
)

func (s *IDIPServer) GetExcelList(apiData []byte) []byte {
	req := &GetExcelList{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Errorf("Unmarshal fail apiData:%s error:%+v", string(apiData), err)
	}

	res := &GetExcelListRes{}

	if req.OptType == 1 { // 获取aniwar 的配置文件列表
		res.FileNames = s.GetAniwarExcel(req)
	}

	if req.OptType == 2 { // 获取battleServer的配置文件
		res.FileNames = s.GetBattleServerExcel(req.Version)
	}
	data, err := json.Marshal(res)
	if err != nil {
		return s.GenRet(err.Error())
	}
	return data
}

func (s *IDIPServer) GetExcelExpired(apiData []byte) []byte {
	req := &ExcelExpired{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Errorf("Unmarshal fail apiData:%s error:%+v", string(apiData), err)
	}
	key := fmt.Sprintf("cn:develop:aniwar:%s:fileList", req.Version)

	listString, err := s.Server.Redis.Get(context.Background(), key).Result()
	if err != nil {

	}
	lists := strings.Split(listString, "|")
	exTime := int32(0)
	for _, fileKey := range lists {
		val := s.Server.Redis.TTL(context.Background(), fileKey)
		exTime = int32(val.Val().Seconds())
		break
	}

	res := ExcelExpiredRes{
		ExpiredTime: exTime,
		Version:     req.Version,
	}
	data, err := json.Marshal(res)
	if err != nil {
		return s.GenRet(err.Error())
	}
	return data
}

func (s *IDIPServer) GetAniwarExcel(req *GetExcelList) []string {
	key := fmt.Sprintf("%s:%s:aniwar:%s:fileList", req.NameSpace, req.Group, req.Version)
	listString, err := s.Server.RedisCenter.Get(context.Background(), key).Result()
	ping := s.Server.RedisCenter.Ping(context.Background())
	fmt.Println("ping:", ping.String())
	if err != nil {

	}
	lists := strings.Split(listString, "|")
	fileNames := make([]string, 0)
	for _, fileName := range lists {
		fileNames = append(fileNames, fileName)
	}

	return fileNames
}

func (s *IDIPServer) GetBattleServerExcel(version string) []string {
	key := fmt.Sprintf("cn:develop:battleServer:%s:fileList", version)
	listString, err := s.Server.Redis.Get(context.Background(), key).Result()
	if err != nil {

	}
	lists := strings.Split(listString, "|")
	fileNames := make([]string, 0)
	for _, fileName := range lists {
		fileNames = append(fileNames, fileName)
	}

	return fileNames
}
