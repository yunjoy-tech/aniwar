package logic

import (
	"encoding/json"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/db"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/pb"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"
	"strconv"
)

func (s *IDIPServer) GetAllianceInfo(apiData []byte) []byte {
	req := &pb.GMTGetAllianceInfoReq{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Errorf("Unmarshal fail apiData:%s error:%+v", string(apiData), err)
	}

	// 获取联盟信息
	allianceId := strconv.Itoa(int(req.GetAllianceId()))
	allianceInfo, err := s.GetMongoGame(db.KeyAllianceData(allianceId), nil)
	if err != nil {
		return s.GenRet(err.Error())
	}
	res := &pb.GMTGetAllianceInfoRes{
		Info: &pb.PServerAllianceInfo{},
	}
	if err = base.UnmarshalData(allianceInfo.Data, res.Info); err != nil {
		return s.GenRet(err.Error())
	}

	data, err := json.Marshal(res)
	if err != nil {
		return s.GenRet(err.Error())
	}

	return data
}
