package logic

import (
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/pb"
	"gitlab.musadisca-games.com/wangxw/musae/framework/guid"
)

// 创建用户用户信息并落库
func (s *LoginServer) createAccount(openId, uid, channel string, curTime int64) *pb.UserData {
	account := &pb.UserData{}
	id := s.GenGUID(guid.GUID_PLAYER)
	var playerId uint64
	if id > 0 {
		playerId = id + common.USER_ID_BASE // playerId基数从10000开始
	}

	// 账号数据
	account.CreateTs = curTime
	account.UpdateTs = curTime
	account.Account = &pb.AccountInfo{
		OpenId:    openId,
		Uid:       uid,
		Channel:   channel,
		Nickname:  "unknown",
		Gender:    1,
		Country:   "",
		City:      "",
		BannedTs:  0,
		BannedMsg: "",
	}
	// 充值信息
	account.Recharge = &pb.RechargeInfo{
		Balance:     0,
		GenBalance:  0,
		FirstSave:   1,
		SaveBalance: 0,
		SaveGen:     0,
		SaveMoney:   0,
	}
	// 角色列表
	account.PlayerList = &pb.PlayerInfo{UpdateTs: curTime, PlayerId: playerId}
	account.PlayerList.Players = map[uint64]*pb.Player{0: {Id: 0}}
	return account
}
