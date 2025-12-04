package useractor

import (
	"context"
	"fmt"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/actorserver/useractor/event"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datalog/taptap"
	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"
	"time"

	"gitlab.musadisca-games.com/wangxw/musae/framework/threading"

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

const (
	POOL_TYPE_UP = 1 // 限定池
)

type OpenCampPool struct {
	PoolSp int32
	Start  int64
	End    int64
}

type CampPoolHandler struct {
	*UABaseHandler
}

func NewCampPoolHandler(actor *UserActor) *CampPoolHandler {
	h := &CampPoolHandler{UABaseHandler: NewUABaseHandler(actor, "CampPoolHandler")}
	h.ChildHandler = h

	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_GetCampPoolInfoReq), h.GetCampPoolInfoReq)
	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_CampPoolExtractReq), h.CampPoolExtractReq)
	return h
}

// Init 初始化模块数据
func (h *CampPoolHandler) Init() error {
	// 初始化
	h.actor.Data.CampPools = &cmd.PServerCampPoolInfos{
		Createtime: time.Now().Unix(),
		TypeInfos:  make(map[int32]*cmd.PServerCampPoolType),
	}

	// 保存
	if err := h.SaveDB(); err != nil {
		return err
	}

	h.Debug("init card pool data success. player: %s", h.actor.ID())
	return nil
}

func (h *CampPoolHandler) EnterGame() error {
	return nil
}

func (h *CampPoolHandler) DailyRefresh() error {
	return nil
}

func (h *CampPoolHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*cmd.PServerCampPoolInfos); ok {
		h.actor.Data.CampPools = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *CampPoolHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyUserCampPool(h.actor.ID()), h.actor.Data.CampPools
}

// 通用解锁判定
func (h *CampPoolHandler) commonCheck() (error, int32) {
	err, code := h.actor.FuncUnlockHandler.CheckFuncUnlock(FUNC_ID_1002)
	if err != nil {
		return err, int32(code)
	}
	return nil, int32(cmd.ErrorCode_Success)
}

func (h *CampPoolHandler) GetCampPoolInfoReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req cmd.C2LS_GetCampPoolInfoReq
	if err := in.UnmarshalData(&req); err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	err, code := h.commonCheck()
	if err != nil {
		return nil, err, code
	}

	poolsData := h.actor.GetCampPoolsData()
	infos := make([]*cmd.PClientCardPoolInfo, 0)
	// 获取所有开启中的卡池
	openPoolCfg := getCampOpenPoolCfg()
	if req.PoolId > 0 {
		// 获取指定的卡池信息
		needPool := openPoolCfg[req.PoolId]
		if needPool != nil {
			if _, ok := poolsData.TypeInfos[needPool.PoolSp]; !ok {
				poolsData.TypeInfos[needPool.PoolSp] = createCampTypeInfo()
			}

			typeInfo := poolsData.TypeInfos[needPool.PoolSp]
			infos = append(infos, &cmd.PClientCardPoolInfo{
				PoolId: req.PoolId,
				Start:  needPool.Start,
				End:    needPool.End,
				Num:    typeInfo.UpCount,
			})
		}
	} else {
		for id, v := range openPoolCfg {
			if _, ok := poolsData.TypeInfos[v.PoolSp]; !ok {
				poolsData.TypeInfos[v.PoolSp] = createCampTypeInfo()
			}

			typeInfo := poolsData.TypeInfos[v.PoolSp]
			infos = append(infos, &cmd.PClientCardPoolInfo{
				PoolId: id,
				Start:  v.Start,
				End:    v.End,
				Num:    typeInfo.UpCount,
			})
		}
	}

	return &cmd.LS2C_GetCampPoolInfoRes{Infos: infos}, nil, 0
}

func (h *CampPoolHandler) CampPoolExtractReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req cmd.C2LS_CampPoolExtractReq
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

	if !isCampValidPoolId(req.PoolId) {
		return nil, fmt.Errorf("invalid pool id:%d", req.PoolId), int32(cmd.ErrorCode_InvalidParam)
	}

	// 抽卡次数
	total := 0
	if poolType == common.CARD_POOL_TYPE_ONE {
		total = 1
	} else if poolType == common.CARD_POOL_TYPE_TEN {
		total = 10
	}
	// 次数上限检查
	if !h.actor.UseLimitHandler.CheckUseEnough(int32(cmd.RedPointModuleType_Camp_Module), int32(total)) {
		return nil, fmt.Errorf("use count limit"), int32(cmd.ErrorCode_UseLimitError)
	}
	poolCfg := excel.GetCampPoolMgr().GetById(req.PoolId)
	if poolCfg == nil {
		return nil, fmt.Errorf("poolCfg not found %d", req.PoolId), int32(cmd.ErrorCode_NotFoundConfig)
	}
	costItem := make(map[int32]int32)

	// 抽卡券check + 基因check
	if poolCfg.GetPoolSp() == common.CARD_POOL_TYPE_NORMAL {
		cost := excel.GetConfigMgr().GetCfg().CAMP_LOTTERY_COST
		costItem[cost.Key] = int32(total) * cost.Val
	} else if poolCfg.GetPoolSp() == common.CARD_POOL_TYPE_SPECIAL {
		cost := excel.GetConfigMgr().GetCfg().CAMP_LOTTERY_COST
		costItem[cost.Key] = int32(total) * cost.Val
	} else {
		return nil, fmt.Errorf("unrealized card pool type"), int32(cmd.ErrorCode_UnrealizedTypeError)
	}

	if !GetConsumeMgr(h.actor).CheckMapEnough(costItem) {
		return nil, fmt.Errorf("item not enough: %+v", costItem), int32(cmd.ErrorCode_NotEnoughItem)
	}

	// 基因扣除 + 抽卡券扣除
	err = GetConsumeMgr(h.actor).ConsumeList(costItem, h.actor.comData, common.CR_CAMP_POOL)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	// 去抽卡
	result, _, ConvertChange, err := h.handlePoolExtract(req.PoolId, total, h.actor.comData)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}
	h.Debugf("抽卡结果: player %s result %+v", h.actor.ID(), ConvertChange)

	// 次数记录
	err = h.actor.UseLimitHandler.AddUseCount(int32(cmd.RedPointModuleType_Camp_Module), int32(total))
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}

	// 埋点log
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
	rsp := &cmd.LS2C_CampPoolExtractRes{
		PoolId:            req.PoolId,
		PoolType:          req.PoolType,
		ConvertDropChange: ConvertChange,
		CommonData:        h.actor.comData.FixDownComData(),
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

// handlePoolExtract
//
//	@Description: 抽卡逻辑处理
//	@receiver h
//	@param poolId 卡池id
//	@param total 抽卡总次数
//	@param commonData
//	@return []int32 掉落道具id列表
//	@return []int32 品质分布，gm指令用
//	@return error
func (h *CampPoolHandler) handlePoolExtract(poolId int32, total int, commonData *clidto.Comdata) ([]int32, []int32, *cmd.ConvertDropChange, error) {
	poolCfg := excel.GetCampPoolMgr().GetById(poolId)
	mustBeUpNum := excel.GetConfigMgr().GetCfg().RECRUIT_GUARANTEE_NUM
	contentCfgs := getCampContentCfg(poolId)
	result := make([]int32, 0, total)

	poolsData := h.actor.GetCampPoolsData()
	if _, ok := poolsData.TypeInfos[poolCfg.GetPoolSp()]; !ok {
		poolsData.TypeInfos[poolCfg.GetPoolSp()] = createCampTypeInfo()
	}
	typeInfo := poolsData.TypeInfos[poolCfg.GetPoolSp()]

	// 循环抽取
	var target *excel.CampPoolContentCfg
	for i := 1; i <= total; i++ {
		// 确定稀有度
		must, rarity := h.calPoolRarity(poolCfg, typeInfo)

		up := poolCfg.PoolSp == POOL_TYPE_UP && rarity == common.POTENTIAL_SP && typeInfo.UpCount == mustBeUpNum
		temp := getCampContentCfg2(contentCfgs, rarity, up)
		target = randCampCard(temp)

		// 容错处理
		if target == nil {
			continue
		}
		h.Debug("抽卡配置表: ", target.GetId())

		result = append(result, target.GetItemId())

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
		if poolCfg.PoolSp == POOL_TYPE_UP && rarity == common.POTENTIAL_SP {
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
	quality := make([]int32, 0)
	reward := make(map[uint32]uint32)
	limitChange := &cmd.ConvertDropChange{}
	diffMap := h.actor.CampHandler.GetItemDiff(result)
	for _, itemId := range result {
		temp := excel.GetItemMgr().GetById(itemId)
		if temp == nil {
			continue
		}
		quality = append(quality, temp.Quality)
		convertIterm := &cmd.ConvertItermReward{
			OriginItemId: itemId,
			OriginNum:    1,
		}
		//判断是否超过上限
		if diff, _ := diffMap[itemId]; diff == 0 { // 超过限制
			convertItermId, convertNum := h.actor.CampHandler.ConvertBuilding(temp.Quality)
			if convertItermId == 0 {
				h.Debugf("handlePoolExtract iterm[%d]超过上限，根据品质[%d]获取转换材料失败[%d]", itemId, temp.Quality, convertItermId)
				continue
			}
			reward[uint32(convertItermId)] += uint32(convertNum)
			convertIterm.ConvertItemId = convertItermId
			convertIterm.ConvertNum = convertNum
		} else {
			reward[uint32(itemId)] += 1
			diffMap[itemId] = diff - 1 //没超过限制则-1
		}
		limitChange.Items = append(limitChange.Items, convertIterm)
	}
	_, err := GetDropMgr(h.actor).DropList(reward, true, nil, commonData, common.CR_CAMP_POOL)
	if err != nil {
		return nil, nil, nil, err
	}

	h.Debugf("卡牌转材料 cards: %v, change: %+v", reward, commonData.Data.Items)
	// 保存数据
	if err = h.SaveDB(); err != nil {
		return nil, nil, nil, err
	}

	return result, quality, limitChange, nil
}

// 1.计算稀有度
func (h *CampPoolHandler) calPoolRarity(poolCfg *excel.CampPoolCfg, typeInfo *cmd.PServerCampPoolType) (map[int32]int32, int32) {
	must := make(map[int32]int32)

	// 查表获取稀有度权重
	rarityMap := typeInfo.Rarity
	h.Debug("稀有度保底次数 ", rarityMap)
	//weight3 := getCampWeightCfg(rarityMap[common.POTENTIAL_R], poolCfg, common.POTENTIAL_R)
	weight4 := getCampWeightCfg(rarityMap[common.POTENTIAL_SR], poolCfg, common.POTENTIAL_SR)
	weight5 := getCampWeightCfg(rarityMap[common.POTENTIAL_SSR], poolCfg, common.POTENTIAL_SSR)
	weight6 := getCampWeightCfg(rarityMap[common.POTENTIAL_SP], poolCfg, common.POTENTIAL_SP)

	// 构建随机权重 k=稀有度,v=权重值
	weightMap := make(map[interface{}]int32)
	//weightMap[common.POTENTIAL_R] = weight3
	weightMap[common.POTENTIAL_SR] = weight4
	weightMap[common.POTENTIAL_SSR] = weight5
	weightMap[common.POTENTIAL_SP] = weight6

	// 保底抽处理
	if weight4 == common.CARD_POOL_MUST_VALUE {
		must[common.POTENTIAL_SR] = 0
		//delete(weightMap, common.POTENTIAL_R)
	}
	if weight5 == common.CARD_POOL_MUST_VALUE {
		must[common.POTENTIAL_SSR] = 0
		//delete(weightMap, common.POTENTIAL_R)
		delete(weightMap, common.POTENTIAL_SR)
	}
	if weight6 == common.CARD_POOL_MUST_VALUE {
		must[common.POTENTIAL_SP] = 0
		//delete(weightMap, common.POTENTIAL_R)
		delete(weightMap, common.POTENTIAL_SR)
		delete(weightMap, common.POTENTIAL_SSR)
	}

	// 按照权重值确定稀有度
	target := utils.RandomMap(weightMap, true)
	if _, ok := target.(int32); !ok {
		h.Warnf("calPoolRarity type assert failed. target:%v, weightMap:%+v", target, weightMap)
		return must, common.POTENTIAL_SR
	}

	h.Debugf("抽卡稀有度 must %v target %+v", must, target)
	return must, target.(int32)
}

func createCampTypeInfo() *cmd.PServerCampPoolType {
	m := make(map[int32]int32)
	m[common.POTENTIAL_SR] = 1
	m[common.POTENTIAL_SSR] = 1
	m[common.POTENTIAL_SP] = 1
	return &cmd.PServerCampPoolType{
		Rarity: m,
	}
}

func isCampValidPoolId(id int32) bool {
	open := getCampOpenPoolCfg()
	// 是否开启
	if _, ok := open[id]; !ok {
		return false
	}

	return true
}

func getCampOpenPoolCfg() map[int32]*OpenCampPool {
	open := make(map[int32]*OpenCampPool)
	now := time.Now()
	excel.GetCampPoolMgr().Foreach(func(cfg *excel.CampPoolCfg) bool {
		// 常驻卡池判定
		if cfg.GetEndTime() == "" {
			open[cfg.GetId()] = &OpenCampPool{
				PoolSp: cfg.PoolSp,
				Start:  0,
				End:    0,
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
			open[cfg.GetId()] = &OpenCampPool{
				PoolSp: cfg.PoolSp,
				Start:  start.Unix(),
				End:    end.Unix(),
			}
		}

		return true
	}, true)

	return open
}

func getCampContentCfg(poolId int32) []*excel.CampPoolContentCfg {
	target := make([]*excel.CampPoolContentCfg, 0)
	excel.GetCampPoolContentMgr().Foreach(func(cfg *excel.CampPoolContentCfg) bool {
		if cfg.GetPoolId() == poolId {
			target = append(target, cfg)
		}
		return true
	}, true)

	return target
}

func getCampContentCfg2(all []*excel.CampPoolContentCfg, rarity int32, up bool) []*excel.CampPoolContentCfg {
	target := make([]*excel.CampPoolContentCfg, 0)
	temp := make(map[int32]int32)
	for _, cfg := range all {
		itemCfg := excel.GetItemMgr().GetById(cfg.ItemId)
		if itemCfg == nil || itemCfg.Quality != rarity {
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

func getCampWeightCfg(count int32, poolCfg *excel.CampPoolCfg, rarity int32) int32 {
	cfg := excel.GetCampDynamicWeightMgr().GetById(count)
	if cfg == nil {
		logger.Warn("dynamic weight config not found ", count)
		return 0
	}

	if poolCfg.GetPoolSp() == common.CARD_POOL_TYPE_NORMAL {
		switch rarity {
		case common.POTENTIAL_R:
			return 0
		case common.POTENTIAL_SR:
			return cfg.GetCampNormal4()
		case common.POTENTIAL_SSR:
			return cfg.GetCampNormal5()
		case common.POTENTIAL_SP:
			return cfg.GetCampNormal6()
		}
	} else if poolCfg.GetPoolSp() == common.CARD_POOL_TYPE_SPECIAL {
		switch rarity {
		case common.POTENTIAL_R:
			return 0
		case common.POTENTIAL_SR:
			return cfg.GetCampSp4()
		case common.POTENTIAL_SSR:
			return cfg.GetCampSp5()
		case common.POTENTIAL_SP:
			return cfg.GetCampSp6()
		}
	} else {
		logger.Warn("unrealized pool type ", poolCfg.GetPoolSp())
	}
	return 0
}

// 随机抽取卡牌
func randCampCard(source []*excel.CampPoolContentCfg) *excel.CampPoolContentCfg {
	weightMap := make(map[interface{}]int32)
	for _, cfg := range source {
		weightMap[cfg] = cfg.GetWeight()
	}
	targetCfg := utils.RandomMap(weightMap, true)

	if _, ok := targetCfg.(*excel.CampPoolContentCfg); !ok {
		logger.Warnf("randCard type assert failed. target:%v, weightMap:%+v", targetCfg, weightMap)
		return nil
	}
	return targetCfg.(*excel.CampPoolContentCfg)
}
