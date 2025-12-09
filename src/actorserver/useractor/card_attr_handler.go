package useractor

import (
	"fmt"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"
)

func (h *CardHandler) CalcMaxHp(c *cmd.CardData) (uint32, error) {
	cardCfg := excel.GetBeastarMgr().GetById(int32(c.BaseId))
	if cardCfg == nil {
		return 0, fmt.Errorf("not found card: %d config", c.BaseId)
	}

	// 等级、突破提升 成长系数结果
	basicHp, err := h.GetHpByGrow(c)
	if err != nil {
		return 0, err
	}

	// 觉醒提升
	breakHp, err := h.GetHpByAwakenLevel(c, basicHp)
	if err != nil {
		return 0, err
	}

	// 性格提升
	characterHp, err := h.GetHpByCharacterLevel(c, basicHp)
	if err != nil {
		return 0, err
	}

	// 装备提升
	equipHp, err := h.GetHpByEquipAttr(c, basicHp)
	if err != nil {
		return 0, err
	}

	// 好感度等级提升
	favorHp, err := h.GetHpByFavorLevel(c, basicHp)
	if err != nil {
		return 0, err
	}

	sum := basicHp + breakHp + characterHp + equipHp + favorHp
	logger.Debugf("血量计算: cardId: %d, 基础: %d, 突破加成: %d, 性格加成: %d, 装备加成: %d, 好感度加成: %d, 总血量: %d", c.BaseId, basicHp, breakHp, characterHp, equipHp, favorHp, sum)
	return sum, nil
}

func (h *CardHandler) GetHpByGrow(c *cmd.CardData) (uint32, error) {
	cardCfg := excel.GetBeastarMgr().GetById(int32(c.BaseId))
	if cardCfg == nil {
		return 0, fmt.Errorf("not found card: %d config", c.BaseId)
	}

	// id := cardCfg.GetRarity()*10000 + int32(c.BreakthroughLevel)*1000 + int32(c.CardLevel)
	//
	// growCfg := excel.GetAttributegrowMgr().GetById(id)
	// if growCfg == nil {
	// 	return 0, nil
	// }
	// hp := float32(cardCfg.GetHp()) * float32(growCfg.GetHpgrow()) / 10000

	// logger.Debugf("GetHpByGrow: 基础血量 %d, 百分比 %d, 成长系数加成 %v", cardCfg.GetHp(), growCfg.GetHpgrow(), hp)
	return uint32(cardCfg.GetHp()), nil
}

func (h *CardHandler) GetHpByAwakenLevel(c *cmd.CardData, basicHp uint32) (uint32, error) {
	cardCfg := excel.GetBeastarMgr().GetById(int32(c.BaseId))
	if cardCfg == nil {
		return 0, fmt.Errorf("not found card: %d config", c.BaseId)
	}

	cardAwakenLevel := c.AwakenLevel
	if 0 >= cardAwakenLevel {
		return 0, nil
	}

	hp := uint32(0)
	for level := uint32(1); level <= cardAwakenLevel; level++ {
		cardAwakenLevelId := cardCfg.Potential*100 + int32(level)
		cardAwakenCfg := excel.GetPotentialMgr().GetById(cardAwakenLevelId)
		if cardAwakenCfg == nil {
			return 0, fmt.Errorf("card: %d not found compound: %d config", c.BaseId, cardAwakenLevelId)
		}

		if upValue, ok := cardAwakenCfg.UpAtt[int32(cmd.PCardAttri_PCardAttri_Hp)]; ok {
			if cardAwakenCfg.AbiType == 1 {
				// 绝对值
				hp += uint32(upValue)
				logger.Debugf("突破%d级 增加%d血 当前加成%d血", level, upValue, hp)
			} else if cardAwakenCfg.AbiType == 2 {
				// 百分比
				add := uint32(float32(basicHp) * (float32(upValue) / 100))
				hp += add
				logger.Debugf("突破%d级 百分比%d 增加%d血 当前加成%d血", level, upValue, add, hp)
			}
		}
	}

	logger.Debugf("GetHpByAwakenLevel: lv %d, 基础血量 %d, 突破加成 %d", cardAwakenLevel, basicHp, hp)
	return hp, nil
}

func (h *CardHandler) GetHpByCharacterLevel(c *cmd.CardData, basicHp uint32) (uint32, error) {
	level := c.CharacterLevel
	if 0 >= level {
		return 0, nil
	}

	characterHp := uint32(0)
	for i := uint32(1); i <= level; i++ {
		// 能力加成计算
		id := int32(c.BaseId*100 + i)
		characterCfg := excel.GetCharacterMgr().GetById(id)
		if characterCfg == nil {
			continue
		}
		// 区分不同性格
		tempAbi := getCharacterAbi(c, characterCfg)
		if tempAbi > 0 {
			// 能力的配置
			cfg := excel.GetCharacterAbiMgr().GetById(tempAbi)
			if cfg == nil {
				continue
			}

			if upValue, ok := cfg.GetUpAtt()[int32(cmd.PCardAttri_PCardAttri_Hp)]; ok {
				if cfg.AbiType == 1 {
					// 绝对值
					characterHp += uint32(upValue)
					logger.Debugf("性格%d级 能力加成增加%d血 当前加成%d血", i, upValue, characterHp)
				} else if cfg.AbiType == 2 {
					// 百分比
					add := uint32(float32(basicHp) * (float32(upValue) / 100))
					characterHp += add
					logger.Debugf("性格%d级 百分比%d 能力加成增加%d血 当前加成%d血", i, upValue, add, characterHp)
				}
			}
		}
		// 等级属性加成
		// 区分不同性格
		tempUpAtr := getCharacterUpAtr(c, characterCfg)
		if tempUpAtr != nil {
			for _, addition := range tempUpAtr {
				if addition.AttributeType != int32(cmd.PCardAttri_PCardAttri_Hp) {
					continue
				}
				if addition.AdditionType == 1 {
					characterHp += uint32(addition.Vaule)
					logger.Debugf("性格%d级 等级属性增加%d血 当前加成%d血", i, addition.Vaule, characterHp)
				} else if addition.AdditionType == 2 {
					add := uint32(float32(basicHp) * (float32(addition.Vaule) / 100))
					characterHp += add
					logger.Debugf("性格%d级 百分比%d 等级属性增加%d血 当前加成%d血", i, addition.Vaule, add, characterHp)
				}
			}
		}
	}

	logger.Debugf("GetHpByCharacterLevel: 基础血量 %d, 性格加成 %d", basicHp, characterHp)
	return characterHp, nil
}

func (h *CardHandler) GetHpByEquipAttr(c *cmd.CardData, basicHp uint32) (uint32, error) {
	// 获取卡牌上穿戴的装备
	equips, err := h.actor.EquipHandler.GetEquipList(int32(c.BaseId))
	if err != nil {
		return 0, err
	}
	var equipHp uint32
	for _, equip := range equips {
		// 判定主属性
		mainAttrHp := h.actor.EquipHandler.GetEquipMainAttrHp(equip)
		if mainAttrHp > 0 {
			equipHp += uint32(mainAttrHp)
			logger.Debugf("装备%d 主属性增加%d血 当前加成%d血", equip.ConfigId, mainAttrHp, equipHp)
		}
		// 判定副属性
		subAttrHp := h.actor.EquipHandler.GetEquipSubAttrHp(equip)
		if subAttrHp != nil {
			// 绝对值
			if v, ok := subAttrHp[1]; ok {
				equipHp += uint32(v)
				logger.Debugf("装备%d 副属性增加%d血 当前加成%d血", equip.ConfigId, v, equipHp)
			}
			// 百分比
			if v, ok := subAttrHp[2]; ok {
				add := uint32(float32(basicHp) * (float32(v) / 100))
				equipHp += add
				logger.Debugf("装备%d 百分比%d 副属性增加%d血 当前加成%d血", equip.ConfigId, v, add, equipHp)
			}
		}
	}

	logger.Debugf("GetHpByEquipAttr: 基础血量 %d, 装备加成 %d", basicHp, equipHp)
	return equipHp, nil
}

func (h *CardHandler) GetHpByFavorLevel(c *cmd.CardData, basicHp uint32) (uint32, error) {
	cardCfg := excel.GetBeastarMgr().GetById(int32(c.BaseId))
	if cardCfg == nil {
		return 0, fmt.Errorf("not found card: %d config", c.BaseId)
	}

	cardFavoriteLevel := c.FavoriteLevel
	if 0 >= cardFavoriteLevel {
		return 0, nil
	}

	hp := uint32(0)
	for level := uint32(1); level <= cardFavoriteLevel; level++ {
		cardFavorLevelId := cardCfg.Id*100 + int32(level)
		favorCfg := excel.GetFavorMgr().GetById(cardFavorLevelId)
		if favorCfg == nil {
			return 0, fmt.Errorf("card: %d not found compound: %d config", c.BaseId, cardFavorLevelId)
		}

		if upValue, ok := favorCfg.UpAtt[int32(cmd.PCardAttri_PCardAttri_Hp)]; ok {
			// 目前固定绝对值
			hp += uint32(upValue)
			logger.Debugf("好感度%d级 增加%d血 当前加成%d血", level, upValue, hp)
		}
	}

	logger.Debugf("GetHpByFavorLevel: lv %d, 基础血量 %d, 突破加成 %d", cardFavoriteLevel, basicHp, hp)
	return hp, nil
}

func getCharacterAbi(c *cmd.CardData, characterCfg *excel.CharacterCfg) int32 {
	if c.CurCharacter == int32(cmd.CharacterType_CharacterType_Human) {
		return characterCfg.CharacterAbility
	}
	if c.CurCharacter == int32(cmd.CharacterType_CharacterType_Animal) {
		return characterCfg.CharacterAbilityB
	}
	return 0
}

func getCharacterUpAtr(c *cmd.CardData, characterCfg *excel.CharacterCfg) []*excel.CardAttributeAddition {
	if c.CurCharacter == int32(cmd.CharacterType_CharacterType_Human) {
		return characterCfg.UpAtr
	}
	if c.CurCharacter == int32(cmd.CharacterType_CharacterType_Animal) {
		return characterCfg.UpAtrB
	}
	return nil
}
