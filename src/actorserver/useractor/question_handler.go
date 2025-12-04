package useractor

import (
	"context"
	"fmt"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/db"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/utils"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"
	"google.golang.org/protobuf/proto"
	"time"
)

type QuestionHandler struct {
	*UABaseHandler
}

func NewQuestionHandler(actor *UserActor) *QuestionHandler {
	h := &QuestionHandler{UABaseHandler: NewUABaseHandler(actor, "QuestionHandler")}
	h.ChildHandler = h

	actor.RegisterProtoHandler(int32(cmd.Protocols_PS2S_SendQuestionRewardReq), h.SendQuestionReward)
	return h
}

// Init 初始化模块数据
func (h *QuestionHandler) Init() error {
	// 初始化
	h.actor.Data.Question = &cmd.PUserQuestions{
		Createtime: time.Now().Unix(),
		Questions:  make(map[string]*cmd.PQuestion),
	}

	// 保存
	if err := h.SaveDB(true); err != nil {
		return err
	}

	h.Debug("init user question data success. player: %s", h.actor.ID())
	return nil
}

func (h *QuestionHandler) EnterGame() error {
	return nil
}

func (h *QuestionHandler) DailyRefresh() error {
	return nil
}

func (h *QuestionHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*cmd.PUserQuestions); ok {
		h.actor.Data.Question = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *QuestionHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyUserQuestion(h.actor.ID()), h.actor.Data.Question
}

func (h *QuestionHandler) SendQuestionReward(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req cmd.S2S_SendQuestionRewardReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	questionData := h.actor.GetQuestionData()
	// 是否重复领取
	question := questionData.Questions[req.Sid]
	if question == nil {
		return nil, fmt.Errorf("question not found %s", req.Sid), int32(cmd.ErrorCode_ParamError)
	}
	if question.QuestionState == common.QUESTION_STATE_READ {
		return nil, fmt.Errorf("repeated commit question %s", req.Sid), int32(cmd.ErrorCode_InternalError)
	}

	// 记录
	question.QuestionState = common.QUESTION_STATE_READ
	if err = h.SaveDB(); err != nil {
		return nil, err, int32(cmd.ErrorCode_SaveDBError)
	}

	// 发奖
	h.actor.MailHandler.AddUserMail(common.MAIL_TEMPLATE_5, utils.ConvertItem2(question.Attachments), h.actor.comData)

	return &cmd.S2S_SendQuestionRewardRes{}, nil, 0
}

func (h *QuestionHandler) AddQuestion(sid string, attachments []*cmd.ItemReward) error {
	questionData := h.actor.GetQuestionData()
	// 判定是否重复问卷
	_, ok := questionData.Questions[sid]
	if ok {
		return fmt.Errorf("repeated question sid %s", sid)
	}

	questionData.Questions[sid] = &cmd.PQuestion{
		QuestionSid:   sid,
		QuestionState: common.QUESTION_STATE_UNREAD,
		Attachments:   attachments,
	}

	if err := h.SaveDB(); err != nil {
		return err
	}

	h.Debugf("add question sid %s", sid)
	return nil
}
