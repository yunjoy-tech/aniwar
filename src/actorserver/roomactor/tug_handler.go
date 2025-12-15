package roomactor

import (
	"context"
	"fmt"
	"gitee.com/aniwar2/musae/framework/gamelib/guid"
	"time"

	"gitee.com/aniwar2/aniwar/src/common/datahelper"

	myUtils "gitee.com/aniwar2/aniwar/src/common/utils"

	"gitee.com/aniwar2/musae/framework/base"

	"gitee.com/aniwar2/musae/framework/threading"

	"gitee.com/aniwar2/aniwar/src/common/db"

	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/framework/service"
	"google.golang.org/protobuf/proto"
)

type TugHandler struct {
	*USBaseHandler
	_tickStop chan struct{}
}

func NewTugHandler(actor *RoomActor) *TugHandler {
	h := &TugHandler{USBaseHandler: NewUSBaseHandler(actor, "TugHandler")}
	h.ChildHandler = h

	actor.RegisterProtoHandler(int32(pb.Protocols_PC2LS_TugClickReq), h.TugClickReq) // 拔河游戏开始 C2S

	return h
}

// Init 初始化模块数据
func (h *TugHandler) Init() error {

	return nil
}

func (h *TugHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*pb.Tug); ok {
		h.actor.Tug = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *TugHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyGameTugData(h.actor.ID()), h.actor.RoomData.Tug
}

func (h *TugHandler) EnterGame() error {
	// implement me
	panic("implement me")
}

func (h *TugHandler) DailyRefresh() error {
	// implement me
	panic("implement me")
}

func (h *TugHandler) TugClickReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var (
		err       error
		playerUid = in.UserId // 玩家id
	)

	if h.actor.Tug.TugState != pb.TugState_ts_playing {
		// 拔河已结束或未开始 - 忽略本次请求 - 不做errCode响应
		return &pb.LS2C_TugClickRes{}, nil, int32(pb.ErrorCode_Success)
	}

	var req pb.C2LS_TugClickReq
	err = in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	for _, scoreInfo := range h.actor.Tug.Scores {
		if scoreInfo.PlayerUid == playerUid {
			scoreInfo.Score += req.TotalClickCount
			break
		}
	}

	h.tryGameOver()

	// 持久化
	err = h.Cache2Redis()
	if err != nil {
		return nil, err, int32(pb.ErrorCode_SaveDBError)
	}

	rsp := &pb.LS2C_TugClickRes{TotalClickCount: req.TotalClickCount}

	return rsp, nil, int32(pb.ErrorCode_Success)
}

func (h *TugHandler) pushTugInfoNtf() {
	ntf := &pb.LS2C_TugInfoNtf{
		TugInfo: h.actor.Tug,
	}

	err := h.actor.Srv.Send2Gates(h.actor.Data.RoomId, h.actor.UserMap, ntf)
	if err != nil {
		return
	}
}

// 游戏开始
func (h *TugHandler) tugGameStart(roomModel pb.RoomModel) {
	scores := make([]*pb.TugScore, 0)
	for _, player := range h.actor.Data.Players {
		scores = append(scores, &pb.TugScore{
			PlayerUid: player.PlayerUid,
			Score:     0,
		})
	}

	h.actor.Tug = &pb.Tug{
		RoomGameInfo: &pb.RoomGameInfo{
			GameId:      guid.GenStrUuid(),
			TugStartSec: time.Now().Unix(),
		},
		TugState:  pb.TugState_ts_countDown,
		Scores:    scores,
		RoomModel: roomModel,
	}
	err := h.Cache2Redis()
	if err != nil {
		h.Errorf(err.Error())
		return
	}

	// 通知客户端
	h.pushTugInfoNtf()
	// 启动定时器
	h.tugCreateTick()
}

// 游戏结束
func (h *TugHandler) tugGameOver() {
	// 修改状态
	h.actor.Tug.TugState = pb.TugState_ts_game_over

	h.Infof("游戏结束, 结果:%v", h.actor.Tug)

	err := h.Cache2Redis()
	if err != nil {
		h.Errorf(err.Error())
		return
	}

	// 每秒同步次数据
	h.pushTugInfoNtf()

	// 返回room状态
	h.actor.RoomHandler.gameBack2Room()
}

func (h *TugHandler) tugCreateTick() {
	h.Debugf("开始房间内的定时器")
	threading.GoSafeWithParam(func(hh interface{}) {
		_h := hh.(*TugHandler)

		t := time.NewTicker(time.Second * 1)
		defer t.Stop()
		for {
			select {
			case <-_h._tickStop:
				_h.Debugf("定时器关闭")
				return
			case <-t.C:
				_h.tugExecTick()
			}
		}
	}, h)
}

func (h *TugHandler) tugExecTick() {
	h.Debugf("定时器调用...")
	if h.actor.Tug.TugState == pb.TugState_ts_game_over {
		// 游戏结束, 停止定时器
		h._tickStop <- struct{}{}
		return
	}

	h.actor.Tug.TugTick++
	if h.actor.Tug.TugTick == datahelper.GetMiniGameCountdown(h.actor.Tug.RoomModel) {
		// 3秒倒计时结束，游戏开始
		h.actor.Tug.TugState = pb.TugState_ts_playing
		err := h.Cache2Redis()
		if err != nil {
			h.Errorf(err.Error())
			return
		}

	} else {
		h.Debugf("多出来的时间累计...")
	}

	if !h.tryGameOver() {
		// 游戏未结束, 每秒同步次数据
		h.pushTugInfoNtf()
	}
}

func (h *TugHandler) tryGameOver() bool {
	var (
		diff int32 = 0
	)

	if h.actor.Tug.TugTick == datahelper.GetMiniGameTotalSec(h.actor.Tug.RoomModel) {
		h.Infof("时间到了，游戏结束")
		goto OnGameOver
	}

	diff = getMaxDiff(h.actor.Tug.Scores)
	if diff > datahelper.GetMiniGameWinCondition(h.actor.Tug.RoomModel, datahelper.MiniGameWinTypeClick) {
		h.Infof("分差到了，游戏结束")
		goto OnGameOver
	}
	return false

OnGameOver:
	h.tugGameOver()
	return true
}

func getMaxDiff(scores []*pb.TugScore) int32 {
	var maxDiff int32 = 0
	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			eachDiff := myUtils.Abs(scores[i].Score - scores[j].Score)
			if eachDiff > maxDiff {
				maxDiff = eachDiff
			}
		}
	}

	return maxDiff
}
