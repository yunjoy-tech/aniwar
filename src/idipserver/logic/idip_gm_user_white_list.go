package logic

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dapr/go-sdk/service/common"
	"github.com/go-redis/redis/v8"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/db"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"
	"net/http"
	"strconv"
	"strings"
)

const (
	TYPE_ADD    = "add"
	TYPE_DELETE = "delete"
)

// 请求参数结构
type ModifyWhiteListReq struct {
	ReqType   string `json:"type"`      // 固定值 “gmt_player_white_list”
	Uids      []int  `json:"uids"`      // 玩家uid,[123, 124, 125, …]
	Operation string `json:"operation"` // 添加则该值为’add’,删除为’delete’
}

// 返回结果结构
type WhiteListItem struct {
	// ret: 0代表全部uid成功，-1代表全部uid失败，-2代表部分uid失败
	// info: 额外提示字段，数组中每个对象结构如下，如果ret为0该字段不需要
	UserId int `json:"user_id"` // 用户id
	Ret    int `json:"ret"`     // 错误码，0表示成功，-1表示失败
}

// ModifyWhiteList 编辑玩家登录白名单
func (s *IDIPServer) ModifyWhiteList(out *common.Content, reqJson []byte) {

	// 解析数据
	req := ModifyWhiteListReq{}
	if err := json.Unmarshal(reqJson, &req); err != nil {
		RetCommonMsg(out, http.StatusInternalServerError, int32(cmd.ErrorCode_InternalError), Internal_Error)
		return
	}

	// 获取当前的白名单
	data, err := s.GetFromConfigCenter(db.KeyCfgWhiteList)
	if err != nil && !strings.Contains(err.Error(), redis.Nil.Error()) && !errors.Is(err, service.DB_ERROR_NOT_EXIST) {
		logger.Error("GetFromConfigCenter", db.KeyCfgWhiteList, err)
		RetCommonMsg(out, http.StatusInternalServerError, int32(cmd.ErrorCode_InternalError), Internal_Error)
		return
	}
	curWhiteList := make(map[string]int32)
	if data != "" {
		temp := strings.Split(data, ",")
		for _, v := range temp {
			curWhiteList[v] = 0
		}
	}
	logger.Debugf("ModifyWhiteList curWhiteList: %v", curWhiteList)

	items := make([]WhiteListItem, 0)
	for _, uid := range req.Uids {
		uidStr := strconv.Itoa(uid)
		if req.Operation == TYPE_ADD {
			curWhiteList[uidStr] = 0
		} else if req.Operation == TYPE_DELETE {
			delete(curWhiteList, uidStr)
		} else {
			RetCommonMsg(out, http.StatusInternalServerError, int32(cmd.ErrorCode_UnrealizedTypeError), Unrealized_Type_Error)
			return
		}
		items = append(items, WhiteListItem{
			UserId: uid,
			Ret:    0,
		})
	}

	// 保存
	tempStr := ""
	for k := range curWhiteList {
		tempStr += fmt.Sprintf("%s,", k)
	}
	tempStr = strings.TrimSuffix(tempStr, ",")
	err = s.SaveToConfigCenter(db.KeyCfgWhiteList, tempStr)
	if err != nil {
		RetCommonMsg(out, http.StatusInternalServerError, int32(cmd.ErrorCode_InternalError), Internal_Error)
		return
	}

	// 返回结果数据
	RetCommonMsg(out, http.StatusOK, int32(RET_CODE_SUCCESS), items)
}
