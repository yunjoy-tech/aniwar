package logic

import (
	"context"
	"encoding/json"
	"gitee.com/aniwar2/musae/gamelib/guid"
	"net/http"

	"gitee.com/aniwar2/aniwar/src/common/actor/stub"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/base"
	"gitee.com/aniwar2/musae/logger"
	"github.com/dapr/go-sdk/service/common"
	"google.golang.org/protobuf/proto"
)

// 请求参数结构
type SendGiftPackageReq struct {
	ReqType string              `json:"type"`    // 固定值 “give_iap_package”
	Content []CommonGiftPackage `json:"content"` // 礼包内容
}

// SendGiftPackage 发送礼包
func (s *IDIPServer) SendGiftPackage(out *common.Content, reqJson []byte) {

	// 解析数据
	req := SendGiftPackageReq{}
	if err := json.Unmarshal(reqJson, &req); err != nil {
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), Internal_Error)
		return
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
	for _, p := range req.Content {
		uaid, err := s.GetUAIDByRoleId(uint64(p.PlayerId))
		if err != nil {
			errCheck(err, p.PlayerId)
			continue
		}
		rpcCall := &pb.S2SReceiveGMAddGiftReq{PackageId: int32(p.PackageId)}
		userStub := stub.NewUserStub(uaid)
		data, err := proto.Marshal(rpcCall)
		if err != nil {
			errCheck(err, p.PlayerId)
			continue
		}
		in := &base.ProtoMsg{
			AppId:   s.AppId,
			MsgId:   int32(pb.Protocols_PS2AS_ReceiveGMAddGiftReq),
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
			errCheck(err, p.PlayerId)
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
