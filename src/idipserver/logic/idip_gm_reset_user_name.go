package logic

import (
	"encoding/json"
	"fmt"
	"gitee.com/aniwar2/aniwar/src/common/db"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"github.com/dapr/go-sdk/service/common"
	"net/http"
	"strconv"
)

// 请求参数结构
type ResetUserNameReq struct {
	ReqType        string `json:"type"`           // 固定值 “init_user_name”
	Uids           []int  `json:"uids"`           // 收件人ID数组，如[12312,31232…]
	ProhibitedTime int64  `json:"prohibitedTime"` // 改名禁止时间，可选，不填传0
}

// ResetUserName 重置玩家名称
func (s *IDIPServer) ResetUserName(out *common.Content, reqJson []byte) {

	// 解析数据
	req := ResetUserNameReq{}
	if err := json.Unmarshal(reqJson, &req); err != nil {
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), Internal_Error)
		return
	}

	// 失败项
	items := make([]*RetBaseItems, 0)
	for _, uid := range req.Uids {
		// 拿玩家的账号数据
		// uaid := s.GetUAID(sdkconstant.GenLilithUid(uid), 0)
		uaid := strconv.Itoa(uid)
		roleInfo := &pb.PServerRoleBaseInfo{}
		if err := s.getUserGameData(db.KeyUserBaseInfo(uaid), roleInfo); err != nil {
			items = append(items, &RetBaseItems{Ret: int32(pb.ErrorCode_InternalError), Info: Internal_Error})
			continue
		}

		roleInfo.Common.RoleName = fmt.Sprintf("%s:%d", "aniwar", roleInfo.Common.RoleId)
		key := db.KeyUserBaseInfo(uaid)
		kvTable, err := db.BuildKvTable(roleInfo, key)
		if err != nil {
			items = append(items, &RetBaseItems{Ret: int32(pb.ErrorCode_InternalError), Info: Internal_Error})
			continue
		}
		err = s.SaveMongoGame(key, kvTable, nil)
		if err != nil {
			items = append(items, &RetBaseItems{Ret: int32(pb.ErrorCode_InternalError), Info: Internal_Error})
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
