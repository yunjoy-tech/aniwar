package useractor

import (
	"context"
	"fmt"
	"gitee.com/aniwar2/musae/framework/threading"
	"gitee.com/bychannel/aniwar/src/common/datalog/taptap"
	"strconv"
	"time"

	"gitee.com/aniwar2/musae/framework/base"

	"google.golang.org/protobuf/proto"

	"gitee.com/aniwar2/musae/framework/service"
	"gitee.com/bychannel/aniwar/src/common/db"

	"gitee.com/bychannel/aniwar/src/proto/pb"
)

type AccountHandler struct {
	*UABaseHandler
}

func NewAccountHandler(actor *UserActor) *AccountHandler {
	h := &AccountHandler{UABaseHandler: NewUABaseHandler(actor, "AccountHandler")}
	h.ChildHandler = h

	actor.RegisterProtoHandler(int32(pb.Protocols_PS2S_TcpGateTopicReq1), h.TcpGateTopicReq1) // gate通知userActor, 广播到所有的actors更新topic
	// actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_UseItemReq), h.UseItemReq)                     // 使用道具
	// actor.RegisterProtoHandler(int32(pb.Protocols_PLS2S_DestroyExpireItemReq), h.DestroyExpireItemReq) // 销毁过期道具
	return h
}

func (h *AccountHandler) Init() error {
	return nil
}

func (h *AccountHandler) EnterGame() error {
	return nil
}

func (h *AccountHandler) DailyRefresh() error {
	return nil
}

func (h *AccountHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*pb.UserData); ok {
		h.actor.Account = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *AccountHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoAccount, db.KeyAccountInfo(h.actor.GetUID()), h.actor.Account
}

func (h *AccountHandler) TcpGateTopicReq1(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err error
	)

	var req pb.S2S_TcpGateTopicReq1
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	req2 := &pb.S2S_TcpGateTopicReq2{
		Opt:    req.Opt,
		Uid:    req.Uid,
		GateId: req.GateId,
	}
	// gateTopicData, err := proto.Marshal(req2)
	// if err != nil {
	//	return nil, err, int32(pb.ErrorCode_InternalError)
	// }
	// h.Debugf("收到gate通知, useractor广播到所有的actors, 更新topic, uaid:%s, req:%+v", h.actor.uid, &req)

	h.actor.UpdateGateTopic(req2)

	// TODO HDY 通知roomActor更新gateTopic

	// _, err = h.actor.Srv.UserInvoke(h.actor.Srv.UAID(h.actor.uid, h.actor.roleId), &base.ProtoMsg{
	//	AppId:        global.ACTOR_SVC,
	//	MsgId:        int32(pb.Protocols_PS2S_TcpGateTopicReq2),
	//	ServerReqIdx: utils.GenIntUUID(),
	//	UserId:       h.actor.uid,
	//	RoleId:       0,
	//	UAID:         h.actor.Srv.UAID(h.actor.ID(), h.actor.roleId),
	//	Data:         gateTopicData,
	//	ErrCode:      0,
	// })
	// if err != nil {
	//	return nil, err, int32(pb.ErrorCode_RpcInvokeError)
	// }

	// _, err, invokeErr := h.actor.Srv.ActorInvoke(global.RoomActorType, h.actor.uid, &base.ProtoMsg{ // FIXME 此处的actorId不正确, 需要传入roomId
	//	AppId:        global.ACTOR_SVC,
	//	MsgId:        int32(pb.Protocols_PS2S_TcpGateTopicReq2),
	//	ServerReqIdx: utils.GenIntUUID(),
	//	UserId:       h.actor.uid,
	//	RoleId:       0,
	//	UAID:         h.actor.Srv.UAID(h.actor.ID(), h.actor.roleId),
	//	Data:         gateTopicData,
	//	ErrCode:      0,
	// })
	// if invokeErr != nil {
	//	return nil, invokeErr, int32(pb.ErrorCode_RpcInvokeError)
	// } else if err != nil {
	//	return nil, err, int32(pb.ErrorCode_RpcInvokeError)
	// }

	return &pb.S2S_TcpGateTopicRes1{}, nil, int32(pb.ErrorCode_Success)
}

// SavePlayer 保存角色信息
func (h *AccountHandler) SavePlayer(playerId uint64) {
	var (
		curTime = time.Now().Unix()
	)

	h.actor.Account.PlayerList.Players[1] = &pb.Player{Id: playerId, CreateTs: curTime}
	h.actor.Account.PlayerList.PlayerId = playerId
	h.actor.Account.PlayerList.UpdateTs = curTime

	err := h.SaveDB(true)
	if err != nil {
		h.SaveDB()
		h.Errorf("AccountHandler 报错账号数据报错, err:%v", err.Error())
	}
}

// ChangeNickname 修改昵称
func (h *AccountHandler) ChangeNickname(nickname string) {
	account := h.actor.Account.Account
	account.Nickname = nickname

	if err := h.SaveDB(); err != nil {
		h.Errorf("AccountHandler 修改昵称报错, err:%v", err.Error())
	}
}

// Banned 玩家封禁
// bannedMsg表示封禁的原因，解封可以传空字符串
func (h *AccountHandler) Banned(bannedMsg string, bannedSec int64) error {
	account := h.actor.Account.Account
	if bannedSec > 0 { // 封禁
		account.BannedTs = time.Now().Add(time.Second * time.Duration(bannedSec)).Unix()
		account.BannedMsg = bannedMsg

		threading.RunSafe(func() {
			e := &taptap.BanRole{
				PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
				RoleId:            strconv.FormatUint(h.actor.roleId, 10),
				UnlockTime:        strconv.FormatUint(uint64(time.Second*time.Duration(bannedSec)), 10),
				Reason:            "0",
				BanSource:         "0",
				BanReason:         bannedMsg,
			}
			taptap.WriteDataLog(taptap.LogType_BanRole, h.actor.uid, h.actor.Account.TapUserInfo, e)
		})
	} else {
		account.BannedTs = bannedSec
		account.BannedMsg = bannedMsg

		threading.RunSafe(func() {
			e := &taptap.UnBanRole{
				PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
				RoleId:            strconv.FormatUint(h.actor.roleId, 10),
				Reason:            bannedMsg,
			}
			taptap.WriteDataLog(taptap.LogType_UnBanRole, h.actor.uid, h.actor.Account.TapUserInfo, e)
		})
	}

	// threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.BanRole{
	//		LogType:    lilith.LogType_BanRole,
	//		Version:    "1",
	//		EventTime:  time.Now().Format(lilith.FORMAT_DATETIME_LOG),
	//		GameId:     conf.GConf().Sdk.GameId,
	//		OpenId:     lilith.GetOpenId(h.actor.GetUID()),
	//		ServerId:   "0",
	//		RoleId:     strconv.FormatUint(h.actor.roleId, 10),
	//		UnlockTime: strconv.FormatUint(uint64(time.Second*time.Duration(bannedSec)), 10),
	//		Reason:     "0",
	//		BanSource:  "0",
	//		BanReason:  "0",
	//	})
	// })

	err := h.SaveDB(true)
	if err != nil {
		h.Errorf("BannedUser got err:%v", err.Error())
	}

	// actor下线
	err = h.actor.Srv.DeleteActor(h.actor.Type(), h.actor.ID())
	if err != nil {
		h.Errorf("BannedUser got err:%v", err.Error())
	}

	h.Infof("BannedUser:%s, sec:%v", h.actor.ID(), bannedSec)
	return nil
}
