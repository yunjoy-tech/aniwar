package useractor

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"gitee.com/bychannel/musae/framework/logger"

	"gitee.com/bychannel/aniwar/src/actorserver/useractor/event"

	"github.com/pkg/errors"

	"gitee.com/bychannel/aniwar/src/common/datahelper"

	"gitee.com/bychannel/aniwar/src/common/utils"

	"gitee.com/bychannel/aniwar/src/common"

	"gitee.com/bychannel/aniwar/src/excel/data"
	"gitee.com/bychannel/musae/framework/base"

	"gitee.com/bychannel/aniwar/src/common/db"
	"gitee.com/bychannel/aniwar/src/proto/pb"
	"gitee.com/bychannel/musae/framework/service"
	"google.golang.org/protobuf/proto"
)

type ActivityHandler struct {
	*UABaseHandler
}

func NewActivityHandler(actor *UserActor) *ActivityHandler {
	h := &ActivityHandler{UABaseHandler: NewUABaseHandler(actor, "ActivityHandler")}
	h.ChildHandler = h

	// 协议注册
	h.actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_ActivityListReq), h.ActivityListReq)
	h.actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_FetchActivityRewardReq), h.FetchActivityRewardReq)
	h.actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_OneKeyActivityRewardReq), h.OneKeyActivityRewardReq)

	return h
}

// Init 初始化模块数据
func (h *ActivityHandler) Init() error {
	// 初始化
	h.actor.Data.ActivityData = &pb.PServerUserActivity{
		CreateTime:    time.Now().Unix(),
		ActivityDatas: make(map[int32]*pb.ActivityData, 0),
	}

	if err := h.SaveDB(); err != nil {
		return err
	}

	h.Debug("init Activity data success.")
	return nil
}

func (h *ActivityHandler) EnterGame() error {
	// 尝试更新活动
	h.tryUpdateActivityList()

	return nil
}

func (h *ActivityHandler) DailyRefresh() error {
	// 尝试更新活动
	h.tryUpdateActivityList()

	// 七日签到, 每天上线自动签到
	_ = h.doSignin()

	return nil
}

func (h *ActivityHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*pb.PServerUserActivity); ok {
		h.actor.Data.ActivityData = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *ActivityHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyUserActivity(h.actor.ID()), h.actor.Data.ActivityData
}

// 处理任务完成
func (h *ActivityHandler) handleTaskType(e event.IEvent) error {
	var (
		nowSec = time.Now().Unix()
	)

	activityData := h.actor.Data.ActivityData
	for _, t := range e.Type() {
		for _, each := range activityData.ActivityDatas {
			// if each.ActivityState != pb.ActivityState_AS_under_way {
			//	// 活动不在进行中
			//	continue
			// }
			if !canActivityDo(each) {
				// 活动不在进行中
				h.Debugf("ActivityHandler.handleTaskType 活动不在进行中, activityId=%d", each.ActivityId)
				continue
			}

			// changedActivityData := &pb.ActivityData{
			//	ActivityId: each.ActivityId,
			//	Items:      make([]*pb.ActivityItem, 0),
			// }
			hadChange := false

			for _, item := range each.Items {
				// if nowSec < item.BeginTime {
				//	// 当前活动页未开始
				//	h.Debugf("ActivityHandler.handleTaskType 活动不在进行中, DayIndex=%d", item.DayIndex)
				//	continue
				// }

				activityItem := &pb.ActivityItem{
					TaskInfos: make([]*pb.TaskInfoItem, 0),
				}

				for _, taskInfo := range item.TaskInfos {
					if taskInfo == nil {
						h.Errorf("任务数据为nil, item:%v", item)
						continue
					}

					if taskInfo.CondId == t {
						if h.actor.TaskTypeMgr.CheckTaskConditionComplete(taskInfo, e, nowSec >= item.BeginTime) {
							activityItem.TaskInfos = append(activityItem.TaskInfos, taskInfo)
							hadChange = true
						}
					}
				}
			}

			if hadChange {
				// 通知客户端
				h.actor.comData.AddActivityData(each)
				h.Debugf("任务进度更新, 下发的数据:%v", each)
			}
		}
	}

	return nil
}

// 处理任务触发
func (h *ActivityHandler) handleTaskTrigger(e event.IEvent) error {
	h.tryUpdateActivityList()
	err := h.SaveDB()
	if err != nil {
		return err
	}
	return nil
}

func (h *ActivityHandler) tryUpdateActivityList() {
	newestActivityDatas := h.getNewestActivityDatas()

	if h.actor.Data.ActivityData.ActivityDatas == nil {
		h.actor.Data.ActivityData.ActivityDatas = make(map[int32]*pb.ActivityData, 0)
	}

	for _, each := range newestActivityDatas {
		// 新的活动
		h.actor.Data.ActivityData.ActivityDatas[each.ActivityId] = each
		// 下发客户端数据
		h.actor.comData.AddActivityData(each)

		if pb.ActivityExcelType(each.ActivityType) == pb.ActivityExcelType_AE_TYPE_signin {
			// 刚领取到签到活动，先执行下签到逻辑
			_ = h.doSignin()
		}
	}

	err := h.SaveDB()
	if err != nil {
		h.Debugf(err.Error())
	}
}

// 活动是否可见
func canActivityShow(activityData *pb.ActivityData) bool {
	if activityData == nil {
		return false
	}

	if err, _ := activityIsOpen(activityData.ActivityId); err != nil {
		logger.Debugf(err.Error())
		return false
	}

	nowSec := time.Now().Unix()
	b := activityData.BeginShowTime <= nowSec && nowSec < activityData.EndShowTime
	if !b {
		logger.Debugf("活动ActivityId=%d, 不在开启时间内, nowSec=%d, activityData:%v", nowSec, activityData)
	}

	return b
}

// 活动是否可执行
func canActivityDo(activityData *pb.ActivityData) bool {
	if activityData == nil {
		return false
	}

	if err, _ := activityIsOpen(activityData.ActivityId); err != nil {
		logger.Debugf(err.Error())
		return false
	}

	nowSec := time.Now().Unix()
	b := activityData.BeginTime <= nowSec && nowSec < activityData.EndTime

	if !b {
		logger.Debugf("活动ActivityId=%d, 不在可执行时间内, nowSec=%d, activityData:%v", nowSec, activityData)
	}

	return b
}

func (h *ActivityHandler) formatActivity2Client() *pb.PClientActivity {
	// 尝试更新活动
	h.tryUpdateActivityList()

	activityData := h.actor.Data.ActivityData

	clientActivityData := &pb.PClientActivity{
		ActivityDatas: make([]*pb.ActivityData, 0),
	}

	for _, each := range activityData.ActivityDatas {
		if !canActivityShow(each) {
			continue
		}

		clientActivityData.ActivityDatas = append(clientActivityData.ActivityDatas, each)
	}

	return clientActivityData
}

// ActivityListReq 获取互动列表
func (h *ActivityHandler) ActivityListReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req pb.C2LS_ActivityListReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_SerializeError)
	}

	// 尝试更新活动
	h.tryUpdateActivityList()

	res := &pb.LS2C_ActivityListRes{
		CommonData: h.actor.comData.FixDownComData(),
	}

	h.Debugf("获取活动数据:%v", res)

	return res, nil, int32(pb.ErrorCode_Success)
}

// FetchActivityRewardReq 获取活动奖励
func (h *ActivityHandler) FetchActivityRewardReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req pb.C2LS_FetchActivityRewardReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_SerializeError)
	}

	err, errCode, dropChange := h.fetchActivityRewards(req.ActivityId, req.DayIndex, req.ElementId)
	if err != nil {
		return nil, err, int32(errCode)
	}

	err = h.SaveDB()
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	res := &pb.LS2C_FetchActivityRewardRes{
		DropChange: dropChange,
		CommonData: h.actor.comData.FixDownComData(),
	}

	return res, nil, int32(pb.ErrorCode_Success)
}

func (h *ActivityHandler) OneKeyActivityRewardReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req pb.C2LS_OneKeyActivityRewardReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_SerializeError)
	}

	err, errCode, dropChange := h.oneKeyActivityRewards(req.ActivityId)
	if err != nil {
		return nil, err, int32(errCode)
	}

	err = h.SaveDB()
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	res := &pb.LS2C_OneKeyActivityRewardRes{
		DropChange: dropChange,
		CommonData: h.actor.comData.FixDownComData(),
	}

	return res, nil, int32(pb.ErrorCode_Success)
}

// getNewestActivityDatas 获取新的活动数据
func (h *ActivityHandler) getNewestActivityDatas() []*pb.ActivityData {
	var (
		err                                             error
		beginShowTime, beginTime, endTime, ShowLastTime time.Time
		nowSec                                          = time.Now()

		newestActivityDatas = make([]*pb.ActivityData, 0)
	)

	hadActivityData := h.actor.Data.ActivityData.ActivityDatas

	// 任务类型
	data.GetActivityMgr().Foreach(func(cfg *data.ActivityCfg) bool {
		if err, _ = activityIsOpen(cfg.Id); err != nil {

			return true
		}

		if !h.actor.TaskTriggerMgr.checkTriggerType([]int32{cfg.Unlocktype}) {
			// 没有触发成功, 继续遍历
			return true
		}

		if _, ok := hadActivityData[cfg.Id]; ok {
			// 活动已经接到了, 继续遍历
			return true
		}

		// 活动时间
		if pb.ActivityExcelTimeType(cfg.Timetype) == pb.ActivityExcelTimeType_AE_TIME_TYPE_newbie { // 没有过期时间
			// totalDays, err := strconv.Atoi(cfg.Value1)
			// if err != nil {
			//	err = errors.Wrap(err, fmt.Sprintf("解析配置表失败, activityId=%d, value1=%s", cfg.Id, cfg.Value1))
			//	h.Debugf(err.Error())
			//	return true
			// }

			todayRefreshTime := common.GetTodayRefreshTime(nowSec)

			beginShowTime = todayRefreshTime
			beginTime = todayRefreshTime

			// _endTimeSec := common.GetNextNDailyRefreshTime(int32(totalDays))
			// endTime = time.Unix(_endTimeSec, 0)
			// ShowLastTime = time.Unix(_endTimeSec, 0)
			endTime = time.Date(2999, 1, 1, 0, 0, 0, 0, time.Local) // 固定给个很长的过期时间
			ShowLastTime = endTime

		} else if pb.ActivityExcelTimeType(cfg.Timetype) == pb.ActivityExcelTimeType_AE_TIME_TYPE_background {
			beginShowTime, beginTime, endTime, ShowLastTime, err = datahelper.GetActivityCfgTime(cfg.Id)
			if err != nil {
				h.Debugf(err.Error())
				return true
			}
		}

		activityData := &pb.ActivityData{
			ActivityId:   cfg.Id,
			ActivityType: cfg.Type,
			// ActivityState: pb.ActivityState_AS_under_way,
			BeginTime:     beginTime.Unix(),
			EndTime:       endTime.Unix(),
			BeginShowTime: beginShowTime.Unix(),
			EndShowTime:   ShowLastTime.Unix(),
			Items:         nil,
		}

		if pb.ActivityExcelType(cfg.Type) == pb.ActivityExcelType_AE_TYPE_task {
			// 寻找活动任务
			_activityItems := h.getActivityTaskItems(cfg.Id)
			activityData.Items = _activityItems

			newestActivityDatas = append(newestActivityDatas, activityData)
		} else if pb.ActivityExcelType(cfg.Type) == pb.ActivityExcelType_AE_TYPE_signin {
			// 签到活动
			_activitySignInfo := h.getActivitySigninItems(cfg.Id, beginTime, endTime)
			activityData.Sign = _activitySignInfo
		}

		newestActivityDatas = append(newestActivityDatas, activityData)

		return true
	}, true)

	// 签到类型

	return newestActivityDatas
}

func (h *ActivityHandler) getActivityTaskItems(activityId int32) []*pb.ActivityItem {
	var (
		activityItems = make([]*pb.ActivityItem, 0)
	)

	data.GetActivityTaskMgr().Foreach(func(cfg *data.ActivityTaskCfg) bool {
		if cfg.ActivityId == activityId {
			// 开始时间
			diffDay := utils.Max(cfg.Days-1, 0) // 天数
			openTime := common.GetNextNDailyRefreshTime(diffDay)
			activityItem := &pb.ActivityItem{
				BeginTime: openTime,
				DayIndex:  cfg.Days,
			}

			taskCfgs := make([]*data.TaskCfg, 0)
			for _, eachTaskGroupId := range cfg.TaskgroupId {
				_taskCfgs := h.actor.TaskTriggerMgr.TriggerTaskGroupById(eachTaskGroupId)
				taskCfgs = append(taskCfgs, _taskCfgs...)
			}

			nowSec := time.Now().Unix()

			for _, taskCfg := range taskCfgs {
				taskInfo := h.actor.TaskTypeMgr.CreateTaskInfoItemNew(taskCfg, nowSec >= openTime)
				activityItem.TaskInfos = append(activityItem.TaskInfos, taskInfo)
			}

			activityItems = append(activityItems, activityItem)

			return false
		}

		return true
	}, true)

	return activityItems
}

func (h *ActivityHandler) getActivitySigninItems(activityId int32, beginTime time.Time, endTime time.Time) *pb.PActivitySignInfo {
	// var (
	//	activityItems = make([]*pb.ActivityItem, 0)
	// )

	// data.GetActivitySigninMgr().Foreach(func(cfg *data.ActivitySigninCfg) bool {
	//	if cfg.ActivityId == activityId {
	//		// 开始时间
	//		diffDay := utils.Max(cfg.Days-1, 0) // 天数
	//		openTime := common.GetNextNDailyRefreshTime(diffDay)
	//		activityItem := &pb.ActivityItem{
	//			BeginTime: openTime,
	//			DayIndex:  cfg.Days,
	//		}
	//
	//		signinInfo := &pb.PActivitySignInfo{
	//			NextSign:         common.GetTodayRefreshTime(time.Now()).Unix(),
	//			Signed:           0,
	//			HadRewardDayIdxs: make([]int32, 0),
	//		}
	//
	//		activityItem.SignInfo = signinInfo
	//
	//		activityItems = append(activityItems, activityItem)
	//
	//		return false
	//	}
	//
	//	return true
	// }, true)
	signinInfo := &pb.PActivitySignInfo{
		NextSign:         common.GetTodayRefreshTime(time.Now()).Unix(),
		Signed:           0,
		HadRewardDayIdxs: make([]int32, 0),
	}

	return signinInfo
}

func (h *ActivityHandler) fetchActivityRewards(activityId int32, dayIndex int32, elementId int32) (error, pb.ErrorCode, *pb.DropChange) {
	var (
		dropChange = &pb.DropChange{}
		nowSec     = time.Now().Unix()
	)

	activityCfg := data.GetActivityMgr().GetById(activityId)

	activityData, ok := h.actor.Data.ActivityData.ActivityDatas[activityId]
	if !ok {
		err := errors.New(fmt.Sprintf("不存在的活动数据, activityId=%d", activityId))
		h.Errorf(err.Error())
		return err, pb.ErrorCode_ParamError, nil
	}

	if activityData.BeginTime > nowSec || activityData.EndShowTime < nowSec {
		return errors.New(fmt.Sprintf("活动未开始或已经结束了, activityId=%d", activityId)), pb.ErrorCode_Activity_is_not_open, nil
	}

	switch pb.ActivityExcelType(activityCfg.Type) {
	case pb.ActivityExcelType_AE_TYPE_task:
		err, errCode, _dropChange := h.fetchActivityTaskRewards(activityId, dayIndex, elementId)
		if err != nil {
			return err, errCode, nil
		}
		mergeDropChange(dropChange, _dropChange, true)

	case pb.ActivityExcelType_AE_TYPE_signin:
		err, errCode, _dropChange := h.fetchActivitySigninRewards(activityId, dayIndex)
		if err != nil {
			return err, errCode, nil
		}
		mergeDropChange(dropChange, _dropChange, true)

	default:
		h.Warnf("未支持的活动类型, activityId=%d, activityType=%d", activityId, activityCfg.Type)
	}

	// 活动全部完成, 下发整体奖励
	_wholeActivityFinishRewards := h.wholeActivityFinishRewards(activityId)
	mergeDropChange(dropChange, _wholeActivityFinishRewards, true)

	h.actor.comData.AddActivityData(h.actor.Data.ActivityData.ActivityDatas[activityId])

	return nil, pb.ErrorCode_Success, dropChange
}

func (h *ActivityHandler) fetchActivityTaskRewards(activityId int32, dayIndex int32, elementId int32) (error, pb.ErrorCode, *pb.DropChange) {
	var (
		nowSec = time.Now().Unix()
	)

	activityData, ok := h.actor.Data.ActivityData.ActivityDatas[activityId]
	if !ok {
		err := errors.New(fmt.Sprintf("不存在的活动数据, activityId=%d", activityId))
		h.Errorf(err.Error())
		return nil, pb.ErrorCode_ParamError, nil
	}

	if err, errCode := activityIsOpen(activityId); err != nil {
		return nil, errCode, nil
	}

	for _, item := range activityData.Items {
		if item.DayIndex == dayIndex {
			if item.BeginTime > nowSec {
				// 活动页开始时间还未到
				h.Debugf("活动页时间还未到, activityId=%d, item.DayIdx=%d, item.BeginTime=%d, nowSec=%d", activityId, item.DayIndex, item.BeginTime, nowSec)
				return nil, pb.ErrorCode_Activity_is_not_open, nil
			}
		}
	}

	var taskInfoItem *pb.TaskInfoItem

LabelA:
	for _, item := range activityData.Items {

		if item.DayIndex == dayIndex {
			for _, taskInfo := range item.TaskInfos {
				if taskInfo.Id == elementId {
					if taskInfo.Status == TASK_STATUS_RECEIVED {
						h.Debugf("奖励已经领取了, activityId=%d, dayIndex=%d, elementId=%d", activityId, dayIndex, elementId)
						return nil, pb.ErrorCode_TaskStatusRewardReceived, nil
					}

					if taskInfo.Status != TASK_STATUS_COMPLETE {
						h.Debugf("任务尚未完成, activityId=%d, dayIndex=%d, elementId=%d", activityId, dayIndex, elementId)
						return nil, pb.ErrorCode_TaskStatusNotComplete, nil
					}

					taskInfoItem = taskInfo
					break LabelA
				}
			}
		}
	}
	if taskInfoItem == nil {
		return fmt.Errorf("任务未完成, %d", elementId), pb.ErrorCode_TaskStatusNotComplete, nil
	}
	taskCfg := data.GetTaskMgr().GetById(taskInfoItem.Id)
	if taskCfg == nil {
		return fmt.Errorf("taskCfg is nil, %d", taskInfoItem.Id), pb.ErrorCode_NotFoundConfig, nil
	}
	// 任务状态
	taskInfoItem.Status = TASK_STATUS_RECEIVED

	reward := datahelper.ConvertItem3(taskCfg.Reward)
	dropChange, err := GetDropMgr(h.actor).DropList2(reward, true, nil, h.actor.comData, common.CR_FINISH_ACTIVITY)
	if err != nil {
		return err, pb.ErrorCode_InternalError, nil
	}

	err = h.SaveDB()
	if err != nil {
		return err, pb.ErrorCode_SaveDBError, nil
	}

	return nil, pb.ErrorCode_Success, dropChange
}

func (h *ActivityHandler) oneKeyActivityRewards(activityId int32) (error, pb.ErrorCode, *pb.DropChange) {
	var (
		dropChange = &pb.DropChange{}
	)

	for _, activityData := range h.actor.Data.ActivityData.ActivityDatas {
		for _, item := range activityData.Items {
			for _, taskInfo := range item.TaskInfos {
				err, errCode, _dropChange := h.fetchActivityRewards(activityId, item.DayIndex, taskInfo.Id)
				if err != nil {
					return err, errCode, nil
				}

				mergeDropChange(dropChange, _dropChange, true)
			}
		}
	}

	return nil, pb.ErrorCode_Success, dropChange
}

// 互动全部完成下发互动表的奖励
func (h *ActivityHandler) wholeActivityFinishRewards(activityId int32) *pb.DropChange {
	activityData, ok := h.actor.Data.ActivityData.ActivityDatas[activityId]
	if !ok {
		h.Debugf("wholeActivityFinishRewards 未找到活动数据")
		return nil
	}

	switch pb.ActivityExcelType(activityData.ActivityType) {
	case pb.ActivityExcelType_AE_TYPE_task:
		for _, activityItem := range activityData.Items {
			for _, taskInfoItem := range activityItem.TaskInfos {
				if taskInfoItem.Status != TASK_STATUS_RECEIVED {
					// 还有任务没有完成
					h.Debugf("wholeActivityFinishRewards 还有任务没有完成, taskInfoItem:%v", taskInfoItem)
					return nil
				}
			}
		}
	case pb.ActivityExcelType_AE_TYPE_signin:
		activityCfg := data.GetActivityMgr().GetById(activityId)
		totalDays, err := strconv.Atoi(activityCfg.Value1)
		if err != nil {
			err = errors.Wrap(err, fmt.Sprintf("解析配置表失败, activityId=%d, value1=%s", activityCfg.Id, activityCfg.Value1))
			h.Debugf(err.Error())
			return nil
		}
		if len(activityData.Sign.HadRewardDayIdxs) < totalDays {
			// 还有未签到或还有签到奖励未领取
			h.Debugf("wholeActivityFinishRewards 还有未签到或还有签到奖励未领取, activityData.Sign:%v", activityData.Sign)
			return nil
		}
	default:
		h.Errorf("未支持的活动类型, activityId=%d", activityId)
	}

	activityCfg := data.GetActivityMgr().GetById(activityData.ActivityId)
	dropChange, err := GetDropMgr(h.actor).DropListByItems(activityCfg.Awarditem, true, nil, h.actor.comData, common.CR_FINISH_ACTIVITY)
	if err != nil {
		h.Errorf(err.Error())
		return nil
	}
	h.Debugf("wholeActivityFinishRewards 领取整个活动完成的奖励, dropChange:%v", dropChange)

	return dropChange
}

// doSignin 自动签到
func (h *ActivityHandler) doSignin() error {
	nowSec := time.Now().Unix()

	for _, activityData := range h.actor.Data.ActivityData.ActivityDatas {
		if pb.ActivityExcelType(activityData.ActivityType) == pb.ActivityExcelType_AE_TYPE_signin {
			if activityData.Sign.NextSign > nowSec {
				h.Debugf("当前签到时间还未到, activityData.Sign:%v", activityData.Sign)
				continue
			}

			activityData.Sign.Signed++
			h.Infof("执行签到, 总签到次数:%d", activityData.Sign.Signed)

			// 更新下次可签到的时间戳
			activityCfg := data.GetActivityMgr().GetById(activityData.ActivityId)
			totalDays, err := strconv.Atoi(activityCfg.Value1)
			if err != nil {
				err = errors.Wrap(err, fmt.Sprintf("解析配置表失败, activityId=%d, value1=%s", activityCfg.Id, activityCfg.Value1))
				h.Debugf(err.Error())
				return err
			}
			if activityData.Sign.Signed < int32(totalDays) { // 还有可签到的
				activityData.Sign.NextSign = common.GetNextDailyRefreshTime()
			}

			// 下发给客户端
			h.actor.comData.AddActivityData(activityData)
		}
	}

	err := h.SaveDB()
	if err != nil {
		return err
	}
	return nil
}

// fetchActivitySigninRewards 签到领取奖励
func (h *ActivityHandler) fetchActivitySigninRewards(activityId int32, dayIndex int32) (error, pb.ErrorCode, *pb.DropChange) {
	activityData, ok := h.actor.Data.ActivityData.ActivityDatas[activityId]
	if !ok {
		return errors.New(fmt.Sprintf("未找到活动数据, activityId=%d", activityId)), pb.ErrorCode_ParamError, nil
	}

	if pb.ActivityExcelType(activityData.ActivityType) != pb.ActivityExcelType_AE_TYPE_signin {
		return errors.New(fmt.Sprintf("不是签到活动, activityId=%d", activityId)), pb.ErrorCode_ParamError, nil
	}

	if activityData.Sign == nil {
		return errors.New(fmt.Sprintf("签到数据为空, activityId=%d", activityId)), pb.ErrorCode_ParamError, nil
	}

	if activityData.Sign.Signed < dayIndex {
		return errors.New(fmt.Sprintf("签到天数不足, activityId=%d, dayIndex=%d, signInfo:%v", activityId, dayIndex, activityData.Sign)),
			pb.ErrorCode_Activity_signin_count_not_enough, nil
	}

	hadFound := false
	for _, hadRewardIdx := range activityData.Sign.HadRewardDayIdxs {
		if hadRewardIdx == dayIndex {
			hadFound = true
		}
	}
	if hadFound {
		return errors.New(fmt.Sprintf("已经领取过奖励了, activityId=%d, dayIndex=%d, signInfo:%v", activityId, dayIndex, activityData.Sign)),
			pb.ErrorCode_Activity_had_got_reward, nil
	}

	// 记录已经领奖的标识
	activityData.Sign.HadRewardDayIdxs = append(activityData.Sign.HadRewardDayIdxs, dayIndex)

	// 发放奖励
	singinRewards := datahelper.GetActivitySinginRewards(activityId, dayIndex)
	dropChange, err := GetDropMgr(h.actor).DropListByItems(singinRewards, true, nil, h.actor.comData, common.CR_FINISH_ACTIVITY)
	if err != nil {
		return err, pb.ErrorCode_InternalError, nil
	}

	return nil, pb.ErrorCode_Success, dropChange
}

// 活动是否开启
func activityIsOpen(activityId int32) (error, pb.ErrorCode) {
	activityCfg := data.GetActivityMgr().GetById(activityId)

	if activityCfg.IsOpen != 1 {
		return errors.New(fmt.Sprintf("活动未开启, activiyId=%d", activityId)), pb.ErrorCode_Activity_is_not_open
	}

	return nil, pb.ErrorCode_Success
}
