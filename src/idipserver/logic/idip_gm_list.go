package logic

import (
	"encoding/json"
	"gitee.com/bychannel/aniwar/src/proto/pb"
	"gitee.com/bychannel/musae/framework/global"
	"gitee.com/bychannel/musae/framework/logger"
	"github.com/dapr/go-sdk/service/common"
	"net/http"
)

type GMListReq struct {
	ReqType string `json:"type"` // 固定值
}

type GmHelpRsp struct {
	CmdName string `json:"cmd_name"`
	Desc    string `json:"desc"`
	Help    string `json:"help"`
}

func (s *IDIPServer) GetUserGMList(out *common.Content, reqJson []byte) {
	rpcCall := &pb.S2AS_GetGmListReq{GetGlobalGM: false}
	rspData, err := s.SvcInvoke(global.ACTOR_SVC, "", 0, "", rpcCall)
	if err != nil {
		logger.Error("GetUserGMList error", err)
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_ParamError), err)
		return
	}
	gmList := make([]*GmHelpRsp, 0)
	if err := json.Unmarshal(rspData, &gmList); err != nil {
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_ParamError), err)
		return
	}
	RetCommonMsg(out, http.StatusOK, int32(RET_CODE_SUCCESS), gmList)
}

func (s *IDIPServer) GetGlobalGMList(out *common.Content, reqJson []byte) {
	// rpcCall := &pb.S2AS_GetGmListReq{GetGlobalGM: true}
	// rspData, err := s.SvcInvoke(server.ACTOR_SVC, "", 0, "", rpcCall)
	// if err != nil {
	//	logger.Error("GetUserGMList error", err)
	//	RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_ParamError), err)
	//	return
	// }
	// gmList := make([]*GmHelpRsp,0)
	// if err:=json.Unmarshal(rspData,&gmList); err != nil {
	//	RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_ParamError), err)
	//	return
	// }
	// RetCommonMsg(out, http.StatusOK, int32(RET_CODE_SUCCESS), gmList)

	RetCommonMsg(out, http.StatusOK, int32(RET_CODE_SUCCESS), GlobalCmdList)
}
