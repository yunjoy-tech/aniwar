package logic

import (
	"encoding/json"
	"github.com/dapr/go-sdk/service/common"
	gameCommon "gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"
	"net/http"
	"strconv"
)

// 请求参数结构
type SubBatchResourceReq struct {
	ReqType  string        `json:"type"`     // 固定值 “gm_sub_batch_resource”
	Uids     []int         `json:"uids"`     // 玩家uid,[123, 124, 125, …]
	Currency int           `json:"currency"` // 要发送的一级货币数量
	Coins    []CommonCoin  `json:"coins"`    // 要发送的次级货币coin数组，coin定义见这里
	Items    []CommonItem  `json:"items"`    // 要发送的道具item数组，item定义见这里
	Heroes   []CommonHero  `json:"heroes"`   // 要发送的英雄，heroes定义见这里
	Equip    []CommonEquip `json:"equip"`    // 要发送的装备，euip定义见这里
}

// SubBatchResource 扣除批量资源
func (s *IDIPServer) SubBatchResource(out *common.Content, reqJson []byte) {

	// 解析数据
	req := &SubBatchResourceReq{}
	if err := json.Unmarshal(reqJson, req); err != nil {
		RetCommonMsg(out, http.StatusInternalServerError, int32(cmd.ErrorCode_InternalError), Internal_Error)
		return
	}
	rpcCall := &cmd.S2SReceiveGMCostResReq{Items: map[int32]int32{}, Coins: map[int32]int32{}}
	for _, coin := range req.Coins {
		itemId, err := strconv.Atoi(coin.CoinName)
		if err != nil {
			logger.Error("coin arg error:", coin.CoinName, err)
			continue
		}
		rpcCall.Items[int32(itemId)] += int32(coin.CoinValue)
	}
	rpcCall.Items[gameCommon.CURRENCY_ITEM_ID_2005] += int32(req.Currency)
	for _, item := range req.Items {
		itemId, err := strconv.Atoi(item.ItemId)
		if err != nil {
			logger.Error("coin arg error:", item.ItemCount, err)
			continue
		}
		rpcCall.Items[int32(itemId)] += item.ItemCount
	}
	//todo hero no define
	for range req.Heroes {
		rpcCall.Heros = append(rpcCall.Heros, &cmd.ItemHero{})
	}
	//todo equip no define
	for range req.Equip {
		rpcCall.Equips = append(rpcCall.Equips, &cmd.ItemEquip{})
	}
	items := GMCostItem(s, req.Uids, rpcCall)
	// 返回结果数据
	if len(items) > 0 {
		RetCommonMsg(out, http.StatusInternalServerError, int32(RET_CODE_FAIL), items)
	} else {
		RetCommonMsg(out, http.StatusOK, int32(RET_CODE_SUCCESS), items)
	}
}
