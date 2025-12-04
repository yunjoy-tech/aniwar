package useractor

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datahelper"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/actorserver/useractor/event"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/clidto"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/db"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"google.golang.org/protobuf/proto"
)

type StoryFlagHandler struct {
	*UABaseHandler
}

func NewStoryHandler(actor *UserActor) *StoryFlagHandler {
	h := &StoryFlagHandler{UABaseHandler: NewUABaseHandler(actor, "StoryFlagHandler")}
	h.ChildHandler = h

	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_SaveStoryFlagReq), h.SaveStoryFlag) // 使用道具

	return h
}

// Init 初始化模块数据
func (h *StoryFlagHandler) Init() error {
	// 初始化
	h.actor.Data.StoryFlagData = &cmd.LS2DB_StoryFlagData{
		Createtime: time.Now().Unix(),
		Flags:      make(map[string]*cmd.FlagInfo),
	}

	// 保存
	if err := h.SaveDB(true); err != nil {
		return err
	}

	h.Debug("init story-flag data success. player: %s", h.actor.ID())
	return nil
}

func (h *StoryFlagHandler) EnterGame() error {
	return nil
}

func (h *StoryFlagHandler) DailyRefresh() error {
	return nil
}

func (h *StoryFlagHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*cmd.LS2DB_StoryFlagData_Old); ok {
		// 兼容旧数据结构
		flagInfos := make(map[string]*cmd.FlagInfo, 0)
		for key, _ := range dbVal.Flags {
			err, flag, val := datahelper.GetFlagVal(key)
			if err != nil {
				return err
			}
			flagInfos[key] = &cmd.FlagInfo{
				Key: flag,
				Val: val,
			}
		}
		newData := &cmd.LS2DB_StoryFlagData{
			Createtime: dbVal.Createtime,
			Flags:      flagInfos,
		}
		h.actor.Data.StoryFlagData = newData
		err := h.SaveDB()
		if err != nil {
			return err
		}
		h.Debugf("SetDBData, LS2DB_StoryFlagData_Old:%v, 数据结构兼容转换! %v", dbVal, newData)
		return nil
	} else if dbVal, ok := dbData.(*cmd.LS2DB_StoryFlagData); ok {
		for key, flagInfo := range dbVal.Flags {
			if flagInfo.Key == "" {
				flagInfo.Key = key
			}
		}

		h.actor.Data.StoryFlagData = dbVal

	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *StoryFlagHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyUserStoryFlag(h.actor.ID()), h.actor.Data.StoryFlagData
}

func (h *StoryFlagHandler) SaveStoryFlag(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err        error
		errCode    cmd.ErrorCode
		changeDrop = &cmd.DropChange{}
	)

	var req cmd.C2LS_SaveStoryFlagReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	if err, errCode = h.saveStoryFlagVal(h.actor.comData, req.Flag); err != nil {
		return nil, err, int32(errCode)
	}

	// 下发奖励
	flagCfg := data.GetFlagMgr().GetById(req.Flag.Key)

	addRewards, err := newDropMgr(h.actor).DropList2(flagCfg.FlagReward, true, nil, h.actor.comData, common.CR_Save_Story_Flag)
	changeDrop.Items = append(changeDrop.Items, addRewards.Items...)
	//err = h.sendStoryFlagNtf([]string{req.Flag})
	//if err != nil {
	//	h.Errorf("sendStoryFlagNtf 报错, err:%+v", err)
	//}

	h.actor.comData.Data.Flags = append(h.actor.comData.Data.Flags, req.Flag)

	rsp := &cmd.LS2C_SaveStoryFlagRes{
		Success: 1,
		//ItemReward: addRewards.Items,
		DropChange: changeDrop,
		CommonData: h.actor.comData.FixDownComData(),
	}

	return rsp, nil, 0
}

// 保存story-flag
func (h *StoryFlagHandler) saveStoryFlag(commonData *clidto.Comdata, flagInfoStrs ...string) (error, cmd.ErrorCode) {
	for _, flagStr := range flagInfoStrs {
		err, flag, val := datahelper.GetFlagVal(flagStr)
		if err != nil {
			return err, cmd.ErrorCode_ConfigError
		}

		err, errCode := h.saveStoryFlagVal(
			commonData,
			&cmd.FlagInfo{
				Key: flag,
				Val: val,
			})
		if err != nil {
			return err, errCode
		}
	}

	return nil, cmd.ErrorCode_Success
}

// 保存story-flag
func (h *StoryFlagHandler) saveStoryFlagVal(commonData *clidto.Comdata, flag *cmd.FlagInfo) (error, cmd.ErrorCode) {
	var (
		err       error
		hadChange = false
	)

	//for _, flag := range flags {
	flagCfg := data.GetFlagMgr().GetById(flag.Key)
	if flagCfg == nil {
		h.Debugf("无效的参数, story-flag: %v", flag)
		//continue
	}

	if flag.Val < 0 || flag.Val > 9 {
		return errors.New("flag值不在0-9之间"), cmd.ErrorCode_Chapter_flag_val_not_legal
	}

	storyFlagData := h.actor.GetStoryFlagData()
	//if _, ok := storyFlagData.Flags[flag.Key]; ok {
	//	h.Debugf("已经存过了, story-flag:%v", flag)
	//	//continue
	//} else {
	hadChange = true

	storyFlagData.Flags[flag.Key] = flag

	if commonData != nil {
		commonData.Data.Flags = append(commonData.Data.Flags, flag)
	}
	//}
	//}

	if hadChange {
		if err = h.SaveDB(); err != nil {
			return err, cmd.ErrorCode_InternalError
		}
		// 处理剧情任务刷新
		if errx := h.actor.eventManager.SyncPublish(event.NewBasicEvent(TASK_EVENT_STORY_FLAG_CHANGE, []int32{}, nil)); errx != nil {
			h.Error(errx)
		}
	}

	return nil, cmd.ErrorCode_Success
}

// 检查给定的flag是否已经获得
func (h *StoryFlagHandler) checkExistFlags(flagStrs ...string) bool {
	flagData := h.actor.GetStoryFlagData()

	for _, flagStr := range flagStrs {
		err, flag, val := datahelper.GetFlagVal(flagStr)
		if err != nil {
			return false
		}
		if dbFlag, ok := flagData.Flags[flag]; !ok || dbFlag.Val != val {
			h.Debugf("flag未完成, flag:%s, val:%d", flag, val)
			return false
		}
	}
	return true
}
