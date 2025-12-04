package logic

import (
	"encoding/json"
	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"
)

func (s *IDIPServer) ClientVersionPublish(apiData []byte) []byte {
	req := &ClientVersionPublishReq{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Warn("C2SMsg - Unmarshal error ", err)
	}
	//设置当前版本
	s.setCurrentVersion(req)
	s.setMaxClientVersion(req)
	//坐下oss 版本号和2.8sdk 版本号映射
	s.setVersionMap(req)

	//s.changeClientVersionState(req)
	res := &ClientVersionPublishRes{
		CurVersion: req.NewVersion,
	}
	resByte, _ := json.Marshal(res)

	return resByte
}

func (s *IDIPServer) GetClientVersionInfo() {

}
