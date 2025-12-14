package logic

import (
	"context"
	"encoding/json"
	"net/http"

	"gitee.com/aniwar2/musae/framework/utils"

	"gitee.com/aniwar2/aniwar/src/common/actor/stub"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/framework/base"
	"github.com/dapr/go-sdk/service/common"
	"google.golang.org/protobuf/proto"
)

// 请求参数结构
type SendMultiResourceReq struct {
	ReqType   string                `json:"type"`      // 固定值 “send_laotie_fuli”
	ExcelData []CommonMultiResource `json:"excelData"` // 上传excel文件对象，excelData格式如下
}

// SendMultiResource 给不同玩家发送资源(只发送道具)
func (s *IDIPServer) SendMultiResource(out *common.Content, reqJson []byte) {

	// 解析数据
	req := SendMultiResourceReq{}
	if err := json.Unmarshal(reqJson, &req); err != nil {
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), Internal_Error)
		return
	}

	items := make([]*RetBaseItems, 0)
	for _, each := range req.ExcelData {
		if each.Items != "" {
			// 增加的道具列表
			addMap := make(map[int32]int32)
			err := json.Unmarshal([]byte(each.Items), &addMap)
			if err != nil {
				items = append(items, &RetBaseItems{Ret: int32(pb.ErrorCode_InternalError), Info: Internal_Error})
				continue
			}
			rpcCall := &pb.S2SReceiveGMAddResReq{Items: addMap, Coins: map[int32]int32{}}
			for range GMAddItem(s, []int{each.Uid}, rpcCall) {
				items = append(items, &RetBaseItems{Ret: int32(pb.ErrorCode_InternalError), Info: Internal_Error})
			}
		} else if each.GiftCode != "" {
			if err := s.addGiftCode(each.Uid, each.GiftCode); err != nil {
				items = append(items, &RetBaseItems{Ret: int32(pb.ErrorCode_InternalError), Info: Internal_Error})
			}
		} else {
			items = append(items, &RetBaseItems{Ret: int32(pb.ErrorCode_InternalError), Info: Internal_Error})
		}
	}

	// 返回结果数据
	if len(items) > 0 {
		RetCommonMsg(out, http.StatusInternalServerError, int32(RET_CODE_FAIL), items)
	} else {
		RetCommonMsg(out, http.StatusOK, int32(RET_CODE_SUCCESS), items)
	}
}

func (s *IDIPServer) addGiftCode(uid int, code string) error {
	uaid, err := s.GetUAIDByRoleId(uint64(uid))
	if err != nil {
		return err
	}
	rpcCall := &pb.C2LS_UseGiftCodeReq{Code: code}
	userStub := stub.NewUserStub(uaid)
	data, err := proto.Marshal(rpcCall)
	if err != nil {
		return err
	}
	in := &base.ProtoMsg{
		AppId:   s.AppId,
		MsgId:   int32(pb.Protocols_PS2AS_ReceiveGMAddGiftCodeReq),
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
		return err
	}
	return nil
}
