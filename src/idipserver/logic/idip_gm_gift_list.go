package logic

import (
	"encoding/json"
	"github.com/dapr/go-sdk/service/common"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/pb"
	"net/http"
)

type GiftListReq struct {
	ReqType string `json:"type"` // 固定值 “query_iap_list”
}

type GiftDetail struct {
	PackageID int    `json:"package_id"`
	Name      string `json:"name"`
	BuyCount  int    `json:"buy_count"`
}

// GetGiftList 获取礼包列表
func (s *IDIPServer) GetGiftList(out *common.Content, reqJson []byte) {
	// 解析数据
	req := &GiftListReq{}
	if err := json.Unmarshal(reqJson, req); err != nil {
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), Internal_Error)
		return
	}
	giftList := make([]*GiftDetail, 0)
	excel.GetPackageMgr().Foreach(func(cfg *excel.PackageCfg) bool {
		giftList = append(giftList, &GiftDetail{
			PackageID: int(cfg.Id),
			Name:      cfg.NameShow,
			BuyCount:  0,
		})
		return true
	}, true)
	RetCommonMsg(out, http.StatusOK, int32(RET_CODE_SUCCESS), giftList)
}
