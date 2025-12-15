package logic

import (
	"context"
	"encoding/json"
	"gitee.com/aniwar2/musae/framework/gamelib/guid"
	"net/http"

	"gitee.com/aniwar2/aniwar/src/common/actor/stub"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/framework/base"
	"github.com/dapr/go-sdk/service/common"
)

// 请求参数结构
type UserInfoReq struct {
	ReqType  string `json:"type"`      // 固定值 “query_user”
	SvrId    int32  `json:"svr_id"`    // 全服搜索（或者不分服架构的游戏）无该字段
	UserId   int    `json:"user_id"`   // 游戏内分配的角色唯一ID
	UserName string `json:"user_name"` // 游戏内角色名称
	OpenId   string `json:"open_id"`   // sdk厂商提供的账号唯一ID（说明3）
	// 1. UserId，UserName，OpenID三者仅会有一项有值
	// 2. UserName必须支持模糊查询
}

// 返回结果结构
type UserInfoRes struct {
	Users []*CommonUser `json:"users"`
}

// QueryUserInfo 查询玩家信息
func (s *IDIPServer) QueryUserInfo(out *common.Content, reqJson []byte) {

	// 解析数据
	req := UserInfoReq{}
	if err := json.Unmarshal(reqJson, &req); err != nil {
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), Internal_Error)
		return
	}

	users := make([]*CommonUser, 0)

	if req.UserId > 0 {
		// 按userid查询
		uaid, err := s.GetUAIDByRoleId(uint64(req.UserId))
		if err != nil {
			RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), Internal_Error)
			return
		}
		userInfo, err := s.GetUserInfo2(uaid)
		if err != nil {
			RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), err)
			return
		}
		users = append(users, userInfo)
	} else if req.UserName != "" {
		// fixme 按username查询，模糊搜索可能会有多个

	} else if req.OpenId != "" {
		roleId, err := s.GetUidFromOpenId(req.OpenId)
		if err != nil {
			RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), Internal_Error)
			return
		}
		userInfo, err := s.GetUserInfo2(roleId)
		if err != nil {
			RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), err)
			return
		}
		users = append(users, userInfo)
	} else {
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_ParamError), Param_Error)
		return
	}

	msg := UserInfoRes{Users: users}

	// 返回结果数据
	RetCommonMsg(out, http.StatusOK, int32(pb.ErrorCode_Success), msg)
}

func (s *IDIPServer) GetUserInfo2(roleId string) (*CommonUser, error) {
	userStub := stub.NewUserStub(roleId)
	in := &base.ProtoMsg{
		AppId:   s.AppId,
		MsgId:   int32(pb.Protocols_PS2AS_GetUserInfo),
		UserId:  roleId,
		RoleId:  0,
		UAID:    roleId,
		Data:    nil,
		ErrCode: 0,
		// GUID:    utils.GenIntUUID(),
		ServerReqIdx: guid.GenIntUuid(),
		Topic:        "",
	}
	s.ImpActorStub(userStub)
	rsp, err := userStub.UserInvoke(context.Background(), in)
	if rsp.ErrCode != RET_CODE_SUCCESS || err != nil {
		return nil, err
	}
	res := &pb.S2AS_GetUserInfoRes{}
	err = base.UnmarshalData(rsp.Data, res)
	if err != nil {
		return nil, err
	}
	userInfo := &CommonUser{}
	if err = json.Unmarshal([]byte(res.User), userInfo); err != nil {
		return nil, err
	}
	return userInfo, nil
}
