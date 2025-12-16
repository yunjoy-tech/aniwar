package useractor

import (
	"fmt"

	"gitee.com/aniwar2/aniwar/src/common/db"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/service"
	"gitee.com/aniwar2/musae/state"
	"google.golang.org/protobuf/proto"
)

func (u *UserActor) SaveRedis(key string, value proto.Message, meta map[string]string) error {
	// // kvtable包装
	// kvTable := &state.KvTable{
	//	Id:      0,
	//	Data:    make([]byte, 0),
	//	UpSecTS: time.Now().Unix(),
	//	InSecTS: 0,
	// }
	// temp, err := proto.Marshal(value)
	// if err != nil || temp == nil {
	//	return fmt.Errorf("SaveGlobalRedis Marshal err:%+v", err.Error())
	// }
	//
	// kvTable.Data = temp
	kvTable, err := db.BuildKvTable(value, key)
	if err != nil {
		return err
	}

	// 保存缓存数据
	err = u.Srv.SaveGlobalRedis(key, kvTable, meta)
	if err != nil {
		return err
	}

	// logger.Debugf("UserActor SaveGlobalRedis,%s", key)
	// logger.Debugf("UserActor SaveGlobalRedis,%s, %s", key, utils.PrettyJson(value))
	return nil
}

/*func (u *UserActor) saveRedis(key string, kvTable *state.KvTable, meta map[string]string) error {
	// 保存缓存数据
	err := u.Srv.SaveGlobalRedis(key, kvTable, meta)
	if err != nil {
		return err
	}
	return nil
}*/

func (u *UserActor) GetRedis(key string, value proto.Message) error {
	var (
		err     error
		kvTable *state.KvTable
	)

	kvTable, err = u.getRedis(key)
	if err != nil {
		return err
	}

	if kvTable == nil {
		return nil
	}

	err = proto.Unmarshal(kvTable.Data, value)
	if err != nil {
		return err
	}

	// logger.Infof("UserActor LoadDB ret: %v, %, %v", err, key, utils.PrettyJson(value))
	return nil
}

func (u *UserActor) getRedis(key string) (*state.KvTable, error) {
	var (
		err     error
		kvTable *state.KvTable
	)

	kvTable, err = u.Srv.GetGlobalRedis(key, nil)
	if err != nil {
		return nil, err
	}

	return kvTable, nil
}

func (u *UserActor) RedisKeyExist(key string, message proto.Message) (*state.KvTable, bool) {
	reply, err := u.GetStateManager().getRedis(key)
	if err != nil || reply == nil || reply.Data == nil || len(reply.Data) == 0 {
		return nil, false
	}

	if message == nil {
		return reply, true
	}

	err = proto.Unmarshal(reply.Data, message)
	if err != nil {
		return nil, false
	}

	return reply, true
}

// 获取指定玩家的base数据块
func (u *UserActor) getRoleBaseDataByRoleId(roleId uint64) (*pb.PServerRoleBaseInfo, error) {
	uaid, err := u.Srv.GetUAIDByRoleId(roleId)
	if err != nil {
		return nil, fmt.Errorf("roleId not found %v", roleId)
	}

	data := &pb.PServerRoleBaseInfo{}
	_, err = u.GetCache(service.MongoDbType_MongoGame, db.KeyUserBaseInfo(uaid), data)
	if err != nil {
		return nil, err
	}
	// 获取是否有好友消息
	if data.Common != nil {
		data.Common.HasMessage = u.UserChatHandler.HasMessage(roleId)
	}
	return data, nil
}

// 获取指定玩家的详情数据块
func (u *UserActor) getRoleDetailInfoByRoleId(roleId uint64) (*pb.PServerRoleDetailInfo, error) {
	uaid, err := u.Srv.GetUAIDByRoleId(roleId)
	if err != nil {
		return nil, fmt.Errorf("roleId not found %v", roleId)
	}

	data := &pb.PServerRoleDetailInfo{}
	_, err = u.GetCache(service.MongoDbType_MongoGame, db.KeyRoleDetailInfo(uaid), data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// 获取指定玩家的联盟数据块
func (u *UserActor) getAllianceDataByRoleId(roleId uint64) (*pb.PUserAllianceData, error) {
	uaid, err := u.Srv.GetUAIDByRoleId(roleId)
	if err != nil {
		return nil, fmt.Errorf("roleId not found %v", roleId)
	}

	data := &pb.PUserAllianceData{}
	_, err = u.GetCache(service.MongoDbType_MongoGame, db.KeyUserAlliance(uaid), data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// 获取指定玩家的好友数据块
func (u *UserActor) getFriendDataByRoleId(roleId uint64) (*pb.PFriendData, error) {
	uaid, err := u.Srv.GetUAIDByRoleId(roleId)
	if err != nil {
		return nil, fmt.Errorf("roleId not found %v", roleId)
	}

	data := &pb.PFriendData{}
	_, err = u.GetCache(service.MongoDbType_MongoGame, db.KeyUserFriend(uaid), data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// 获取指定玩家的卡牌数据块
func (u *UserActor) getClientCardInfo(roleId uint64) []*pb.PClientCardInfo {
	var (
		cardInfos = make([]*pb.PClientCardInfo, 0)
	)
	uaid, err := u.Srv.GetUAIDByRoleId(roleId)
	if err != nil {
		return cardInfos
	}

	// 皮肤数据
	skinData := &pb.PSkinData{}
	_, err = u.GetCache(service.MongoDbType_MongoGame, db.KeyUserCardSkin(uaid), skinData)
	if err != nil {
		return cardInfos
	}

	// 卡牌数据
	cardData := &pb.PCardData{}
	_, err = u.GetCache(service.MongoDbType_MongoGame, db.KeyUserCard(uaid), cardData)
	if err != nil {
		return cardInfos
	}

	for _, card := range cardData.Card {
		if card == nil {
			continue
		}

		// clientData := u.CardHandler.ToClientData(card)
		// clientData.Common.Skins = skinData.Skins[int32(cardId)].GetSkins()
		// cardInfos = append(cardInfos, clientData)
	}

	return cardInfos
}

// 获取指定玩家的卡牌数据块
func (u *UserActor) getCardDataByRoleId(roleId uint64, cards []int32) ([]*pb.PClientCardInfo, error) {
	cardInfos := u.getClientCardInfo(roleId)

	ret := make([]*pb.PClientCardInfo, 0)
	for _, id := range cards {
		hadFound := false
		for _, cardInfo := range cardInfos {
			if int32(cardInfo.Common.CardId) != id {
				continue
			}
			hadFound = true
			ret = append(ret, cardInfo)
		}
		if !hadFound {
			ret = append(ret, &pb.PClientCardInfo{}) // 占位用
		}

	}

	return ret, nil
}
