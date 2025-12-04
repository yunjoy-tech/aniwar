package chapter

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datahelper"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	myUtils "gitlab.musadisca-games.com/wangxw/aniwar/src/common/utils"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"
)

// BattleNiwa 地图
type BattleNiwa struct {
	*cmd.BattleMapInfo
}

func NewBattleNiwa(levelId, niwaId int32 /*, niwaId int32 , finishedOnceEvents map[int32]*cmd.FinishedOnceEvent, finishQuestIds []int32*/) (*BattleNiwa, error) {
	if niwaId == 0 {
		levelCfg := data.GetLevelMgr().GetById(levelId)
		niwaId = levelCfg.Niwa // 没有指定地图id, 则读取配表的地图id
		logger.Debugf("没有指定地图id, 读取配表中的首个地图id, levelId=%d, niwaId=%d", levelId, niwaId)
	}

	// 校验地图id是否是当前关卡中
	levelCfg := data.GetLevelMgr().GetById(levelId)
	if !myUtils.ArrayContain(levelCfg.NiwaTotal, niwaId) {
		return nil, errors.New(fmt.Sprintf("无效的地图id, 配置的数据:%v, niwaId=%d", levelCfg.NiwaTotal, niwaId))
	}

	return &BattleNiwa{
		BattleMapInfo: &cmd.BattleMapInfo{
			UpdateTime:  time.Now().Unix(),
			NiwaId:      niwaId,
			BornPosIdx:  0,
			BornPosList: nil,
		},
	}, nil
}

// 创建地图节点事件
func GetMappointEvents(
	niwaId int32,
	mapEvents []*cmd.MappointEvent, // 地图上的事件列表
	finishedOnceEventIds []int32,
	updateType common.MAPPOINT_EVENT_UPDATE_TYPE,
	updateTypeParamIds []string) ([]*cmd.MappointEvent, []*cmd.MappointEventGroupInfo) {

	logger.Debugf("创建地图信息, niwaId=%d", niwaId)

	// 已经存在的事件列表
	existedEventMap := make(map[int32]bool)
	// 已存在事件的点列表
	existedPosMap := make(map[int32]bool)

	for _, event := range mapEvents {
		existedEventMap[event.EventId] = true
		existedPosMap[event.PosIdx] = true
	}
	for _, eventId := range finishedOnceEventIds {
		existedEventMap[eventId] = true
	}

	eventDataMap := make(map[int32]*cmd.MappointEvent, 0)
	//eventGroupMap := make(map[int32]*cmd.MappointEventGroupInfo, 0)

	cfgs := make([]*data.NiwaMappointCfg, 0)

	data.GetNiwaMappointMgr().Foreach(
		func(cfg *data.NiwaMappointCfg) bool {
			if cfg.GetNiwaId() == niwaId && int32(updateType) == cfg.RefreshType {
				switch updateType {
				case common.MAPPOINT_EVENT_UPDATE_TYPE_0:
					cfgs = append(cfgs, cfg)

				case common.MAPPOINT_EVENT_UPDATE_TYPE_2,
					common.MAPPOINT_EVENT_UPDATE_TYPE_5:
					if myUtils.WithinArray2(cfg.RefreshParam, updateTypeParamIds) {
						cfgs = append(cfgs, cfg)
					}

				default:
					logger.Errorf("未支持的类型, %v", updateType)
					return false
				}
			}
			return true
		},
		false)

	// log
	_showMappointEventLog(niwaId, cfgs)

	//LabelA:
	for _, cfg := range cfgs {
		if _, ok := existedPosMap[cfg.MappointId]; ok {
			// 该点已经存在事件
			logger.Debugf("该点已经存在事件, niwaId=%d, mappointId:%v", niwaId, cfg.MappointId)
			continue
		} else {
			existedPosMap[cfg.MappointId] = true
		}

		var eventData *cmd.MappointEvent

		if cfg.IsEventRandom == 0 { // 固定事件
			eventData = &cmd.MappointEvent{
				EventId: cfg.EventId,
				PosIdx:  cfg.MappointId,
			}
		} else { // 随机事件
			eventVo := NiwaRandomMappointEvent(cfg)
			posVo := NiwaRandomMappointPos(cfg)

			eventData = &cmd.MappointEvent{
				EventId: eventVo.VoId,
				PosIdx:  posVo.VoId,
			}
		}

		if eventData == nil || eventData.EventId == 0 {
			continue
		}

		if _, ok := existedEventMap[eventData.EventId]; ok {
			logger.Debugf("地图:%d, 地图上已经存在该事件:%d", niwaId, eventData.EventId)
			continue
		} else {
			existedEventMap[cfg.MappointId] = true
		}

		eventDataMap[eventData.EventId] = eventData
	}

	eventDataList := make([]*cmd.MappointEvent, 0)

	for _, each := range eventDataMap {
		eventDataList = append(eventDataList, each)
	}

	// log
	_showAddEventInfo(eventDataList)

	return eventDataList, datahelper.GetEventGroupList(eventDataList)
}

func _showAddEventInfo(mappointEvents []*cmd.MappointEvent) {
	var eventIdIntro string
	for _, event := range mappointEvents {
		if len(eventIdIntro) > 0 {
			eventIdIntro += ", "
		}
		eventIdIntro += fmt.Sprintf("%d:%d", event.EventId, event.PosIdx)
	}
	logger.Debugf("新增事件:%d个, %s", len(mappointEvents), eventIdIntro)
}

func _showMappointEventLog(niwaId int32, cfgs []*data.NiwaMappointCfg) {
	logger.Debugf("niwa_mappoint表niwaId:%d, 共有%d个节点事件", niwaId, len(cfgs))
	var cdgsIds string
	for _, cfg := range cfgs {
		if len(cdgsIds) > 0 {
			cdgsIds += ", "
		}
		cdgsIds += strconv.Itoa(int(cfg.GetId()))
	}
	logger.Debugf("cfgs.ids = %s", cdgsIds)
}

// NiwaRandomMappointEvent 随机构建节点事件
func NiwaRandomMappointEvent(cfg *data.NiwaMappointCfg) *data.WeightVo {
	poolCfg := data.GetMappointEventPoolMgr().GetById(cfg.GetEventPool())
	eventVo, err := datahelper.RandomByWeightVo(poolCfg.GetEventPool())
	if err != nil {
		logger.Debugf("随机地图事件时报错, err:%+v", err)
	}
	logger.Debugf("niwa_mappoint表id:%d, 随机到的事件:%+v", cfg.GetId(), eventVo)

	return eventVo
}

func NiwaRandomMappointPos(cfg *data.NiwaMappointCfg) *data.WeightVo {
	poolCfg := data.GetMappointPoolMgr().GetById(cfg.GetMappointPool())
	posVo, err := datahelper.RandomByWeightVo(poolCfg.GetMappointPool())
	if err != nil {
		logger.Debugf("随机地图事件时报错, err:%+v", err)
	}
	logger.Debugf("niwa_mappoint表id:%d, 随机到的节点:%+v", cfg.GetId(), posVo)
	return posVo
}

func (s *BattleNiwa) FormatNiWa2Proto() *cmd.BattleMapInfo {
	return s.BattleMapInfo
}

func FormatNiWa2Protos(mapInfos []*BattleNiwa) map[int32]*cmd.BattleMapInfo {
	ret := make(map[int32]*cmd.BattleMapInfo, 0)

	for _, each := range mapInfos {
		mapInfo := each.FormatNiWa2Proto()
		ret[mapInfo.NiwaId] = mapInfo
	}

	return ret
}
