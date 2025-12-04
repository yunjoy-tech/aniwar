package useractor

import (
	"context"
	"fmt"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/actorserver/useractor/event"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datalog/taptap"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/threading"
	"google.golang.org/protobuf/proto"
	"time"
)

// CampLightingComposeTreeRewardReq 光合树奖励领取
func (h *CampHandler) CampLightingComposeTreeRewardReq(_ context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	if err, errCode := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_1002); err != nil {
		return nil, err, int32(errCode)
	}

	var req cmd.C2LS_PlayerCampLightingComposeTreeRewardReq
	if err := in.UnmarshalData(&req); err != nil {
		return nil, err, int32(cmd.ErrorCode_SerializeError)
	}

	commonParams := NewOutputParams(true, req.BuildingId, 0, in.MsgId)
	if err, errCode := h.commonCheck(commonParams); err != nil {
		return nil, err, errCode
	}

	formulaCfg := h.getFormulaByCfg(commonParams.buildLevelConfig, false)
	if len(formulaCfg) == 0 {
		return nil, fmt.Errorf("config not found"), int32(cmd.ErrorCode_NotFoundConfig)
	}

	//step:1 更具itemId 获取配置
	product, totalTimeCost, code := h.GetItemCfgProduct(req.GetItemId(), formulaCfg)
	if code != int32(cmd.ErrorCode_Success) {
		return nil, fmt.Errorf("CampLightingComposeTreeRewardReq get product cfg err"), code
	}

	// step : 2 更具tiemId 获取结束时间
	endTimeStamp := h.GetProductEndTime(req.GetItemId(), commonParams)

	// step:3 更具ItemId 计算产物数量
	rewards, code := h.GetProduct(product, totalTimeCost, endTimeStamp)
	if code != int32(cmd.ErrorCode_Success) {
		return nil, fmt.Errorf("CampLightingComposeTreeRewardReq get product cfg err"), code
	}
	if len(rewards) == 0 {
		return nil, fmt.Errorf("CampLightingComposeTreeRewardReq no reward"), int32(cmd.ErrorCode_CampNoRewardCanGet)
	}

	// step:4 根据产物设计结束时间
	if err := h.SetProductEndTime(req.GetItemId(), totalTimeCost, commonParams); err != nil {
		return nil, fmt.Errorf("CampLightingComposeTreeRewardReq Set Product EndTime err"), int32(cmd.ErrorCode_InternalError)
	}

	// 发放奖励
	var dropChange *cmd.DropChange
	var err error

	dropChange, err = GetDropMgr(h.actor).DropList2(rewards, true, nil, h.actor.comData, common.CR_Camp_Building_Get)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	// 发布收取光合树事件
	errx := h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_TREE_COLLECT, []int32{TASK_TYPE_301}, nil))
	if errx != nil {
		h.Error(errx)
	}
	// 发布事件
	if errx := h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_BUILDING_REWARD, []int32{TASK_TYPE_510}, map[string]interface{}{
		"buildId": commonParams.building.ItemId,
		"reward":  rewards,
	})); errx != nil {
		h.Error(errx)
	}

	// 光合树收获 埋点
	build := excel.GetBuildMainMgr().GetById(commonParams.building.ItemId)
	threading.RunSafe(func() {
		e := &taptap.CamptreeReward{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			Id:                build.Id,                                           //建筑唯一id
			BuildingId:        req.BuildingId,                                     //建筑id
			Reward:            taptap.ConvertMap2Str(rewards),                     //领取的奖励
			Lv:                commonParams.camp.LightingComposeTree.Level,        //光合树当前等级
			BeforeEndTs:       endTimeStamp,                                       //领取前的结束时间戳
			AfterEndTs:        time.Now().Unix() + totalTimeCost,                  //领取后的结束时间戳
			UseTime:           totalTimeCost - (endTimeStamp - time.Now().Unix()), //奖励的折算时间
		}
		taptap.WriteDataLog(taptap.LogType_CamptreeReward, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	h.actor.comData.GetCampData().Camp = append(h.actor.comData.GetCampData().Camp, &cmd.PPlayerCamp{LightingComposeTree: commonParams.camp.LightingComposeTree})
	res := &cmd.LS2C_PlayerCampLightingComposeTreeRewardRes{
		DropChange: dropChange,
		CommonData: h.actor.comData.FixDownComData(),
	}
	return res, nil, 0
}

//////////////////////////////////////////////////////////////////////////// 内部调用

// GetProduct 获取产物
func (h *CampHandler) GetProduct(cfgProduct *excel.ItemReward, totalTimeCost, endTimeStamp int64) (map[int32]int32, int32) {
	var rewards = make(map[int32]int32)
	now := time.Now().Unix()
	tm := totalTimeCost - (endTimeStamp - now) // 用了多少时间

	if now >= endTimeStamp { // 可以全部领取，获取到的奖励就是
		rewards[cfgProduct.ItemId] += cfgProduct.Num
	} else {
		if cfgProduct.Num <= 0 {
			return nil, int32(cmd.ErrorCode_ConfigError)
		}
		n := float32(tm) / float32(totalTimeCost) * float32(cfgProduct.Num)
		h.Infof("光和树领取奖励计算diffTime[%d],totalTimeCost[%d],cfgProduct.Num[%d],result:[%f]", float32(tm), float32(totalTimeCost), float32(cfgProduct.Num), float32(tm)/float32(totalTimeCost))
		if int32(n) > 0 {
			rewards[cfgProduct.ItemId] = int32(n)
		}
	}
	return rewards, int32(cmd.EnterLevelType_EnterLevelType_None)
}

// GetItemCfgProduct 获取光和树产物
func (h *CampHandler) GetItemCfgProduct(itemId int32, formulaCfg map[int32]*excel.ItemSynthesisCfg) (*excel.ItemReward, int64, int32) {
	var totalTimeCost int64
	var tempProduct *excel.ItemReward

	for _, v := range formulaCfg {
		if len(v.ItemProduct) == 0 {
			return nil, 0, int32(cmd.ErrorCode_NotFoundConfig)
		}
		for _, p := range v.ItemProduct {
			if p.ItemId == itemId {
				tempProduct = p
			}
		}
		totalTimeCost = int64(v.TimeCost)
	}
	return tempProduct, totalTimeCost, int32(cmd.ErrorCode_Success)
}

// GetProductEndTime 获取产物的结束时间
func (h *CampHandler) GetProductEndTime(itemId int32, commonParams *OutputParams) int64 {
	endTimeStamp := commonParams.camp.LightingComposeTree.EndTimestampList
	for _, v := range endTimeStamp {
		if v.ItemId == itemId {
			return v.EndTimestamp
		}
	}
	return 0
}

// SetProductEndTime 设置产物的结束时间
func (h *CampHandler) SetProductEndTime(itemId int32, totalTimeCost int64, commonParams *OutputParams) error {
	endTimeStamp := commonParams.camp.LightingComposeTree.EndTimestampList
	productEndTime := make([]*cmd.ComposeTreeProductEndTime, 0)
	for _, v := range endTimeStamp {
		if v.ItemId == itemId {
			v.EndTimestamp = time.Now().Unix() + totalTimeCost
		}
		productEndTime = append(productEndTime, v)
	}
	commonParams.camp.LightingComposeTree.EndTimestampList = productEndTime
	//更新数据
	if err := h.SaveDB(); err != nil {
		return err
	}
	return nil
}

// SetLightingComposeTreeTs 设置光合树回满值时间戳
func (h *CampHandler) SetLightingComposeTreeTs(itemId int32, tm int64) error {
	camp := h.getCurCamp()
	if camp == nil || camp.LightingComposeTree == nil {
		return fmt.Errorf("lightingComposeTree not exist, CampHandler.SetLightingComposeTreeTimestamp")
	}
	for _, v := range camp.LightingComposeTree.EndTimestampList {
		if v.ItemId == itemId {
			v.EndTimestamp = tm
		}
	}

	h.SaveDB(true)
	return nil
}
