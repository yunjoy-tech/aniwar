package logic

import (
	"context"
	"encoding/json"
	"net/http"

	"gitee.com/bychannel/musae/framework/utils"

	"gitee.com/bychannel/aniwar/src/common/actor/stub"
	"gitee.com/bychannel/aniwar/src/proto/pb"
	"gitee.com/bychannel/musae/framework/base"
	"gitee.com/bychannel/musae/framework/logger"
	"github.com/dapr/go-sdk/service/common"
	"google.golang.org/protobuf/proto"
)

// 请求参数结构
type SendUserOfflineDataReq struct {
	// ReqType    string       `json:"type"`        // 固定值 “send_mail”
	SvrId int32 `json:"svr_id"` // 服务器ID
	Uids  []int `json:"uids"`   // 玩家uid,[123, 124, 125, …]
	// OperateType pb.OfflineOperateType // 操作类型
	Params     map[int32]int32 // 操作参数
	AffectTime int64           `json:"affect_time"` // 生效时间，可选
	ExpireTime int64           `json:"expire_time"` // 过期时间，可选
}

// SendUserOfflineData 记录玩家离线数据
func (s *IDIPServer) SendUserOfflineData(out *common.Content, reqJson []byte) {

	// 解析数据
	req := SendUserOfflineDataReq{}
	if err := json.Unmarshal(reqJson, &req); err != nil {
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), Internal_Error)
		return
	}

	rpcCall := &pb.S2SSaveOfflineDataReq{
		// ODataStatus: pb.OfflineDataStatus_Need_exec,
		// OperateType: req.OperateType,
		Params: req.Params,
	}

	items := make([]*RetItems, 0)
	errCheck := func(err error, id int) {
		if err != nil {
			items = append(items, &RetItems{
				SvrId:  0,
				UserId: int32(id),
				Ret:    int32(pb.ErrorCode_InternalError),
				Info:   err.Error(),
			})
			logger.Error("add error", err)
		}
	}

	for _, uid := range req.Uids {
		uaid, err := s.GetUAIDByRoleId(uint64(uid))
		if err != nil {
			errCheck(err, uid)
			continue
		}
		userStub := stub.NewUserStub(uaid)
		data, err := proto.Marshal(rpcCall)
		if err != nil {
			errCheck(err, uid)
			continue
		}
		in := &base.ProtoMsg{
			AppId:   s.AppId,
			MsgId:   int32(pb.Protocols_PS2AS_S2SSaveOfflineDataReq),
			UserId:  uaid,
			RoleId:  0,
			UAID:    uaid,
			Data:    data,
			ErrCode: 0,
			// GUID:    utils.GenIntUUID(),
			ServerReqIdx: utils.GenIntUUID(),
			Topic:        "",
		}
		s.ImpActorStub(userStub)
		rsp, err := userStub.UserInvoke(context.Background(), in)
		if rsp.ErrCode != RET_CODE_SUCCESS || err != nil {
			errCheck(err, uid)
			continue
		}
	}

	// 返回结果数据
	if len(items) > 0 {
		RetCommonMsg(out, http.StatusInternalServerError, int32(RET_CODE_FAIL), items)
	} else {
		RetCommonMsg(out, http.StatusOK, int32(RET_CODE_SUCCESS), items)
	}
}
