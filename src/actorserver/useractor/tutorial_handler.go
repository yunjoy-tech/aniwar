package useractor

import (
	"context"
	"fmt"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datalog/taptap"
	"gitlab.musadisca-games.com/wangxw/musae/framework/threading"
	"time"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/db"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"
	"google.golang.org/protobuf/proto"
)

const (
	MASTER_ID_1 = 1 // 账号数据创建
	MASTER_ID_2 = 2 // 起名
	MASTER_ID_3 = 3 // 起名完成
	MASTER_ID_4 = 4 // 第一段动画完成
	MASTER_ID_5 = 5 // 第一段剧情完成
	MASTER_ID_6 = 6 // 第0章起始
	MASTER_ID_7 = 7 // 第0章结束
)

var master_id_map = map[int32]int32{
	MASTER_ID_1: 0,
	MASTER_ID_2: 0,
	MASTER_ID_3: 0,
	MASTER_ID_4: 0,
	MASTER_ID_5: 0,
	MASTER_ID_6: 0,
	MASTER_ID_7: 0,
}

type TutorialHandler struct {
	*UABaseHandler
}

func (h *TutorialHandler) Init() error {
	// 初始化
	h.actor.Data.Tutorial = &cmd.PPlayerBeginnerTutorialBlob{
		Createtime:             time.Now().Unix(),
		FinishMasterTutorial:   make([]*cmd.PPlayerDBBeginnerTutorialBlob, 0),
		FinishFunctionTutorial: make([]*cmd.PPlayerDBBeginnerTutorialBlob, 0),
	}

	// 保存
	if err := h.SaveDB(true); err != nil {
		return err
	}

	logger.Debug("init tutorial data success. player: %s", h.actor.ID())
	return nil
}

func (h *TutorialHandler) EnterGame() error {
	return nil
}

func (h *TutorialHandler) DailyRefresh() error {
	return nil
}

func (h *TutorialHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*cmd.PPlayerBeginnerTutorialBlob); ok {
		h.actor.Data.Tutorial = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *TutorialHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyUserTutorial(h.actor.ID()), h.actor.Data.Tutorial
}

func NewTutorialHandler(actor *UserActor) *TutorialHandler {
	h := &TutorialHandler{UABaseHandler: NewUABaseHandler(actor, "TutorialHandler")}
	h.ChildHandler = h

	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_PlayerBeginnerTutorialAddRecordReq), h.PlayerBeginnerTutorialAddRecordReq) // 新手引导

	return h
}

// 新手引导记录
func (h *TutorialHandler) PlayerBeginnerTutorialAddRecordReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {

	var req cmd.C2LS_PlayerBeginnerTutorialAddRecordReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	// 是否合法
	var cfg *excel.TutorialCfg
	excel.GetTutorialMgr().Foreach(func(t *excel.TutorialCfg) bool {
		if t.GroupId == int32(req.TutorialId) {
			cfg = t
			return false
		}
		return true
	}, false)
	if _, ok := master_id_map[int32(req.TutorialId)]; !ok && cfg == nil {
		return nil, fmt.Errorf("invalid tutorial id %d", req.TutorialId), int32(cmd.ErrorCode_InvalidParam)
	}

	err, errcode := h.handleTutorialRecord(req.GetOperationType(), req.GetTutorialId())
	if errcode != cmd.ErrorCode_Success {
		return nil, err, int32(errcode)
	}

	// 埋点
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.TutorialAddRecord{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_TutorialDddRecord, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		TutorialType:   req.OperationType, // 引导点类型，1=主引导，2=功能引导
	//		TutorialId:     req.TutorialId,    // 引导点id
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.TutorialAddRecord{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			TutorialType:      req.OperationType, // 引导点类型，1=主引导，2=功能引导
			TutorialId:        req.TutorialId,    // 引导点id
		}
		taptap.WriteDataLog(taptap.LogType_TutorialDddRecord, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	// 消息返回
	rsp := &cmd.LS2C_PlayerBeginnerTutorialAddRecordRes{
		OperationType: req.OperationType,
		TutorialId:    req.TutorialId,
	}

	return rsp, nil, 0
}

func (h *TutorialHandler) buildPlayerBeginnerTutorial() *cmd.PPlayerBeginnerTutorial {
	master := uint32(0)
	ids := make([]uint32, 0)
	// 主引导
	for _, v := range h.actor.GetTutorialData().FinishMasterTutorial {
		if v.TutorialId > master {
			master = v.TutorialId
		}
	}

	// 功能引导
	for _, v := range h.actor.GetTutorialData().FinishFunctionTutorial {
		ids = append(ids, v.TutorialId)
	}

	return &cmd.PPlayerBeginnerTutorial{
		MasterTutorialId: master,
		TutorialId:       ids,
	}
}

func (h *TutorialHandler) handleTutorialRecord(typ, id uint32) (error, cmd.ErrorCode) {
	newTutorial := &cmd.PPlayerDBBeginnerTutorialBlob{
		TutorialId:      id,
		FinishTimestamp: time.Now().Unix(),
	}

	// 类型判断
	if typ == uint32(cmd.PlayerBeginnerTutorialType_PlayerBeginnerTutorialType_Master) {
		if h.existTutorialId(h.actor.GetTutorialData().FinishMasterTutorial, id) {
			return nil, cmd.ErrorCode_Success
		}
		h.actor.GetTutorialData().FinishMasterTutorial = append(h.actor.GetTutorialData().FinishMasterTutorial, newTutorial)
	} else if typ == uint32(cmd.PlayerBeginnerTutorialType_PlayerBeginnerTutorialType_Function) {
		if h.existTutorialId(h.actor.GetTutorialData().FinishFunctionTutorial, id) {
			return nil, cmd.ErrorCode_Success
		}
		h.actor.GetTutorialData().FinishFunctionTutorial = append(h.actor.GetTutorialData().FinishFunctionTutorial, newTutorial)
	} else {
		return fmt.Errorf("unrealized type error"), cmd.ErrorCode_UnrealizedTypeError
	}

	// 保存数据
	if err := h.SaveDB(true); err != nil {
		return fmt.Errorf("save db error"), cmd.ErrorCode_SaveDBError
	}
	return nil, cmd.ErrorCode_Success
}

func (h *TutorialHandler) existTutorialId(arr []*cmd.PPlayerDBBeginnerTutorialBlob, target uint32) bool {
	for _, blob := range arr {
		if blob.TutorialId == target {
			h.Debugf("existTutorialId：%+v %d", arr, target)
			return true
		}
	}
	return false
}
