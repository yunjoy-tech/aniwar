package useractor

import (
	"fmt"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/db"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"
	"google.golang.org/protobuf/proto"
	"time"
)

// 付费功能使用次数上限判定
type UseLimitHandler struct {
	*UABaseHandler
}

func NewUseLimitHandler(actor *UserActor) *UseLimitHandler {
	h := &UseLimitHandler{UABaseHandler: NewUABaseHandler(actor, "UseLimitHandler")}
	h.ChildHandler = h
	return h
}

func (h *UseLimitHandler) Init() error {
	// 初始化
	h.actor.Data.UseLimit = &cmd.PUseLimitInfo{
		Createtime: time.Now().Unix(),
		Items:      make(map[int32]*cmd.KeyValueItem),
	}

	// 保存
	if err := h.SaveDB(); err != nil {
		return err
	}

	h.Debug("init use limit data success.")
	return nil
}

func (h *UseLimitHandler) EnterGame() error {
	return nil
}

func (h *UseLimitHandler) DailyRefresh() error {
	return h.tryClearUseCount()
}

func (h *UseLimitHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*cmd.PUseLimitInfo); ok {
		h.actor.Data.UseLimit = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *UseLimitHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyUseLimit(h.actor.ID()), h.actor.Data.UseLimit
}

func (h *UseLimitHandler) buildUseLimitData() *cmd.PClientUseLimitInfo {
	data := h.actor.GetUseLimitData()
	limits := make([]*cmd.KeyValueItem, 0)
	for _, item := range data.Items {
		limits = append(limits, item)
	}
	return &cmd.PClientUseLimitInfo{
		Limits:    limits,
		RefreshTs: common.GetNextDailyRefreshTime(),
	}
}

// CheckUseEnough
//
//	@Description: 检查指定模块的使用次数是否超过上限
//	@receiver h
//	@param funcId 模块id
//	@param useCount 需要使用的次数
//	@return bool 足够返回true，否则返回false
func (h *UseLimitHandler) CheckUseEnough(funcId, useCount int32) bool {
	var (
		cur, max int32
	)

	data := h.actor.GetUseLimitData()
	item := data.Items[funcId]
	if item != nil {
		cur = item.Value
	}
	// 查询配置的最大值
	switch funcId {
	case int32(cmd.RedPointModuleType_Card_Pool_Module):
		max = excel.GetConfigMgr().GetCfg().GACHA_LIMIT_DAILY
	case int32(cmd.RedPointModuleType_Camp_Module):
		max = excel.GetConfigMgr().GetCfg().GACHA_LIMIT_DAILY_CAMP
	default:
		h.Warnf("unrealized func type %d", funcId)
	}

	return cur+useCount <= max
}

// AddUseCount
//
//	@Description: 记录使用的次数
//	@receiver h
//	@param funcId 模块id
//	@param addCount 增加的次数
//	@return error
func (h *UseLimitHandler) AddUseCount(funcId, addCount int32) error {
	data := h.actor.GetUseLimitData()
	item, ok := data.Items[funcId]
	if !ok {
		item = &cmd.KeyValueItem{
			Key:   funcId,
			Value: addCount,
		}
		data.Items[funcId] = item
	} else {
		item.Value += addCount
	}
	h.actor.comData.GetUseLimitData().Limits = append(h.actor.comData.GetUseLimitData().Limits, item)
	h.Debugf("AddUseCount funcId: %d, addCount: %d", funcId, addCount)
	return h.SaveDB()
}

// 尝试清除使用次数记录数据
func (h *UseLimitHandler) tryClearUseCount() error {
	data := h.actor.GetUseLimitData()
	for _, item := range data.Items {
		item.Value = 0
	}
	h.Infof("tryClearUseCount success.")
	return h.SaveDB()
}
