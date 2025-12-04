package useractor

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/clidto"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/db"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"
	"google.golang.org/protobuf/proto"
	"time"
)

// RelationHandler 羁绊关系
type RelationHandler struct {
	*UABaseHandler
	ExcelCfg        map[int32]map[int32]*excel.AssociationCfg // 关系组:等级:excel
	CardKey         map[int32][]int32                         // 关系组Id：对应的CardId
	RelationLevel   map[int32]int32                           // 羁绊等级
	Open            map[int32][]*excel.AssociationCfg         // 剧情Id:配置
	WinRelation     int32                                     //战斗胜利关系成长的基础数值
	CampRelation    int32                                     //营地放置，每分钟累计的成长值
	CampRelationMax int32                                     //营地放置，可存储的经验值上限
}

func NewRelationHandlerHandler(actor *UserActor) *RelationHandler {
	h := &RelationHandler{UABaseHandler: NewUABaseHandler(actor, "RelationHandler")}
	h.ChildHandler = h

	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_ReadRelationPlotReq), h.ReadRelationPlotReq) //羁绊剧情记录已读
	return h
}

// Init 初始化模块数据
func (h *RelationHandler) Init() error {
	// 初始化
	h.actor.Data.RelationData = &cmd.PUserRelationData{
		RelationData:     make(map[string]*cmd.UserRelationData, 0),
		CampCardLifeTime: make(map[string]int64, 0),
	}

	// 保存
	if err := h.SaveDB(true); err != nil {
		return err
	}

	h.Debug("init achieve data success. player: %s", h.actor.ID())
	return nil
}

func (h *RelationHandler) EnterGame() error {
	// 预处理配置表
	h.ProcessExcel()
	return nil
}

func (h *RelationHandler) DailyRefresh() error {
	return nil
}

func (h *RelationHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*cmd.PUserRelationData); ok {
		h.actor.Data.RelationData = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *RelationHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyUserRelation(h.actor.ID()), h.actor.Data.RelationData
}

////////////////////////////////////////////////////////////////////////////////////////相关协议

func (h *RelationHandler) ReadRelationPlotReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	req := &cmd.C2LS_ReadRelationPlotReq{}

	if err := in.UnmarshalData(req); err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}
	// 根据key 获取数据
	relation := h.GetRelationDataByKey(req.GetKey())
	if relation == nil {
		h.Debug("ReadRelationPlotReq get relation data is err:", req.GetKey())
		return nil, errors.New("ReadRelationPlotReq params is err"), int32(cmd.ErrorCode_ParamError)
	}

	// 判断剧情等级是否达到
	if !h.CheckRelationLevel(req.GetKey(), req.GetPlotLevel()) {
		return nil, errors.New("ReadRelationPlotReq relation level not enough"), int32(cmd.ErrorCode_Relation_level_not_enough)
	}

	//读过的剧情Id，加入到列表
	relation.PlotList = h.AddRelationPlot(relation.PlotList, req.GetPlotLevel())

	h.SetRelationData(req.GetKey(), relation)
	if err := h.SaveDB(); err != nil {
		h.Debug("ReadRelationPlotReq SaveDB is err:", err)
		return nil, errors.New("SaveDB is err"), int32(cmd.ErrorCode_InternalError)
	}
	h.actor.comData.Data.CardRelation = append(h.actor.comData.Data.CardRelation, h.Relation2ClientOne(req.GetKey()))
	res := &cmd.LS2C_ReadRelationPlotRes{
		CommonData: h.actor.comData.FixDownComData(),
	}
	return res, nil, int32(cmd.ErrorCode_Success)
}

////////////////////////////////////////////////////////////////////////////////////////内部调用

func (h *RelationHandler) ProcessExcel() {

	if h.ExcelCfg == nil {
		h.ExcelCfg = make(map[int32]map[int32]*excel.AssociationCfg, 0)
	}
	if h.CardKey == nil {
		h.CardKey = make(map[int32][]int32, 0)
	}
	if h.RelationLevel == nil {
		h.RelationLevel = make(map[int32]int32, 0)
	}
	if h.Open == nil {
		h.Open = make(map[int32][]*excel.AssociationCfg, 0)
	}
	//cfg := make(map[int32]map[int32]*excel.AssociationCfg, 0)
	excel.GetAssociationMgr().Foreach(func(c *excel.AssociationCfg) bool {
		var item map[int32]*excel.AssociationCfg
		var ok bool
		if item, ok = h.ExcelCfg[c.Array]; !ok {
			item = make(map[int32]*excel.AssociationCfg, 0)
		}
		item[c.Level] = c
		h.ExcelCfg[c.Array] = item

		if _, ok = h.CardKey[c.Array]; !ok {
			h.CardKey[c.Array] = []int32{c.HeroA, c.HeroB}
		}
		h.Open[c.Limit] = append(h.Open[c.Limit], c)

		//处理Open

		return true
	}, true)

	excel.GetAssLevelMgr().Foreach(func(cfg *excel.AssLevelCfg) bool {

		h.RelationLevel[cfg.Id] = cfg.Exp

		return true
	}, true)
	h.WinRelation = excel.GetConfigMgr().GetCfg().FRIENDSHIP_BASE
	h.CampRelation = excel.GetConfigMgr().GetCfg().FRIENDSHIP_LIFE
	h.CampRelationMax = excel.GetConfigMgr().GetCfg().FRIENDSHIP_LIFE_LIMIT
}

func (h *RelationHandler) GetRelationDataByKey(key string) *cmd.UserRelationData {
	relationData := h.actor.GetRelationData()
	if data, ok := relationData.RelationData[key]; ok {
		return data
	}
	return nil
}

func (h *RelationHandler) GetCampCardLifeTime() map[string]int64 {
	relationData := h.actor.GetRelationData()
	return relationData.CampCardLifeTime
}

func (h *RelationHandler) SetRelationData(key string, data *cmd.UserRelationData) {
	relationData := h.actor.GetRelationData()
	relationData.RelationData[key] = data
}

func (h *RelationHandler) AddRelationValue(ass int32, key string, value int32) *cmd.UserRelationData {
	if value == 0 {
		return nil
	}
	relationData := h.GetRelationDataByKey(key)
	if relationData == nil {
		relationData = &cmd.UserRelationData{
			RelationLevel: 0,
			RelationExp:   0,
			PlotList:      make([]int32, 0),
		}
	}
	h.Infof("卡牌增加羁绊值,ass[%d],原来的羁绊值[%d],等级[%d]", ass, relationData.RelationExp, relationData.RelationLevel)
	relationData.RelationExp += value
	relationData = h.AddRelationExp(ass, relationData)
	h.SetRelationData(key, relationData)
	h.Infof("卡牌增加羁绊值,ass[%d],新的羁绊值[%d],等级[%d]", ass, relationData.RelationExp, relationData.RelationLevel)
	return relationData
}

// AddRelationExpOne 校验经验 只升一级
func (h *RelationHandler) AddRelationExpOne(ass int32, relation *cmd.UserRelationData) *cmd.UserRelationData {
	curLevel := relation.RelationLevel
	nextLevelNeedExp := h.RelationLevel[curLevel+1]
	if relation.RelationExp >= nextLevelNeedExp { //可以升级
		if h.IsOpenNextLevel(ass, curLevel+1) { //下一剧情解锁
			relation.RelationLevel += 1
			relation.RelationExp = 0
		} else {
			relation.RelationExp = nextLevelNeedExp
		}
	}
	return relation
}

// AddRelationExp 校验经验 可以连续升级
func (h *RelationHandler) AddRelationExp(ass int32, relation *cmd.UserRelationData) *cmd.UserRelationData {
	maxLevel := h.AssMaxUnlockLevel(ass)
	startLevel := relation.RelationLevel
	for i := startLevel; i <= maxLevel; i++ {
		nextLevelNeedExp := h.RelationLevel[i+1]
		//判断是否可以增加
		if !h.IsOpenNextLevel(ass, i) {
			if relation.RelationExp >= nextLevelNeedExp {
				relation.RelationExp = nextLevelNeedExp
				continue
			}
		}
		//处理等级
		if relation.RelationExp >= nextLevelNeedExp { //可以升级
			relation.RelationLevel = h.AddRelationLevel(relation.RelationLevel, 1, ass)
			relation.RelationExp = relation.RelationExp - nextLevelNeedExp // 这里必定是正数
		}
	}
	return relation
}

func (h *RelationHandler) AddRelationLevel(level, value, ass int32) int32 {
	newLevel := level + value
	maxLevel := h.GetAssMaxLevel(ass)
	if newLevel > maxLevel {
		newLevel = maxLevel
	}
	return newLevel
}

func (h *RelationHandler) GetAssMaxLevel(ass int32) int32 {
	levelCfg, ok := h.ExcelCfg[ass]
	if !ok {
		h.Debug("relation is add get excel is err:", ass)
		return 0
	}
	return int32(len(levelCfg))
}

// IsOpenNextLevel ass 关系组
func (h *RelationHandler) IsOpenNextLevel(ass, level int32) bool {
	// 判断下一级有没有解锁
	levelCfg, ok := h.ExcelCfg[ass]
	if !ok {
		h.Debug("relation is add get excel is err:", ass)
		return false
	}
	assCfg, ok := levelCfg[level+1]
	if !ok {
		return false
	}
	// 判断下一级有没有解锁
	if assCfg.Limit > 0 {
		if !h.actor.QuestHandler.checkQuestFinish(assCfg.Limit) {
			return false
		}
	}
	return true
}

func (h *RelationHandler) AssMaxUnlockLevel(ass int32) int32 {
	levelCfg, ok := h.ExcelCfg[ass]
	if !ok {
		h.Debug("relation is add get excel is err:", ass)
		return 0
	}
	maxLevel := int32(0)
	for _, v := range levelCfg {
		if v.Limit > 0 {
			if !h.actor.QuestHandler.checkQuestFinish(v.Limit) {
				continue
			}
		}
		if maxLevel < v.Level {
			maxLevel = v.Level
		}
	}
	return maxLevel
}

func (h *RelationHandler) SetCampCardLifeTime(key string, value int64) {
	lifeTime := h.GetCampCardLifeTime()
	if lifeTime == nil {
		return
	}
	lifeTime[key] = value
}

// Relation2Client 羁绊数据返回给客户端
func (h *RelationHandler) Relation2Client() []*cmd.CardRelationData {
	clientRelation := make([]*cmd.CardRelationData, 0)
	for k, v := range h.actor.GetRelationData().RelationData {
		clientRelation = append(clientRelation, &cmd.CardRelationData{
			Key:           k,
			RelationLevel: v.GetRelationLevel(),
			RelationExp:   v.GetRelationExp(),
			PlotList:      v.PlotList,
		})
	}
	return clientRelation
}

// Relation2ClientOne 羁绊数据返回给客户端 单个
func (h *RelationHandler) Relation2ClientOne(key string) *cmd.CardRelationData {
	if data, ok := h.actor.GetRelationData().RelationData[key]; ok {
		return &cmd.CardRelationData{
			Key:           key,
			RelationLevel: data.GetRelationLevel(),
			RelationExp:   data.GetRelationExp(),
			PlotList:      data.PlotList,
		}
	}
	return nil
}

func (h *RelationHandler) GetAllRelationData() []*cmd.CardRelationData {
	// 老号修正数据
	allCards := h.FilterNoAssCard(h.actor.CardHandler.GetAllCardIds())
	if len(allCards) == 0 {
		return h.Relation2Client()
	}
	for id := range allCards {
		h.InitCardRelation(id, nil)
	}
	return h.Relation2Client()
}

// InitCardRelation 初始卡片羁绊数据
func (h *RelationHandler) InitCardRelation(cardId int32, commonData *clidto.Comdata) bool {

	association := h.GetCardArrayCfg(cardId)
	if association == nil {
		return false
	}
	for _, arr := range association {
		// 判断两个角色是否存在
		cardIds, ok := h.CardKey[arr]
		if !ok {
			continue
		}
		if !h.CardExist(cardIds) { //有一个没有获取
			continue
		}
		// 两个都获取到了进行初始化
		h.InitRelation(cardIds, commonData)
	}

	if h.SaveDB() != nil {
		h.Debug("save relation is err")
	}

	return true

}

// CampUpdateRelation 营地更新羁绊值
func (h *RelationHandler) CampUpdateRelation(cardIds []int32, commonData *clidto.Comdata, relationType int32, flag bool) {
	h.AddRelation(cardIds, commonData, relationType)
	if flag {
		h.CampLifeAddOrSubCard(cardIds)
	}
	if h.SaveDB() != nil {
		h.Debug("save relation is err")
	}
	h.Infof("营地更新羁绊值,cardIds[%v],relationType[%d]", cardIds, relationType)
}

// AddRelation 增加伙伴的羁绊值 战斗胜利
func (h *RelationHandler) AddRelation(cardIds []int32, commonData *clidto.Comdata, relationType int32) {
	//过滤调没有关系组的卡片
	newCardIds := h.FilterNoAssCard(cardIds)
	//计算羁绊值
	for id := range newCardIds {
		h.addRelation(id, newCardIds, commonData, relationType)
		delete(newCardIds, id) //判断完就删掉，避免后面的卡牌重复计算
	}
	if h.SaveDB() != nil {
		h.Debug("save relation is err")
	}
	h.Infof("战斗胜利更新羁绊值,cardIds[%v],relationType[%d]", cardIds, relationType)
}

// CampLifeAddOrSubCard 营地生活区伙伴上下阵
func (h *RelationHandler) CampLifeAddOrSubCard(cardIds []int32) {
	if len(cardIds) < 2 {
		return
	}
	//过滤调没有关系组的卡片
	newCardIds := h.FilterNoAssCard(cardIds)
	for id := range newCardIds {
		h.CalculateTime(id, newCardIds)
		delete(newCardIds, id) //判断完就删掉，避免后面的卡牌重复计算
	}
	if h.SaveDB() != nil {
		h.Debug("save relation is err")
	}
}

// FilterNoAssCard 过滤掉没有关系组的伙伴
func (h *RelationHandler) FilterNoAssCard(cardIds []int32) map[int32]interface{} {
	newCardIds := make(map[int32]interface{}, 0)
	for _, id := range cardIds {
		ass := h.GetCardArrayCfg(id)
		if ass == nil {
			continue
		}
		newCardIds[id] = struct{}{}
	}
	return newCardIds
}

func (h *RelationHandler) CalculateTime(cardId int32, newCardIds map[int32]interface{}) {
	association := h.GetCardArrayCfg(cardId)
	if association == nil {
		return
	}
	now := time.Now().Unix()
	for _, ass := range association {
		cardIds, ok := h.CardKey[ass]
		otherCardId := int32(0)
		for _, id := range cardIds {
			if id != cardId {
				otherCardId = id
			}
		}
		// 不在队伍里
		if _, ok = newCardIds[otherCardId]; !ok {
			continue
		}
		// 更新时间
		h.SetCampCardLifeTime(h.GetRelationKey(cardIds), now) //
		h.Debugf("营地卡牌[%d]和[%d]开始计时:", cardIds[0], cardIds[1], h.WinRelation)
	}
}

func (h *RelationHandler) addRelation(cardId int32, newCardIds map[int32]interface{}, commonData *clidto.Comdata, relationType int32) {
	association := h.GetCardArrayCfg(cardId)
	if association == nil {
		return
	}
	//关系组
	for _, ass := range association {
		cardIds, ok := h.CardKey[ass] // 关系组的两个角色
		otherCardId := int32(0)
		for _, id := range cardIds {
			if id != cardId {
				otherCardId = id
			}
		}
		// 不在队伍里
		if _, ok = newCardIds[otherCardId]; !ok {
			continue
		}
		// 增加羁绊值
		key := h.GetRelationKey(cardIds)
		if relationData := h.AddRelationValue(ass, key, h.GetAddValue(relationType, cardIds)); relationData != nil {
			h.AddCommonData(key, relationData, commonData)
		}
		h.Debugf("卡牌[%d]和[%d]增加羁绊值[%d]", cardIds[0], cardIds[1], h.GetAddValue(relationType, cardIds))
	}
}
func (h *RelationHandler) GetAddValue(relationType int32, cardIds []int32) int32 {
	switch relationType {
	case common.Realtion_type_win:
		h.Infof("战斗胜利增加羁绊值[%d]", h.WinRelation)
		return h.WinRelation
	case common.Realtion_type_camp_life:
		//营地计算
		lifeTime := h.GetCampCardLifeTime()
		timeStamp, ok := lifeTime[h.GetRelationKey(cardIds)]
		if !ok {
			return 0
		}
		now := time.Now().Unix()
		diff := int32((now - timeStamp) / 60)
		if diff*h.CampRelation >= h.CampRelationMax {
			h.SetCampCardLifeTime(h.GetRelationKey(cardIds), now)
			return h.CampRelationMax
		}
		h.SetCampCardLifeTime(h.GetRelationKey(cardIds), now)
		h.Infof("营地生活区增加羁绊值[%d]", diff*h.CampRelation)
		return diff * h.CampRelation
	}
	return 0
}

func (h *RelationHandler) GetRelationKey(cardId []int32) string {
	if cardId[0] > cardId[1] {
		return fmt.Sprintf("%d_%d", cardId[0], cardId[1])
	}
	return fmt.Sprintf("%d_%d", cardId[1], cardId[0])
}

// CardExist 判断关系组里的角色是否已经获取了
func (h *RelationHandler) CardExist(cardIds []int32) bool {
	for _, id := range cardIds {
		if !h.actor.CardHandler.IsExistCard(uint32(id)) {
			return false
		}
	}
	return true
}

func (h *RelationHandler) InitRelation(cardIds []int32, commonData *clidto.Comdata) {
	if len(cardIds) < 2 {
		return
	}
	key := h.GetRelationKey(cardIds)
	data := h.actor.GetRelationData()
	if relation, ok := data.RelationData[key]; !ok {
		relation = &cmd.UserRelationData{
			RelationLevel: 0,
			RelationExp:   0,
			PlotList:      make([]int32, 0),
		}
		data.RelationData[key] = relation
		h.AddCommonData(key, relation, commonData)
	}
}

// GetCardArrayCfg 获取卡牌的关系组配置
func (h *RelationHandler) GetCardArrayCfg(cardId int32) []int32 {
	cardCfg := excel.GetBeastarMgr().GetById(cardId)
	if cardCfg == nil {
		return nil
	}
	if len(cardCfg.Association) == 0 {
		return nil
	}
	return cardCfg.Association
}

func (h *RelationHandler) AddCommonData(key string, relation *cmd.UserRelationData, commonData *clidto.Comdata) {
	if commonData == nil {
		return
	}
	commonData.Data.CardRelation = append(commonData.Data.CardRelation, &cmd.CardRelationData{
		Key:           key,
		RelationLevel: relation.GetRelationLevel(),
		RelationExp:   relation.GetRelationExp(),
		PlotList:      relation.PlotList,
	})
}

// CheckRelationLevel 检测剧情等级是否合法
func (h *RelationHandler) CheckRelationLevel(key string, level int32) bool {
	relation := h.GetRelationDataByKey(key)

	if level > relation.RelationLevel {
		return false
	}
	return true
}

func (h *RelationHandler) AddRelationPlot(plotList []int32, plot int32) []int32 {
	for _, p := range plotList {
		if p == plot {
			return plotList
		}
	}
	plotList = append(plotList, plot)
	return plotList
}

// OpenRelationLevel 解锁伙伴关系
func (h *RelationHandler) OpenRelationLevel(questId int32, commonData *clidto.Comdata) {
	if !h.actor.QuestHandler.checkQuestFinish(questId) {
		return
	}
	// 根据剧情Id 找到配置
	cfgs := h.Open[questId]
	if len(cfgs) == 0 {
		return
	}

	// 循环找到的配置，判断有没有可以解锁的
	for _, cfg := range cfgs {
		key := h.GetRelationKey([]int32{cfg.HeroA, cfg.HeroB})
		// 根据配置的cardId 找到服务端数据2_1
		relationData := h.GetRelationDataByKey(key)
		//获取配置当前等级的最大值
		nextLevelNeedExp := h.RelationLevel[relationData.RelationLevel+1]
		//判断当前等级的经验值是否大于等于 最大值值
		if relationData.RelationLevel+1 == cfg.Level && relationData.RelationExp >= nextLevelNeedExp {
			relationData.RelationLevel = h.AddRelationLevel(relationData.RelationLevel, 1, cfg.Array)
			relationData.RelationExp = 0
			h.SetRelationData(key, relationData)
		}
		h.AddCommonData(key, relationData, commonData)
	}
	if err := h.SaveDB(); err != nil {
		h.Debug("OpenRelationLevel saveDB err")
	}
}

// GMAddRelation 增加伙伴的羁绊值 战斗胜利
func (h *RelationHandler) GMAddRelation(cardIds []int32, commonData *clidto.Comdata, value int32) {
	key := h.GetRelationKey(cardIds)

	//根据
	ass := int32(0)
	excel.GetAssociationMgr().Foreach(func(cfg *excel.AssociationCfg) bool {
		tmpCards := make([]int32, 0)
		tmpCards = append(tmpCards, cfg.HeroA)
		tmpCards = append(tmpCards, cfg.HeroB)
		if h.GetRelationKey(tmpCards) == key {
			ass = cfg.Array
		}
		return true
	}, false)

	if relationData := h.AddRelationValue(ass, key, value); relationData != nil {
		h.AddCommonData(key, relationData, commonData)
	}
	h.Debug("GM add cards relation:", cardIds, value)
	if h.SaveDB() != nil {
		h.Debug("save relation is err")
	}
}
