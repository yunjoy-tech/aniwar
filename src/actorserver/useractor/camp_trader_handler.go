package useractor

import (
	"context"
	"fmt"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datalog/taptap"
	myUtils "gitlab.musadisca-games.com/wangxw/aniwar/src/common/utils"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/threading"
	_ "google.golang.org/genproto/googleapis/cloud/dataproc/v1"
	"google.golang.org/protobuf/proto"
)

/////////////////////////////////////////////////////////101 商人相关

// PlayerCampTraderExchangeReq 兑换商人奖励
func (h *CampHandler) PlayerCampTraderExchangeReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_1002)
	if err != nil {
		return nil, err, int32(code)
	}
	var req cmd.C2LS_PlayerCampTraderExchangeReq
	if err := in.UnmarshalData(&req); err != nil {
		return nil, err, int32(cmd.ErrorCode_SerializeError)
	}

	commonParams := NewOutputParams(true, req.BuildingId, 0, in.MsgId)
	if err, errCode := h.commonCheck(commonParams); err != nil {
		return nil, err, errCode
	}

	trader := commonParams.camp.Trader
	// 是否存在
	var item *cmd.PPlayerCampTraderList
	for _, v := range trader.Items {
		if v.Id == req.Id {
			item = v
		}
	}
	if item == nil {
		return nil, fmt.Errorf("param error"), int32(cmd.ErrorCode_ParamError)
	}
	// 是否已经兑换了
	if item.Status != TRADER_STATUS_1 {
		return nil, fmt.Errorf("trader had exchange"), int32(cmd.ErrorCode_CampTraderHadExchange)
	}

	// 消耗check
	costs := myUtils.ConvertItem(item.Costs)
	if !GetConsumeMgr(h.actor).CheckMapEnough(costs) {
		return nil, fmt.Errorf("item not enough"), int32(cmd.ErrorCode_NotEnoughItem)
	}
	if err := GetConsumeMgr(h.actor).ConsumeList(costs, h.actor.comData, common.CR_Camp_Trader_Exchange); err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}
	item.Status = TRADER_STATUS_2

	// save
	if err := h.SaveDB(); err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}
	// 发奖励
	if _, err := GetDropMgr(h.actor).DropList2(map[int32]int32{item.Reward.Key: item.Reward.Value}, true, nil, h.actor.comData, common.CR_Camp_Trader_Exchange); err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	// 商人兑换奖励 埋点
	build := excel.GetBuildMainMgr().GetById(commonParams.building.ItemId)
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.CampTraderExchange{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_CampTraderExchange, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		Id:             build.Id,                              //建筑唯一id
	//		BuildingId:     req.BuildingId,                        //建筑id
	//		Lv:             commonParams.building.BuildingLevel,   //建筑等级
	//		TraderId:       req.Id,                                //兑换清单id
	//		Category:       item.Category,                         //清单分类
	//		Quality:        item.Quality,                          //清单品质
	//		Costs:          lilith.ConvertMap2Str(costs),          //消耗
	//		Reward:         lilith.ConvertStruct2Str(item.Reward), //奖励
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.CampTraderExchange{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			Id:                build.Id,                              //建筑唯一id
			BuildingId:        req.BuildingId,                        //建筑id
			Lv:                commonParams.building.BuildingLevel,   //建筑等级
			TraderId:          req.Id,                                //兑换清单id
			Category:          item.Category,                         //清单分类
			Quality:           item.Quality,                          //清单品质
			Costs:             taptap.ConvertMap2Str(costs),          //消耗
			Reward:            taptap.ConvertStruct2Str(item.Reward), //奖励
		}
		taptap.WriteDataLog(taptap.LogType_CampTraderExchange, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	h.actor.comData.GetCampData().Camp = append(h.actor.comData.GetCampData().Camp, &cmd.PPlayerCamp{Trader: commonParams.camp.Trader})
	return &cmd.LS2C_PlayerCampTraderExchangeRes{CommonData: h.actor.comData.FixDownComData()}, nil, 0
}

// PlayerCampTraderRefreshReq 刷新商人清单
func (h *CampHandler) PlayerCampTraderRefreshReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_1002)
	if err != nil {
		return nil, err, int32(code)
	}
	var req cmd.C2LS_PlayerCampTraderRefreshReq
	if err := in.UnmarshalData(&req); err != nil {
		return nil, err, int32(cmd.ErrorCode_SerializeError)
	}

	commonParams := NewOutputParams(true, req.BuildingId, 0, in.MsgId)
	if err, errCode := h.commonCheck(commonParams); err != nil {
		return nil, err, errCode
	}

	beforeList := commonParams.camp.Trader.Items
	commonParams.camp.Trader.Items = h.TryRefreshTraderList(commonParams.camp.Trader.Level, commonParams.camp.Trader.Items)

	// save
	if err := h.SaveDB(); err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}
	h.actor.comData.GetCampData().Camp = append(h.actor.comData.GetCampData().Camp, &cmd.PPlayerCamp{Trader: commonParams.camp.Trader})

	// 商人清单刷新 埋点
	traderBeforeTemp := make([]*taptap.PPlayerCampTraderListTemp, 0, len(commonParams.camp.Trader.Items)) //刷新前的清单列表
	for _, value := range beforeList {
		traderBeforeTemp = append(traderBeforeTemp, taptap.NewPPlayerCampTraderListTemp(value.Id, value.Category, value.Status, value.Quality, taptap.ConvertListStruct2Str(value.Costs), taptap.ConvertStruct2Str(value.Reward)))
	}

	traderAfterTemp := make([]*taptap.PPlayerCampTraderListTemp, 0, len(commonParams.camp.Trader.Items)) //刷新后的清单列表
	for _, value := range commonParams.camp.Trader.Items {
		traderAfterTemp = append(traderAfterTemp, taptap.NewPPlayerCampTraderListTemp(value.Id, value.Category, value.Status, value.Quality, taptap.ConvertListStruct2Str(value.Costs), taptap.ConvertStruct2Str(value.Reward)))
	}

	build := excel.GetBuildMainMgr().GetById(commonParams.building.ItemId)
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.CampTraderRefresh{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_CampTraderRefresh, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		Id:             build.Id,                                       //建筑唯一id
	//		BuildingId:     req.BuildingId,                                 //建筑id
	//		Lv:             commonParams.building.BuildingLevel,            //建筑等级
	//		BeforeList:     lilith.ConvertListStruct2Str(traderBeforeTemp), //刷新前的清单列表
	//		AfterList:      lilith.ConvertListStruct2Str(traderAfterTemp),  //刷新后的清单列表
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.CampTraderRefresh{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			Id:                build.Id,                                       //建筑唯一id
			BuildingId:        req.BuildingId,                                 //建筑id
			Lv:                commonParams.building.BuildingLevel,            //建筑等级
			BeforeList:        taptap.ConvertListStruct2Str(traderBeforeTemp), //刷新前的清单列表
			AfterList:         taptap.ConvertListStruct2Str(traderAfterTemp),  //刷新后的清单列表
		}
		taptap.WriteDataLog(taptap.LogType_CampTraderRefresh, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	return &cmd.LS2C_PlayerCampTraderRefreshRes{CommonData: h.actor.comData.FixDownComData()}, nil, 0
}
