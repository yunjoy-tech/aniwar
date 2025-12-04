package logic

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"gitlab.musadisca-games.com/wangxw/musae/framework/utils"

	"github.com/dapr/go-sdk/service/common"
	myCommon "gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/guid"
	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"
	"google.golang.org/protobuf/proto"
)

// 请求参数结构
type SendUserMail2Req struct {
	ReqType      string                       `json:"type"`         // 固定值 “send_multi_lang_mail”
	SvrId        int32                        `json:"svr_id"`       // 服务器ID
	SenderName   string                       `json:"sender_name"`  // 发件人名称
	RecverUids   []int32                      `json:"recver_uids"`  // 收件人ID数组，如[12312,31232…]
	Mails        map[string]map[string]string `json:"mails"`        // 多语言邮件对象，map[string]map[string]string 解析 keys为"title" "context" "sender_name"
	Currency     int                          `json:"currency"`     // 要发送的一级货币数量
	Coins        []CommonCoin                 `json:"coins"`        // 要发送的次级货币coin数组，coin定义见这里
	Items        []CommonItem                 `json:"items"`        // 要发送的道具item数组，item定义见这里
	ExpireTime   int64                        `json:"expire_time"`  // 过期时间，可选
	Questionaire CommonQuestion               `json:"questionaire"` // 问卷
}

// SendUserMail2 发送多语言玩家邮件
func (s *IDIPServer) SendUserMail2(out *common.Content, reqJson []byte) {

	// 解析数据
	req := SendUserMail2Req{}
	if err := json.Unmarshal(reqJson, &req); err != nil {
		RetCommonMsg(out, http.StatusInternalServerError, int32(cmd.ErrorCode_InternalError), Internal_Error)
		return
	}

	// 附件奖励
	attachments := make([]*cmd.ItemReward, 0)
	for _, item := range req.Items {
		itemId, err := strconv.Atoi(item.ItemId)
		if err != nil {
			RetCommonMsg(out, http.StatusInternalServerError, int32(cmd.ErrorCode_ParamError), Param_Error)
			return
		}
		if item.ItemCount <= 0 {
			continue
		}
		attachments = append(attachments, &cmd.ItemReward{
			ItemId: uint32(itemId),
			Num:    uint32(item.ItemCount),
		})
	}
	// 货币配置支持
	if req.Currency > 0 {
		attachments = append(attachments, &cmd.ItemReward{
			ItemId: myCommon.CURRENCY_ITEM_ID_2005,
			Num:    uint32(req.Currency),
		})
	}
	for _, coin := range req.Coins {
		id, err := strconv.Atoi(coin.CoinName)
		if err != nil || coin.CoinValue <= 0 {
			continue
		}
		attachments = append(attachments, &cmd.ItemReward{
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
	langMap := make(map[string]*cmd.S2S_SendGMAddUserMailReqLanguage)
	for lang, m := range req.Mails {
		langMap[lang] = &cmd.S2S_SendGMAddUserMailReqLanguage{Content: m}
	}

	mailID := s.GenGUID(guid.GUID_MAIL)
	if mailID == 0 {
		RetCommonMsg(out, http.StatusInternalServerError, int32(cmd.ErrorCode_InternalError), Internal_Error)
		return
	}
	// 构建邮件数据
	mail := &cmd.PMailInfo{
		Id:          int64(mailID),
		Title:       "",
		Content:     "",
		Sender:      "",
		SendType:    myCommon.MAIL_SEND_TYPE_USER,
		MailType:    myCommon.MAIL_TYPE_5,
		CreateTime:  time.Now().Unix(),
		ExpireTime:  req.ExpireTime,
		IsRead:      myCommon.MAIL_STATUS_UNREAD,
		IsReceived:  myCommon.MAIL_STATUS_UNRECEIVE,
		Attachments: attachments,
		GiftsType:   myCommon.MAIL_REWARD_TYPE_OTHER,
	}

	reqData := &cmd.S2S_SendGMAddUserMailReq{
		AddMail:      mail,
		LangMap:      langMap,
		QuestionId:   questionId,
		QuestionType: questionType,
	}
	data, err := proto.Marshal(reqData)
	if err != nil {
		RetCommonMsg(out, http.StatusInternalServerError, int32(cmd.ErrorCode_InternalError), Internal_Error)
		return
	}

	retItems := make([]*RetItems, 0)
	for _, uid := range req.RecverUids {
		uaid, err := s.GetUAIDByRoleId(uint64(uid))
		if err != nil {
			errCheck(err, int(uid), retItems)
			continue
		}

		in := &base.ProtoMsg{
			AppId:   s.AppId,
			MsgId:   int32(cmd.Protocols_PS2AS_ReceiveGMAddMailReq),
			UserId:  uaid,
			RoleId:  0,
			UAID:    uaid,
			Data:    data,
			ErrCode: 0,
			//GUID:    utils.GenIntUUID(),
			ServerReqIdx: utils.GenIntUUID(),
			Topic:        "",
		}
		rsp, err := s.UserInvoke(uaid, in)
		if rsp.ErrCode != RET_CODE_SUCCESS || err != nil {
			errCheck(err, int(uid), retItems)
			continue
		}
	}

	// 返回结果数据
	if len(retItems) > 0 {
		RetCommonMsg(out, http.StatusInternalServerError, int32(RET_CODE_FAIL), retItems)
	} else {
		RetCommonMsg(out, http.StatusOK, int32(RET_CODE_SUCCESS), retItems)
	}
}
