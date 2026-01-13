package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gitee.com/aniwar2/musae/gamelib/guid"
	"gitee.com/aniwar2/musae/global"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gitee.com/aniwar2/aniwar/src/common/db"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/base"
	"gitee.com/aniwar2/musae/logger"
	"gitee.com/aniwar2/musae/service"
	"gitee.com/aniwar2/musae/utils"
	"github.com/dapr/go-sdk/service/common"
	"google.golang.org/protobuf/proto"
)

type InsideGmtHandlerFunc = func(apiData []byte) []byte

var InsideGmtHandlerMap = make(map[pb.GMT]InsideGmtHandlerFunc)

// 审核订单的状态
const (
	Verify   = iota + 1 // 此次操作需要审核
	NoVerify            // 此次操作不需要审核
)

const (
	Verifying = iota + 1 // 审核中
	Pass                 // 通过
	Rejected             // 驳回
)

const (
	version_disable   = iota // 禁用
	version_enable           // 	启用
	version_Verifying        // 审核中
	version_pass             // 审核通过
	version_use              // 使用
)

func registerInsideGmtHandler(cmd pb.GMT, f InsideGmtHandlerFunc) {
	if _, ok := InsideGmtHandlerMap[cmd]; !ok {
		InsideGmtHandlerMap[cmd] = f
		logger.Debugf("register InsideGmtHandler: %d", cmd)
	} else if ok {
		logger.Errorf("register InsideGmtHandler are registered: %d", cmd)
	}
}

func (s *IDIPServer) InitInsideGmtHandlerMap() {
	registerInsideGmtHandler(pb.GMT_SendUserMailReq, s.SendUserMailReq)
	registerInsideGmtHandler(pb.GMT_SendMultiLangUserMailReq, s.SendMultiLangUserMailReq)
	registerInsideGmtHandler(pb.GMT_SendSysMailReq, s.SendSysMailReq)
	registerInsideGmtHandler(pb.GMT_SendMultiLangSysMailReq, s.SendMultiLangSysMailReq)
	registerInsideGmtHandler(pb.GMT_GetConfCenterReq, s.GetConfCenterReq)
	registerInsideGmtHandler(pb.GMT_ModConfCenterReq, s.ModConfCenterReq)
	registerInsideGmtHandler(pb.GMT_GetGameJsonReq, s.GetGameJsonReq)
	registerInsideGmtHandler(pb.GMT_UpdateGameJsonReq, s.UpdateGameJsonReq)
	registerInsideGmtHandler(pb.GMT_ServerListReq, s.ServerListReq)
	registerInsideGmtHandler(pb.GMT_GMCommondReq, s.GMCommondReq)
	registerInsideGmtHandler(pb.GMT_MailListReq, s.MailListReq)
	registerInsideGmtHandler(pb.GMT_GetSysMailReq, s.GetSysMailReq)
	registerInsideGmtHandler(pb.GMT_DelSysMailReq, s.DelSysMailReq)
	registerInsideGmtHandler(pb.GMT_CheckItemReq, s.CheckItemReq)
	registerInsideGmtHandler(pb.GMT_ImportUserJsonReq, s.ImportUserJsonReq)
	registerInsideGmtHandler(pb.GMT_ExportUserJsonReq, s.ExportUserJsonReq)
	registerInsideGmtHandler(pb.GMT_CopyUserInfoReq, s.CopyUserInfoReq)
	registerInsideGmtHandler(pb.GMT_UserInfoReq, s.UserInfoReq)
	registerInsideGmtHandler(pb.GMT_GetAllGMTRecord, s.GetAllGMTRecord)
	registerInsideGmtHandler(pb.GMT_GMTVerify, s.Verify)
	registerInsideGmtHandler(pb.GMT_GetUserOrderList, s.GetUserOrderList)
	registerInsideGmtHandler(pb.GMT_ReDropOrderReward, s.ReDropOrderReward)
	registerInsideGmtHandler(pb.GMT_GetUserBagInfo, s.GetUserBagInfo)
	registerInsideGmtHandler(pb.GMT_GetUserCardInfo, s.GetUserCardInfo)
	registerInsideGmtHandler(pb.GMT_ReduceItem, s.ReduceItem)
	registerInsideGmtHandler(pb.GMT_GetServerVersion, s.GetServerVersion)
	registerInsideGmtHandler(pb.GMT_ChangeVersionState, s.ChangeVersionState)
	registerInsideGmtHandler(pb.GMT_ClientVersionPublish, s.ClientVersionPublish) // 发布上线
	registerInsideGmtHandler(pb.GMT_PushExcelReq, s.PushExcelReq)
	registerInsideGmtHandler(pb.GMT_SrvHotReloadReq, s.SrvHotReloadReq)
	registerInsideGmtHandler(pb.GMT_SrvRestartReq, s.SrvRestartReq)
	registerInsideGmtHandler(pb.GMT_NotifyDownloadPkgReq, s.NotifyDownloadPkgReq)
	registerInsideGmtHandler(pb.GMT_CopyTapUserInfoReq, s.CopyTapUserInfoReq)
	registerInsideGmtHandler(pb.GMT_DelTapUserInfoReq, s.DelTapUserInfoReq)
	registerInsideGmtHandler(pb.GMT_SetClientMinVersion, s.SetClientMinVersion)
	registerInsideGmtHandler(pb.GMT_SetServerCurVersion, s.SetServerCurVersion)
	registerInsideGmtHandler(pb.GMT_GetClientMaxVersion, s.GetClientMaxVersion) // 获取客户端最大版本
	registerInsideGmtHandler(pb.GMT_SetExcelExpired, s.SetExcelExpired)
	registerInsideGmtHandler(pb.GMT_GetExcelExpired, s.GetExcelExpired)
	registerInsideGmtHandler(pb.GMT_GetExcelList, s.GetExcelList)
	registerInsideGmtHandler(pb.GMT_GetAllianceInfo, s.GetAllianceInfo)
	registerInsideGmtHandler(pb.GMT_GetRollingVersion, s.GetServerRollingVersion)
	registerInsideGmtHandler(pb.GMT_GetExcelConfigReq, s.GetExcelConfigReq)
}

func (s *IDIPServer) GetExcelConfigReq(apiData []byte) []byte {
	req := &pb.GMTExcelConfigReq{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Errorf("Unmarshal fail apiData:%s error:%+v", string(apiData), err)
	}
	logger.Debugf("InsideGMT GetExcelConfig, Req: %s", utils.PrettyJson(req))
	return s.GetExcelConfig(req.SheetName)
}

func (s *IDIPServer) UserInfoReq(apiData []byte) []byte {
	req := &pb.GMTUserInfoReq{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Errorf("Unmarshal fail apiData:%s error:%+v", string(apiData), err)
	}
	typ, ok := pb.GMT_UserData_value[req.Type]
	logger.Debugf("InsideGMT UserInfo,type:%v Req: %s", typ, utils.PrettyJson(req))
	if ok {
		return s.GetUserInfo(s.GetUAID(req.Uid, req.RoleId), pb.GMT_UserData(typ))
	} else {
		return []byte(req.Type + " error")
	}
}

func (s *IDIPServer) CopyUserInfoReq(apiData []byte) []byte {
	req := &pb.GMTCopyUserInfoReq{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Errorf("Unmarshal fail apiData:%s error:%+v", string(apiData), err)
	}
	logger.Debugf("InsideGMT CopyUserInfo, Req: %s", utils.PrettyJson(req))
	return s.CopyUserInfo(s.GetUAID(req.Uid, 0), req.StartId, req.CopyNum)
}

func (s *IDIPServer) ExportUserJsonReq(apiData []byte) []byte {
	req := &pb.GMTExportUserJsonReq{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Errorf("Unmarshal fail apiData:%s error:%+v", string(apiData), err)
	}
	logger.Debugf("InsideGMT ExportUserInfo, Req: %s", utils.PrettyJson(req))
	return s.ExportUserInfo(req.Users, req.Chan)
}

func (s *IDIPServer) ImportUserJsonReq(apiData []byte) []byte {
	req := &pb.GMTImportUserJsonReq{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Errorf("Unmarshal fail apiData:%s error:%+v", string(apiData), err)
	}
	logger.Debugf("InsideGMT ImportUserInfo, Req: %s", utils.PrettyJson(req))
	return s.ImportUserInfo(req.Files)
}

func (s *IDIPServer) MailListReq(apiData []byte) []byte {
	req := &pb.GMTCommonReq{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Errorf("Unmarshal fail apiData:%s error:%+v", string(apiData), err)
	}
	return s.GetMailList(s.GetUAID(req.Uid, req.RoleId))
}

func (s *IDIPServer) GetSysMailReq(apiData []byte) []byte {
	req := &pb.GMTGetSysMailReq{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Errorf("Unmarshal fail apiData:%s error:%+v", string(apiData), err)
	}
	return s.GetSysMail()
}

func (s *IDIPServer) DelSysMailReq(apiData []byte) []byte {
	req := &pb.GMTDelSysMailReq{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Errorf("Unmarshal fail apiData:%s error:%+v", string(apiData), err)
	}
	return s.DelSysMail(req.MailId)
}

func (s *IDIPServer) CheckItemReq(apiData []byte) []byte {
	req := &pb.GMTCheckItemReq{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Errorf("Unmarshal fail apiData:%s error:%+v", string(apiData), err)
	}
	return s.CheckItem(req.Items)
}

func (s *IDIPServer) GMCommondReq(apiData []byte) []byte {
	req := &pb.GMTGMCommondReq{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Errorf("Unmarshal fail apiData:%s error:%+v", string(apiData), err)
	}
	uaid := s.GetUAID(req.Uid, req.RoleId)
	req.Uid, req.RoleId = s.ConvUAID(uaid)
	return s.UseGMCommand(req.Uid, uaid, req.Cmd, req.Params)
}

func (s *IDIPServer) ServerListReq(apiData []byte) []byte {
	req := &pb.GMTServerListReq{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Errorf("Unmarshal fail apiData:%s error:%+v", string(apiData), err)
	}
	return s.GetServerList()
}

func (s *IDIPServer) UpdateGameJsonReq(apiData []byte) []byte {
	req := &pb.GMTUpdateGameJsonReq{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Errorf("Unmarshal fail apiData:%s error:%+v", string(apiData), err)
	}
	return s.UpdateGameJson(req.Files)
}

func (s *IDIPServer) GetGameJsonReq(apiData []byte) []byte {
	req := &pb.GMTGetGameJsonReq{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Errorf("Unmarshal fail apiData:%s error:%+v", string(apiData), err)
	}
	return s.GetGameJson(req.Filename)
}

func (s *IDIPServer) ModConfCenterReq(apiData []byte) []byte {
	req := &pb.GMTModConfCenterReq{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Errorf("Unmarshal fail apiData:%s error:%+v", string(apiData), err)
	}
	return s.ModConfCenter(req.Name, req.Val)
}
func (s *IDIPServer) GetConfCenterReq(apiData []byte) []byte {
	req := &pb.GMTGetConfCenterReq{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Errorf("Unmarshal fail apiData:%s error:%+v", string(apiData), err)
	}
	return s.GetConfCenter(req.Key)
}

func (s *IDIPServer) SendMultiLangSysMailReq(apiData []byte) []byte {
	data := &common.Content{}
	s.SendSysMail2(data, apiData)
	return data.Data
}

func (s *IDIPServer) SendSysMailReq(apiData []byte) []byte {
	data := &common.Content{}
	s.SendSysMail(data, apiData)
	return data.Data
}

func (s *IDIPServer) SendMultiLangUserMailReq(apiData []byte) []byte {
	data := &common.Content{}
	s.SendUserMail2(data, apiData)
	return data.Data
}

func (s *IDIPServer) SendUserMailReq(apiData []byte) []byte {
	data := &common.Content{}
	s.SendUserMail(data, apiData)
	return data.Data
}

func (s *IDIPServer) GetAllGMTRecord(apiData []byte) []byte {
	record, err := s.GetGMTRecord()
	if err != nil {
		logger.Warn("InsideGmtHandler GetAllGMTRecord  is not found ", err)
		return nil
	}
	value := &pb.GMTDataVerify{}
	if len(record) > 0 {
		err = proto.Unmarshal(record, value)
		if err != nil {
			return nil
		}
	}
	data, _ := json.Marshal(value)
	return data
}

func (s *IDIPServer) SaveGMTRecord(api *pb.GMTApiReq) error {

	record, err := s.GetGMTRecord()
	if err != nil {
		logger.Warn("InsideGmtHandler SaveGMTRecord  is not found ", api.GetCmd(), err)
	}
	value := &pb.GMTDataVerify{}
	if len(record) > 0 {
		err = proto.Unmarshal(record, value)
		if err != nil {
			return err
		}
	}
	value.OrderNumMax++
	// 先获取库里的数据TODO
	value.DataVerify = append(value.DataVerify, &pb.DataVerify{
		Cmd:        api.GetCmd(),
		Data:       api.GetData(),
		OpType:     api.GetOpType(),
		State:      api.GetState(),
		Result:     api.GetResult(),
		Number:     value.OrderNumMax,
		CreateTime: time.Now().Format("2006-01-02 15:04:05"),
		Url:        api.GetUrl(),
	})
	kvTable, err := db.BuildKvTable(value, db.KeyGMTVerify())
	if err != nil {
		return err
	}
	if err = s.SaveMongoGmt(db.KeyGMTVerify(), kvTable, nil); err != nil {
		return err
	}

	// 如果是改变服务器版本状态的,要改变元数据的状态
	if api.GetCmd() == int32(pb.GMT_ChangeVersionState) {
		// TODO
		versionData := &ChangeVersionState{}
		json.Unmarshal(api.GetData(), versionData)
		var key string
		if versionData.ServiceType == "1" {
			key = fmt.Sprintf("version:server:%s", versionData.Version)
		}
		if versionData.ServiceType == "2" {
			key = fmt.Sprintf("version:client:%s", versionData.Version)
		}

		s.changeVersionState(key, version_Verifying)
	}

	return nil
}

func (s *IDIPServer) GetGMTRecord() ([]byte, error) {
	kvTable, err := s.GetMongoGmt(db.KeyGMTVerify(), nil)
	if err != nil {
		if errors.Is(err, service.DB_ERROR_NOT_EXIST) {
			return []byte{}, nil
		}
		return nil, err
	}
	return kvTable.Data, nil
}

func (s *IDIPServer) Verify(apiData []byte) []byte {
	req := &VerifyReq{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Warn("C2SMsg - Unmarshal error ")
	}
	record, err := s.GetGMTRecord()
	if err != nil {
		logger.Warn("InsideGmtHandler Verify  is not found", err)
	}
	value := &pb.GMTDataVerify{}
	if len(record) > 0 {
		err = proto.Unmarshal(record, value)
		if err != nil {
			logger.Warn("InsideGmtHandler Unmarshal  DataVerify is err", err)
			return nil
		}
	}

	// 从库里把这条数据找出来
	target := &pb.DataVerify{}
	for _, v := range value.DataVerify {
		if v.Number == req.Number {
			target = v
			break
		}
	}

	if target.State != Verifying {
		logger.Warn("InsideGmtHandler Verify  the record has verified :", target.Number)
		return nil
	}
	switch req.ReqType {
	case Rejected:
		target.State = Rejected
		target.Result = "successful"
	case Pass:
		if handler, ok := InsideGmtHandlerMap[pb.GMT(target.Cmd)]; ok {
			out := handler(target.Data)
			target.Result = string(out)
			target.State = Pass
		} else {
			logger.Warn("InsideGmtHandler is not found ", target.Cmd)
		}
	}
	// 保存到数据库
	kvTable, _ := db.BuildKvTable(value, db.KeyGMTVerify())
	if err = s.SaveMongoGmt(db.KeyGMTVerify(), kvTable, nil); err != nil {
		logger.Warn("InsideGmtHandler Verify  SaveMongoGmt err: ", err)
	}
	res, _ := json.Marshal(target)
	return res
}

func (s *IDIPServer) GetUserOrderList(apiData []byte) []byte {
	data := &common.Content{}
	s.getUserOrderList(data, apiData)
	return data.Data
}

func (s *IDIPServer) getUserOrderList(out *common.Content, apiData []byte) {
	req := &GetOrderReq{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Warn("C2SMsg - Unmarshal error ")
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), Internal_Error)
		return
	}
	//
	uaid := s.GetUAID(req.Uid, req.RoleId)

	reqData := &pb.S2S_OrderListReq{
		OrderStatus: pb.OrderStatus_OrderStatus_Max,
	}

	data, err := proto.Marshal(reqData)
	if err != nil {
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), Internal_Error)
		return
	}
	// 从DB中获取用户订单数据
	rsp, err := s.UserInvoke(uaid, &base.ProtoMsg{
		AppId:   s.AppId,
		MsgId:   int32(pb.Protocols_PS2S_OrderListReq),
		UserId:  req.Uid,
		RoleId:  req.RoleId,
		UAID:    uaid,
		Data:    data,
		ErrCode: 0,
		// GUID:    utils.GenIntUUID(),
		ServerReqIdx: guid.GenIntUuid(),
		Topic:        "",
	})
	if rsp.ErrCode != RET_CODE_SUCCESS || err != nil {
		RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), Internal_Error)
		return
	}
	out.Data = rsp.Data
	return
}

func (s *IDIPServer) ReDropOrderReward(apiData []byte) []byte {
	req := &GetDropOrderReq{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Warn("C2SMsg - Unmarshal error ")
	}
	//
	uaid := s.GetUAID(req.Uid, req.RoleId)

	reqData := &pb.S2S_ReplacementOrderReq{
		OrderId: req.OrderId,
	}

	data, err := proto.Marshal(reqData)
	if err != nil {
		// RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), Internal_Error)
		return nil
	}
	// 从DB中获取用户订单数据
	rsp, err := s.UserInvoke(uaid, &base.ProtoMsg{
		AppId:   s.AppId,
		MsgId:   int32(pb.Protocols_PS2S_ReplacementOrderReq),
		UserId:  req.Uid,
		RoleId:  req.RoleId,
		UAID:    uaid,
		Data:    data,
		ErrCode: 0,
		// GUID:    utils.GenIntUUID(),
		ServerReqIdx: guid.GenIntUuid(),
		Topic:        "",
	})
	if rsp.ErrCode != RET_CODE_SUCCESS || err != nil {
		// RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), Internal_Error)

		return []byte{}
	}
	return rsp.GetData()
}

func (s *IDIPServer) GetUserBagInfo(apiData []byte) []byte {
	req := &pb.GMTCommonReq{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Errorf("Unmarshal fail apiData:%s error:%+v", string(apiData), err)
	}
	// 获取背包数据
	uaid := s.GetUAID(req.Uid, req.RoleId)
	userItems, err := s.GetMongoGame(db.KeyUserItems(uaid), nil)
	if err != nil {
		return s.GenRet(err.Error())
	}
	info := &pb.PCommonItemInfos{}
	if err = base.UnmarshalData(userItems.Data, info); err != nil {
		return s.GenRet(err.Error())
	}
	// 可是转换
	ret := s.ConvertBagItem(info.GetItems())
	// 返回
	data, err := json.Marshal(ret)
	if err != nil {
		return s.GenRet(err.Error())
	}

	return data
}

func (s *IDIPServer) GetUserCardInfo(apiData []byte) []byte {
	req := &pb.GMTCommonReq{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Errorf("Unmarshal fail apiData:%s error:%+v", string(apiData), err)
	}
	// 获取卡片数据
	uaid := s.GetUAID(req.Uid, req.RoleId)
	userCards, err := s.GetMongoGame(db.KeyUserCard(uaid), nil)
	if err != nil {
		return s.GenRet(err.Error())
	}
	userEquip, err := s.GetMongoGame(db.KeyUserEquipInfo(uaid), nil)
	if err != nil {
		return s.GenRet(err.Error())
	}
	equips := &pb.PEquipData{}
	err = proto.Unmarshal(userEquip.Data, equips)

	info := &pb.PCardData{}
	if err = base.UnmarshalData(userCards.Data, info); err != nil {
		return s.GenRet(err.Error())
	}
	// 可是转换
	ret := s.ConvertCard(info.GetCard(), equips)
	// 返回
	data, err := json.Marshal(ret)
	if err != nil {
		return s.GenRet(err.Error())
	}

	return data
}

func (s *IDIPServer) ReduceItem(apiData []byte) []byte {
	req := &ReduceItem{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Errorf("Unmarshal fail apiData:%s error:%+v", string(apiData), err)
	}
	//
	roleId, _ := strconv.ParseUint(req.RoleId, 10, 64)
	uaid := s.GetUAID(req.Uid, uint64(roleId))

	reqData := &pb.S2S_ReduceUserItemReq{
		ItemId: req.ItemId,
		// Num:    req.Num,
		Num: req.Num,
	}

	data, err := proto.Marshal(reqData)
	if err != nil {
		// RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), Internal_Error)
		return nil
	}
	// 调userInvoke 扣除道具
	rsp, err := s.UserInvoke(uaid, &base.ProtoMsg{
		AppId:   s.AppId,
		MsgId:   int32(pb.Protocols_PS2S_ReduceUserItemReq),
		UserId:  req.Uid,
		RoleId:  uint64(roleId),
		UAID:    uaid,
		Data:    data,
		ErrCode: 0,
		// GUID:    utils.GenIntUUID(),
		ServerReqIdx: guid.GenIntUuid(),
		Topic:        "",
	})
	if rsp.ErrCode != RET_CODE_SUCCESS || err != nil {
		// RetCommonMsg(out, http.StatusInternalServerError, int32(pb.ErrorCode_InternalError), Internal_Error)
		fmt.Println("报错:", rsp.ErrCode, err)
		return []byte{}
	}
	return rsp.GetData()
}

func (s *IDIPServer) GetServerVersion(apiData []byte) []byte {
	req := &GetVersion{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Errorf("Unmarshal fail apiData:%s error:%+v", string(apiData), err)
	}

	var version *ServerVersion
	if req.Ops == "1" { // 获取服务端版本
		version = s.getServerVersion()
	}
	if req.Ops == "2" { // 客户端
		version = s.getClientVersion()
	}

	data, err := json.Marshal(version)
	if err != nil {
		return s.GenRet(err.Error())
	}
	return data
}

func (s *IDIPServer) ChangeVersionState(apiData []byte) []byte {
	req := &ChangeVersionState{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Errorf("Unmarshal fail apiData:%s error:%+v", string(apiData), err)
	}
	var versionRecord *VersionRecord
	if req.ServiceType == "1" {
		versionRecord = s.changeServerVersionState(req)
	}
	if req.ServiceType == "2" {
		// versionRecord = s.changeClientVersionState(req)
		// 设置当前版本
		// s.setCurrentVersion(req)
		// s.setMaxClientVersion(req)
		// //坐下oss 版本号和2.8sdk 版本号映射
		// s.setVersionMap(req)
	}

	data, err := json.Marshal(versionRecord)
	if err != nil {
		return s.GenRet(err.Error())
	}
	return data
}

func (s *IDIPServer) changeVersionState(key string, state int32) bool {
	// key := fmt.Sprintf("version:server:%s", version)
	s.Server.Redis.HSet(context.Background(), key, "state", state)
	return true
}

func (s *IDIPServer) PushExcelReq(apiData []byte) []byte {
	req := &pb.GMTPushExcelReq{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Errorf("Unmarshal fail apiData:%s error:%+v", string(apiData), err)
	}
	logger.Debugf("InsideGMT PushExcelReq, Req: %s", utils.PrettyJson(req))
	return s.SrvPushExcel(req.Files)
}

func (s *IDIPServer) SrvHotReloadReq(apiData []byte) []byte {
	req := &pb.GMTSrvHotReloadReq{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Errorf("Unmarshal fail apiData:%s error:%+v", string(apiData), err)
	}
	logger.Debugf("InsideGMT SrvHotReloadReq, Req: %s", utils.PrettyJson(req))
	return s.SrvHotReload(req)
}

func (s *IDIPServer) SrvRestartReq(apiData []byte) []byte {
	req := &pb.GMTSrvRestartReq{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Errorf("Unmarshal fail apiData:%s error:%+v", string(apiData), err)
	}
	logger.Debugf("InsideGMT SrvRestartReq, Req: %s", utils.PrettyJson(req))
	return s.SrvRestart(req)
}

func (s *IDIPServer) NotifyDownloadPkgReq(apiData []byte) []byte {
	req := &pb.GMTNotifyDownloadPkgReq{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Errorf("Unmarshal fail apiData:%s error:%+v", string(apiData), err)
	}
	logger.Debugf("InsideGMT NotifyDownloadPkgReq, Req: %s", utils.PrettyJson(req))
	return s.NotifyDownloadPkg(req)
}

func (s *IDIPServer) CopyTapUserInfoReq(apiData []byte) []byte {
	req := &pb.GMTCopyTapUserInfoReq{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Errorf("Unmarshal fail apiData:%s error:%+v", string(apiData), err)
	}
	logger.Debugf("InsideGMT CopyTapUserInfoReq, Req: %s", utils.PrettyJson(req))
	return s.CopyTapUserInfo(req.OldAccount, req.NewAccount)
}
func (s *IDIPServer) GetServerRollingVersion(apiData []byte) []byte {
	return s.GetRollingVersion()
}

func (s *IDIPServer) DelTapUserInfoReq(apiData []byte) []byte {
	req := &pb.GMTDelTapUserInfoReq{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Errorf("Unmarshal fail apiData:%s error:%+v", string(apiData), err)
	}
	logger.Debugf("InsideGMT DelTapUserInfoReq, Req: %s", utils.PrettyJson(req))
	return s.DelTapUser(req.Uids)
}

func (s *IDIPServer) SetClientMinVersion(apiData []byte) []byte {
	req := &SetMinVersion{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Errorf("Unmarshal fail apiData:%s error:%+v", string(apiData), err)
	}
	logger.Debugf("InsideGMT SetClientMinVersion, Req: %s", utils.PrettyJson(req))
	// 找到流水号=> 到线上版本的映射
	key := fmt.Sprintf("%s:%s", db.KeyCfgCVersionJenkins, req.Version)
	// value := s.Server.RedisCenter.Get(context.Background(), key)
	// newVersion := value.Val()
	newVersion, err := s.Server.GetFromConfigCenter(key)
	if err != nil {
		logger.Debugf("InsideGMT SetClientMinVersion, err: %s", err)
		return s.GenRet(err.Error())
	}

	if req.Platform == "ios" {
		key = db.KeyCfgCVersionIOSMini
	}
	if req.Platform == "android" {
		key = db.KeyCfgCVersionAndroidMini
	}

	return s.SetMinVersion(key, newVersion)
}

func (s *IDIPServer) SetServerCurVersion(apiData []byte) []byte {
	req := &GoLive{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Errorf("Unmarshal fail apiData:%s error:%+v", string(apiData), err)
	}
	logger.Debugf("InsideGMT SetClientMinVersion, Req: %s", utils.PrettyJson(req))
	// s.Server.RedisCenter.Set(context.Background(), "cfg:version:server", req.Version, -1)
	return s.GenRet("success")
}

func (s *IDIPServer) GetClientMaxVersion(apiData []byte) []byte {

	req := &GetClientMaxVersionReq{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Errorf("Unmarshal fail apiData:%s error:%+v", string(apiData), err)
	}

	key := db.KeyCfgCVersionAndroidMax
	if strings.ToLower(req.Platform) == "ios" {
		key = db.KeyCfgCVersionIOSMax
	}
	// val := s.Server.Redis.Get(context.Background(), "cfg:version:client:android:max")
	// val := s.Server.RedisCenter.Get(context.Background(), key)
	resp := &GetClientMaxVersion{}
	maxVersion, err := s.Server.GetFromConfigCenter(key)
	logger.Warnf("GetClientMaxVersionReq: maxVersion:%s,err:%+v", maxVersion, err)
	if err != nil {
		resp.Version = "0.0.0"
	} else {
		resp.Version = maxVersion
	}
	data, err := json.Marshal(resp)
	if err != nil {
		return s.GenRet(err.Error())
	}
	return data
}

func (s *IDIPServer) SetExcelExpired(apiData []byte) []byte {
	req := &ExcelExpired{}
	if err := json.Unmarshal(apiData, req); err != nil {
		logger.Errorf("Unmarshal fail apiData:%s error:%+v", string(apiData), err)
	}

	key := fmt.Sprintf("cn:develop:aniwar:%s:fileList", req.Version)

	listString, err := s.Server.Redis.Get(context.Background(), key).Result()
	if err != nil {

	}
	lists := strings.Split(listString, "|")
	exTime := int32(0)
	for _, fileKey := range lists {
		val := s.Server.Redis.TTL(context.Background(), fileKey)
		fmt.Println("val:", val.Val().Seconds())
		exTime = int32(val.Val().Seconds()) + req.Day*24*3600
		s.Server.RedisExpire(context.Background(), fileKey, time.Duration(exTime)*time.Second)
	}

	res := ExcelExpiredRes{
		ExpiredTime: exTime,
		Version:     req.Version,
	}
	data, err := json.Marshal(res)
	if err != nil {
		return s.GenRet(err.Error())
	}
	return data
}

func (s *IDIPServer) getServerVersion() *ServerVersion {
	version := &ServerVersion{
		// VersionRecord: make([]*VersionRecord, 0),
		CurVersion: "",
	}
	// res := s.Server.RedisCenter.Keys(context.Background(), "cn:develop:version:server:*")
	// for _, key := range res.Val() {
	//	versionMap := s.Server.Redis.HGetAll(context.Background(), key)
	//	version.VersionRecord = append(version.VersionRecord, VersionRecordMap2Struct(versionMap.Val()))
	//	fmt.Println("version：")
	// }
	// TODO 获取当前版本

	// value := s.Server.Redis.Get(context.Background(), "cfg:version:server")

	version.CurVersion = global.ROLLING_VERSION
	return version
}

func (s *IDIPServer) getClientVersion() *ServerVersion {
	version := &ServerVersion{
		VersionRecord: make([]*VersionRecord, 0),
		CurVersion:    "",
	}
	var err error
	// 获取当前版本android
	// value := s.Server.RedisCenter.Get(context.Background(), db.KeyCfgCVersionAndroid)
	// version.CurVersionAndroid = value.Val()
	version.CurVersionAndroid, err = s.Server.GetFromConfigCenter(db.KeyCfgCVersionAndroid)
	if err != nil {
		logger.Warn("getClientVersion - CurVersionAndroid error ", err)
	}

	// value = s.Server.RedisCenter.Get(context.Background(), db.KeyCfgCVersionIOS)
	// version.CurVersionIos = value.Val()
	version.CurVersionIos, err = s.Server.GetFromConfigCenter(db.KeyCfgCVersionIOS)
	if err != nil {
		logger.Warn("getClientVersion - CurVersionIos error ", err)
	}
	// 获取最低版本
	// valueMini := s.Server.RedisCenter.Get(context.Background(), db.KeyCfgCVersionAndroidMini)
	// version.MinVersionAndroid = valueMini.Val()
	version.MinVersionAndroid, err = s.Server.GetFromConfigCenter(db.KeyCfgCVersionAndroidMini)
	if err != nil {
		logger.Warn("getClientVersion - MinVersionAndroid error ", err)
	}
	// valueMini = s.Server.RedisCenter.Get(context.Background(), db.KeyCfgCVersionIOSMini)
	// version.MinVersionIos = valueMini.Val()
	version.MinVersionIos, err = s.Server.GetFromConfigCenter(db.KeyCfgCVersionIOSMini)
	if err != nil {
		logger.Warn("getClientVersion - MinVersionIos error ", err)
	}
	// 获取当前版本的映射Jenkins 版本
	version.CurJenkinsVersionAndroid, err = s.GetJenkinsCUrVersion(version.CurVersionAndroid)
	if err != nil {
		logger.Warn("getClientVersion - CurJenkinsVersionAndroid error ", err)
	}
	version.CurJenkinsVersionIos, err = s.GetJenkinsCUrVersion(version.CurVersionIos)
	if err != nil {
		logger.Warn("getClientVersion - CurJenkinsVersionIos error ", err)
	}
	version.MinJenkinsVersionAndroid, err = s.GetJenkinsCUrVersion(version.MinVersionAndroid)
	if err != nil {
		logger.Warn("getClientVersion - MinJenkinsVersionAndroid error ", err)
	}
	version.MinJenkinsVersionIos, err = s.GetJenkinsCUrVersion(version.MinVersionIos)
	if err != nil {
		logger.Warn("getClientVersion - MinJenkinsVersionIos error ", err)
	}
	return version
}

func (s *IDIPServer) setCurrentVersion(req *ClientVersionPublishReq) error {
	key := db.KeyCfgCVersionAndroid
	if "ios" == strings.ToLower(req.Platform) {
		key = db.KeyCfgCVersionIOS
	}
	// s.Server.RedisCenter.Set(context.Background(), key, req.NewVersion, -1)
	return s.Server.SaveToConfigCenter(key, req.NewVersion)
}
func (s *IDIPServer) setVersionMap(req *ClientVersionPublishReq) error {
	// 设置线上版本=>jenkins 流水号映射
	key := db.KeyCfgCVersionOnline
	// if strings.ToLower(req.Channel) == "ios" {
	//	key = db.KeyCfgCVersionIOSOnline
	// }
	key = fmt.Sprintf("%s:%s", key, req.NewVersion)
	// s.Server.RedisCenter.Set(context.Background(), key, req.Version, -1)
	if err := s.Server.SaveToConfigCenter(key, req.Version); err != nil {
		return err
	}

	// 设置流水号=> 线上版本的映射
	keyJenkins := db.KeyCfgCVersionJenkins
	// if strings.ToLower(req.Channel) == "ios" {
	//	keyJenkins = db.KeyCfgCVersionIOSJenkins
	// }
	keyJenkins = fmt.Sprintf("%s:%s", keyJenkins, req.Version)
	// s.Server.RedisCenter.Set(context.Background(), keyJenkins, req.NewVersion, -1)
	return s.Server.SaveToConfigCenter(keyJenkins, req.NewVersion)

}

func (s *IDIPServer) setMaxClientVersion(req *ClientVersionPublishReq) error {
	key := db.KeyCfgCVersionAndroidMax
	if "ios" == strings.ToLower(req.Platform) {
		key = db.KeyCfgCVersionIOSMax
	}
	// s.Server.RedisCenter.Set(context.Background(), key, req.NewVersion, -1)
	return s.Server.SaveToConfigCenter(key, req.NewVersion)
}

func (s *IDIPServer) GetJenkinsCUrVersion(curVersion string) (string, error) {
	key := fmt.Sprintf("%s:%s", db.KeyCfgCVersionOnline, curVersion)
	// value := s.Server.RedisCenter.Get(context.Background(), key)
	// return value.Val()
	return s.Server.GetFromConfigCenter(key)
}

func (s *IDIPServer) GetJenkinsMinVersion(minVersion string) (string, error) {
	key := fmt.Sprintf("cfg:version:j_c:map:%s", minVersion)
	// value := s.Server.Redis.Get(context.Background(), key)
	// return value.Val()
	return s.Server.GetFromConfigCenter(key)
}

func VersionRecordMap2Struct(val map[string]string) *VersionRecord {
	state, _ := strconv.Atoi(val["state"])
	versionRecord := &VersionRecord{
		Version:      val["version"],
		VersionNotes: val["version_notes"],
		UploadTime:   val["upload_time"],
		PkgName:      val["pkg_name"],
		State:        int32(state),
	}

	if vt, ok := val["version_type"]; ok {
		versionType, _ := strconv.Atoi(vt)
		versionRecord.VersionType = int32(versionType)
	}
	return versionRecord
}

func (s *IDIPServer) changeServerVersionState(req *ChangeVersionState) *VersionRecord {
	key := fmt.Sprintf("version:server:%s", req.Version)
	s.changeVersionState(key, req.State)
	versionMap := s.Server.Redis.HGetAll(context.Background(), key)
	return VersionRecordMap2Struct(versionMap.Val())
}

func (s *IDIPServer) changeClientVersionState(req *ClientVersionPublishReq) *VersionRecord {
	key := fmt.Sprintf("version:client:%s:%s", req.Branch, req.Version)
	s.changeVersionState(key, req.State)
	versionMap := s.Server.Redis.HGetAll(context.Background(), key)
	return VersionRecordMap2Struct(versionMap.Val())
}
