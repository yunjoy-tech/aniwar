package useractor

import (
	"context"
	"fmt"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datalog/taptap"
	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"
	"strconv"
	"time"

	"gitlab.musadisca-games.com/wangxw/musae/framework/threading"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/actorserver/useractor/event"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/clidto"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/db"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/utils"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"
	"google.golang.org/protobuf/proto"
)

type PoolHandler struct {
	*UABaseHandler
}

type OpenPool struct {
	PoolType int32
	Start    int64
	End      int64
}

func NewPoolHandler(actor *UserActor) *PoolHandler {
	h := &PoolHandler{UABaseHandler: NewUABaseHandler(actor, "PoolHandler")}
	h.ChildHandler = h

	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_GetCardPoolInfoReq), h.GetCardPoolInfoReq)
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_CardPoolExtractReq), h.CardPoolExtractReq)
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_NewbiePoolExtractReq), h.NewbiePoolExtractReq)
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_SelectNewbieResultReq), h.SelectNewbieResultReq)
	return h
}

// Init 初始化模块数据
func (h *PoolHandler) Init() error {
	// 初始化
	h.actor.Data.Pools = &cmd.PServerCardPoolInfos{
		Createtime: time.Now().Unix(),
		TypeInfos:  make(map[int32]*cmd.PServerCardPoolType),
		Newbie:     &cmd.PNewbiePoolInfo{Results: make([]*cmd.PNewbiePoolLog, 0)},
	}

	// 保存
	if err := h.SaveDB(); err != nil {
		return err
	}

	h.Debug("init card pool data success. player: %s", h.actor.ID())
	return nil
}

func (h *PoolHandler) EnterGame() error {
	return nil
}

func (h *PoolHandler) DailyRefresh() error {
	return nil
}

func (h *PoolHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*cmd.PServerCardPoolInfos); ok {
		h.actor.Data.Pools = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *PoolHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyUserCardPool(h.actor.ID()), h.actor.Data.Pools
}

// 通用解锁判定
func (h *PoolHandler) commonCheck() (error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_CARD_POOL)
	if err != nil {
		return err, int32(code)
	}

	return nil, int32(cmd.ErrorCode_Success)
}

func (h *PoolHandler) GetCardPoolInfoReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req cmd.C2LS_GetCardPoolInfoReq
	if err := in.UnmarshalData(&req); err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	err, code := h.commonCheck()
	if err != nil {
		return nil, err, code
	}

	data := h.actor.GetPoolsData()
	infos := make([]*cmd.PClientCardPoolInfo, 0)
	// 获取所有开启中的卡池
	openPoolCfg := h.getOpenPoolCfg()
	if req.PoolId > 0 {
		// 获取指定的卡池信息
		needPool := openPoolCfg[req.PoolId]
		if needPool != nil {
			if _, ok := data.TypeInfos[needPool.PoolType]; !ok {
				data.TypeInfos[needPool.PoolType] = createCardTypeInfo()
			}

			typeInfo := data.TypeInfos[needPool.PoolType]
			infos = append(infos, &cmd.PClientCardPoolInfo{
				PoolId: req.PoolId,
				Start:  needPool.Start,
				End:    needPool.End,
				Num:    typeInfo.UpCount,
			})
		}
	} else {
		for id, v := range openPoolCfg {
			if _, ok := data.TypeInfos[v.PoolType]; !ok {
				data.TypeInfos[v.PoolType] = createCardTypeInfo()
			}

			typeInfo := data.TypeInfos[v.PoolType]
			infos = append(infos, &cmd.PClientCardPoolInfo{
				PoolId: id,
				Start:  v.Start,
				End:    v.End,
				Num:    typeInfo.UpCount,
			})
		}
	}

	res := &cmd.LS2C_GetCardPoolInfoRes{Infos: infos}
	if data.Newbie.Select == 0 {
		res.Newbie = &cmd.PNewbiePoolInfo{Results: data.Newbie.Results}
	}
	return res, nil, 0
}

func (h *PoolHandler) SelectNewbieResultReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req cmd.C2LS_SelectNewbieResultReq
	if err := in.UnmarshalData(&req); err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}
	err, code := h.commonCheck()
	if err != nil {
		return nil, err, code
	}

	data := h.actor.GetPoolsData()
	// 是否选择
	if data.Newbie.Select > 0 {
		return nil, fmt.Errorf("had select newbie reward %d", data.Newbie.Select), int32(cmd.ErrorCode_IllegalOperationError)
	}
	// 非法选项
	if len(data.Newbie.Results) < int(req.Index) {
		return nil, fmt.Errorf("param error"), int32(cmd.ErrorCode_ParamError)
	}

	// 记录数据
	data.Newbie.Select = req.Index
	if err = h.SaveDB(); err != nil {
		return nil, err, int32(cmd.ErrorCode_SaveDBError)
	}

	// 发奖励
	logs := data.Newbie.Results[req.Index-1]
	reward := make(map[int32]int32)
	for _, id := range logs.CardIds {
		reward[id] += 1
	}
	_, err = GetDropMgr(h.actor).DropList2(reward, true, nil, h.actor.comData, common.CR_Card_Pool)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	cardNum := make([]*cmd.KeyValueItem, 0)
	tempCardNum := make(map[int32]int32)
	for _, itemId := range logs.CardIds {
		temp := excel.GetItemMgr().GetById(itemId)
		if temp == nil {
			continue
		}
		// 卡牌获得次数
		if _, ok := tempCardNum[itemId]; !ok {
			tempCardNum[itemId] = 0
			card, err := h.actor.CardHandler.GetCard(uint32(temp.SystemId))
			if err != nil {
				cardNum = append(cardNum, &cmd.KeyValueItem{Key: itemId, Value: 0})
			} else {
				cardNum = append(cardNum, &cmd.KeyValueItem{Key: itemId, Value: int32(card.AddNum)})
			}
		}
	}

	res := &cmd.LS2C_SelectNewbieResultRes{
		CommonData: h.actor.comData.FixDownComData(),
		AddNum:     cardNum,
		ItemIds:    logs.CardIds,
	}

	// 稀有卡牌广播
	h.BroadcastMessage(in.RoleId, logs.CardIds)
	return res, nil, 0
}

func (h *PoolHandler) NewbiePoolExtractReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req cmd.C2LS_NewbiePoolExtractReq
	if err := in.UnmarshalData(&req); err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}
	err, code := h.commonCheck()
	if err != nil {
		return nil, err, code
	}

	// 参数check
	poolCfg := excel.GetPoolMgr().GetById(req.PoolId)
	if poolCfg == nil {
		return nil, fmt.Errorf("poolCfg not found %d", req.PoolId), int32(cmd.ErrorCode_NotFoundConfig)
	}
	if poolCfg.GetPoolType() != common.CARD_POOL_NEWBIE {
		return nil, fmt.Errorf("pool type not match"), int32(cmd.ErrorCode_ParamError)
	}
	// 抽满了10次
	data := h.actor.GetPoolsData()
	if len(data.Newbie.Results) >= int(excel.GetConfigMgr().GetCfg().GACHA_NEWPLAYER_TIMES) {
		return nil, fmt.Errorf("extract num limit"), int32(cmd.ErrorCode_IllegalOperationError)
	}

	if !h.isValidPoolId(req.PoolId) {
		return nil, fmt.Errorf("invalid pool id:%d", req.PoolId), int32(cmd.ErrorCode_InvalidParam)
	}

	// 抽卡次数
	total := 10
	costItem := map[int32]int32{poolCfg.Gacha10Cost.Key: poolCfg.Gacha10Cost.Val}

	// 消耗检查
	if !GetConsumeMgr(h.actor).CheckMapEnough(costItem) {
		return nil, fmt.Errorf("item not enough: %+v", costItem), int32(cmd.ErrorCode_NotEnoughItem)
	}

	err = GetConsumeMgr(h.actor).ConsumeList(costItem, h.actor.comData, common.CR_Card_Pool)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	// 去抽卡
	result, _, _, _, err := h.handleNewbiePoolExtract(req.PoolId, total, h.actor.comData)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}
	h.Debugf("抽卡结果: player %s result %+v", h.actor.ID(), result)

	// 埋点log
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.CardExtract{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_CardExtract, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		PoolId:         req.PoolId,
	//		ExtractType:    common.CARD_POOL_TYPE_TEN,
	//		Cost:           lilith.ConvertMap2Str(costItem),
	//		Cards:          lilith.ConvertList2Str(result),
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.CardExtract{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			PoolId:            req.PoolId,
			ExtractType:       common.CARD_POOL_TYPE_TEN,
			Cost:              taptap.ConvertMap2Str(costItem),
			Cards:             taptap.ConvertList2Str(result),
		}
		taptap.WriteDataLog(taptap.LogType_CardExtract, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	// 返回消息
	rsp := &cmd.LS2C_NewbiePoolExtractRes{
		PoolId:     req.PoolId,
		ItemIds:    &cmd.PNewbiePoolLog{CardIds: result},
		CommonData: h.actor.comData.FixDownComData(),
	}

	// 发布事件
	errx := h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_POOL_EXTRACT, []int32{TASK_TYPE_407, TASK_TYPE_408}, map[string]interface{}{
		"pool_id": req.PoolId,
		"count":   int32(total),
	}))
	if errx != nil {
		h.Error(errx)
	}
	return rsp, nil, 0
}

func (h *PoolHandler) CardPoolExtractReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req cmd.C2LS_CardPoolExtractReq
	if err := in.UnmarshalData(&req); err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}
	err, code := h.commonCheck()
	if err != nil {
		return nil, err, code
	}

	// 参数check
	poolType := req.PoolType
	if poolType != common.CARD_POOL_TYPE_ONE && poolType != common.CARD_POOL_TYPE_TEN {
		return nil, fmt.Errorf("invalid pool type:%d", poolType), int32(cmd.ErrorCode_InvalidParam)
	}

	if !h.isValidPoolId(req.PoolId) {
		return nil, fmt.Errorf("invalid pool id:%d", req.PoolId), int32(cmd.ErrorCode_InvalidParam)
	}

	poolCfg := excel.GetPoolMgr().GetById(req.PoolId)
	if poolCfg == nil {
		return nil, fmt.Errorf("poolCfg not found %d", req.PoolId), int32(cmd.ErrorCode_NotFoundConfig)
	}
	// 排除新手池
	if poolCfg.PoolType == common.CARD_POOL_NEWBIE {
		return nil, fmt.Errorf("pool type not match"), int32(cmd.ErrorCode_IllegalOperationError)
	}

	// 抽卡次数
	var total int32
	var costItem map[int32]int32
	if poolType == common.CARD_POOL_TYPE_ONE {
		total = 1
		costItem = map[int32]int32{poolCfg.Gacha1Cost.Key: poolCfg.Gacha1Cost.Val}
	} else if poolType == common.CARD_POOL_TYPE_TEN {
		total = 10
		costItem = map[int32]int32{poolCfg.Gacha10Cost.Key: poolCfg.Gacha10Cost.Val}
	}

	// 次数上限检查
	isLimit := true
	if poolCfg.GetPoolType() == common.CARD_POOL_FRIEND {
		isLimit = false
	}
	if isLimit && !h.actor.UseLimitHandler.CheckUseEnough(int32(cmd.RedPointModuleType_Card_Pool_Module), total) {
		return nil, fmt.Errorf("use count limit"), int32(cmd.ErrorCode_UseLimitError)
	}

	// 消耗检查
	if !GetConsumeMgr(h.actor).CheckMapEnough(costItem) {
		return nil, fmt.Errorf("item not enough: %+v", costItem), int32(cmd.ErrorCode_NotEnoughItem)
	}

	err = GetConsumeMgr(h.actor).ConsumeList(costItem, h.actor.comData, common.CR_Card_Pool)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	// 去抽卡
	result, newCards, _, cardNum, err := h.handlePoolExtract(req.PoolId, int(total), h.actor.comData)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}
	h.Infof("抽卡结果: result %+v", result)

	// 次数记录
	if isLimit {
		err = h.actor.UseLimitHandler.AddUseCount(int32(cmd.RedPointModuleType_Card_Pool_Module), total)
		if err != nil {
			return nil, err, int32(cmd.ErrorCode_InternalError)
		}
	}

	// 埋点log
	//threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.CardExtract{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_CardExtract, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		PoolId:         req.PoolId,
	//		ExtractType:    poolType,
	//		Cost:           lilith.ConvertMap2Str(costItem),
	//		Cards:          lilith.ConvertList2Str(result),
	//	})
	//})
	threading.RunSafe(func() {
		e := &taptap.CardExtract{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			PoolId:            req.PoolId,
			ExtractType:       poolType,
			Cost:              taptap.ConvertMap2Str(costItem),
			Cards:             taptap.ConvertList2Str(result),
		}
		taptap.WriteDataLog(taptap.LogType_CardExtract, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	// 返回消息
	rsp := &cmd.LS2C_CardPoolExtractRes{
		PoolId:     req.PoolId,
		PoolType:   req.PoolType,
		ItemIds:    result,
		NewCards:   newCards,
		CommonData: h.actor.comData.FixDownComData(),
		AddNum:     cardNum,
	}

	// 发布事件
	errx := h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_POOL_EXTRACT, []int32{TASK_TYPE_407, TASK_TYPE_408}, map[string]interface{}{
		"pool_id": req.PoolId,
		"count":   total,
	}))
	if errx != nil {
		h.Error(errx)
	}

	// 稀有卡牌广播
	h.BroadcastMessage(in.RoleId, result)
	return rsp, nil, 0
}

func (h *PoolHandler) handleNewbiePoolExtract(poolId int32, total int, commonData *clidto.Comdata) ([]int32, []int32, []int32, []*cmd.KeyValueItem, error) {
	poolCfg := excel.GetPoolMgr().GetById(poolId)
	//mustBeUpNum := excel.GetConfigMgr().GetCfg().RECRUIT_GUARANTEE_NUM
	contentCfgs := getContentCfg(poolId)
	result := make([]int32, 0, total)

	poolsData := h.actor.GetPoolsData()
	if _, ok := poolsData.TypeInfos[poolCfg.GetPoolType()]; !ok {
		poolsData.TypeInfos[poolCfg.GetPoolType()] = createCardTypeInfo()
	}
	typeInfo := poolsData.TypeInfos[poolCfg.GetPoolType()]

	// 循环抽取
	var target *excel.PoolContentCfg
	var maxRarity int32
	for i := 1; i <= total; i++ {
		// 确定稀有度
		must, rarity := h.calPoolRarity(poolCfg, typeInfo, maxRarity)
		if rarity == common.POTENTIAL_SP {
			maxRarity++
		}

		//up := poolCfg.GetPoolType() == common.CARD_POOL_SPECIAL && rarity == common.POTENTIAL_SP && typeInfo.UpCount == mustBeUpNum
		temp := getContentCfg2(contentCfgs, rarity, false)
		target = randCard(temp)

		// 容错处理
		if target == nil {
			continue
		}
		h.Debug("抽卡配置表: ", target.GetId())

		result = append(result, target.GetCardId())

		// 抽卡后续处理
		tempCounter := map[int32]int32{common.POTENTIAL_SR: 0, common.POTENTIAL_SSR: 0, common.POTENTIAL_SP: 0}
		delete(tempCounter, rarity)
		typeInfo.Rarity[rarity] = 1

		// 有保底但是抽取了更高稀有度
		if _, ok := must[rarity]; !ok {
			for k := range must {
				typeInfo.Rarity[k] = 1
				delete(tempCounter, k)
			}
		}

		// 记录稀有度保底次数
		for k := range tempCounter {
			if _, ok := must[k]; !ok {
				typeInfo.Rarity[k] += 1
			}
		}

		// up保底次数记录
		//if poolCfg.GetPoolType() == common.CARD_POOL_SPECIAL && rarity == common.POTENTIAL_SP {
		//	// 命中
		//	if up || target.Up > 0 {
		//		typeInfo.UpCount = 0
		//	} else {
		//		typeInfo.UpCount++
		//	}
		//	h.Debugf("大保底次数: %d", typeInfo.UpCount)
		//}
	}

	// fixme 新手可以去掉这些逻辑
	// 获得卡牌
	newCards := make([]int32, 0)
	quality := make([]int32, 0)
	reward := make(map[uint32]uint32)
	cardNum := make([]*cmd.KeyValueItem, 0)
	tempCardNum := make(map[int32]int32)
	for _, itemId := range result {
		// 计算新获得
		temp := excel.GetItemMgr().GetById(itemId)
		if temp == nil {
			continue
		}
		if !h.actor.CardHandler.IsExistCard(uint32(temp.GetSystemId())) {
			newCards = append(newCards, itemId)
		}
		cardRarity, err := GetCardRarityByItemId(itemId)
		if err != nil {
			h.Error(err)
		}
		quality = append(quality, cardRarity)
		reward[uint32(itemId)] += 1
		// 卡牌获得次数
		if _, ok := tempCardNum[itemId]; !ok {
			tempCardNum[itemId] = 0
			card, err := h.actor.CardHandler.GetCard(uint32(temp.SystemId))
			if err != nil {
				cardNum = append(cardNum, &cmd.KeyValueItem{Key: itemId, Value: 0})
			} else {
				cardNum = append(cardNum, &cmd.KeyValueItem{Key: itemId, Value: int32(card.AddNum)})
			}
		}
	}

	h.Debugf("卡牌转材料 cards: %v, change: %+v", reward, commonData.Data.Items)
	// 保存数据
	poolsData.Newbie.Results = append(poolsData.Newbie.Results, &cmd.PNewbiePoolLog{CardIds: result})
	if err := h.SaveDB(); err != nil {
		return nil, nil, nil, nil, err
	}

	return result, newCards, quality, cardNum, nil
}

// handlePoolExtract
//
//	@Description: 抽卡逻辑处理
//	@receiver h
//	@param poolId 卡池id
//	@param total 抽卡总次数
//	@param commonData
//	@return []int32 掉落道具id列表
//	@return []int32 首次获得的卡牌id列表
//	@return []int32 卡牌品质分布，gm指令用
//	@return []*cmd.KeyValueItem 卡牌获取次数
//	@return error
func (h *PoolHandler) handlePoolExtract(poolId int32, total int, commonData *clidto.Comdata) ([]int32, []int32, []int32, []*cmd.KeyValueItem, error) {
	poolCfg := excel.GetPoolMgr().GetById(poolId)
	mustBeUpNum := excel.GetConfigMgr().GetCfg().RECRUIT_GUARANTEE_NUM
	contentCfgs := getContentCfg(poolId)
	result := make([]int32, 0, total)

	poolsData := h.actor.GetPoolsData()
	if _, ok := poolsData.TypeInfos[poolCfg.GetPoolType()]; !ok {
		poolsData.TypeInfos[poolCfg.GetPoolType()] = createCardTypeInfo()
	}
	typeInfo := poolsData.TypeInfos[poolCfg.GetPoolType()]

	// 循环抽取
	var target *excel.PoolContentCfg
	for i := 1; i <= total; i++ {
		// 确定稀有度
		must, rarity := h.calPoolRarity(poolCfg, typeInfo, -1)

		up := poolCfg.GetPoolType() == common.CARD_POOL_SPECIAL && rarity == common.POTENTIAL_SP && typeInfo.UpCount == mustBeUpNum
		temp := getContentCfg2(contentCfgs, rarity, up)
		target = randCard(temp)

		// 容错处理
		if target == nil {
			continue
		}
		h.Debug("抽卡配置表: ", target.GetId())

		result = append(result, target.GetCardId())

		// 抽卡后续处理
		tempCounter := map[int32]int32{common.POTENTIAL_SR: 0, common.POTENTIAL_SSR: 0, common.POTENTIAL_SP: 0}
		delete(tempCounter, rarity)
		typeInfo.Rarity[rarity] = 1

		// 有保底但是抽取了更高稀有度
		if _, ok := must[rarity]; !ok {
			for k := range must {
				typeInfo.Rarity[k] = 1
				delete(tempCounter, k)
			}
		}

		// 记录稀有度保底次数
		for k := range tempCounter {
			if _, ok := must[k]; !ok {
				typeInfo.Rarity[k] += 1
			}
		}

		// up保底次数记录
		if poolCfg.GetPoolType() == common.CARD_POOL_SPECIAL && rarity == common.POTENTIAL_SP {
			// 命中
			if up || target.Up > 0 {
				typeInfo.UpCount = 0
			} else {
				typeInfo.UpCount++
			}
			h.Debugf("大保底次数: %d", typeInfo.UpCount)
		}
	}

	// 获得卡牌
	newCards := make([]int32, 0)
	quality := make([]int32, 0)
	reward := make(map[uint32]uint32)
	cardNum := make([]*cmd.KeyValueItem, 0)
	tempCardNum := make(map[int32]int32)
	for _, itemId := range result {
		// 计算新获得
		temp := excel.GetItemMgr().GetById(itemId)
		if temp == nil {
			continue
		}
		if !h.actor.CardHandler.IsExistCard(uint32(temp.GetSystemId())) {
			newCards = append(newCards, itemId)
		}
		cardRarity, err := GetCardRarityByItemId(itemId)
		if err != nil {
			h.Error(err)
		}
		quality = append(quality, cardRarity)
		reward[uint32(itemId)] += 1
		// 卡牌获得次数
		if _, ok := tempCardNum[itemId]; !ok {
			tempCardNum[itemId] = 0
			card, err := h.actor.CardHandler.GetCard(uint32(temp.SystemId))
			if err != nil {
				cardNum = append(cardNum, &cmd.KeyValueItem{Key: itemId, Value: 0})
			} else {
				cardNum = append(cardNum, &cmd.KeyValueItem{Key: itemId, Value: int32(card.AddNum)})
			}
		}
	}
	_, err := GetDropMgr(h.actor).DropList(reward, true, nil, commonData, common.CR_Card_Pool)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	h.Debugf("卡牌转材料 cards: %v, change: %+v", reward, commonData.Data.Items)
	// 保存数据
	if err = h.SaveDB(); err != nil {
		return nil, nil, nil, nil, err
	}

	return result, newCards, quality, cardNum, nil
}

// 1.计算稀有度
func (h *PoolHandler) calPoolRarity(poolCfg *excel.PoolCfg, typeInfo *cmd.PServerCardPoolType, maxRarity int32) (map[int32]int32, int32) {
	must := make(map[int32]int32)

	// 查表获取稀有度权重
	h.Debug("稀有度保底次数 ", typeInfo.Rarity)
	weight3 := getWeightCfg(typeInfo, poolCfg, common.POTENTIAL_R)
	weight4 := getWeightCfg(typeInfo, poolCfg, common.POTENTIAL_SR)
	weight5 := getWeightCfg(typeInfo, poolCfg, common.POTENTIAL_SSR)
	weight6 := getWeightCfg(typeInfo, poolCfg, common.POTENTIAL_SP)

	// 新手池并且6星数量到上限了
	if maxRarity >= 0 && maxRarity >= excel.GetConfigMgr().GetCfg().GACHA_NEWPLAYER_RARITY_LIMIT {
		weight6 = 0
	}

	// 构建随机权重 k=稀有度,v=权重值
	weightMap := make(map[interface{}]int32)
	weightMap[common.POTENTIAL_R] = weight3
	weightMap[common.POTENTIAL_SR] = weight4
	weightMap[common.POTENTIAL_SSR] = weight5
	weightMap[common.POTENTIAL_SP] = weight6

	// 保底抽处理
	if weight4 == common.CARD_POOL_MUST_VALUE {
		must[common.POTENTIAL_SR] = 0
		delete(weightMap, common.POTENTIAL_R)
	}
	if weight5 == common.CARD_POOL_MUST_VALUE {
		must[common.POTENTIAL_SSR] = 0
		delete(weightMap, common.POTENTIAL_R)
		delete(weightMap, common.POTENTIAL_SR)
	}
	if weight6 == common.CARD_POOL_MUST_VALUE {
		must[common.POTENTIAL_SP] = 0
		delete(weightMap, common.POTENTIAL_R)
		delete(weightMap, common.POTENTIAL_SR)
		delete(weightMap, common.POTENTIAL_SSR)
	}

	// 按照权重值确定稀有度
	target := utils.RandomMap(weightMap, true)
	if _, ok := target.(int32); !ok {
		h.Warnf("calPoolRarity type assert failed. target:%v, weightMap:%+v", target, weightMap)
		return must, common.POTENTIAL_R
	}

	h.Debugf("抽卡稀有度 must %v target %+v", must, target)
	return must, target.(int32)
}

func createCardTypeInfo() *cmd.PServerCardPoolType {
	m := make(map[int32]int32)
	m[common.POTENTIAL_R] = 1   // 永远是1
	m[common.POTENTIAL_SR] = 1  // 初始是1
	m[common.POTENTIAL_SSR] = 1 // 初始是1
	m[common.POTENTIAL_SP] = 1  // 初始是1
	return &cmd.PServerCardPoolType{Rarity: m}
}

func (h *PoolHandler) isValidPoolId(id int32) bool {
	open := h.getOpenPoolCfg()
	// 时间上是否开启
	if _, ok := open[id]; !ok {
		return false
	}
	// 新手池特殊处理
	cfg := excel.GetPoolMgr().GetById(id)
	if cfg == nil {
		return false
	}
	if cfg.GetPoolType() == common.CARD_POOL_NEWBIE {
		data := h.actor.GetPoolsData()
		// 已经选择奖励了
		if data.Newbie.Select > 0 {
			return false
		}
	}

	return true
}

func (h *PoolHandler) getOpenPoolCfg() map[int32]*OpenPool {
	open := make(map[int32]*OpenPool)
	now := time.Now()
	excel.GetPoolMgr().Foreach(func(cfg *excel.PoolCfg) bool {
		// 新手池判断
		if cfg.GetPoolType() == common.CARD_POOL_NEWBIE {
			data := h.actor.GetPoolsData()
			if data.Newbie.Select > 0 {
				return true
			}
		}

		// 常驻卡池判定
		if cfg.GetEndTime() == "" {
			open[cfg.GetId()] = &OpenPool{
				PoolType: cfg.GetPoolType(),
				Start:    0,
				End:      0,
			}
			return true
		}

		start, err := common.ParseDate(cfg.GetStartTime())
		if err != nil {
			logger.Error("pool excel field starttime err:", cfg.GetStartTime())
			return true
		}
		end, err := common.ParseDate(cfg.GetEndTime())
		if err != nil {
			logger.Error("pool excel field endtime err:", cfg.GetEndTime())
			return true
		}
		if now.After(start) && now.Before(end) {
			open[cfg.GetId()] = &OpenPool{
				PoolType: cfg.GetPoolType(),
				Start:    start.Unix(),
				End:      end.Unix(),
			}
		}

		return true
	}, true)

	return open
}

func getContentCfg(poolId int32) []*excel.PoolContentCfg {
	target := make([]*excel.PoolContentCfg, 0)
	excel.GetPoolContentMgr().Foreach(func(cfg *excel.PoolContentCfg) bool {
		if cfg.GetPoolId() == poolId {
			target = append(target, cfg)
		}
		return true
	}, true)

	return target
}

func getContentCfg2(all []*excel.PoolContentCfg, rarity int32, up bool) []*excel.PoolContentCfg {
	target := make([]*excel.PoolContentCfg, 0)
	temp := make(map[int32]int32)
	for _, cfg := range all {
		cardRarity, err := GetCardRarityByItemId(cfg.CardId)
		if err != nil || cardRarity != rarity {
			continue
		}
		f := true
		if up && cfg.Up == 0 {
			f = false
		}
		if f {
			target = append(target, cfg)
			temp[cfg.Id] = cfg.Weight
		}
	}
	logger.Debugf("稀有度目标卡池配置id列表: %v", temp)
	return target
}

func getWeightCfg(typeInfo *cmd.PServerCardPoolType, poolCfg *excel.PoolCfg, rarity int32) int32 {
	count := typeInfo.Rarity[rarity]
	cfg := excel.GetDynamicWeightMgr().GetById(count)
	if cfg == nil {
		// 容错处理,从次数1开始
		typeInfo.Rarity[rarity] = 1
		cfg = excel.GetDynamicWeightMgr().GetById(1)
		logger.Warn("dynamic weight config not found ", count)
	}

	if poolCfg.GetPoolType() == common.CARD_POOL_NORMAL {
		switch rarity {
		case common.POTENTIAL_R:
			return cfg.GetNormal3()
		case common.POTENTIAL_SR:
			return cfg.GetNormal4()
		case common.POTENTIAL_SSR:
			return cfg.GetNormal5()
		case common.POTENTIAL_SP:
			return cfg.GetNormal6()
		}
	} else if poolCfg.GetPoolType() == common.CARD_POOL_SPECIAL {
		switch rarity {
		case common.POTENTIAL_R:
			return cfg.GetSp3()
		case common.POTENTIAL_SR:
			return cfg.GetSp4()
		case common.POTENTIAL_SSR:
			return cfg.GetSp5()
		case common.POTENTIAL_SP:
			return cfg.GetSp6()
		}
	} else if poolCfg.GetPoolType() == common.CARD_POOL_NEWBIE {
		switch rarity {
		case common.POTENTIAL_R:
			ret3 := cfg.GetNewplayer3()
			// 新手池特殊容错
			if ret3 == 0 {
				typeInfo.Rarity[rarity] = 1
				ret3 = excel.GetDynamicWeightMgr().GetById(1).GetNewplayer3()
			}
			return ret3
		case common.POTENTIAL_SR:
			ret4 := cfg.GetNewplayer4()
			// 新手池特殊容错
			if ret4 == 0 {
				typeInfo.Rarity[rarity] = 1
				ret4 = excel.GetDynamicWeightMgr().GetById(1).GetNewplayer4()
			}
			return ret4
		case common.POTENTIAL_SSR:
			ret5 := cfg.GetNewplayer5()
			if ret5 == 0 {
				typeInfo.Rarity[rarity] = 1
				ret5 = excel.GetDynamicWeightMgr().GetById(1).GetNewplayer5()
			}
			return ret5
		case common.POTENTIAL_SP:
			ret6 := cfg.GetNewplayer6()
			if ret6 == 0 {
				typeInfo.Rarity[rarity] = 1
				ret6 = excel.GetDynamicWeightMgr().GetById(1).GetNewplayer6()
			}
			return ret6
		}
	} else if poolCfg.GetPoolType() == common.CARD_POOL_FRIEND {
		switch rarity {
		case common.POTENTIAL_R:
			return cfg.GetFriend3()
		case common.POTENTIAL_SR:
			return cfg.GetFriend4()
		case common.POTENTIAL_SSR:
			return cfg.GetFriend5()
		case common.POTENTIAL_SP:
			return cfg.GetFriend6()
		}
	} else {
		logger.Warn("unrealized pool type ", poolCfg.GetPoolType())
	}
	return 0
}

// 随机抽取卡牌
func randCard(source []*excel.PoolContentCfg) *excel.PoolContentCfg {
	weightMap := make(map[interface{}]int32)
	for _, cfg := range source {
		weightMap[cfg] = cfg.GetWeight()
	}
	targetCfg := utils.RandomMap(weightMap, true)

	if _, ok := targetCfg.(*excel.PoolContentCfg); !ok {
		logger.Warnf("randCard type assert failed. target:%v, weightMap:%+v", targetCfg, weightMap)
		return nil
	}
	return targetCfg.(*excel.PoolContentCfg)
}

func (h *PoolHandler) ResetCardLogByGM() error {
	return h.Init()
}

func (h *PoolHandler) BroadcastMessage(roleId uint64, cardIds []int32) {
	h.Infof("玩家[%d]:%d,抽到卡片[%+v],开始广播 ", roleId, cardIds)
	quality := make(map[int32]interface{})
	excel.GetSystemInfoMgr().Foreach(func(cfg *excel.SystemInfoCfg) bool {
		if cfg.Type == common.Broadcast_type {
			quality[cfg.GetParameter()] = struct{}{}
		}
		return true
	}, true)
	h.Infof("可以广播的卡片稀有度:%v", quality)
	roleData := h.actor.GetUserData()
	if roleData == nil || roleData.Common == nil {
		h.Debug("pool broad cast message getRoleDetailInfo err")
		return
	}
	//
	broadMessage := make([]*cmd.BroadMessage, 0, len(cardIds))
	for _, id := range cardIds {
		rarity, err := GetCardRarityByItemId(id)
		if err != nil {
			continue
		}
		//拼接消息
		if _, ok := quality[rarity]; !ok {
			continue
		}
		broadMessage = append(broadMessage, h.NewMessage(roleData.Common.RoleName, id))
	}
	if len(broadMessage) <= 0 {
		return
	}
	h.actor.UserChatHandler.BroadcastMessages(roleId, cmd.ChatChannel_Channel_system, broadMessage)
}

func (h *PoolHandler) NewMessage(roleName string, cardId int32) *cmd.BroadMessage {
	data := []string{roleName, strconv.Itoa(int(cardId))}
	message := &cmd.BroadMessage{
		MType:      cmd.MessageType_Message_Type_card,
		FromRoleId: 0,
		Data:       data,
		TimeStamp:  time.Now().Unix(),
	}
	return message
}
