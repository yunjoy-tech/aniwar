package useractor

import (
	"context"
	"fmt"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datalog/taptap"
	"time"

	"gitlab.musadisca-games.com/wangxw/musae/framework/threading"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/db"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"

	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/pb"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"google.golang.org/protobuf/proto"
)

type TroopHandler struct {
	*UABaseHandler
}

func NewTroopHandler(actor *UserActor) *TroopHandler {
	h := &TroopHandler{UABaseHandler: NewUABaseHandler(actor, "TroopHandler")}
	h.ChildHandler = h

	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_CardTroopListReq), h.TroopListReq)
	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_CardTroopOperateReq), h.TroopOperateReq)
	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_CardTroopFoodLogReq), h.TroopFoodLogReq)
	return h
}

// Init 初始化模块数据
func (h *TroopHandler) Init() error {
	// 初始化
	h.actor.Data.Troops = &pb.PCardTroopsInfo{
		Createtime: time.Now().Unix(),
		Troop:      make(map[int32]*pb.PServerCardTroopInfo),
	}

	// 保存
	if err := h.SaveDB(true); err != nil {
		return err
	}

	h.Debug("init troops data success. player: ", h.actor.ID())
	return nil
}

func (h *TroopHandler) EnterGame() error {
	return nil
}

func (h *TroopHandler) DailyRefresh() error {
	return nil
}

func (h *TroopHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*pb.PCardTroopsInfo); ok {
		h.actor.Data.Troops = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *TroopHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyUserCardTroop(h.actor.ID()), h.actor.Data.Troops
}

func (h *TroopHandler) TroopListReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {

	var req pb.C2LS_CardTroopListReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	// 参数check
	if !isValidCardTroopType(req.TroopType) {
		return nil, fmt.Errorf("invalia param"), int32(pb.ErrorCode_InvalidParam)
	}

	troopInfo, err := h.getTroopTypeInfo(req.TroopType)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InvalidParam)
	}

	// 返回消息
	rsp := &pb.LS2C_CardTroopListRes{
		Troop: &pb.PClientCardTroopInfo{
			TroopType:  troopInfo.TroopType,
			Troop:      convertList(troopInfo.Troop),
			UseTroopId: troopInfo.UseTroopId,
			Foods:      troopInfo.Foods,
		},
	}

	return rsp, nil, 0
}

// TroopFoodLogReq 编队食物记录
func (h *TroopHandler) TroopFoodLogReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {

	var req pb.C2LS_CardTroopFoodLogReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	// check
	if !isValidCardTroopType(req.TroopType) || !isValidFood(req.Foods) {
		h.Debugf("TroopFoodLogReq invalid param: %d %+v", req.TroopType, req.Foods)
		return nil, err, int32(pb.ErrorCode_InvalidParam)
	}

	// 获取玩法数据
	troopInfo, err := h.getTroopTypeInfo(req.TroopType)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	// 埋点log
	// threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.FoodOperate{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_FoodOperate, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		TroopType:      req.TroopType,
	//		BeforeFoods:    lilith.ConvertList2Str(troopInfo.Foods),
	//		AfterFoods:     lilith.ConvertList2Str(req.Foods),
	//	})
	// })
	threading.RunSafe(func() {
		e := &taptap.FoodOperate{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			TroopType:         req.TroopType,
			BeforeFoods:       taptap.ConvertList2Str(troopInfo.Foods),
			AfterFoods:        taptap.ConvertList2Str(req.Foods),
		}
		taptap.WriteDataLog(taptap.LogType_FoodOperate, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	troopInfo.Foods = req.Foods
	if err = h.SaveDB(); err != nil {
		h.Error("TroopFoodLogReq save troop to db failed. err: ", err)
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	// 返回消息
	rsp := &pb.LS2C_CardTroopFoodLogRes{
		TroopType: req.TroopType,
		Foods:     req.Foods,
	}
	return rsp, nil, 0
}

func (h *TroopHandler) TroopOperateReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {

	var req pb.C2LS_CardTroopOperateReq
	err := in.UnmarshalData(&req)
	if err != nil {
		h.Error("TroopOperateReq Unmarshal err: ", err)
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	err, code := h.CardTroopOperate(pb.CardTroopType(req.TroopType), req.TroopId, pb.CardTroopSubType(req.SubType), req.Positions)
	if err != nil {
		return nil, err, int32(code)
	}

	info, err := h.getTroopInfo(req.TroopType, req.TroopId)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InvalidParam)
	}
	return &pb.LS2C_CardTroopOperateRes{Info: convert(info)}, nil, 0
}

// CardTroopOperate
//
//	@Description: 卡牌编队接口
//	@receiver h
//	@param troopType 详见pb.CardTroopType
//	@param troopId 编队id
//	@param subType 详见pb.CardTroopSubType
//	@param position 卡牌列表
//	@return error
//	@return pb.ErrorCode
func (h *TroopHandler) CardTroopOperate(troopType pb.CardTroopType, troopId int32, subType pb.CardTroopSubType, position []int32) (error, pb.ErrorCode) {
	errorCode := h.checkTroopOperate(int32(troopType), troopId, position, int32(subType))
	if errorCode != pb.ErrorCode_Success {
		return fmt.Errorf("card troop param check failed"), errorCode
	}

	errorCode = h.handleTroopOperate(int32(troopType), troopId, position, subType)
	if errorCode != pb.ErrorCode_Success {
		return fmt.Errorf("handle troop operate failed"), errorCode
	}

	return nil, pb.ErrorCode_Success
}

// 编队操作参数校验
func (h *TroopHandler) checkTroopOperate(typ, troopId int32, positions []int32, subType int32) pb.ErrorCode {
	errCode := h.CheckTroopTypAndId(typ, troopId)
	if errCode != pb.ErrorCode_Success {
		return errCode
	}
	// 队伍数据check
	temp := make(map[int32]int32)
	for index, cardId := range positions {
		// 位置check
		if !isValidCardPos(int32(index + 1)) {
			return pb.ErrorCode_InvalidTroopPosition
		}

		// 卡牌是否拥有
		if cardId != 0 && !h.actor.CardHandler.IsExistCard(uint32(cardId)) {
			return pb.ErrorCode_CardNotExist
		}
		// 重复元素
		if cardId != 0 {
			if _, ok := temp[cardId]; ok {
				return pb.ErrorCode_InvalidParam
			}
			temp[cardId] = 1
		}

		// 阵亡角色仍可以上阵
		// if typ == int32(pb.CardTroopType_CardTroopType_Normal) && subType == int32(pb.CardTroopSubType_Map_In) && cardId > 0 {
		//	card, _ := h.actor.CardHandler.GetCard(uint32(cardId))
		//	if card.Hp <= 0 {
		//		return pb.ErrorCode_Chapter_no_live_in_troop
		//	}
		// }
	}

	return pb.ErrorCode_Success
}

// 编队逻辑处理
func (h *TroopHandler) handleTroopOperate(typ, troopId int32, positions []int32, subType pb.CardTroopSubType) pb.ErrorCode {
	cardTroopInfo, err := h.getTroopInfo(typ, troopId)
	if err != nil {
		return pb.ErrorCode_InternalError
	}

	// 埋点log
	// threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.TroopOperate{
	//		CustomHeadInfo:  lilith.BuildCustomHeadInfo(lilith.LogType_TroopOperate, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		TroopType:       typ,
	//		TroopId:         troopId,
	//		BeforePositions: lilith.ConvertList2Str(cardTroopInfo.Card),
	//		AfterPositions:  lilith.ConvertList2Str(positions),
	//		SubType:         int32(subType),
	//	})
	// })
	threading.RunSafe(func() {
		e := &taptap.TroopOperate{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			TroopType:         typ,
			TroopId:           troopId,
			BeforePositions:   taptap.ConvertList2Str(cardTroopInfo.Card),
			AfterPositions:    taptap.ConvertList2Str(positions),
			SubType:           int32(subType),
		}
		taptap.WriteDataLog(taptap.LogType_TroopOperate, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	cardTroopInfo.Card = positions

	// 保存数据
	err = h.SaveDB()
	if err != nil {
		h.Error("save user card troop to db failed. err: ", err)
	}
	return pb.ErrorCode_Success
}

func isValidTroopId(id int32) bool {
	return id != 0 && excel.GetConfigMgr().GetCfg().PARTY_GAMEPLAY_LIMIT >= id
}

// 获取玩法信息
func (h *TroopHandler) getTroopTypeInfo(typ int32) (*pb.PServerCardTroopInfo, error) {
	// 初始化
	troopData := h.actor.GetTroopData()
	if _, ok := troopData.Troop[typ]; !ok {
		troopData.Troop[typ] = &pb.PServerCardTroopInfo{
			TroopType:  typ,
			Troop:      make(map[int32]*pb.ServerCardTroopInfo),
			UseTroopId: 0,
			Foods:      make([]int32, 0),
		}

		if err := h.SaveDB(); err != nil {
			return nil, err
		}
	}

	return troopData.Troop[typ], nil
}

// 找对应的troop数据
func (h *TroopHandler) getTroopInfo(typ, troopId int32) (*pb.ServerCardTroopInfo, error) {
	troopTypeInfo, err := h.getTroopTypeInfo(typ)
	if err != nil {
		return nil, err
	}

	if _, ok := troopTypeInfo.Troop[troopId]; !ok {
		if h.isFull(typ) {
			return nil, err
		}
		// 初始化
		troop := &pb.ServerCardTroopInfo{
			TroopId:   troopId,
			TroopName: "",
			Card:      make([]int32, 0),
		}

		if troopTypeInfo.Troop == nil {
			troopTypeInfo.Troop = make(map[int32]*pb.ServerCardTroopInfo)
		}
		troopTypeInfo.Troop[troopId] = troop
		err = h.SaveDB()
		if err != nil {
			return nil, err
		}
	}

	return troopTypeInfo.Troop[troopId], nil
}

func (h *TroopHandler) isFull(typ int32) bool {
	return h.getCardTroopSize(typ) >= excel.GetConfigMgr().GetCfg().PARTY_GAMEPLAY_LIMIT
}

// 指定玩法的队伍总数量
func (h *TroopHandler) getCardTroopSize(typ int32) int32 {
	data := h.actor.GetTroopData()
	troopInfo := data.Troop[typ]
	if troopInfo == nil {
		return 0
	}
	return int32(len(troopInfo.Troop))
}

func isValidCardPos(pos int32) bool {
	return pos > int32(pb.CardTroopPos_CardTroopPos_None) && int32(pb.CardTroopPos_CardTroopPos_Max) > pos
}

func isValidCardTroopType(typ int32) bool {
	return typ > int32(pb.CardTroopType_CardTroopType_None) && int32(pb.CardTroopType_CardTroopType_Max) > typ
}

func isValidFood(food []int32) bool {
	if len(food) > int(excel.GetConfigMgr().GetCfg().BATTLE_FOOD_LIMIT) {
		return false
	}
	// 类型check
	temp := make(map[int32]int32)
	for _, itemId := range food {
		if itemId == 0 {
			continue
		}
		cfg := excel.GetItemMgr().GetById(itemId)
		if cfg == nil {
			return false
		}

		if cfg.GetType() != int32(pb.ItemType_Food) {
			return false
		}
		// 重复参数
		if _, ok := temp[itemId]; ok {
			return false
		}
		temp[itemId] = 1
	}

	return true
}

func convert(info *pb.ServerCardTroopInfo) *pb.ClientCardTroopInfo {
	return &pb.ClientCardTroopInfo{
		TroopId:   info.TroopId,
		TroopName: info.TroopName,
		Card:      info.Card,
	}
}

func convertList(infos map[int32]*pb.ServerCardTroopInfo) []*pb.ClientCardTroopInfo {
	troops := make([]*pb.ClientCardTroopInfo, 0)
	for _, v := range infos {
		troops = append(troops, convert(v))
	}

	return troops
}

// GetTroopFoodLog 获取指定玩法的食物配置数据
func (h *TroopHandler) GetTroopFoodLog(typ int32) []int32 {
	if !isValidCardTroopType(typ) {
		return []int32{}
	}

	troopInfo, err := h.getTroopTypeInfo(typ)
	if err != nil {
		return []int32{}
	}
	return troopInfo.Foods
}

// SaveUseTroopId 记录玩法最后使用的troopId
func (h *TroopHandler) SaveUseTroopId(typ int32, troopId int32) {
	if errCode := h.CheckTroopTypAndId(typ, troopId); errCode != pb.ErrorCode_Success {
		h.Errorf("SaveUseTroopId param check failed. type %d, troopId %d", typ, troopId)
		return
	}

	troopInfo, err := h.getTroopTypeInfo(typ)
	if err != nil {
		h.Error("SaveUseTroopId failed. err: ", err)
		return
	}

	troopInfo.UseTroopId = troopId
	err = h.SaveDB()
	if err != nil {
		h.Error("SaveUseTroopId save troop to db failed. err: ", err)
	}

	h.Debugf("SaveUseTroopId success. type %d troopId %d", typ, troopId)
}

// GetTroopCardIds 按照位置升序获取card ids
func (h *TroopHandler) GetTroopCardIds(typ, troopId int32) []int32 {
	// 参数check
	if errCode := h.CheckTroopTypAndId(typ, troopId); errCode != pb.ErrorCode_Success {
		return []int32{}
	}
	info, err := h.getTroopInfo(typ, troopId)
	if err != nil {
		return []int32{}
	}
	return info.GetCard()
}

// CheckTroopTypAndId 检查输入参数合法性
func (h *TroopHandler) CheckTroopTypAndId(typ, troopId int32) pb.ErrorCode {
	// 玩法类型check
	if !isValidCardTroopType(typ) {
		h.Errorf("CheckTroopTypAndId check Type err type :%v, ", typ)
		return pb.ErrorCode_InvalidCardTroopType
	}
	// 队伍id check
	if !isValidTroopId(troopId) {
		h.Errorf("CheckTroopTypAndId check failed, troopId: %v, cfg: %+v ", troopId, excel.GetConfigMgr().GetCfg())
		return pb.ErrorCode_InvalidCardTroopId
	}

	return pb.ErrorCode_Success
}
