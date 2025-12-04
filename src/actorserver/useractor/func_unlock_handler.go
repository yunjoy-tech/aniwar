package useractor

import (
	"fmt"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"
	"google.golang.org/protobuf/proto"
)

type FuncUnlockHandler struct {
	*UABaseHandler
}

const (
	FUNC_ID_1001      = 1001 // 探险（主线）
	FUNC_ID_1002      = 1002 // 营地
	FUNC_ID_CARD_POOL = 1003 // 抽卡
	FUNC_ID_CARD      = 1004 // 养成
	FUNC_ID_1005      = 1005 // fixme 体力本
	FUNC_ID_1006      = 1006 // 日常本
	FUNC_ID_DUTY      = 1007 // 值日生
	FUNC_ID_TRIAL     = 1008 // 试炼塔
	FUNC_ID_Friend    = 1009 // 好友
	FUNC_ID_BLOCKWAY  = 1010 // 公路事件
	FUNC_ID_ALLIANCE  = 1011 // 联盟
	FUNC_ID_CALL      = 1012 // 通讯器
	FUNC_ID_GUIDETASK = 1013 // 引导任务
)
const (
	UNLOCK_TYPE_QUEST = 1 // 剧情任务
)

func NewFuncUnlockHandler(actor *UserActor) *FuncUnlockHandler {
	h := &FuncUnlockHandler{UABaseHandler: NewUABaseHandler(actor, "FuncUnlockHandler")}
	h.ChildHandler = h
	return h
}

func (h *FuncUnlockHandler) Init() error {
	return nil
}

func (h *FuncUnlockHandler) EnterGame() error {
	return nil
}

func (h *FuncUnlockHandler) DailyRefresh() error {
	return nil
}

func (h *FuncUnlockHandler) SetDBData(dbData proto.Message) error {
	return nil
}

func (h *FuncUnlockHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoNil, "", nil
}

// CheckFuncUnlock
//
//	@Description: 检查指定功能是否解锁
//	@receiver h
//	@param funcId 模块功能id
//	@return error 错误
//	@return cmd.ErrorCode 错误码
func (h *FuncUnlockHandler) CheckFuncUnlock(funcId int32) (error, cmd.ErrorCode) {
	// 动态配置是否关闭
	if _, ok := h.actor.Srv.CloseFuncMap.Load(funcId); ok {
		return fmt.Errorf("func had close %d", funcId), cmd.ErrorCode_DeprecatedMsgError
	}

	return h.CheckFuncUnlockBase(funcId)
}

func (h *FuncUnlockHandler) CheckFuncUnlockBase(funcId int32) (error, cmd.ErrorCode) {
	// 未配置解锁要求
	cfg := excel.GetSystemUnlockMgr().GetById(funcId)
	if cfg == nil {
		return nil, cmd.ErrorCode_Success
	}
	// 玩家等级判定
	if h.actor.LoginHandler.getRoleLevel() < uint32(cfg.Unlocklevel) {
		return fmt.Errorf("role level not enough"), cmd.ErrorCode_FuncUnlockError
	}
	// 其他条件判定
	switch cfg.UnlockType {
	case UNLOCK_TYPE_QUEST:
		if !h.actor.QuestHandler.checkQuestFinish(cfg.UnlockParam) {
			return fmt.Errorf("func unlock condition not match %d", UNLOCK_TYPE_QUEST), cmd.ErrorCode_FuncUnlockError
		}
	}
	return nil, cmd.ErrorCode_Success
}

// 检查给定的一批模块id，判定是否有至少一个解锁，如果有则返回true，否则返回false
func (h *FuncUnlockHandler) CheckFuncsUnlock(funcIds []int32) bool {
	for _, id := range funcIds {
		if err, _ := h.CheckFuncUnlock(id); err == nil {
			return true
		}
	}
	return false
}
