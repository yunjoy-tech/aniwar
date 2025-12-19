package logic

import (
	"context"
	"errors"
	"fmt"
	"gitee.com/aniwar2/musae/gamelib/guid"
	"net/http"
	"strconv"
	"strings"

	"gitee.com/aniwar2/aniwar/src/common/sdkconstant"
	"gitee.com/aniwar2/aniwar/src/common/sdkconstant/sdksign"

	myCommon "gitee.com/aniwar2/aniwar/src/common"
	"gitee.com/aniwar2/aniwar/src/common/conf"
	myUtils "gitee.com/aniwar2/aniwar/src/common/utils"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/base"
	"gitee.com/aniwar2/musae/logger"
	"github.com/dapr/go-sdk/service/common"
	"google.golang.org/protobuf/proto"
)

type GmtHandlerFunc = func(*common.Content, []byte)

var GmtHandlerMap = make(map[string]GmtHandlerFunc)

// 初始化handler映射
func (s *IDIPServer) InitMap() {
	registerGmtHandler(REQUEST_TYPE_QUERY_USER, s.QueryUserInfo)               // 查询玩家信息
	registerGmtHandler(REQUEST_TYPE_MODIFY_WHITE_LIST, s.ModifyWhiteList)      // 修改登录白名单
	registerGmtHandler(REQUEST_TYPE_SEND_USER_MAIL, s.SendUserMail)            // 发送玩家邮件
	registerGmtHandler(REQUEST_TYPE_SEND_SYS_MAIL, s.SendSysMail)              // 发送全服邮件
	registerGmtHandler(REQUEST_TYPE_SEND_MULTI_LANG_MAIL, s.SendUserMail2)     // 发送多语言玩家邮件
	registerGmtHandler(REQUEST_TYPE_SEND_MULTI_LANG_SYS_MAIL, s.SendSysMail2)  // 发送多语言全服邮件
	registerGmtHandler(REQUEST_TYPE_ADD_SINGLE_RESOURCE, s.AddSingleResource)  // 单类资源增加
	registerGmtHandler(REQUEST_TYPE_SUB_SINGLE_RESOURCE, s.SubSingleResource)  // 单类资源扣除
	registerGmtHandler(REQUEST_TYPE_ADD_BATCH_RESOURCE, s.AddBatchResource)    // 批量增加资源
	registerGmtHandler(REQUEST_TYPE_SUB_BATCH_RESOURCE, s.SubBatchResource)    // 批量扣除资源
	registerGmtHandler(REQUEST_TYPE_SEND_GIFT_PACKAGE, s.SendGiftPackage)      // 发放礼包
	registerGmtHandler(REQUEST_TYPE_SEND_MULTI_RESOURCE, s.SendMultiResource)  // 发放自定义资源
	registerGmtHandler(REQUEST_TYPE_RESET_USER_NAME, s.ResetUserName)          // 重置玩家昵称
	registerGmtHandler(REQUEST_TYPE_QUERY_USER_GM_LIST, s.GetUserGMList)       // 获取玩家个人cmd
	registerGmtHandler(REQUEST_TYPE_QUERY_GLOBAL_GM_LIST, s.GetGlobalGMList)   // 获取全服cmd
	registerGmtHandler(REQUEST_TYPE_QUERY_USER_EXCUTE_CMD, s.ExcuteUserGM)     // 执行个人cmd
	registerGmtHandler(REQUEST_TYPE_QUERY_GLOBAL_EXCUTE_CMD, s.ExcuteGlobalGM) // 执行全服cmd

}

func (s *IDIPServer) GMTHandler(ctx context.Context, in *common.InvocationEvent) (out *common.Content, err error) {
	defer func() {
		if errx := recover(); errx != any(nil) {
			logger.Error("GMTHandler failed, err: ", errx)
		}
	}()

	if in == nil {
		err = errors.New("nil invocation parameter")
	}
	logger.Debugf("[IDIP] InvokeHandler - ContentType:%s, Verb:%s, QueryString:%s, len:%v", in.ContentType, in.Verb, in.QueryString, len(in.Data))

	out = &common.Content{
		ContentType: in.ContentType,
		DataTypeURL: in.DataTypeURL,
	}

	// 前置处理逻辑
	reqJson, code, errMsg := s.PreHandle(in, conf.GMT().ApiSecret)
	if code != pb.ErrorCode_Success {
		RetCommonMsg(out, http.StatusInternalServerError, int32(code), errMsg)
		return
	}
	// 记录此次操作数据
	err = s.RecordOperation(lilithKey(), reqJson)
	if err != nil {
		logger.Error("report operation failed", err)
	}

	m, err := myUtils.JsonToMap(reqJson)
	if err != nil {
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), Internal_Error)
		return
	}

	// 根据type调用处理
	var handler GmtHandlerFunc
	if typ, ok := m["type"].(string); !ok {
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), Internal_Error)
		return
	} else {
		handler = GmtHandlerMap[typ]
	}
	if handler == nil {
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_UnrealizedTypeError), Unrealized_Type_Error)
		return
	}
	handler(out, reqJson)

	return out, nil
}

func (s *IDIPServer) QuestionRewardHandler(ctx context.Context, in *common.InvocationEvent) (out *common.Content, err error) {
	defer func() {
		if errx := recover(); errx != any(nil) {
			logger.Error("QuestionRewardHandler failed, err: ", errx)
		}
	}()

	if in == nil {
		err = errors.New("nil invocation parameter")
	}
	logger.Debugf("[IDIP] InvokeHandler - ContentType:%s, Verb:%s, QueryString:%s, len:%v", in.ContentType, in.Verb, in.QueryString, len(in.Data))

	out = &common.Content{
		ContentType: in.ContentType,
		DataTypeURL: in.DataTypeURL,
	}

	reqJson, code, errMsg := s.PreHandle(in, conf.Question().SecretKey)
	if code != pb.ErrorCode_Success {
		logger.Debugf("QuestionRewardHandler prehandle failed, %s", errMsg)
		RetCommonMsg(out, http.StatusInternalServerError, int32(code), errMsg)
		return
	}

	// 记录此次操作数据
	err = s.RecordOperation(lilithKey(), reqJson)
	if err != nil {
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), Internal_Error)
		return
	}

	// reqStr原始数据结构:
	// type		string	固定值"recv_reward"
	// role_key	string	玩家身份标识 "open_id;user_id"
	// sid 		string  问卷id（如是多语言问卷，即为多语言问卷id）
	m, err := myUtils.JsonToMap(reqJson)
	if err != nil {
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), Internal_Error)
		return
	}
	mType := m["type"]
	mRole := m["role_key"]
	mSid := m["sid"]
	if mType == nil || mRole == nil || mSid == nil {
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_ParamError), Param_Error)
		return
	}

	if mType.(string) != "recv_reward" {
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_ParamError), Param_Error)
		return
	}

	// 给玩家发奖
	arr := strings.Split(mRole.(string), ";")
	uid, err := strconv.Atoi(arr[1])
	if err != nil {
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_ParamError), Param_Error)
		return
	}

	uaid, err := s.GetUAIDByRoleId(uint64(uid))
	if err != nil {
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), Internal_Error)
		return
	}

	data, err := proto.Marshal(&pb.S2S_SendQuestionRewardReq{Sid: mSid.(string)})
	if err != nil {
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), Internal_Error)
		return
	}
	rsp, err := s.UserInvoke(uaid, &base.ProtoMsg{
		AppId:   s.AppId,
		MsgId:   int32(pb.Protocols_PS2S_SendQuestionRewardReq),
		UserId:  uaid,
		RoleId:  0,
		UAID:    uaid,
		Data:    data,
		ErrCode: 0,
		// GUID:    utils.GenIntUUID(),
		ServerReqIdx: guid.GenIntUuid(),
		Topic:        "",
	})
	if rsp.ErrCode != RET_CODE_SUCCESS || err != nil {
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), Internal_Error)
		return
	}

	// 成功通知
	RetCommonMsg(out, http.StatusOK, int32(RET_CODE_SUCCESS_200), SUCCESS)
	return out, nil
}

func (s *IDIPServer) DelAccountHandler(ctx context.Context, in *common.InvocationEvent) (out *common.Content, err error) {
	defer func() {
		if errx := recover(); errx != any(nil) {
			logger.Error("DelAccountHandler failed, err: ", errx)
		}
	}()

	if in == nil {
		err = errors.New("nil invocation parameter")
	}
	logger.Debugf("[IDIP] InvokeHandler - ContentType:%s, Verb:%s, QueryString:%s, len:%v", in.ContentType, in.Verb, in.QueryString, len(in.Data))

	out = &common.Content{
		ContentType: in.ContentType,
		DataTypeURL: in.DataTypeURL,
	}

	// 参数
	// reqStr原始数据结构
	// app_id	 integer	游戏应用app_Id
	// app_uid	 integer	用户app_uid
	// timestamp integer	注销账号的时间点
	// id		 integer	注销事件标识，gmt注销时该值为0
	// sign	  	 string     同park支付签名方法，见下文描述
	argsMap := sdksign.ParseUrlArgs(string(in.Data))

	// 验签
	if b := sdksign.ParkSignVerify(argsMap, []string{"sign"}); !b {
		logger.Debugf("request sign failed.")
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_SignCheckError), Sign_Check_Error)
		return
	}

	// 记录此次操作数据
	err = s.RecordOperation(lilithKey(), in.Data)
	if err != nil {
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), err)
		return
	}

	uaid := sdkconstant.GenLilithUid(argsMap["app_uid"].(int))
	// 游戏方不删除数据，进行永久封号处理
	data, err := proto.Marshal(&pb.S2AS_ExcuteGMReq{
		CmdName: myCommon.GM_BANNED,
		OptVal:  fmt.Sprintf("%v %s", 365*24*3600*10, "账号数据删除"),
	})
	if err != nil {
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), err)
		return
	}

	callData := &base.ProtoMsg{
		AppId:   s.AppId,
		MsgId:   int32(pb.Protocols_PS2AS_GmExecuteReq),
		UserId:  uaid,
		RoleId:  0,
		UAID:    uaid,
		Data:    data,
		ErrCode: 0,
		// GUID:    utils.GenIntUUID(),
		ServerReqIdx: guid.GenIntUuid(),
		Topic:        "",
	}
	rsp, err := s.UserInvoke(uaid, callData)
	if rsp.ErrCode != RET_CODE_SUCCESS || err != nil {
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), err)
		return
	}

	// 成功通知
	RetCommonMsg(out, http.StatusOK, int32(RET_CODE_SUCCESS_200), SUCCESS)
	return out, nil
}
