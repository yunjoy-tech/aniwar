package logic

import (
	"context"
	"encoding/json"
	"gitee.com/aniwar2/musae/gamelib/guid"
	"net/http"
	"strconv"

	"gitee.com/aniwar2/aniwar/src/common/actor/stub"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/base"
	"gitee.com/aniwar2/musae/logger"
	"github.com/dapr/go-sdk/service/common"
	"google.golang.org/protobuf/proto"
)

// 请求参数结构
type AddSingleResourceReq struct {
	ReqType string `json:"type"`    // 固定值 “gm_add_resource”
	Uids    []int  `json:"uids"`    // 玩家uid,[123, 124, 125, …]
	Restype string `json:"restype"` // 资源类型
	Num     int    `json:"num"`     // 资源数量
}

// AddSingleResource 增加单类资源
func (s *IDIPServer) AddSingleResource(out *common.Content, reqJson []byte) {

	// 解析数据
	req := &AddSingleResourceReq{}
	if err := json.Unmarshal(reqJson, req); err != nil {
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), Internal_Error)
		return
	}
	itemId, err := strconv.Atoi(req.Restype)
	if err != nil {
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), Param_Error)
		return
	}
	rpcCall := &pb.S2SReceiveGMAddResReq{Items: map[int32]int32{int32(itemId): int32(req.Num)}}
	items := GMAddItem(s, req.Uids, rpcCall)
	// 返回结果数据
	if len(items) > 0 {
		RetCommonMsg(out, http.StatusInternalServerError, int32(RET_CODE_FAIL), items)
	} else {
		RetCommonMsg(out, http.StatusOK, int32(RET_CODE_SUCCESS), items)
	}
}

func GMAddItem(s *IDIPServer, uids []int, rpcCall *pb.S2SReceiveGMAddResReq) []*RetItems {
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
	for _, uid := range uids {
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
			MsgId:   int32(pb.Protocols_PS2AS_ReceiveGMAddResReq),
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
			errCheck(err, uid)
			continue
		}
	}
	return items
}
