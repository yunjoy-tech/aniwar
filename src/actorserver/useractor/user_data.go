package useractor

import (
	"time"

	"github.com/yunjoy-tech/aniwar/src/proto/pb"
	"github.com/yunjoy-tech/musae/service"
)

/*

user actor data struct

*/

type UserData struct {
	Data      *pb.PlayerData
	Account   *pb.UserData
	OrderData *pb.OrderData
}

// // 全量加载用户数据
// func (s *UserActor) loadDBDataByDBType() error {
//	// 从db中加载数据
//	for _, handler := range s.handlersMap {
//		_, err := loadPlayerData1(s, handler)
//		if err != nil {
//			if errors.Is(err, service.DB_ERROR_NOT_EXIST) {
//
//				//msgDescriptor := dbData.ProtoReflect().Descriptor().Fields().ByName("createtime")
//				////dbData.ProtoReflect().Range(func(descriptor protoreflect.FieldDescriptor, value protoreflect.Value) bool {
//				////	if descriptor.TextName() == "createtime" {
//				////		value.
//				////		dbData.ProtoReflect().Set(descriptor,)
//				////	}
//				////	return false
//				////})
//				//if msgDescriptor == nil {
//				//	logger.Errorf("没有 %s 字段", "createtime")
//				//} else if msgDescriptor.IsPlaceholder() {
//				//
//				//} else {
//				//
//				//}
//				////handler.createBound()
//				err = handler.SetPlayerDataBound()
//				if err != nil {
//					return err
//				}
//				err = handler.SaveDB()
//				if err != nil {
//					return err
//				}
//			} else {
//				return err
//			}
//		}
//	}
//
//	// 初始化玩家数据
//	for _, handler := range s.handlersMap {
//		handler.InitPlayerData()
//	}
//
//	return nil
// }

func (u *UserActor) loadAllData(bMini ...bool) error {
	var (
		err    error
		startT = time.Now()
	)

	mongoDBs := []service.MongoDbType{
		service.MongoDbType_MongoAccount, // 账号db
		service.MongoDbType_MongoGame,    // 游戏db
	}

	for _, eachDB := range mongoDBs {
		if err = u.loadDBDataByDBType(eachDB, bMini...); err != nil {
			return err
		}
	}

	u.WarnDelayf(time.Since(startT).Milliseconds(), "")

	return nil
}

// // 加载账号数据
// func (u *UserActor) loadAccountData() error {
//	if dbData, err := loadPlayerData(u, service.MongoDbType_MongoAccount, db.KeyAccountInfo(u.GetUID()), u.Account); err != nil {
//		//if errors.Is(err, service.DB_ERROR_NOT_EXIST) {
//		//	if err = u.AccountHandler.Init(); err != nil {
//		//		return err
//		//	}
//		//} else {
//		//	return err
//		//}
//		return err // 在actor中account中必须有值(login-server中已经有写入数据了)
//	} else {
//		u.Account = dbData
//	}
//
//	if u.Account == nil {
//		return errors.New("after loadDBDataByDBType, account is still nil")
//	}
//
//	return nil
// }

// 全量加载用户数据
func (u *UserActor) loadDBDataByDBType(dbType service.MongoDbType, bMini ...bool) error {
	var isMiniMode bool
	if len(bMini) != 0 && bMini[0] {
		isMiniMode = true
	}
	for _, handler := range u.HandlersMap[dbType] {
		dbTable, dbKey, dbVal := handler.DBTable()
		if isMiniMode {
			if handler.IsSupportMini() {
				if err := handler.LoadDBData(dbTable, dbKey, dbVal); err != nil {
					return err
				}
			}
		} else {
			if err := handler.LoadDBData(dbTable, dbKey, dbVal); err != nil {
				return err
			}
		}

	}
	return nil
}

func (x *UserData) GetUserData() *pb.PServerRoleBaseInfo {
	data := x.Data.Base
	if data == nil {
		return nil
	}
	if data.Heads == nil {
		data.Heads = make([]int32, 0)
	}
	if data.NewHeads == nil {
		data.NewHeads = make([]int32, 0)
	}
	return data
}

func (x *UserData) GetUserCardData() *pb.PCardData {
	cards := x.Data.Cards
	if cards.Card == nil {
		cards.Card = make(map[uint32]*pb.CardData)
	}

	return cards
}

func (x *UserData) GetTroopData() *pb.PCardTroopsInfo {
	data := x.Data.Troops
	if data.Troop == nil {
		data.Troop = make(map[int32]*pb.PServerCardTroopInfo)
	}

	return data
}

func (x *UserData) GetCampData() *pb.PPlayerCampBlob {
	data := x.Data.Camp
	if data.DecorationBuilding == nil {
		data.DecorationBuilding = make(map[int64]*pb.PPlayerCampDecorationBuilding)
	}
	if data.BuildingUnlockList == nil {
		data.BuildingUnlockList = make(map[int32]int32)
	}
	if data.Camp == nil {
		data.Camp = make(map[int32]*pb.PPlayerCampServerCamp)
	}
	return data
}

func (x *UserData) GetCurrencyData() *pb.PCurrencyInfo {
	data := x.Data.Currency
	if data.Currencyx == nil {
		data.Currencyx = make(map[int32]*pb.CurrencyItem)
	}
	return data
}

func (x *UserData) GetTutorialData() *pb.PPlayerBeginnerTutorialBlob {
	data := x.Data.Tutorial
	if data.FinishMasterTutorial == nil {
		data.FinishMasterTutorial = make([]*pb.PPlayerDBBeginnerTutorialBlob, 0)
	}
	if data.FinishFunctionTutorial == nil {
		data.FinishFunctionTutorial = make([]*pb.PPlayerDBBeginnerTutorialBlob, 0)
	}
	return data
}

func (x *UserData) GetPoolsData() *pb.PServerCardPoolInfos {
	data := x.Data.Pools
	if data.TypeInfos == nil {
		data.TypeInfos = make(map[int32]*pb.PServerCardPoolType)
	}
	if data.Newbie == nil {
		data.Newbie = &pb.PNewbiePoolInfo{
			Select:  0,
			Results: make([]*pb.PNewbiePoolLog, 0),
		}
	}
	return data
}

func (x *UserData) GetCampPoolsData() *pb.PServerCampPoolInfos {
	data := x.Data.CampPools
	if data.TypeInfos == nil {
		data.TypeInfos = make(map[int32]*pb.PServerCampPoolType)
	}
	return data
}

func (x *UserData) GetHandBookData() *pb.PHandbookInfo {
	data := x.Data.Handbooks
	if data.HandBookItem == nil {
		data.HandBookItem = make(map[uint32]*pb.ServerHandBookItem, 0)
	}
	return data
}

func (x *UserData) GetQuestionData() *pb.PUserQuestions {
	data := x.Data.Question
	if data.Questions == nil {
		data.Questions = make(map[string]*pb.PQuestion, 0)
	}
	return data
}

func (x *UserData) GetLevelsData() *pb.LS2DB_LevelInfos {
	levelsData := x.Data.LevelsData
	if levelsData.LevelInfos == nil {
		levelsData.LevelInfos = make(map[int32]*pb.LS2DB_LevelInfo)
	}
	if levelsData.PLevelSummary == nil {
		levelsData.PLevelSummary = &pb.PServerLevelSummary{}
	}
	if levelsData.PLevelSummary.MonsterTicketInfoMap == nil {
		levelsData.PLevelSummary.MonsterTicketInfoMap = make(map[int32]*pb.LevelMonsterTicketInfo, 0)
	}
	if levelsData.PLevelSummary.LevelSummaryMap == nil {
		levelsData.PLevelSummary.LevelSummaryMap = make(map[int32]*pb.LevelSummary, 0)
	}

	if levelsData.FinishedOnceEvents == nil {
		levelsData.FinishedOnceEvents = make(map[int32]*pb.FinishedOnceEvent)
	}

	return levelsData
}

func (x *UserData) GetStoryFlagData() *pb.LS2DB_StoryFlagData {
	flagData := x.Data.StoryFlagData
	if flagData.Flags == nil {
		flagData.Flags = make(map[string]*pb.FlagInfo)
	}

	return flagData
}

func (x *UserData) GetShopData() *pb.LS2DB_ShopData {
	shopData := x.Data.ShopData
	if shopData.ShopInfos == nil {
		shopData.ShopInfos = make(map[int32]*pb.ShopInfo)
	}

	return shopData
}

func (x *UserData) GetEquipData() *pb.PEquipData {
	data := x.Data.EquipData
	if data.Equips == nil {
		data.Equips = make(map[uint64]*pb.PCommonEquipInfo)
	}
	return data
}

func (x *UserData) GetCardSkinData() *pb.PSkinData {
	data := x.Data.SkinData
	if data.Skins == nil {
		data.Skins = make(map[int32]*pb.CardSkinData)
	}
	return data
}

func (x *UserData) GetDutyData() *pb.PDutyData {
	data := x.Data.DutyData
	if data.DailyTask == nil {
		data.DailyTask = make(map[int32]*pb.TaskInfoItem)
	}
	if data.UnlockTag == nil {
		data.UnlockTag = make(map[int32]*pb.TaskInfoItem)
	}
	if data.Active == nil {
		data.Active = make(map[int32]*pb.ActiveInfoItem)
	}
	if data.WeeklyTask == nil {
		data.WeeklyTask = make(map[int32]*pb.TaskInfoItem)
	}
	return data
}

func (x *UserData) GetGuideTaskData() *pb.PGuideTaskData {
	data := x.Data.GuideTaskData
	if data.Tasks == nil {
		data.Tasks = make(map[int32]*pb.TaskInfoItem)
	}
	if data.Complete == nil {
		data.Complete = make(map[int32]int32)
	}
	return data
}

func (x *UserData) GetSignData() *pb.PSignData {
	data := x.Data.Sign
	if data.Sign == nil {
		data.Sign = make(map[int32]*pb.PCommonSignInfo)
	}
	return data
}

func (x *UserData) GetQuestData() *pb.PQuestData {
	data := x.Data.QuestData
	if data.CompleteQuests == nil {
		data.CompleteQuests = make([]int32, 0)
	}
	if data.OpenQuests == nil {
		data.OpenQuests = make(map[int32]*pb.PCommonQuestInfo)
	}
	return data
}

func (x *UserData) GetAchieveData() *pb.PUserAchieves {
	data := x.Data.AchieveData
	if data.SectionReceive == nil {
		data.SectionReceive = make(map[string]*pb.PAchieveReceive)
	}
	if data.Achieves == nil {
		data.Achieves = make(map[string]int32)
	}
	if data.GroupReceive == nil {
		data.GroupReceive = make(map[string]int32)
	}
	return data
}

func (x *UserData) GetCampaignData() *pb.PPlayerGeneralCampaign {
	data := x.Data.CampaignInfo
	if data.ServerGeneralCampaign == nil {
		data.ServerGeneralCampaign = &pb.PServerGeneralCampaign{
			ResCampaignCompleteCount: 0,
		}
	}

	if data.ServerGeneralCampaign.DailyCampaigns == nil {
		data.ServerGeneralCampaign.DailyCampaigns = make(map[int32]*pb.PDailyCampaignBase)
	}
	if data.ServerGeneralCampaign.ResCampaigns == nil {
		data.ServerGeneralCampaign.ResCampaigns = make(map[int32]*pb.PResourceCampaignBase)
	}

	return data
}

func (x *UserData) GetTrialData() *pb.PUserTrial {
	data := x.Data.TrialData
	if data.Trial == nil {
		data.Trial = make(map[int32]*pb.PTrialInfo)
	}
	return data
}

func (x *UserData) GetUseLimitData() *pb.PUseLimitInfo {
	data := x.Data.UseLimit
	if data.Items == nil {
		data.Items = make(map[int32]*pb.KeyValueItem)
	}
	return data
}

func (x *UserData) GetBlockWayData() *pb.PBlockWay {
	data := x.Data.BlockWayData
	if data.EventList == nil {
		data.EventList = make(map[int64]*pb.PBlockWayEvent)
	}
	if data.EventType == nil {
		data.EventType = make(map[int32]*pb.PBlockWayEventGroup)
	}
	return data
}

func (x *UserData) GetRoleDetailData() *pb.PServerRoleDetailInfo {
	data := x.Data.Detail
	if data.Common == nil {
		data.Common = &pb.PCommonRoleBaseInfo{}
	}
	if data.Cards == nil {
		data.Cards = make([]int32, 4, 4)
	}
	if data.Lifex == nil {
		data.Lifex = make(map[int32]int32)
	}
	return data
}

func (x *UserData) GetPlayerLevelData() *pb.PPlayerLevelInfo {
	data := x.Data.PlayerLevelData
	if data.Stamina == nil {
		data.Stamina = &pb.PStaminaInfo{}
	}
	return data
}

func (x *UserData) GetUserMailData() *pb.PUserMailInfo {
	data := x.Data.UserMail
	if data.UserMail == nil {
		data.UserMail = make(map[int64]*pb.PMailInfo)
	}
	return data
}

func (x *UserData) GetFriendData() *pb.PFriendData {
	data := x.Data.FriendData
	if data.Friends == nil {
		data.Friends = make(map[uint64]int32)
	}
	if data.Applys == nil {
		data.Applys = make(map[uint64]int64)
	}
	if data.Examinesx == nil {
		data.Examinesx = make(map[uint64]int64)
	}
	if data.Blacks == nil {
		data.Blacks = make(map[uint64]int32)
	}
	if data.Receives == nil {
		data.Receives = make(map[uint64]int32)
	}
	if data.Sends == nil {
		data.Sends = make(map[uint64]int32)
	}
	return data
}

func (x *UserData) GetUserAllianceData() *pb.PUserAllianceData {
	data := x.Data.UserAlliance
	if data.SignLog == nil {
		data.SignLog = make(map[int64]int64)
	}
	return data
}

func (x *UserData) GetOfflineEventData() *pb.POfflineEventData {
	data := x.Data.OfflineEventData
	if data.EventList == nil {
		data.EventList = make(map[int64]*pb.OfflineEvent)
	}
	return data
}

func (x *UserData) GetOrderData() *pb.OrderData {
	if x.OrderData == nil {
		x.OrderData = &pb.OrderData{}
	}

	data := x.OrderData
	if data.Orders == nil {
		data.Orders = make(map[string]*pb.Order)
	}
	if data.HistoryProducts == nil {
		data.HistoryProducts = make(map[int32]int32)
	}
	if data.RefundData == nil {
		data.RefundData = &pb.RefundData{
			RefundCount:  0,
			RefundAmount: 0,
		}
	}
	if data.ItemInfo == nil {
		data.ItemInfo = make(map[int32]*pb.OrderItemInfo)
	}
	return data
}

func (x *UserData) GetChatData() *pb.PUserChatInfo {
	data := x.Data.ChatInfo
	if data.LastSendTime == nil {
		data.LastSendTime = make(map[string]int64, 0)
	}
	if data.HasMessage == nil {
		data.HasMessage = make(map[string]bool, 0)
	}
	return data
}

func (x *UserData) GetRelationData() *pb.PUserRelationData {
	data := x.Data.RelationData
	if data.RelationData == nil {
		data.RelationData = make(map[string]*pb.UserRelationData, 0)
	}
	if data.CampCardLifeTime == nil {
		data.CampCardLifeTime = make(map[string]int64, 0)
	}
	return data
}

func (x *UserData) GetCallSysData() *pb.PUserCallSysData {
	data := x.Data.CallSysData
	if data.Signal == nil {
		data.Signal = make(map[int32]*pb.CardSignal, 0)
	}
	return data
}

// // 加载玩家数据
// func loadPlayerData[T proto.Message](userActor *UserActor, mongoDbName service.MongoDbType, dbKey string, dbData T) (T, error) {
//	if utils.IsNull(dbData) {
//		dbData = dbData.ProtoReflect().New().Interface().(T)
//	}
//
//	_, err := userActor.GetCache(mongoDbName, dbKey, dbData)
//	if err != nil {
//		var zero T // 返回默认值 - nil
//		return zero, err
//	}
//
//	return dbData, nil
// }

// // 加载玩家数据
// func loadPlayerData1(userActor *UserActor, handler IBaseHandler) (proto.Message, error) {
//	dbKey, dbData := handler.GetPlayerDataKV()
//
//	if !utils.IsNull(dbData) {
//		return dbData, nil
//	} else {
//		dbData = dbData.ProtoReflect().New().Interface()
//	}
//
//	dbData = dbData.ProtoReflect().New().Interface()
//	err := userActor.LoadDB(dbKey, dbData)
//	if err != nil {
//		//var zero T // 返回默认值 - nil
//		return nil, err
//	}
//
//	//data := dbData.ProtoReflect().New().Interface()
//
//	return dbData, nil
// }

// 保存玩家数据
/*func savePlayerData[T proto.Message](userActor *UserActor, dbKey string, dbData T) error {
	err := userActor.SaveMongoDB("", dbKey, dbData)
	if err != nil {
		return err
	}

	return nil
}*/
