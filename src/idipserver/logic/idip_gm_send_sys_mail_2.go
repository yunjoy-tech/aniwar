package logic

import (
	"encoding/json"
	myCommon "gitee.com/aniwar2/aniwar/src/common"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/framework/gamelib/guid"
	"gitee.com/aniwar2/musae/framework/global"
	"gitee.com/aniwar2/musae/framework/logger"
	"github.com/dapr/go-sdk/service/common"
	"net/http"
	"strconv"
	"time"
)

// 请求参数结构
type SendSysMail2Req struct {
	ReqType      string                       `json:"type"`          // 固定值 “send_multi_lang_global_mail”
	SvrIds       []int                        `json:"svr_ids"`       // 服务器ID列表，空数组表示向所有服务器发送全服邮件
	BornsvrIds   []int                        `json:"bornsvr_ids"`   // 出生服服务器ID列表，不填表示不添加该项限制，不填则不传该参数
	Plat         string                       `json:"plat"`          // 给指定渠道所有玩家发邮件，渠道列表由研发商提供，无该字段表示所有渠道发
	SenderName   string                       `json:"sender_name"`   // 发件人名称
	Mails        map[string]map[string]string `json:"mails"`         // 多语言邮件对象，map[string]map[string]string 解析
	Currency     int                          `json:"currency"`      // 要发送的一级货币数量
	Coins        []CommonCoin                 `json:"coins"`         // 要发送的次级货币coin数组，coin定义见这里
	Items        []CommonItem                 `json:"items"`         // 要发送的道具item数组，item定义见这里
	ExpireTime   int                          `json:"expire_time"`   // 过期时间，可选
	Level        int                          `json:"level"`         // 最低发放等级，可选
	VipLevelMin  int                          `json:"vip_level_min"` // 发放玩家的最低vip等级，0表示无最低等级限制
	VipLevelMax  int                          `json:"vip_level_max"` // 发放玩家的最高vip等级，0表示无最高等级限制
	Os           string                       `json:"os"`            // 发放平台，android或ios，空字符串表示两者都包含，可选
	MailCategory string                       `json:"mail_category"` // fixme 邮件种类，分类为邮件（GMT_Mail）,公告（GMT_Announce）,邮件和公告（GMT_MailAndAnnounce）
	Questionaire CommonQuestion               `json:"questionaire"`  // 问卷
}

// SendSysMail2 发送多语言系统邮件
func (s *IDIPServer) SendSysMail2(out *common.Content, reqJson []byte) {

	// 解析数据
	req := SendSysMail2Req{}
	if err := json.Unmarshal(reqJson, &req); err != nil {
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), Internal_Error)
		return
	}

	// 附件奖励
	attachments := make([]*pb.ItemReward, 0)
	for _, item := range req.Items {
		itemId, err := strconv.Atoi(item.ItemId)
		if err != nil {
			RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_ParamError), Param_Error)
			return
		}
		if item.ItemCount <= 0 {
			continue
		}
		attachments = append(attachments, &pb.ItemReward{
			ItemId: uint32(itemId),
			Num:    uint32(item.ItemCount),
		})
	}
	// 货币配置支持
	if req.Currency > 0 {
		attachments = append(attachments, &pb.ItemReward{
			ItemId: myCommon.CURRENCY_ITEM_ID_2005,
			Num:    uint32(req.Currency),
		})
	}
	for _, coin := range req.Coins {
		id, err := strconv.Atoi(coin.CoinName)
		if err != nil || coin.CoinValue <= 0 {
			continue
		}
		attachments = append(attachments, &pb.ItemReward{
			ItemId: uint32(id),
			Num:    uint32(coin.CoinValue),
		})
	}
	// 问卷支持
	var questionId string
	var questionType int32
	if req.Questionaire.QuestionType == "1" {
		if req.Questionaire.Sid != "" {
			// 单语言邮件
			questionId = req.Questionaire.Sid
			questionType = myCommon.QUESTION_LANG_TYPE_SINGLE
		} else if req.Questionaire.SourceId != "" {
			// 多语言邮件
			questionId = req.Questionaire.SourceId
			questionType = myCommon.QUESTION_LANG_TYPE_MULTI
		} else {
			logger.Errorf("unrealized question type: %+v", req.Questionaire)
		}
	}

	// 多语言支持
	langMap := make(map[string]*pb.PSysMailInfoLanguage)
	for lang, m := range req.Mails {
		langMap[lang] = &pb.PSysMailInfoLanguage{Content: m}
	}

	mailID := uint64(guid.GenIntUuid())
	if mailID == 0 {
		logger.Error("get mail id error")
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_ParamError), Param_Error)
		return
	}

	// 构建邮件数据
	mail := &pb.PSysMailInfo{
		Id:           int64(mailID),
		Title:        "",
		Content:      "",
		Sender:       "",
		SendType:     myCommon.MAIL_SEND_TYPE_USER,
		MailType:     myCommon.MAIL_TYPE_5,
		CreateTime:   time.Now().Unix(),
		ExpireTime:   int64(req.ExpireTime),
		Attachments:  attachments,
		QuestionId:   questionId,
		QuestionType: questionType,
		Lang:         "",
		LangMap:      langMap,
		Plat:         req.Plat,
		Level:        int32(req.Level),
		MinVip:       int32(req.VipLevelMin),
		MaxVip:       int32(req.VipLevelMax),
		Os:           req.Os,
	}

	reqMsg := &pb.S2S_SendGMAddMailReq{AddMail: mail}
	_, err := s.SvcInvoke(global.ACTOR_SVC, "", 0, "", reqMsg)
	if err != nil {
		logger.Error("add sys mail error", err)
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_ParamError), Param_Error)
		return
	}
	// 返回结果数据
	RetCommonMsg(out, http.StatusOK, int32(RET_CODE_SUCCESS), SUCCESS)
}
