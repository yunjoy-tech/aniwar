package useractor

import excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"

type OpenCondition struct {
	actor *UserActor
}

const (
	EAccountLevel  int32 = iota + 1 // 账户等级
	EBuildingLevel                  //建筑等级
	EFinishTask                     // 完成任务
)

func NewOpenCondition(actor *UserActor) *OpenCondition {
	return &OpenCondition{
		actor: actor,
	}
}

func (condition *OpenCondition) AllowOpens(temps []*excel.CampUpgradeCondition) bool {
	for _, v := range temps {
		if v != nil && condition.AllowOpen(v) == false {
			return false
		}
	}
	return true
}

func (condition *OpenCondition) AllowOpen(temp *excel.CampUpgradeCondition) bool {
	switch temp.Type {
	case EAccountLevel: //玩家等级
		return int32(condition.actor.LoginHandler.getRoleLevel()) >= temp.Param1
	case EBuildingLevel: //建筑等级
		for _, v := range condition.actor.GetCampData().DecorationBuilding {
			if temp.Param1 == v.ItemId {
				return temp.Param2 <= v.BuildingLevel
			}
		}
	case EFinishTask: // 完成任务
		return condition.actor.QuestHandler.IsComplete(temp.Param1)
	}
	return false
}
