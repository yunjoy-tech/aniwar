package logic

import (
	"context"
	"encoding/json"
	"gitee.com/aniwar2/aniwar/src/common/datalog/taptap"
	"gitee.com/aniwar2/musae/gamelib/guid"
	"net/http"
	"strings"

	"gitee.com/aniwar2/aniwar/src/common/actor/stub"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/base"
	"github.com/dapr/go-sdk/service/common"
	"google.golang.org/protobuf/proto"
)

type ExcuteUserGMReq struct {
	ReqType     string `json:"type"`         // 固定值 query_excute_cmd
	CmdName     string `json:"cmd_name"`     // 指令名称
	OptVal      string `json:"opt_val"`      // 额外参数
	UserID      int    `json:"user_id"`      // 用户id
	EffectTime  int    `json:"effect_time"`  // 生效时间戳
	ExpiredTime int    `json:"expired_time"` // 过期时间戳
}

type ExcuteGlobalGMReq struct {
	ReqType  string `json:"type"`       // 固定值 query_excute_cmd2
	CmdName  string `json:"cmd_name"`   // 指令名称
	OptVal   string `json:"opt_val"`    // 额外参数
	SvrIDMax int    `json:"svr_id_max"` // 服务器最小值
	SvrIDMin int    `json:"svr_id_min"` // 服务器最大值
}

func (s *IDIPServer) ExcuteUserGM(out *common.Content, reqJson []byte) {
	req := &ExcuteUserGMReq{}
	if err := json.Unmarshal(reqJson, req); err != nil {
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), err)
		return
	}
	uaid, err := s.GetUAIDByRoleId(uint64(req.UserID))
	if err != nil {
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), err)
		return
	}
	rpcCall := &pb.S2AS_ExcuteGMReq{CmdName: req.CmdName, OptVal: req.OptVal}
	userStub := stub.NewUserStub(uaid)
	data, err := proto.Marshal(rpcCall)
	if err != nil {
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), err)
		return
	}
	in := &base.ProtoMsg{
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
	s.ImpActorStub(userStub)
	rsp, err := userStub.UserInvoke(context.Background(), in)
	if rsp.ErrCode != RET_CODE_SUCCESS || err != nil {

		// 埋点
		taptap.GmCmdComm(req.CmdName, req.OptVal, "", req.UserID, "", "idipserver", string(out.Data), http.StatusInternalServerError)

		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), err)
		return
	}

	// 埋点
	taptap.GmCmdComm(req.CmdName, req.OptVal, "", req.UserID, "", "idipserver", string(out.Data), http.StatusOK)

	RetCommonMsg(out, http.StatusOK, int32(RET_CODE_SUCCESS), rsp.Data)
}

// GM指令
func (s *IDIPServer) ExcuteGlobalGM(out *common.Content, reqJson []byte) {
	req := &ExcuteGlobalGMReq{}
	if err := json.Unmarshal(reqJson, req); err != nil {
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), err)
		return
	}

	arr := strings.Split(req.OptVal, " ")
	// ret, err := s.HandleGlobalCmd(req.CmdName, []string{req.OptVal})
	ret, err := s.HandleGlobalCmd(req.CmdName, arr)
	out.Data = []byte(ret)
	if err != nil {

		// 埋点
		taptap.GmCmdComm(req.CmdName, req.OptVal, "", 0, "", "idipserver", string(out.Data), http.StatusInternalServerError)
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), err)
		return
	}

	// 埋点
	taptap.GmCmdComm(req.CmdName, req.OptVal, "", 0, "", "idipserver", string(out.Data), http.StatusOK)
	RetCommonMsg(out, http.StatusOK, int32(RET_CODE_SUCCESS), ret)
}
