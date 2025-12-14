package roomactor

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"gitee.com/bychannel/aniwar/src/common/datahelper"

	"gitee.com/aniwar2/musae/framework/global"

	"gitee.com/aniwar2/musae/framework/logger"

	myUtils "gitee.com/bychannel/aniwar/src/common/utils"

	"github.com/forgoer/openssl"

	"gitee.com/aniwar2/musae/framework/utils"

	"github.com/pkg/errors"

	"gitee.com/bychannel/aniwar/src/common"
	"gitee.com/bychannel/aniwar/src/common/db"

	"gitee.com/aniwar2/musae/framework/base"
	"gitee.com/aniwar2/musae/framework/service"
	"gitee.com/bychannel/aniwar/src/proto/pb"
	"google.golang.org/protobuf/proto"
)

type RoomHandler struct {
	*USBaseHandler
}

func NewRoomHandler(actor *RoomActor) *RoomHandler {
	h := &RoomHandler{USBaseHandler: NewUSBaseHandler(actor, "RoomHandler")}
	h.ChildHandler = h

	actor.RegisterProtoHandler(int32(pb.Protocols_PS2S_CreateRoomReq), h.CreateRoomReq)  // 创建房间 S2S
	actor.RegisterProtoHandler(int32(pb.Protocols_PS2S_JoinRoomReq), h.JoinRoomReq)      // 检查是否可以加入房间 S2S
	actor.RegisterProtoHandler(int32(pb.Protocols_PS2S_ExitRoomReq), h.ForceExitRoomReq) // 尝试被动退出房间 S2S

	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_EnterRoomReq), h.EnterRoomReq)       // 进入房间 C2S - (客户端第一个长链协议)
	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_UpdateLineupReq), h.UpdateLineupReq) // 编队 C2S
	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_PlayerReadyReq), h.PlayerReadyReq)   // 玩家准备操作 C2S
	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_RoomStartReq), h.RoomStartReq)       // 房间内点击游戏开始操作 C2S
	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_ExitRoomReq), h.ExitRoomReq)         // 退出房间 C2S
	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_KickPlayerReq), h.KickPlayerReq)     // 踢人出房间 C2S
	// actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_FinishLoadingReq), h.FinishLoadingReq) // 进入房间loading结束 C2S

	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_RecruitPlayerReq), h.RecruitPlayerReq)   // 招募C2S
	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_InviteIntoRoomReq), h.InviteIntoRoomReq) // 邀请进入房间C2S
	return h
}

// Init 初始化模块数据
func (h *RoomHandler) Init() error {

	return nil
}

func (h *RoomHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*pb.Room); ok {
		h.actor.Data = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *RoomHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyPvpRoomData(h.actor.ID()), h.actor.RoomData.Data
}

func (h *RoomHandler) EnterGame() error {
	// implement me
	panic("implement me")
}

func (h *RoomHandler) DailyRefresh() error {
	// implement me
	panic("implement me")
}

func (h *RoomHandler) CreateRoomReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err       error
		playerUid = in.UserId // 玩家id
	)

	var req pb.S2S_CreateRoomReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	// 必须是空闲状态
	err, code := h.checkRoomState(pb.RoomState_RoomState_idle)
	if err != nil {
		return nil, err, int32(code)
	}

	h.Infof("-----------------玩家:%s, 申请创建房间", playerUid)
	h.actor.Data.RoomId = h.actor.ID()
	h.actor.Data.RoomSecret = utils.GenStrUUID()
	h.actor.Data.RoomState = pb.RoomState_RoomState_created
	h.actor.Data.PlayType = req.PlayType
	h.actor.Data.OwnerUid = playerUid
	h.actor.Data.Players = nil
	h.actor.Data.UpdateTs = time.Now().Unix()
	_, _ = h.doRecruit(pb.RoomRecruitStateOpt_rrs_opt_cancel) // 取消招募状态
	h.actor.CleanGateTopic()

	// 加入房间
	err, errCode := h.addPlayer(playerUid, req.BaseInfo, req.Cards)
	if err != nil {
		return nil, err, int32(errCode)
	}

	// 持久化
	err = h.Cache2Redis()
	if err != nil {
		return nil, err, int32(pb.ErrorCode_SaveDBError)
	}

	rsp := &pb.S2S_CreateRoomRes{
		RoomSimple: h.GetClientRoomSimple(),
	}

	return rsp, nil, int32(pb.ErrorCode_Success)
}

func (h *RoomHandler) JoinRoomReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err       error
		playerUid = in.UserId // 玩家id
	)

	var req pb.S2S_JoinRoomReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	if h.actor.Data == nil || h.actor.Data.RoomId != req.RoomId {
		err = fmt.Errorf("房间信息:%v, 请求进入的房间id:%v", h.actor.Data, req.RoomId)
		h.Debugf(err.Error())
		return nil, err, int32(pb.ErrorCode_Room_not_exist)
	}

	err, code := h.checkRoomState(pb.RoomState_RoomState_created)
	if err != nil {
		return nil, err, int32(code)
	}
	// if req.RoomSecret != h.actor.Data.RoomSecret {
	//	return nil, err, int32(pb.ErrorCode_Room_secret_fault)
	// }
	decodeString, err := base64.URLEncoding.DecodeString(req.RoomId)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_Room_not_exist)
	}
	decrypt, err := openssl.AesECBDecrypt(decodeString, []byte(common.RoomIdSecret), openssl.PKCS7_PADDING)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_Room_not_exist)
	}
	roomID := &pb.RoomID{}
	err = json.Unmarshal(decrypt, roomID)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_Room_not_exist)
	}
	if roomID.PlayType != int32(h.actor.Data.PlayType) {
		return nil, err, int32(pb.ErrorCode_Room_not_exist)
	}

	hadPlayerCount := len(h.actor.Data.Players)
	if int32(hadPlayerCount) >= datahelper.GetMiniGamePlayerNum(h.actor.Data.PlayType) {
		return nil, errors.New(fmt.Sprintf("房间人数已满, count=%d", hadPlayerCount)), int32(pb.ErrorCode_Room_player_num_full)
	}

	// 加入房间
	err, errCode := h.addPlayer(playerUid, req.BaseInfo, req.Cards)
	if err != nil {
		return nil, err, int32(errCode)
	}

	// 持久化
	h.actor.Data.UpdateTs = time.Now().Unix()
	err = h.Cache2Redis()
	if err != nil {
		return nil, err, int32(pb.ErrorCode_SaveDBError)
	}

	rsp := &pb.S2S_JoinRoomRes{
		RoomSimple: h.GetClientRoomSimple(),
	}

	return rsp, nil, int32(pb.ErrorCode_Success)
}

func (h *RoomHandler) EnterRoomReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err error
	)

	err, code := h.checkRoomState(pb.RoomState_RoomState_created)
	if err != nil {
		return nil, err, int32(code)
	}

	var req pb.C2LS_EnterRoomReq
	err = in.UnmarshalData(&req)

	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	// 持久化
	h.actor.Data.UpdateTs = time.Now().Unix()
	err = h.Cache2Redis()
	if err != nil {
		return nil, err, int32(pb.ErrorCode_SaveDBError)
	}

	rsp := &pb.LS2C_EnterRoomRes{
		Room: h.actor.Data,
	}

	// 广播房间信息(前端要求：优先推送，自己本次不收到推送)
	h.pushRoomInfoNtf()

	// 绑定玩家和gate的主题(推送需要)
	h.actor.AddGateTopic(in.Topic, in.UserId)

	return rsp, nil, int32(pb.ErrorCode_Success)
}

// UpdateLineupReq 配置阵容
func (h *RoomHandler) UpdateLineupReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err       error
		playerUid = in.UserId // 玩家id
	)

	err, code := h.checkRoomState(pb.RoomState_RoomState_created)
	if err != nil {
		return nil, err, int32(code)
	}

	var req pb.C2LS_UpdateLineupReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	var cardNum int32 = 0
	for _, id := range req.CardIds {
		if id > 0 {
			cardNum++
		}
	}

	maxCardNum := datahelper.GetMiniGameHeroNum(h.actor.Data.PlayType)
	if cardNum > maxCardNum {
		return nil, err, int32(pb.ErrorCode_Room_player_card_num_invalid)
	}

	logger.Infof("更新阵容, %+v", req.CardIds)

	for _, player := range h.actor.Data.Players {
		if player.PlayerUid != playerUid {
			continue
		}

		// 上阵卡牌
		if player.LineupCards == nil {
			player.LineupCards = make([]*pb.PClientCardInfo, maxCardNum, maxCardNum)
		}

		for pos, cardId := range req.CardIds {
			player.LineupCards[pos] = getCardInfo(player.AllCards, cardId)
		}
	}

	// 持久化
	h.actor.Data.UpdateTs = time.Now().Unix()
	err = h.Cache2Redis()
	if err != nil {
		return nil, err, int32(pb.ErrorCode_SaveDBError)
	}

	// 广播房间信息
	h.pushRoomInfoNtf()

	rsp := &pb.LS2C_UpdateLineupRes{}

	return rsp, nil, int32(pb.ErrorCode_Success)
}

func getCardInfo(cards []*pb.PClientCardInfo, cardId int32) *pb.PClientCardInfo {
	if cardId == 0 {
		return nil
	}

	for _, card := range cards {
		if int32(card.Common.CardId) == cardId {
			return card
		}
	}

	return nil
}

// PlayerReadyReq 准备、取消准备
func (h *RoomHandler) PlayerReadyReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err       error
		playerUid = in.UserId // 玩家id
	)
	err, code := h.checkRoomState(pb.RoomState_RoomState_created)
	if err != nil {
		return nil, err, int32(code)
	}

	needCardNum := datahelper.GetMiniGameHeroNum(h.actor.Data.PlayType)

	var lineupCardNum int32 = 0
	for _, player := range h.actor.Data.Players {
		if player.PlayerUid != playerUid {
			continue
		}

		for _, card := range player.LineupCards {
			if card != nil {
				lineupCardNum++
			}
		}
	}
	if lineupCardNum != needCardNum { // 阵上卡牌数量必须满足条件
		return nil, err, int32(pb.ErrorCode_Room_player_card_num_invalid)
	}

	var req pb.C2LS_PlayerReadyReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	h.doPlayerReady(playerUid, req.ReadyOpt)

	// 持久化
	h.actor.Data.UpdateTs = time.Now().Unix()
	err = h.Cache2Redis()
	if err != nil {
		return nil, err, int32(pb.ErrorCode_SaveDBError)
	}

	// 广播房间信息
	h.pushRoomInfoNtf()

	rsp := &pb.LS2C_PlayerReadyRes{}

	return rsp, nil, int32(pb.ErrorCode_Success)
}

func (h *RoomHandler) RoomStartReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err       error
		playerUid = in.UserId // 玩家id
	)

	var req pb.C2LS_RoomStartReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	err, code := h.checkRoomState(pb.RoomState_RoomState_created)
	if err != nil {
		return nil, err, int32(code)
	}
	if !h.isOwner(playerUid) {
		return nil, errors.New("只能房主可以开始游戏"), int32(pb.ErrorCode_Room_only_owner_opt)
	}

	if len(h.actor.Data.Players) <= 1 {
		return nil, errors.New("人数不足"), int32(pb.ErrorCode_Room_player_not_enough)
	}

	// 房主准备
	h.doPlayerReady(playerUid, pb.RoomPlayerStateOpt_rps_opt_do_ready)

	// 检查玩家是否全部准备
	for _, player := range h.actor.Data.Players {
		if player.PlayerState != pb.RoomPlayerState_rps_readying {
			return nil, errors.New("还有玩家未准备"), int32(pb.ErrorCode_Room_not_all_player_ready)
		}
	}

	// 房间状态
	h.actor.Data.RoomState = pb.RoomState_RoomState_playing
	// // 玩家状态
	// for _, player := range h.actor.Data.UserMap {
	//	player.PlayerState = pb.RoomPlayerState_rps_enter_game_load
	// }

	// 持久化
	h.actor.Data.UpdateTs = time.Now().Unix()
	err = h.Cache2Redis()
	if err != nil {
		return nil, err, int32(pb.ErrorCode_SaveDBError)
	}

	// 广播房间信息
	h.pushRoomInfoNtf()

	h.gameStart()

	rsp := &pb.LS2C_RoomStartRes{}
	return rsp, nil, int32(pb.ErrorCode_Success)
}

func (h *RoomHandler) ForceExitRoomReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req pb.S2S_ExitRoomReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	// 退出房间
	h.doExitRoom(in.UserId, pb.ExitRoomReason_exitRR_bySelf)

	if h.isOwner(in.UserId) {
		// 房主退出，解散房间
		for _, player := range h.actor.Data.Players {
			h.doExitRoom(player.PlayerUid, pb.ExitRoomReason_exitRR_room_dismissed)
		}
		h.actor.Data.RoomState = pb.RoomState_RoomState_idle
	}

	// 持久化
	err = h.Cache2Redis()
	if err != nil {
		return nil, err, int32(pb.ErrorCode_SaveDBError)
	}

	// 广播房间信息
	h.pushRoomInfoNtf()

	return &pb.S2S_ExitRoomRes{}, nil, int32(pb.ErrorCode_Success)
}

func (h *RoomHandler) ExitRoomReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err       error
		playerUid = in.UserId // 玩家id
	)

	var req pb.C2LS_ExitRoomReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	err, code := h.checkRoomState(pb.RoomState_RoomState_created)
	if err != nil {
		return nil, err, int32(code)
	}

	// 退出房间
	h.doExitRoom(playerUid, pb.ExitRoomReason_exitRR_bySelf)

	if h.isOwner(playerUid) {
		// 房主退出，解散房间
		for _, player := range h.actor.Data.Players {
			h.doExitRoom(player.PlayerUid, pb.ExitRoomReason_exitRR_room_dismissed)
		}
		h.actor.Data.RoomState = pb.RoomState_RoomState_idle
	}

	// 持久化
	h.actor.Data.UpdateTs = time.Now().Unix()
	err = h.Cache2Redis()
	if err != nil {
		return nil, err, int32(pb.ErrorCode_SaveDBError)
	}

	// 广播房间信息
	h.pushRoomInfoNtf()

	rsp := &pb.LS2C_ExitRoomRes{}
	return rsp, nil, int32(pb.ErrorCode_Success)
}

func (h *RoomHandler) KickPlayerReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err       error
		playerUid = in.UserId // 玩家id
	)

	err, code := h.checkRoomState(pb.RoomState_RoomState_created)
	if err != nil {
		return nil, err, int32(code)
	}

	if !h.isOwner(playerUid) {
		return nil, errors.New("只有房主可以踢出玩家"), int32(pb.ErrorCode_Room_only_owner_opt)
	}

	var req pb.C2LS_KickPlayerReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	// 退出房间
	h.doExitRoom(req.KickedPlayerUid, pb.ExitRoomReason_exitRR_kicked)

	// 持久化
	h.actor.Data.UpdateTs = time.Now().Unix()
	err = h.Cache2Redis()
	if err != nil {
		return nil, err, int32(pb.ErrorCode_SaveDBError)
	}

	// 广播房间信息
	h.pushRoomInfoNtf()

	rsp := &pb.LS2C_KickPlayerRes{}
	return rsp, nil, int32(pb.ErrorCode_Success)
}

func (h *RoomHandler) RecruitPlayerReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		playerUid = in.UserId // 玩家id
	)

	if !h.isOwner(playerUid) {
		return nil, errors.New("只有房主可以招募"), int32(pb.ErrorCode_Room_only_owner_opt)
	}
	err, code := h.checkRoomState(pb.RoomState_RoomState_created)
	if err != nil {
		return nil, err, int32(code)
	}

	roomSimple := h.GetClientRoomSimple()
	if roomSimple == nil {
		return nil, errors.New("获取房间信息失败"), int32(pb.ErrorCode_Room_not_exist)
	}

	var req pb.C2LS_RecruitPlayerReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	if err, errCode := h.doRecruit(req.Opt); err != nil {
		return nil, err, int32(errCode)
	}

	// 持久化
	h.actor.Data.UpdateTs = time.Now().Unix()
	err = h.Cache2Redis()
	if err != nil {
		return nil, err, int32(pb.ErrorCode_SaveDBError)
	}

	// 广播招募消息
	if req.Opt == pb.RoomRecruitStateOpt_rrs_opt_start {
		h.pushRecruitPlayerNtf(in.RoleId, roomSimple)
	}
	rsp := &pb.LS2C_RecruitPlayerRes{}
	return rsp, nil, int32(pb.ErrorCode_Success)
}

func (h *RoomHandler) InviteIntoRoomReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err           error
		fromPlayerUid = in.UserId // 玩家id
		fromUaid      = in.UAID
	)

	if !h.isOwner(fromPlayerUid) {
		return nil, errors.New("只有房主可以招募"), int32(pb.ErrorCode_Room_only_owner_opt)
	}
	err, code := h.checkRoomState(pb.RoomState_RoomState_created)
	if err != nil {
		return nil, err, int32(code)
	}

	var req pb.C2LS_InviteIntoRoomReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	// 对方是否在房间中
	uaid, err := h.actor.Srv.GetUAIDByRoleId(req.InvitedRoleId)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_NotFoundPlayer)
	}
	uid, _ := h.actor.Srv.ConvUAID(uaid)
	if h.actor.Srv.CheckInRoom(uid) {
		return nil, err, int32(pb.ErrorCode_Room_player_in_other_room)
	}

	fromPlayer := h.GetPlayer(fromPlayerUid)

	inviteReq := &pb.S2S_InviteIntoRoomReq{
		FromRoleId: fromPlayer.BaseInfo.Common.RoleId,
		ToRoleId:   req.InvitedRoleId,
		RoomId:     h.actor.Data.RoomId,
		PlayType:   h.actor.Data.PlayType,
	}
	inviteReqData, err := proto.Marshal(inviteReq)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}
	// toUaid, err := h.actor.Srv.GetUAIDByRoleId(req.InvitedRoleId)
	// if err != nil {
	//	return nil, err, int32(pb.ErrorCode_InternalError)
	// }
	// toUid, _ := h.actor.Srv.ConvUAID(toUaid)
	inviteRespData, err := h.actor.Srv.UserInvoke(fromUaid, &base.ProtoMsg{
		AppId:        global.ACTOR_SVC,
		MsgId:        int32(pb.Protocols_PS2S_InviteIntoRoomReq),
		ServerReqIdx: utils.GenIntUUID(),
		UserId:       fromPlayerUid,
		RoleId:       req.InvitedRoleId,
		UAID:         fromUaid,
		Data:         inviteReqData,
		ErrCode:      0,
	})
	if err != nil {
		return nil, err, int32(pb.ErrorCode_RpcInvokeError)
	}
	inviteResp := &pb.S2S_FetchUserInfoRes{}
	err = proto.Unmarshal(inviteRespData.Data, inviteResp)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	h.actor.Data.UpdateTs = time.Now().Unix()
	err = h.Cache2Redis()
	if err != nil {
		return nil, err, int32(pb.ErrorCode_SaveDBError)
	}

	resp := &pb.LS2C_InviteIntoRoomRes{}
	return resp, nil, int32(pb.ErrorCode_Success)
}

func (h *RoomHandler) isOwner(playerUid string) bool {
	return playerUid == h.actor.Data.OwnerUid
}

func (h *RoomHandler) GetPlayer(uid string) *pb.RoomPlayer {
	for _, player := range h.actor.Data.Players {
		if player.PlayerUid == uid {
			return player
		}
	}
	return nil
}

// func (h *RoomHandler) FinishLoadingReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
//	var (
//		err       error
//		playerUid = in.UserId // 玩家id
//	)
//
//	var req pb.C2LS_FinishEnterLoadReq
//	err = in.UnmarshalData(&req)
//	if err != nil {
//		return nil, err, int32(pb.ErrorCode_DeSerializeError)
//	}
//
//	for _, player := range h.actor.Data.UserMap {
//		if player.PlayerUid == playerUid {
//			player.PlayerState = pb.RoomPlayerState_rps_finish_game_load
//			break
//		}
//	}
//
//	// 启动定时器
//	h.tugCreateTick()
//
//	rsp := &pb.LS2C_FinishEnterLoadRes{}
//
//	return rsp, nil, int32(pb.ErrorCode_Success)
// }

func (h *RoomHandler) addPlayer(playerUid string, baseInfo *pb.PClientRoleBaseInfo, cards []*pb.PClientCardInfo) (error, pb.ErrorCode) {
	if h.actor.Data.Players == nil {
		h.actor.Data.Players = make([]*pb.RoomPlayer, 0)
	}

	if player := h.GetPlayer(playerUid); player != nil {
		return errors.New(fmt.Sprintf("playerUid=%s, 已经进入房间了", playerUid)), pb.ErrorCode_Room_player_had_enter_room
	}
	// for _, player := range h.actor.Data.Players {
	//	if player.PlayerUid == playerUid {
	//		return errors.New(fmt.Sprintf("playerUid=%s, 已经进入房间了", playerUid)), pb.ErrorCode_Room_player_had_enter_room
	//	}
	// }

	player := &pb.RoomPlayer{
		PlayerUid: playerUid,
		// Score:       0,
		PlayerState: pb.RoomPlayerState_rps_not_ready,
		BaseInfo:    baseInfo,
		AllCards:    cards,
	}

	h.actor.Data.Players = append(h.actor.Data.Players, player)

	return nil, pb.ErrorCode_Success
}

func (h *RoomHandler) doExitRoom(playerUid string, reason pb.ExitRoomReason) {
	// 解除绑定roomId
	err := h.actor.Srv.SaveRoomBindingData(playerUid, "")
	if err != nil {
		h.Debug(err)
	}

	var delIdx = -1
	for i, player := range h.actor.Data.Players {
		if player.PlayerUid == playerUid {
			delIdx = i
			break
		}
	}
	if delIdx == -1 {
		// 未找到
		return
	}

	h.actor.Data.Players = append(h.actor.Data.Players[:delIdx], h.actor.Data.Players[delIdx+1:]...)

	// 推送被踢消息
	err = h.pushExitRoomNtf(playerUid, reason)
	if err != nil {
		_ = errors.Wrapf(err, "被提玩家已经不在房间里了, playerUid=%s", playerUid)
		h.Debug(err.Error())
	}
	h.Infof("玩家 %s 由于 %v 退出房间了", playerUid, reason)
}

func (h *RoomHandler) GetClientRoom() *pb.Room {
	clientRoom := &pb.Room{}

	err := myUtils.DeepCopyByJson(h.actor.Data, clientRoom)
	if err != nil {
		return nil
	}

	// 所有卡牌不下发给客户端
	for _, player := range clientRoom.Players {
		player.AllCards = nil
	}

	return clientRoom
}

func (h *RoomHandler) GetClientRoomSimple() *pb.RoomSimple {
	// 获取房主信息
	var roleInfo *pb.PCommonRoleBaseInfo
	for _, v := range h.actor.Data.GetPlayers() {
		if v.PlayerUid == h.actor.Data.OwnerUid {
			roleInfo = v.BaseInfo.Common
		}
	}

	return &pb.RoomSimple{
		RoomId:    h.actor.Data.RoomId,
		RoomState: h.actor.Data.RoomState,
		PlayType:  h.actor.Data.PlayType,
		IsRecruit: h.actor.Data.IsRecruit,
		RoleInfo:  roleInfo,
	}
}

// 推送房间信息
func (h *RoomHandler) pushRoomInfoNtf() {
	ntf := &pb.LS2C_RoomStateNtf{
		Room: h.GetClientRoom(),
	}
	h.Infof("推送消息 userMap:%v", h.actor.UserMap)
	err := h.actor.Srv.Send2Gates(h.actor.Data.RoomId, h.actor.UserMap, ntf)
	if err != nil {
		return
	}
}

// 推送退出房间的玩家信息
func (h *RoomHandler) pushExitRoomNtf(exitPlayerUid string, reason pb.ExitRoomReason) error {
	ntf := &pb.LS2C_KickedRoomNtf{
		Reason: reason,
	}

	err := h.actor.Srv.Send2Gate(h.actor.Data.RoomId, h.actor.UserMap[exitPlayerUid], ntf)
	if err != nil {
		return err
	}

	// 接触绑定玩家和gate的主题(推送需要)
	h.actor.DelGateTopic(exitPlayerUid)
	return nil
}

// 推送招募消息
func (h *RoomHandler) pushRecruitPlayerNtf(fromRoleId uint64, roomSimple *pb.RoomSimple) error {
	ntf := &pb.LS2C_NotifyMessage{
		ChannelId: pb.ChatChannel_Channel_recruit,
	}
	byteInfo, _ := json.Marshal(roomSimple)
	message := &pb.BroadMessage{
		MType:      pb.MessageType_Message_Type_Recruit,
		FromRoleId: fromRoleId,
		Data:       []string{string(byteInfo)},
		TimeStamp:  time.Now().Unix(),
	}
	ntf.Message = append(ntf.Message, message)
	// 推送方法修改
	err := h.actor.Srv.Send2BC(strconv.Itoa(int(fromRoleId)), ntf)
	if err != nil {
		return err
	}
	return nil
}

func (h *RoomHandler) doPlayerReady(playerUid string, opt pb.RoomPlayerStateOpt) {
	player := h.GetPlayer(playerUid)
	if player == nil {
		h.Debugf(fmt.Sprintf("不存在的玩家数据, playerUid=%s", playerUid))
		return
	}

	switch opt {
	case pb.RoomPlayerStateOpt_rps_opt_do_ready: // 执行准备操作
		player.PlayerState = pb.RoomPlayerState_rps_readying
	case pb.RoomPlayerStateOpt_rps_opt_cancel_ready: // 执行取消准备操作
		player.PlayerState = pb.RoomPlayerState_rps_not_ready
	default:
		h.Errorf(fmt.Sprintf("未支持的操作类型, opt=%d", opt))
	}

	// for _, player := range h.actor.Data.Players {
	//	if player.PlayerUid == playerUid {
	//		switch opt {
	//		case pb.RoomPlayerStateOpt_rps_opt_do_ready: // 执行准备操作
	//			player.PlayerState = pb.RoomPlayerState_rps_readying
	//		case pb.RoomPlayerStateOpt_rps_opt_cancel_ready: // 执行取消准备操作
	//			player.PlayerState = pb.RoomPlayerState_rps_not_ready
	//		default:
	//			h.Errorf(fmt.Sprintf("未支持的操作类型, opt=%d", opt))
	//		}
	//		break
	//	}
	// }
}

func (h *RoomHandler) gameStart() {
	switch h.actor.Data.PlayType {
	case pb.RoomModel_RoomModel_tug:
		h.actor.TugHandler.tugGameStart(pb.RoomModel_RoomModel_tug)
	default:
		h.Debugf("tugGameStart 未支持的操作类型, playType:%v", h.actor.Data.PlayType)
	}
}

func (h *RoomHandler) gameBack2Room() {
	for _, player := range h.actor.Data.Players {
		// if h.isOwner(player.PlayerUid) {
		//	// 房主不需要取消准备
		//	continue
		// }

		h.doPlayerReady(player.PlayerUid, pb.RoomPlayerStateOpt_rps_opt_cancel_ready)
	}

	h.actor.Data.RoomState = pb.RoomState_RoomState_created

	err := h.Cache2Redis()
	if err != nil {
		h.Errorf(err.Error())
		return
	}

	h.pushRoomInfoNtf()
}

func (h *RoomHandler) doRecruit(opt pb.RoomRecruitStateOpt) (error, pb.ErrorCode) {
	switch opt {
	case pb.RoomRecruitStateOpt_rrs_opt_start:
		h.actor.Data.IsRecruit = 1
	case pb.RoomRecruitStateOpt_rrs_opt_cancel:
		h.actor.Data.IsRecruit = 0
	default:
		err := errors.Errorf("无效的操作类型, opt:%v", opt)
		h.Errorf(err.Error())
		return err, pb.ErrorCode_InvalidParam
	}

	return nil, pb.ErrorCode_Success
}

// 检查房间状态是否正确
func (h *RoomHandler) checkRoomState(state pb.RoomState) (error, pb.ErrorCode) {
	// 正确的状态
	if h.actor.Data.RoomState == state {
		return nil, pb.ErrorCode_Success
	}

	// 状态不正确，判定错误提示
	// 空闲中
	if h.actor.Data.RoomState == pb.RoomState_RoomState_idle {
		return fmt.Errorf("room state is idle"), pb.ErrorCode_Room_state_is_idle
	}
	// 准备中
	if h.actor.Data.RoomState == pb.RoomState_RoomState_created {
		return fmt.Errorf("room state in ready"), pb.ErrorCode_Room_state_in_ready
	}
	// 游戏中
	if h.actor.Data.RoomState == pb.RoomState_RoomState_playing {
		return fmt.Errorf("room state is playing"), pb.ErrorCode_Room_is_in_game_playing
	}

	return fmt.Errorf("room state illegal %v", h.actor.Data.RoomState), pb.ErrorCode_InternalError
}

// 系统解散房间
func (h *RoomHandler) dismissRoomBySystem() {
	for _, player := range h.actor.Data.Players {
		h.doExitRoom(player.PlayerUid, pb.ExitRoomReason_exitRR_dismiss_bySystem)
	}
}
