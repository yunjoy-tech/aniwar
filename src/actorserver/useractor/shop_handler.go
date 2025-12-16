package useractor

import (
	"context"
	"fmt"
	"gitee.com/aniwar2/aniwar/src/meta"
	"time"

	"github.com/pkg/errors"

	"gitee.com/aniwar2/aniwar/src/common/datalog/taptap"

	"gitee.com/aniwar2/musae/threading"

	"gitee.com/aniwar2/aniwar/src/common"
	"gitee.com/aniwar2/aniwar/src/common/clidto"
	"gitee.com/aniwar2/aniwar/src/common/db"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/base"
	"gitee.com/aniwar2/musae/service"
	"google.golang.org/protobuf/proto"
)

var MAX_REFRESH_SEC int64 = -1       // 商店不刷新
var DEFAULT_SHOP_LAYER_IDX int32 = 1 // 默认商品层级

type ShopHandler struct {
	*UABaseHandler
}

func NewShopHandler(actor *UserActor) *ShopHandler {
	h := &ShopHandler{UABaseHandler: NewUABaseHandler(actor, "ShopHandler")}
	h.ChildHandler = h

	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_ShopListReq), h.GetShopListReq)                // 请求商店列表
	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_ShopInfoReq), h.GetShopInfoReq)                // 请求商店信息
	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_ShopBuyReq), h.ShopBuyReq)                     // 购买
	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_ShopManualRefreshReq), h.ShopManualRefreshReq) // 手动刷新商店

	return h
}

// Init 初始化模块数据
func (h *ShopHandler) Init() error {
	// 初始化
	h.actor.Data.ShopData = &pb.LS2DB_ShopData{
		Createtime: time.Now().Unix(),
		ShopInfos:  make(map[int32]*pb.ShopInfo, 0),
	}

	// 保存
	if err := h.SaveDB(true); err != nil {
		return err
	}

	h.Debugf("init shop data success. player: %s", h.actor.ID())
	return nil
}

func (h *ShopHandler) EnterGame() error {
	return nil
}

func (h *ShopHandler) DailyRefresh() error {
	return nil
}

func (h *ShopHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*pb.LS2DB_ShopData); ok {
		h.actor.Data.ShopData = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *ShopHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyUserShopInfo(h.actor.ID()), h.actor.Data.ShopData
}

// 获取商店列表
func (h *ShopHandler) GetShopListReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err error
		// dbShopData *pb.LS2DB_ShopData
		resp = &pb.LS2C_ShopListRes{}
	)

	// _, _, reqData := in.MsgId, in.UserId, in.Data
	var req pb.C2LS_ShopListReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	// 商店列表信息
	// dbShopData = h.actor.GetShopData()
	//
	// if len(dbShopData.ShopInfos) <= 0 {
	//	resp.ShopInfos = getShopIds()
	// }
	resp.ShopInfos = getShopIds(req.ShowType)

	// 埋点
	// threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.ShopList{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_shop_list, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		ShopIds:        lilith.ConvertList2Str(resp.ShopInfos),
	//	})
	// })
	threading.RunSafe(func() {
		e := &taptap.ShopList{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			ShopIds:           taptap.ConvertList2Str(resp.ShopInfos),
		}
		taptap.WriteDataLog(taptap.LogType_shop_list, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	return resp, nil, 0
}

// GetShopInfoReq 请求商品列表
func (h *ShopHandler) GetShopInfoReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err      error
		shopInfo *pb.ShopInfo
		rsp      = &pb.LS2C_ShopInfoRes{ShopInfo: &pb.ShopInfo{GoodsIds: make([]*pb.ShopGoodsInfo, 0)}}
	)

	// _, _, reqData := in.MsgId, in.UserId, in.Data
	var req pb.C2LS_ShopInfoReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	shopInfo, err = h.getShopInfo(req.ShopId)
	if err != nil {
		// // 需要创建新的商店
		// shopInfo = h.createShopInfo(req.ShopId)
		// err = h.saveShopData2DB(shopInfo)
		// if err != nil {
		return nil,
			fmt.Errorf("创建商店报错, saveDB, shopId=%d, err:%+v", req.ShopId, err),
			int32(pb.ErrorCode_InternalError)
		// }
	}

	// 下发指定层级的商品
	// for _, goods := range shopInfo.GoodsIds {
	//	goodsCfg := data.GetShopGoodsMgr().GetById(int32(goods.GetGoodsId()))
	//	if goodsCfg.GetLayer() != int32(req.GetLayerId()) {
	//		continue
	//	}
	//
	//	rsp.ShopInfo.GoodsIds = append(rsp.ShopInfo.GoodsIds, goods)
	// }
	//
	// rsp.ShopInfo.ShopId = req.ShopId
	// rsp.ShopInfo.ShopLayer = req.LayerId
	rsp.ShopInfo = shopInfo

	h.Debugf("商店信息请求结果:%+v", rsp)

	// 埋点
	// threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.ShopInfo{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_shop_info, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		ShopId:         rsp.ShopInfo.ShopId,                                 // id
	//		ShopLayer:      rsp.ShopInfo.ShopLayer,                              // 商店层级
	//		ShopGoodsInfo:  lilith.ConvertListStruct2Str(rsp.ShopInfo.GoodsIds), // 商品ids []*pb.ShopGoodsInfo
	//		ExpireTimeSec:  rsp.ShopInfo.ExpireTimeSec,                          // 过期时间戳
	//	})
	// })
	threading.RunSafe(func() {
		e := &taptap.ShopInfo{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			ShopId:            rsp.ShopInfo.ShopId,                                 // id
			ShopLayer:         rsp.ShopInfo.ShopLayer,                              // 商店层级
			ShopGoodsInfo:     taptap.ConvertListStruct2Str(rsp.ShopInfo.GoodsIds), // 商品ids []*pb.ShopGoodsInfo
			ExpireTimeSec:     rsp.ShopInfo.ExpireTimeSec,                          // 过期时间戳
		}
		taptap.WriteDataLog(taptap.LogType_shop_info, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	return rsp, nil, 0
}

// 购买
func (h *ShopHandler) ShopBuyReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err error
		rsp = &pb.LS2C_ShopBuyRes{}
	)

	_, uid, _ := in.MsgId, in.UserId, in.Data
	var req pb.C2LS_ShopBuyReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	if req.BuyNum <= 0 {
		return nil, fmt.Errorf("无效的购买数量, req=%+v", req.BuyNum), int32(pb.ErrorCode_ParamError)
	}

	shopInfo, err := h.getShopInfo(req.ShopId)
	if err != nil {
		return nil,
			fmt.Errorf("商店过期或不存在, shopId=%d, err:%+v", req.ShopId, err),
			int32(pb.ErrorCode_InternalError)
	}

	for _, eachGoods := range shopInfo.GoodsIds {
		if eachGoods.GoodsId != req.GoodsId {
			continue
		}

		var goodsCfg *meta.GameShopPkgGoodsMeta
		// goodsCfg := data.GetShopGoodsMgr().GetById(int32(eachGoods.GoodsId))

		if goodsCfg.Limit != -1 { // 配置成-1, 表示不做限制
			if int32(eachGoods.HadBuyCount) >= goodsCfg.Limit {
				return nil, fmt.Errorf("购买次数不足, shopId=%d, goodsId=%d", req.ShopId, goodsCfg.Id), int32(pb.ErrorCode_Shop_goods_buy_count_limit)
			}

			if int32(eachGoods.HadBuyCount+req.GetBuyNum()) > goodsCfg.Limit {
				return nil, fmt.Errorf("库存不足, shopId=%d, goodsId=%d", req.ShopId, goodsCfg.Id), int32(pb.ErrorCode_Shop_goods_not_enough)
			}
		}

		// if goodsCfg.GetLayer() != shopInfo.ShopLayer {
		// 	return nil, fmt.Errorf("商品层级不匹配, shopId=%d, goodsId=%d", req.ShopId, goodsCfg.GetId()), int32(pb.ErrorCode_Shop_goods_invalid_layer)
		// }

		dropChange, err, errorCode := h.doBuy(uid, int32(req.BuyNum), goodsCfg, h.actor.comData)
		if err != nil {
			return nil, err, int32(errorCode)
		}

		eachGoods.HadBuyCount += req.GetBuyNum()

		rsp.GoodsInfo = eachGoods
		rsp.DropChange = dropChange

		break
	}

	// 尝试更新商店层级
	h.tryUpdateShopLayer(shopInfo)

	// 持久化
	err = h.SaveDB()
	if err != nil {
		h.Errorf("ShopBuyReq SaveShopData2DB 报错, err:%+v", err)
	}

	rsp.ShopId = req.ShopId
	rsp.GoodsId = req.GoodsId
	rsp.CommonData = h.actor.comData.FixDownComData()

	// 埋点
	// threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.ShopBuy{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_shop_buy, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		ShopId:         rsp.ShopId,
	//		GoodsId:        rsp.GoodsId,
	//		GoodsInfo:      lilith.ConvertStruct2Str(rsp.GoodsInfo),
	//	})
	// })
	threading.RunSafe(func() {
		e := &taptap.ShopBuy{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			ShopId:            rsp.ShopId,
			GoodsId:           rsp.GoodsId,
			GoodsInfo:         taptap.ConvertStruct2Str(rsp.GoodsInfo),
		}
		taptap.WriteDataLog(taptap.LogType_shop_buy, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	return rsp, nil, 0
}

func (h *ShopHandler) ShopManualRefreshReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err error
	)

	var req pb.C2LS_ShopManualRefreshReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	var shopCfg *meta.GameShopPkgShopMeta
	// shopCfg := data.GetShopMgr().GetById(req.ShopId)
	if shopCfg == nil {
		return nil, fmt.Errorf("没有对应的配置, shopId=%d", req.ShopId), int32(pb.ErrorCode_Shop_not_exist)
	}

	if 1 != shopCfg.Isauto {
		return nil, fmt.Errorf("该商店不支持手动刷新, shopId=%d", req.ShopId), int32(pb.ErrorCode_Shop_cannot_refresh_by_manual)
	}

	shopInfo, err := h.getShopInfo(req.ShopId)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_Shop_not_exist)
	}

	if shopCfg.RefreshLimit != -1 && // 配置为-1, 表示不限制
		shopInfo.ManualRefreshCount >= shopCfg.RefreshLimit {

		return nil, fmt.Errorf("该商店手动刷新达到上限, shopId=%d", req.ShopId), int32(pb.ErrorCode_Shop_manual_refresh_limit)
	}

	// 消耗物品
	if !GetConsumeMgr(h.actor).CheckKeyValEnough(shopCfg.RefreshCost) {
		return nil, fmt.Errorf("消耗不足, %v", shopCfg.RefreshCost), int32(pb.ErrorCode_NotEnoughItem)
	}
	err = GetConsumeMgr(h.actor).ConsumeKeyValList(shopCfg.RefreshCost, h.actor.comData, common.CR_Shop_Manual_refresh)
	if err != nil {
		return nil, fmt.Errorf("消耗报错, err:%v", err.Error()), int32(pb.ErrorCode_InternalError)
	}

	// 创建商品
	shopGoods := h.createShopGoods(req.ShopId, shopInfo.ShopLayer)
	shopInfo.GoodsIds = shopGoods

	// 累计次数
	shopInfo.ManualRefreshCount += 1

	// 持久化
	err = h.SaveDB()
	if err != nil {
		h.Errorf(err.Error())
		return nil, err, int32(pb.ErrorCode_SaveDBError)
	}

	rsp := &pb.LS2C_ShopManualRefreshRes{
		ShopInfo:   shopInfo,
		CommonData: h.actor.comData.FixDownComData(),
	}

	return rsp, nil, int32(pb.ErrorCode_Success)
}

func (h *ShopHandler) saveShopData2DB(shopInfo *pb.ShopInfo) error {
	shopData := h.actor.GetShopData()
	if shopData.ShopInfos == nil {
		shopData.ShopInfos = make(map[int32]*pb.ShopInfo, 0)
	}
	shopData.ShopInfos[shopInfo.ShopId] = shopInfo

	return h.SaveDB()

}

// 尝试更新商店层级
func (h *ShopHandler) tryUpdateShopLayer(shopInfo *pb.ShopInfo) {
	var (
		err                error
		nextLayer          int32
		nextLayerGoodExist = false // 下一层是否有商品
	)

	// 检查是否全部商品卖完了
	// for _, each := range shopInfo.GoodsIds {
	// 	goodsCfg := data.GetShopGoodsMgr().GetById(int32(each.GoodsId))
	//
	// 	if int32(shopInfo.ShopLayer) != goodsCfg.GetLayer() {
	// 		// 只关心当前层
	// 		continue
	// 	}
	//
	// 	if goodsCfg.Limit != -1 && // 配置成-1, 表示不限制
	// 		int32(each.HadBuyCount) != goodsCfg.Limit {
	// 		return
	// 	}
	// }

	nextLayer = shopInfo.ShopLayer + 1

	// data.GetShopGoodsMgr().Foreach(func(cfg *data.ShopGoodsCfg) bool {
	// 	if cfg.GetShopId() == shopInfo.GetShopId() && cfg.GetLayer() == nextLayer {
	// 		nextLayerGoodExist = true
	// 		return false // 停止查找
	// 	}
	//
	// 	return true
	// }, false)

	if nextLayerGoodExist {
		shopInfo.ShopLayer = nextLayer
		err = h.SaveDB()
		// err = h.actor.SaveShopData2DB(h.actor)
		if err != nil {
			h.Errorf("ShopBuyReq SaveShopData2DB 报错, err:%+v", err)
		}
	}
}

// 创建商店信息
// @param shopId	商店id
// @param layerId	层级id
// @param lastExpireTimeSec	上次过期时间
func (h *ShopHandler) createShopInfo(shopId, layerIdx int32, lastExpireTimeSec int64) (error, *pb.ShopInfo) {
	var (
		shopGoods = make([]*pb.ShopGoodsInfo, 0)
	)

	// 创建商品(所有层)
	var shopCfg *meta.GameShopPkgShopMeta
	// shopCfg := data.GetShopMgr().GetById(shopId)
	if shopCfg == nil {
		return errors.New(fmt.Sprintf("不存在的商店id, shopId=%d", shopId)), nil
	}

	for _, layer := range shopCfg.LayerList {
		eachGoods := h.createShopGoods(shopId, layer)
		shopGoods = append(shopGoods, eachGoods...)
	}

	// 下次刷新的时间
	nextRefreshTime, err := h.getShopNextRefreshTime(shopId, lastExpireTimeSec)
	if err != nil {
		return err, nil
	}

	shopInfo := &pb.ShopInfo{
		ShopId:             shopId,
		ShopLayer:          1, // 初始层级
		GoodsIds:           shopGoods,
		ExpireTimeSec:      nextRefreshTime,
		ManualRefreshCount: 0, // 手动刷新次数
	}
	// 下发数据
	h.actor.comData.AddShopInfo(shopInfo)

	h.Debugf("玩家:%v, 创建商店:%v", h.actor.ID(), shopInfo)

	return nil, shopInfo
}

// 获取商店下次刷新时间
func (h *ShopHandler) getShopNextRefreshTime(shopId int32, lastExpireTimeSec int64) (int64, error) {
	var (
		now = time.Now()
	)

	if lastExpireTimeSec == MAX_REFRESH_SEC || lastExpireTimeSec > now.Unix() {
		// 还未过期
		return lastExpireTimeSec, nil
	}

	var shopCfg *meta.GameShopPkgShopMeta
	// shopCfg := data.GetShopMgr().GetById(shopId)
	if shopCfg == nil {
		return 0, fmt.Errorf("没有对应的商店配置, shopId:%d", shopId)
	}

	switch shopCfg.RefreshType {
	case 0: // 不刷新
		return MAX_REFRESH_SEC, nil
	case 1: // 每月一号刷新
		next1MonthTime, err := common.GetNextNMonth1RefreshTime(now, 1)
		if err != nil {
			return 0, err
		}
		return next1MonthTime.Unix(), nil

	case 2: // 每周一刷新
		next1WeekTime, err := common.GetNextNWeekMonday(now, 1)
		if err != nil {
			return 0, err
		}
		return next1WeekTime.Unix(), nil

	case 3: // 每日刷新
		nextDailyRefreshTime := common.GetNextDailyRefreshTime()
		return nextDailyRefreshTime, nil

	case 10: // 按照配置周期刷新
		var lastRefreshTimeSec int64 = 0 // 上次刷新时间戳

		if lastExpireTimeSec == 0 {
			// 没有上次过期时间, 以当天5点开始
			lastRefreshTimeSec = common.GetTodayRefreshTime(now).Unix()

		} else {
			// 上次过期时间开始算
			lastRefreshTimeSec = lastExpireTimeSec
		}

		cycleTimeSec := int64(shopCfg.RefreshTime * 60 * 60) // 刷新周期时间(s)
		cycle := (now.Unix() - lastRefreshTimeSec) / cycleTimeSec
		nextDailyRefreshTime := lastRefreshTimeSec + cycleTimeSec*(cycle+1)

		return nextDailyRefreshTime, nil
		// for {
		//	lastRefreshTimeSec += cycleTimeSec
		//	if lastRefreshTimeSec > now.Unix() {
		//		break
		//	}
		// }

	default:
		return 0, fmt.Errorf("没有对应的刷新类型, shopId:%d, refreshType:%d", shopId, shopCfg.RefreshType)
	}

	// return 0, fmt.Errorf("没有对应的刷新类型, shopId:%d, refreshType:%d", shopId, shopCfg.RefreshType)
}

func (h *ShopHandler) doBuy(uid string, buyNum int32, goodsCfg *meta.GameShopPkgGoodsMeta, commonData *clidto.Comdata) (*pb.DropChange, error, pb.ErrorCode) {
	costMap := make(map[int32]int32)
	costMap[goodsCfg.Price.Key] = goodsCfg.Price.Val * buyNum

	if !GetConsumeMgr(h.actor).CheckMapEnough(costMap) {
		return nil, fmt.Errorf("道具不足, uid=%d", uid), pb.ErrorCode_CurrencyNotEnough
	}

	if GetDropMgr(h.actor).CheckLimit(goodsCfg.ItemId.Key, goodsCfg.ItemId.Val*buyNum) {
		return nil, fmt.Errorf("已达持有最大数量, uid=%s", uid), pb.ErrorCode_Shop_item_limit
	}

	err := GetConsumeMgr(h.actor).ConsumeList(costMap, commonData, common.CR_Shop_buy)
	if err != nil {
		return nil, fmt.Errorf("扣除货币报错, uid=%d", uid), pb.ErrorCode_InternalError
	}

	// changeItem, err := GetConsumeMgr(h.actor).ConsumeList(costMap, common.CR_Shop_buy)
	// if err != nil {
	//	return fmt.Errorf("扣除道具报错, uid=%d", uid), pb.ErrorCode_InternalError
	// }

	// 判断商品是否超过背包上限
	// rewardItemCfg := data.GetItemMgr().GetById(goodsCfg.ItemId.Key)
	// hadItemNum := h.actor.BagHandler.GetItemNum(goodsCfg.ItemId.Key)
	// if int64(hadItemNum+goodsCfg.ItemId.Val) > rewardItemCfg.NumLimit {
	//	return nil, fmt.Errorf("已达持有最大数量, uid=%s", uid), pb.ErrorCode_Shop_item_limit
	// }

	dropChange, err := GetDropMgr(h.actor).DropList2(map[int32]int32{goodsCfg.ItemId.Key: goodsCfg.ItemId.Val * buyNum}, true, nil, commonData, common.CR_Shop_buy)
	if err != nil {
		return nil, fmt.Errorf("奖励道具报错, uid=%s", uid), pb.ErrorCode_InternalError
	}

	return dropChange, nil, pb.ErrorCode_Success
}

// 从db获取商店信息
func (h *ShopHandler) getShopInfo(shopId int32) (*pb.ShopInfo, error) {
	var (
		err        error
		dbShopData *pb.LS2DB_ShopData
		shopInfo   *pb.ShopInfo
	)
	dbShopData = h.actor.GetShopData()
	if dbShopData == nil {
		return nil, fmt.Errorf("商店不存在, shopId=%d", shopId)
	}

	if _shopInfo, ok := dbShopData.ShopInfos[shopId]; !ok { // 还没有该商店
		err, shopInfo = h.createShopInfo(shopId, DEFAULT_SHOP_LAYER_IDX, 0)
		if err != nil {
			return nil, err
		}

		err = h.saveShopData2DB(shopInfo)
		if err != nil {
			return nil, err
		}

	} else {
		if _shopInfo.ExpireTimeSec != MAX_REFRESH_SEC && _shopInfo.ExpireTimeSec <= time.Now().Unix() {
			// 商店已经过期了
			err, shopInfo = h.createShopInfo(shopId, DEFAULT_SHOP_LAYER_IDX, _shopInfo.ExpireTimeSec)
			if err != nil {
				return nil, err
			}

			err = h.saveShopData2DB(shopInfo)
			if err != nil {
				return nil, err
			}

		} else {
			shopInfo = _shopInfo
		}
	}

	return shopInfo, nil
}

// 创建商店物品
func (h *ShopHandler) createShopGoods(shopId int32, layerIdx int32) []*pb.ShopGoodsInfo {
	var (
		// layer      = uint32(1) // 初始层级
		// tempGoodsCfgs = make([]*data.ShopGoodsCfg, 0)
		goodsInfos = make([]*pb.ShopGoodsInfo, 0)
	)

	// data.GetShopGoodsMgr().Foreach(func(cfg *data.ShopGoodsCfg) bool {
	// 	if cfg.GetShopId() == shopId && cfg.Layer == layerIdx {
	// 		tempGoodsCfgs = append(tempGoodsCfgs, cfg)
	// 	}
	// 	return true
	// }, false)

	posGoodsMap := make(map[int32][]*meta.GameShopPkgGoodsMeta)
	// for _, cfg := range tempGoodsCfgs {
	// 	if _, ok := posGoodsMap[cfg.PosIdx]; !ok {
	// 		posGoodsMap[cfg.PosIdx] = make([]*data.ShopGoodsCfg, 0)
	// 	}
	//
	// 	posGoodsMap[cfg.PosIdx] = append(posGoodsMap[cfg.PosIdx], cfg)
	// }

	finalGoodsCfg := make([]*meta.GameShopPkgGoodsMeta, 0)
	for _, eachPosCfgs := range posGoodsMap {
		if len(eachPosCfgs) <= 0 {
			// 没有商品数据
			continue
		}

		// 有多条数据, 需要根据权重随机
		// weightVos := make([]*data.WeightVo, 0)
		// for _, cfg := range eachPosCfgs {
		// 	wVo := &data.WeightVo{
		// 		Weight: cfg.RefreshWeight,
		// 		VoId:   cfg.Id,
		// 	}
		//
		// 	weightVos = append(weightVos, wVo)
		// }
		// vo, err := datahelper.RandomByWeightVo(weightVos)
		// if err != nil {
		// 	h.Errorf("随机商店物品报错, shopId=%d, layerIdx=%d posIdx=%d, err:%v", shopId, layerIdx, posIdx, err.Error())
		// 	continue
		// }

		var goodsCfg *meta.GameShopPkgGoodsMeta
		// goodsCfg := data.GetShopGoodsMgr().GetById(vo.GetVoId())
		if goodsCfg == nil {
			continue
		}

		finalGoodsCfg = append(finalGoodsCfg, goodsCfg)
	}

	for _, cfg := range finalGoodsCfg {
		goodsInfo := &pb.ShopGoodsInfo{
			GoodsId:     cfg.Id,
			HadBuyCount: 0,
		}

		goodsInfos = append(goodsInfos, goodsInfo)
	}

	h.Debugf("创建商品, shopId=%d, layerIdx=%d, shopGoods:%v", shopId, layerIdx, goodsInfos)

	return goodsInfos
}

func getShopIds(showType int32) []int32 {
	var (
		shopIds = make([]int32, 0)
	)

	// data.GetShopMgr().Foreach(func(cfg *data.ShopCfg) bool {
	// 	if cfg.ShowType != showType {
	// 		return true
	// 	}
	//
	// 	shopIds = append(shopIds, cfg.GetId())
	// 	return true
	// }, false)

	return shopIds
}
