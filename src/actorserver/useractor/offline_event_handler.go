package useractor

import (
	"context"
	"fmt"
	"gitee.com/bychannel/musae/framework/base"
	"gitee.com/bychannel/musae/framework/threading"
	"time"

	"google.golang.org/protobuf/proto"

	"gitee.com/bychannel/aniwar/src/common/db"
	"gitee.com/bychannel/musae/framework/service"

	"gitee.com/bychannel/aniwar/src/common"
	"gitee.com/bychannel/aniwar/src/proto/pb"
)

type OfflineEventHandler struct {
	*UABaseHandler
}

func NewOfflineEventHandler(actor *UserActor) *OfflineEventHandler {
	h := &OfflineEventHandler{UABaseHandler: NewUABaseHandler(actor, "OfflineEventHandler")}
	h.SetSupportMini()
	h.ChildHandler = h

	// 离线事件调用
	// 好友模块
	actor.RegisterProtoHandler(int32(pb.Protocols_PS2S_DelFriendReq), h.SrvDelFriendReq)
	actor.RegisterProtoHandler(int32(pb.Protocols_PS2S_AddFriendApplyReq), h.SrvAddFriendApplyReq)
	actor.RegisterProtoHandler(int32(pb.Protocols_PS2S_AgreeFriendApplyReq), h.SrvAgreeFriendApplyReq)
	actor.RegisterProtoHandler(int32(pb.Protocols_PS2S_SendFriendPointReq), h.SendFriendPointReq)
	// 联盟模块
	actor.RegisterProtoHandler(int32(pb.Protocols_PS2S_ExitAllianceNtf), h.ExitAllianceNtf)
	actor.RegisterProtoHandler(int32(pb.Protocols_PS2S_JoinAllianceNtf), h.JoinAllianceNtf)
	actor.RegisterProtoHandler(int32(pb.Protocols_PS2S_CheckInAllianceReq), h.CheckInAllianceReq)
	// 聊天模块
	actor.RegisterProtoHandler(int32(pb.Protocols_PS2S_PushMessageToUserReq), h.PushMessageToUserReq) // 私聊推送消息
	return h
}

func (h *OfflineEventHandler) Init() error {
	h.actor.Data.OfflineEventData = &pb.POfflineEventData{
		Createtime: time.Now().Unix(),
		EventList:  make(map[int64]*pb.OfflineEvent),
	}

	// 保存
	if err := h.SaveDB(); err != nil {
		return err
	}

	h.Debug("init offline event data success.")
	return nil
}

func (h *OfflineEventHandler) EnterGame() error {
	return nil
}

func (h *OfflineEventHandler) DailyRefresh() error {
	return nil
}

func (h *OfflineEventHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*pb.POfflineEventData); ok {
		h.actor.Data.OfflineEventData = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *OfflineEventHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyOfflineEvent(h.actor.ID()), h.actor.Data.OfflineEventData
}

// 玩家上线，执行存档的离线事件
func (h *OfflineEventHandler) ExecOfflineEvent() {
	h.Debugf("玩家上线，开始执行离线事件...")
	data := h.actor.GetOfflineEventData()
	now := time.Now().Unix()

	// 执行离线数据
	for key, evt := range data.EventList {
		// 事件过期删除
		if evt.ExpireTime > 0 && now >= evt.ExpireTime {
			delete(data.EventList, key)
			h.Infof("离线事件过期删除, data: %+v", evt)
			continue
		}

		var err error
		// 执行
		handler, ok := h.actor.MsgFunc[evt.Msg.MsgId]
		if !ok {
			h.Warnf("OfflineEventHandler OperateType 还未支持该类型操作, msg: %+v", evt.Msg.Str())
			continue
		}
		threading.RunSafe(func() {
			_, err, _ = handler(context.Background(), evt.Msg)
		})
		// 处理出错，跳过后续
		if err != nil {
			h.Errorf("离线事件处理失败, err:%v", err)
			continue
		}
		// 成功了，删除事件
		delete(data.EventList, key)
		h.Infof("离线事件处理成功, data: %+v", evt)
	}

	if err := h.SaveDB(); err != nil {
		h.Error(err)
	}
}

// 生成离线事件
func (h *OfflineEventHandler) SaveOfflineEvent(in *base.ProtoMsg, expire int64) (err error) {
	data := h.actor.GetOfflineEventData()
	e := &pb.OfflineEvent{
		Msg:        in,
		CreateTime: time.Now().Unix(),
		ExpireTime: expire,
	}
	data.EventList[e.CreateTime] = e

	err = h.SaveDB(true)
	if err != nil {
		h.Errorf("保存离线数据 got error:%v", err.Error())
	}
	return err
}

func (h *OfflineEventHandler) delItem(params map[int32]int32) error {
	return GetConsumeMgr(h.actor).ConsumeList(params, h.actor.comData, common.CR_offline_exec)
}

func (h *OfflineEventHandler) addItem(params map[int32]int32) error {
	_, err := GetDropMgr(h.actor).DropList2(params, true, nil, h.actor.comData, common.CR_offline_exec)
	return err
}

// 检查是否存在联盟
func (h *OfflineEventHandler) CheckInAllianceReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var err error
	var data *pb.PUserAllianceData
	// 是否在线
	if h.actor.IsMiniMode {
		// 从cache中捞数据
		data, err = h.actor.getAllianceDataByRoleId(in.RoleId)
		if err != nil {
			h.Errorf("加载数据失败 err: %v", err)
			return &pb.S2S_CheckInAllianceRes{Exist: true, Outdate: 0}, nil, 0
		}
	} else {
		// 从actor中捞数据
		data = h.actor.GetUserAllianceData()
	}

	return &pb.S2S_CheckInAllianceRes{Exist: data.AllianceId > 0, Outdate: data.ApplyOutdateTs}, nil, 0
}

// 加入联盟后续处理
func (h *OfflineEventHandler) JoinAllianceNtf(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var err error
	if h.actor.IsMiniMode {
		err = h.SaveOfflineEvent(in, 0)
	} else {
		req := &pb.S2S_JoinAllianceNtf{}
		if err = in.UnmarshalData(req); err != nil {
			return nil, err, int32(pb.ErrorCode_DeSerializeError)
		}
		err = h.actor.UserAllianceHandler.HandleJoinAlliance(req.Alliance)
	}
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}
	return &pb.S2S_JoinAllianceNtf{}, nil, 0
}

func (h *OfflineEventHandler) ExitAllianceNtf(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var err error
	h.Debugf("退出联盟通知调用了...")
	// 是否在线
	if h.actor.IsMiniMode {
		err = h.SaveOfflineEvent(in, 0)
	} else {
		err = h.actor.UserAllianceHandler.HandleExitAlliance(2)
	}
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}
	return &pb.S2S_ExitAllianceNtf{}, nil, 0
}

func (h *OfflineEventHandler) SrvDelFriendReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var err error

	// 是否在线
	if h.actor.IsMiniMode {
		err = h.SaveOfflineEvent(in, 0)
	} else {
		req := &pb.S2S_DelFriendReq{}
		if err = in.UnmarshalData(req); err != nil {
			return nil, err, int32(pb.ErrorCode_DeSerializeError)
		}
		err = h.actor.FriendHandler.HandleDelFriend(map[int32]int32{int32(req.RoleId): 0})
	}
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}
	return &pb.S2S_DelFriendRes{}, nil, 0
}

func (h *OfflineEventHandler) SrvAddFriendApplyReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var err error

	if h.actor.IsMiniMode {
		err = h.SaveOfflineEvent(in, 0)
	} else {
		req := &pb.S2S_AddFriendApplyReq{}
		if err = in.UnmarshalData(req); err != nil {
			return nil, err, int32(pb.ErrorCode_DeSerializeError)
		}
		err = h.actor.FriendHandler.HandleAddFriendApply(map[int32]int32{int32(req.RoleId): 0})
	}
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	return &pb.S2S_AddFriendApplyRes{}, nil, 0
}

func (h *OfflineEventHandler) SrvAgreeFriendApplyReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var err error
	if h.actor.IsMiniMode {
		err = h.SaveOfflineEvent(in, 0)
	} else {
		req := &pb.S2S_AgreeFriendApplyReq{}
		if err = in.UnmarshalData(req); err != nil {
			return nil, err, int32(pb.ErrorCode_DeSerializeError)
		}
		err = h.actor.FriendHandler.HandleAgreeFriendApply(map[int32]int32{int32(req.RoleId): 0})
	}
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	return &pb.S2S_AgreeFriendApplyRes{}, nil, 0
}

func (h *OfflineEventHandler) SendFriendPointReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var err error

	if h.actor.IsMiniMode {
		err = h.SaveOfflineEvent(in, 0)
	} else {
		req := &pb.S2S_SendFriendPointReq{}
		if err = in.UnmarshalData(req); err != nil {
			return nil, err, int32(pb.ErrorCode_DeSerializeError)
		}
		err = h.actor.FriendHandler.HandleSendFriendPoint(map[int32]int32{int32(req.RoleId): 0})
	}
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	return &pb.S2S_SendFriendPointRes{}, nil, 0
}

func (h *OfflineEventHandler) PushMessageToUserReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	// roleId := in.RoleId
	req := &pb.S2S_PushMessageToUserReq{}
	if err := in.UnmarshalData(req); err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}
	res := &pb.S2S_PushMessageToUserRes{}
	// 是否在线
	if h.actor.IsMiniMode {
		// err = h.SaveOfflineEvent(pb.OfflineOperateType_DelFriend, 0, in.UserId, map[int32]int32{int32(req.RoleId): 0})
		// 不在线不做处理，
		return res, nil, int32(pb.ErrorCode_Success)
	}
	if err, code := h.actor.UserChatHandler.PushMessageToUserReq(req, req.GetChannelId()); err != nil {
		return nil, err, code
	}
	return res, nil, int32(pb.ErrorCode_Success)
}
