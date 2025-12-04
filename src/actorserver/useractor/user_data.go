package useractor

import (
	"time"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"
)

/*

user actor data struct

*/

type UserData struct {
	Data      *cmd.PlayerData
	Account   *cmd.UserData
	OrderData *cmd.OrderData
}

//// 全量加载用户数据
//func (s *UserActor) loadDBDataByDBType() error {
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
//}

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

//// 加载账号数据
//func (u *UserActor) loadAccountData() error {
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
//}

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

func (x *UserData) GetUserData() *cmd.PServerRoleBaseInfo {
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

func (x *UserData) GetUserCardData() *cmd.PCardData {
	cards := x.Data.Cards
	if cards.Card == nil {
		cards.Card = make(map[uint32]*cmd.CardData)
	}

	return cards
}

func (x *UserData) GetTroopData() *cmd.PCardTroopsInfo {
	data := x.Data.Troops
	if data.Troop == nil {
		data.Troop = make(map[int32]*cmd.PServerCardTroopInfo)
	}

	return data
}

func (x *UserData) GetCampData() *cmd.PPlayerCampBlob {
	data := x.Data.Camp
	if data.DecorationBuilding == nil {
		data.DecorationBuilding = make(map[int64]*cmd.PPlayerCampDecorationBuilding)
	}
	if data.BuildingUnlockList == nil {
		data.BuildingUnlockList = make(map[int32]int32)
	}
	if data.Camp == nil {
		data.Camp = make(map[int32]*cmd.PPlayerCampServerCamp)
	}
	return data
}

func (x *UserData) GetCurrencyData() *cmd.PCurrencyInfo {
	data := x.Data.Currency
	if data.Currencyx == nil {
		data.Currencyx = make(map[int32]*cmd.CurrencyItem)
	}
	return data
}

func (x *UserData) GetTutorialData() *cmd.PPlayerBeginnerTutorialBlob {
	data := x.Data.Tutorial
	if data.FinishMasterTutorial == nil {
		data.FinishMasterTutorial = make([]*cmd.PPlayerDBBeginnerTutorialBlob, 0)
	}
	if data.FinishFunctionTutorial == nil {
		data.FinishFunctionTutorial = make([]*cmd.PPlayerDBBeginnerTutorialBlob, 0)
	}
	return data
}

func (x *UserData) GetPoolsData() *cmd.PServerCardPoolInfos {
	data := x.Data.Pools
	if data.TypeInfos == nil {
		data.TypeInfos = make(map[int32]*cmd.PServerCardPoolType)
	}
	if data.Newbie == nil {
		data.Newbie = &cmd.PNewbiePoolInfo{
			Select:  0,
			Results: make([]*cmd.PNewbiePoolLog, 0),
		}
	}
	return data
}

func (x *UserData) GetCampPoolsData() *cmd.PServerCampPoolInfos {
	data := x.Data.CampPools
	if data.TypeInfos == nil {
		data.TypeInfos = make(map[int32]*cmd.PServerCampPoolType)
	}
	return data
}

func (x *UserData) GetHandBookData() *cmd.PHandbookInfo {
	data := x.Data.Handbooks
	if data.HandBookItem == nil {
		data.HandBookItem = make(map[uint32]*cmd.ServerHandBookItem, 0)
	}
	return data
}

func (x *UserData) GetQuestionData() *cmd.PUserQuestions {
	data := x.Data.Question
	if data.Questions == nil {
		data.Questions = make(map[string]*cmd.PQuestion, 0)
	}
	return data
}

func (x *UserData) GetLevelsData() *cmd.LS2DB_LevelInfos {
	levelsData := x.Data.LevelsData
	if levelsData.LevelInfos == nil {
		levelsData.LevelInfos = make(map[int32]*cmd.LS2DB_LevelInfo)
	}
	if levelsData.PLevelSummary == nil {
		levelsData.PLevelSummary = &cmd.PServerLevelSummary{}
	}
	if levelsData.PLevelSummary.MonsterTicketInfoMap == nil {
		levelsData.PLevelSummary.MonsterTicketInfoMap = make(map[int32]*cmd.LevelMonsterTicketInfo, 0)
	}
	if levelsData.PLevelSummary.LevelSummaryMap == nil {
		levelsData.PLevelSummary.LevelSummaryMap = make(map[int32]*cmd.LevelSummary, 0)
	}

	if levelsData.FinishedOnceEvents == nil {
		levelsData.FinishedOnceEvents = make(map[int32]*cmd.FinishedOnceEvent)
	}

	return levelsData
}

func (x *UserData) GetStoryFlagData() *cmd.LS2DB_StoryFlagData {
	flagData := x.Data.StoryFlagData
	if flagData.Flags == nil {
		flagData.Flags = make(map[string]*cmd.FlagInfo)
	}

	return flagData
}

func (x *UserData) GetShopData() *cmd.LS2DB_ShopData {
	shopData := x.Data.ShopData
	if shopData.ShopInfos == nil {
		shopData.ShopInfos = make(map[int32]*cmd.ShopInfo)
	}

	return shopData
}

func (x *UserData) GetEquipData() *cmd.PEquipData {
	data := x.Data.EquipData
	if data.Equips == nil {
		data.Equips = make(map[uint64]*cmd.PCommonEquipInfo)
	}
	return data
}

func (x *UserData) GetCardSkinData() *cmd.PSkinData {
	data := x.Data.SkinData
	if data.Skins == nil {
		data.Skins = make(map[int32]*cmd.CardSkinData)
	}
	return data
}

func (x *UserData) GetDutyData() *cmd.PDutyData {
	data := x.Data.DutyData
	if data.DailyTask == nil {
		data.DailyTask = make(map[int32]*cmd.TaskInfoItem)
	}
	if data.UnlockTag == nil {
		data.UnlockTag = make(map[int32]*cmd.TaskInfoItem)
	}
	if data.Active == nil {
		data.Active = make(map[int32]*cmd.ActiveInfoItem)
	}
	if data.WeeklyTask == nil {
		data.WeeklyTask = make(map[int32]*cmd.TaskInfoItem)
	}
	return data
}

func (x *UserData) GetGuideTaskData() *cmd.PGuideTaskData {
	data := x.Data.GuideTaskData
	if data.Tasks == nil {
		data.Tasks = make(map[int32]*cmd.TaskInfoItem)
	}
	if data.Complete == nil {
		data.Complete = make(map[int32]int32)
	}
	return data
}

func (x *UserData) GetSignData() *cmd.PSignData {
	data := x.Data.Sign
	if data.Sign == nil {
		data.Sign = make(map[int32]*cmd.PCommonSignInfo)
	}
	return data
}

func (x *UserData) GetQuestData() *cmd.PQuestData {
	data := x.Data.QuestData
	if data.CompleteQuests == nil {
		data.CompleteQuests = make([]int32, 0)
	}
	if data.OpenQuests == nil {
		data.OpenQuests = make(map[int32]*cmd.PCommonQuestInfo)
	}
	return data
}

func (x *UserData) GetAchieveData() *cmd.PUserAchieves {
	data := x.Data.AchieveData
	if data.SectionReceive == nil {
		data.SectionReceive = make(map[string]*cmd.PAchieveReceive)
	}
	if data.Achieves == nil {
		data.Achieves = make(map[string]int32)
	}
	if data.GroupReceive == nil {
		data.GroupReceive = make(map[string]int32)
	}
	return data
}

func (x *UserData) GetCampaignData() *cmd.PPlayerGeneralCampaign {
	data := x.Data.CampaignInfo
	if data.ServerGeneralCampaign == nil {
		data.ServerGeneralCampaign = &cmd.PServerGeneralCampaign{
			ResCampaignCompleteCount: 0,
		}
	}

	if data.ServerGeneralCampaign.DailyCampaigns == nil {
		data.ServerGeneralCampaign.DailyCampaigns = make(map[int32]*cmd.PDailyCampaignBase)
	}
	if data.ServerGeneralCampaign.ResCampaigns == nil {
		data.ServerGeneralCampaign.ResCampaigns = make(map[int32]*cmd.PResourceCampaignBase)
	}

	return data
}

func (x *UserData) GetTrialData() *cmd.PUserTrial {
	data := x.Data.TrialData
	if data.Trial == nil {
		data.Trial = make(map[int32]*cmd.PTrialInfo)
	}
	return data
}

func (x *UserData) GetUseLimitData() *cmd.PUseLimitInfo {
	data := x.Data.UseLimit
	if data.Items == nil {
		data.Items = make(map[int32]*cmd.KeyValueItem)
	}
	return data
}

func (x *UserData) GetBlockWayData() *cmd.PBlockWay {
	data := x.Data.BlockWayData
	if data.EventList == nil {
		data.EventList = make(map[int64]*cmd.PBlockWayEvent)
	}
	if data.EventType == nil {
		data.EventType = make(map[int32]*cmd.PBlockWayEventGroup)
	}
	return data
}

func (x *UserData) GetRoleDetailData() *cmd.PServerRoleDetailInfo {
	data := x.Data.Detail
	if data.Common == nil {
		data.Common = &cmd.PCommonRoleBaseInfo{}
	}
	if data.Cards == nil {
		data.Cards = make([]int32, 4, 4)
	}
	if data.Lifex == nil {
		data.Lifex = make(map[int32]int32)
	}
	return data
}

func (x *UserData) GetPlayerLevelData() *cmd.PPlayerLevelInfo {
	data := x.Data.PlayerLevelData
	if data.Stamina == nil {
		data.Stamina = &cmd.PStaminaInfo{}
	}
	return data
}

func (x *UserData) GetUserMailData() *cmd.PUserMailInfo {
	data := x.Data.UserMail
	if data.UserMail == nil {
		data.UserMail = make(map[int64]*cmd.PMailInfo)
	}
	return data
}

func (x *UserData) GetFriendData() *cmd.PFriendData {
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

func (x *UserData) GetUserAllianceData() *cmd.PUserAllianceData {
	data := x.Data.UserAlliance
	if data.SignLog == nil {
		data.SignLog = make(map[int64]int64)
	}
	return data
}

func (x *UserData) GetOfflineEventData() *cmd.POfflineEventData {
	data := x.Data.OfflineEventData
	if data.EventList == nil {
		data.EventList = make(map[int64]*cmd.OfflineEvent)
	}
	return data
}

func (x *UserData) GetOrderData() *cmd.OrderData {
	if x.OrderData == nil {
		x.OrderData = &cmd.OrderData{}
	}

	data := x.OrderData
	if data.Orders == nil {
		data.Orders = make(map[string]*cmd.Order)
	}
	if data.HistoryProducts == nil {
		data.HistoryProducts = make(map[int32]int32)
	}
	if data.RefundData == nil {
		data.RefundData = &cmd.RefundData{
			RefundCount:  0,
			RefundAmount: 0,
		}
	}
	if data.ItemInfo == nil {
		data.ItemInfo = make(map[int32]*cmd.OrderItemInfo)
	}
	return data
}

func (x *UserData) GetChatData() *cmd.PUserChatInfo {
	data := x.Data.ChatInfo
	if data.LastSendTime == nil {
		data.LastSendTime = make(map[string]int64, 0)
	}
	if data.HasMessage == nil {
		data.HasMessage = make(map[string]bool, 0)
	}
	return data
}

func (x *UserData) GetRelationData() *cmd.PUserRelationData {
	data := x.Data.RelationData
	if data.RelationData == nil {
		data.RelationData = make(map[string]*cmd.UserRelationData, 0)
	}
	if data.CampCardLifeTime == nil {
		data.CampCardLifeTime = make(map[string]int64, 0)
	}
	return data
}

func (x *UserData) GetCallSysData() *cmd.PUserCallSysData {
	data := x.Data.CallSysData
	if data.Signal == nil {
		data.Signal = make(map[int32]*cmd.CardSignal, 0)
	}
	return data
}

//// 加载玩家数据
//func loadPlayerData[T proto.Message](userActor *UserActor, mongoDbName service.MongoDbType, dbKey string, dbData T) (T, error) {
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
//}

//// 加载玩家数据
//func loadPlayerData1(userActor *UserActor, handler IBaseHandler) (proto.Message, error) {
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
//}

// 保存玩家数据
/*func savePlayerData[T proto.Message](userActor *UserActor, dbKey string, dbData T) error {
	err := userActor.SaveMongoDB("", dbKey, dbData)
	if err != nil {
		return err
	}

	return nil
}*/
