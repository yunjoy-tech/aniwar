package logic

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"gitlab.musadisca-games.com/wangxw/musae/framework/utils"

	"github.com/dapr/go-sdk/service/common"
	myCommon "gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/pb"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/guid"
	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"
	"google.golang.org/protobuf/proto"
)

// 请求参数结构
type SendUserMailReq struct {
	ReqType      string         `json:"type"`         // 固定值 “send_mail”
	SvrId        int32          `json:"svr_id"`       // 服务器ID
	SenderName   string         `json:"sender_name"`  // 发件人名称
	RecverUids   []int32        `json:"recver_uids"`  // 收件人ID数组，如[12312,31232…]
	Title        string         `json:"title"`        // 邮件标题
	Context      string         `json:"context"`      // 邮件正文
	Currency     int32          `json:"currency"`     // 要发送的一级货币数量
	Coins        []CommonCoin   `json:"coins"`        // 要发送的次级货币coin数组，coin定义见这里
	Items        []CommonItem   `json:"items"`        // 要发送的道具item数组，item定义见这里
	ExpireTime   int64          `json:"expire_time"`  // 过期时间，可选
	Questionaire CommonQuestion `json:"questionaire"` // 问卷
}

// SendUserMail 发送玩家邮件
func (s *IDIPServer) SendUserMail(out *common.Content, reqJson []byte) {

	// 解析数据
	req := SendUserMailReq{}
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

	mailID := s.GenGUID(guid.GUID_MAIL)
	if mailID == 0 {
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_ParamError), Param_Error)
		return
	}
	// 构建邮件数据
	mail := &pb.PMailInfo{
		Id:          int64(mailID),
		Title:       req.Title,
		Content:     req.Context,
		Sender:      req.SenderName,
		SendType:    myCommon.MAIL_SEND_TYPE_USER,
		MailType:    myCommon.MAIL_TYPE_5,
		CreateTime:  time.Now().Unix(),
		ExpireTime:  req.ExpireTime,
		IsRead:      myCommon.MAIL_STATUS_UNREAD,
		IsReceived:  myCommon.MAIL_STATUS_UNRECEIVE,
		Attachments: attachments,
		GiftsType:   myCommon.MAIL_REWARD_TYPE_OTHER,
	}

	// 构建请求
	reqData := &pb.S2S_SendGMAddUserMailReq{
		AddMail:      mail,
		LangMap:      nil,
		QuestionId:   questionId,
		QuestionType: questionType,
	}
	data, err := proto.Marshal(reqData)
	if err != nil {
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), Internal_Error)
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
			MsgId:   int32(pb.Protocols_PS2AS_ReceiveGMAddMailReq),
			UserId:  uaid,
			RoleId:  0,
			UAID:    uaid,
			Data:    data,
			ErrCode: 0,
			// GUID:    utils.GenIntUUID(),
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
