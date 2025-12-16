package useractor

import (
	"context"
	"fmt"
	"gitee.com/aniwar2/aniwar/src/actorserver/useractor/event"
	"gitee.com/aniwar2/aniwar/src/common"
	"strconv"
	"time"

	"gitee.com/aniwar2/aniwar/src/common/db"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/base"
	"gitee.com/aniwar2/musae/service"
	"google.golang.org/protobuf/proto"
)

type RoleDetailHandler struct {
	*UABaseHandler
}

func NewRoleDetailHandler(actor *UserActor) *RoleDetailHandler {
	h := &RoleDetailHandler{UABaseHandler: NewUABaseHandler(actor, "RoleDetailHandler")}
	h.ChildHandler = h

	// 协议注册
	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_GetRoleInfoReq), h.GetRoleInfoReq)
	h.actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_ChangeShowCardsReq), h.ChangeShowCardsReq)

	return h
}

// Init 初始化模块数据
func (h *RoleDetailHandler) Init() error {
	// 初始化
	h.actor.Data.Detail = &pb.PServerRoleDetailInfo{
		Createtime: time.Now().Unix(),
		Common:     h.actor.GetUserData().Common,
		Cards:      make([]int32, 4, 4),
		Lifex:      make(map[int32]int32),
	}

	if err := h.SaveDB(); err != nil {
		return err
	}

	h.Debug("init role detail data success.")
	return nil
}

func (h *RoleDetailHandler) EnterGame() error {
	return h.tryRefreshRoleDetail()
}

func (h *RoleDetailHandler) DailyRefresh() error {
	return nil
}

func (h *RoleDetailHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*pb.PServerRoleDetailInfo); ok {
		h.actor.Data.Detail = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *RoleDetailHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyRoleDetailInfo(h.actor.ID()), h.actor.Data.Detail
}

func (h *RoleDetailHandler) GetRoleInfoReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req pb.C2LS_GetRoleInfoReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	info, err := h.actor.getRoleDetailInfoByRoleId(req.RoleId)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_NotFoundPlayer)
	}

	// 返回消息
	return &pb.LS2C_GetRoleInfoRes{Info: h.actor.LoginHandler.toClientDetailInfo(info)}, nil, 0
}

func (h *RoleDetailHandler) ChangeShowCardsReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req pb.C2LS_ChangeShowCardsReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	// 卡牌判定
	// for _, card := range req.Cards {
	// 	if card > 0 && !h.actor.CardHandler.IsExistCard(uint32(card)) {
	// 		return nil, fmt.Errorf("card not exist %d", card), int32(pb.ErrorCode_ParamError)
	// 	}
	// }

	// 保存
	detailData := h.actor.GetRoleDetailData()
	detailData.Cards = req.Cards
	if err = h.SaveDB(); err != nil {
		return nil, err, int32(pb.ErrorCode_SaveDBError)
	}
	if err = h.tryUploadRoleInfoToES(); err != nil {
		h.Error(err)
	}
	// 返回
	cards := make([]*pb.PClientCardInfo, 0)
	// for _, id := range req.Cards {
	// 	card, _ := h.actor.CardHandler.GetCard(uint32(id))
	// 	if card != nil {
	// 		clientData := h.actor.CardHandler.ToClientData(card)
	// 		cards = append(cards, clientData)
	// 	} else {
	// 		cards = append(cards, &pb.PClientCardInfo{}) // 占位用
	// 	}
	// }
	return &pb.LS2C_ChangeShowCardsRes{Cards: cards}, nil, 0
}

func (h *RoleDetailHandler) tryRefreshRoleDetail() error {
	data := h.actor.GetRoleDetailData()
	// data.Lifex[0] = int32(h.actor.CardHandler.GetCardCount())
	data.Lifex[1] = 0 /*h.actor.AchieveHandler.GetCompleteCount()*/
	data.Lifex[2] = h.actor.LoginHandler.getLoginDay()
	if err := h.SaveDB(); err != nil {
		return err
	}
	return nil
}

// 处理角色生涯数据更新
func (h *RoleDetailHandler) tryHandleRoleLife(e event.IEvent) error {
	data := h.actor.GetRoleDetailData()
	name := e.Name()
	// 更新生涯数据
	switch name {
	case TASK_EVENT_CARD_CREATE:
		data.Lifex[0] = e.Get("total").(int32)
	case TASK_EVENT_ACHIEVE_COMPLETE:
		data.Lifex[1] = e.Get("complete_num").(int32)
	case TASK_EVENT_PLAYER_LOGIN:
		data.Lifex[2] = e.Get("login_day").(int32)
	default:
		h.Warnf("tryHandleRoleLife unrealized event type %s", name)
		return nil
	}
	if err := h.SaveDB(); err != nil {
		h.Errorf("tryHandleRoleLife got err: %v", err)
	}

	// 尝试同步es
	if err := h.tryUploadRoleInfoToES(); err != nil {
		h.Errorf("tryHandleRoleLife got err: %v", err)
	}
	return nil
}

// 同步数据到es中
func (h *RoleDetailHandler) tryUploadRoleInfoToES() error {
	// 判断是否开启相关的功能
	if !h.actor.FuncUnlockHandler.CheckFuncsUnlock([]int32{FUNC_ID_Friend}) {
		return nil
	}
	// 同步
	data := h.actor.GetRoleDetailData()
	err := h.actor.Srv.ESPut(common.ES_ROLE_DETAIL_KEY, strconv.Itoa(int(h.actor.roleId)), data)
	if err != nil {
		h.Error(err)
		return err
	}

	h.Infof("同步玩家到es中... roleId: %d, level: %d", data.Common.RoleId, data.Common.RoleLevel)
	return err
}

func (h *RoleDetailHandler) ChangeNickname(nickname string) {
	h.actor.GetRoleDetailData().Common.RoleName = nickname
	if err := h.SaveDB(); err != nil {
		h.Errorf("RoleDetailHandler 修改昵称报错, err:%v", err.Error())
	}
	if err := h.tryUploadRoleInfoToES(); err != nil {
		h.Error(err)
	}
}

func (h *RoleDetailHandler) ChangeHeadId(headId int32) {
	h.actor.GetRoleDetailData().Common.RoleHead = headId
	if err := h.SaveDB(); err != nil {
		h.Errorf("RoleDetailHandler 修改头像报错, err:%v", err.Error())
	}
	if err := h.tryUploadRoleInfoToES(); err != nil {
		h.Error(err)
	}
}

func (h *RoleDetailHandler) ChangeRoleSex(sex int32) {
	h.actor.GetRoleDetailData().Common.RoleSex = uint32(sex)
	if err := h.SaveDB(); err != nil {
		h.Errorf("RoleDetailHandler 修改头像报错, err:%v", err.Error())
	}
	if err := h.tryUploadRoleInfoToES(); err != nil {
		h.Error(err)
	}
}

func (h *RoleDetailHandler) ChangeRoleLevel(exp uint64, level uint32) {
	data := h.actor.GetRoleDetailData()
	data.Common.RoleExp = exp
	data.Common.RoleLevel = level
	if err := h.SaveDB(); err != nil {
		h.Errorf("RoleDetailHandler 修改数据报错, err:%v", err.Error())
	}
	if err := h.tryUploadRoleInfoToES(); err != nil {
		h.Error(err)
	}
}

func (h *RoleDetailHandler) ChangeOnlineTime(now int64) {
	h.actor.GetRoleDetailData().Common.OnlineTime = now
	if err := h.SaveDB(); err != nil {
		h.Errorf("RoleDetailHandler 修改数据报错, err:%v", err.Error())
	}
}

func (h *RoleDetailHandler) ChangeOfflineTime(now int64) {
	h.actor.GetRoleDetailData().Common.OfflineTime = now
	if err := h.SaveDB(true); err != nil {
		h.Error(err)
	}
	// 离线了，同步一次
	if now > -1 {
		if err := h.tryUploadRoleInfoToES(); err != nil {
			h.Error(err)
		}
	}
}

func (h *RoleDetailHandler) ChangeLoginDay() {
	h.actor.GetRoleDetailData().Common.LoginDay++
	if err := h.SaveDB(); err != nil {
		h.Errorf("RoleDetailHandler 修改数据报错, err:%v", err.Error())
	}
}
