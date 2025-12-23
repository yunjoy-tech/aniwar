package logic

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gitee.com/aniwar2/aniwar/src/meta"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"gitee.com/aniwar2/aniwar/src/common/datalog/taptap"
	"gitee.com/aniwar2/musae/global"
	"gitee.com/aniwar2/musae/utils"
	"github.com/go-redis/redis/v8"

	"gitee.com/aniwar2/aniwar/src/common/server"
	"gitee.com/aniwar2/musae/service"

	myCommon "gitee.com/aniwar2/aniwar/src/common"
	"gitee.com/aniwar2/aniwar/src/common/actor/stub"
	"gitee.com/aniwar2/aniwar/src/common/conf"
	"gitee.com/aniwar2/aniwar/src/common/db"
	"gitee.com/aniwar2/aniwar/src/common/sdkconstant"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/base"
	"gitee.com/aniwar2/musae/gamelib/guid"
	"gitee.com/aniwar2/musae/logger"
	"gitee.com/aniwar2/musae/state"
	"github.com/dapr/go-sdk/service/common"
	"google.golang.org/protobuf/proto"
)

type UserJsonExport struct {
	Account *pb.UserData              `json:"account"`
	Game    map[string]*state.KvTable `json:"game"`
}

func (s *IDIPServer) InsideGMT(ctx context.Context, in *common.InvocationEvent) (out *common.Content, err error) {
	defer func() {
		if err := recover(); err != any(nil) {
			logger.Error("GMTHandler failed, err: ", err)
		}
	}()

	if in == nil {
		err = errors.New("nil invocation parameter")
	}
	logger.Infof("[idip] InvokeHandler - ContentType:%s, Verb:%s, QueryString:%s, len:%v", in.ContentType, in.Verb, in.QueryString, len(in.Data))

	out = &common.Content{
		ContentType: in.ContentType,
		DataTypeURL: in.DataTypeURL,
	}

	reqJson, code, errMsg := s.PreHandle(in, conf.GMT().ApiSecret)
	if code != pb.ErrorCode_Success {
		out.Data = s.GenRet(errMsg)
		return out, err
	}

	// 记录此次操作数据
	err = s.RecordOperation(aniwarKey(), reqJson)
	if err != nil {
		logger.Error("aniwar record operation failed", err)
	}

	api := &pb.GMTApiReq{}
	if err := json.Unmarshal(reqJson, api); err != nil {
		logger.Warn("InsideGMT GMTApiReq Unmarshal error")
	}
	logger.Infof("InsideGMT : %+v", utils.PrettyJson(api))
	// 在这个做分流  1 ,需要审核; 2, 不需要审核
	if api.GetOpType() == Verify {
		err = s.SaveGMTRecord(api)
		return out, nil
	}
	if handler, ok := InsideGmtHandlerMap[pb.GMT(api.GetCmd())]; ok {
		out.Data = handler(api.GetData())
	} else {
		logger.Warn("InsideGmtHandler is not found ", api.GetCmd())
	}

	logger.Infof("InsideGMT, out: %s", string(out.Data))
	return out, nil
}

func (s *IDIPServer) GetUAID(uid string, roleId uint64) string {
	if roleId > 0 {
		kvt, err := s.GetCache(service.MongoDbType_MongoGame, db.KeyPlayerUAID(roleId), server.ICache(s))
		if err == nil && kvt != nil {
			return s.UAID(kvt.UID, kvt.Id)
		}
	}
	if uid != "" {
		user, err := s.GetAccount(db.KeyAccountInfo(uid))
		if err == nil && user != nil {
			if user.PlayerList != nil && user.PlayerList.Players != nil {
				player, ok := user.PlayerList.Players[1]
				if player != nil && ok {
					return s.UAID(uid, player.Id)
				}
			}
		}
	}
	return ""
}

func (s *IDIPServer) GetUidFromOpenId(id string) (string, error) {
	openId, _ := strconv.Atoi(id)
	kvTable, err := s.GetMongoAccount(db.KeyAccountInfo(sdkconstant.GenLilithUid(openId)), nil)
	if err != nil {
		return "", err
	}
	account := &pb.UserData{}
	err = proto.Unmarshal(kvTable.Data, account)
	if err != nil {
		logger.Warn("proto unmarshal err: ", err)
		return "", err
	}
	player, ok := account.PlayerList.Players[1]
	if !ok {
		return "", fmt.Errorf("account not have player")
	}
	return sdkconstant.GenLilithUid(openId) + "_" + strconv.Itoa(int(player.Id)), nil
}

func (s *IDIPServer) GenJsonRet(message proto.Message) []byte {
	bytes, err := json.Marshal(message)
	if err != nil {
		return s.GenRet(err.Error())
	}

	return bytes
}

func (s *IDIPServer) GenRet(str string) []byte {
	return []byte(fmt.Sprintf("{\"ret\" : \"%s\"}", str))
}

func (s *IDIPServer) UpdateGameJson(files map[string]string) []byte {
	logger.Debugf("update files:[%+v]", files)
	// 覆写json到文件
	// err := ioutil.WriteFile("./output/data/"+filename, []byte(data), 0777)
	// if err != nil {
	//	logger.Debug("err:", err)
	//	return []byte(filename)
	// }
	var reloadFiles string
	for name, data := range files {
		szFile := "./output/res/data/" + name
		file, err := os.OpenFile(szFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			logger.Errorf("open file [%s] error=%v\n", szFile, err)
			return nil
		}
		defer file.Close()
		file.Seek(0, io.SeekStart)
		n, err := file.WriteAt([]byte(data), 0)
		if len([]byte(data)) != n {
			logger.Errorf("write file [%s] error=%v\n", szFile, err)
			return nil
		}
		if len(reloadFiles) > 0 {
			reloadFiles = reloadFiles + "|" + name
		} else {
			reloadFiles = name
		}
	}

	// 通知热更
	s.SaveToConfigCenter(db.KeyCfgReloadExcel, reloadFiles)
	/*if strings.Contains(filename, ".conf") {
		s.SaveToConfigCenter(db.KeyCfgReloadConf, time.Now().Local().String())
	} else if strings.Contains(filename, ".data") {
		s.SaveToConfigCenter(db.KeyCfgReloadExcel, time.Now().Local().String())
	}*/

	// check cfg key and write to db
	/*fileName := path.Base(filename)
	cfgKey := db.KeyExcelCfg(fileName)
	table := &state.KvTable{Data: []byte(data),
		UID:     fileName,
		UpSecTS: time.Now().Unix(),
	}
	if err := s.SaveGlobalRedis(cfgKey, table, nil); err != nil {
		return s.GenRet(err.Error())
	}*/

	// 全部成功了
	return s.GenRet("success")
}

func (s *IDIPServer) GetGameJson(filename string) []byte {
	// 读取文件json
	data, err := ioutil.ReadFile("./output/data/" + filename)
	if err != nil {
		return s.GenRet(err.Error())
	}

	return data
}

func (s *IDIPServer) ModConfCenter(name string, val string) []byte {
	logger.Debugf("ModConfCenter name: %s, val: %s", name, val)

	// 调用全服命令处理
	_, err := s.HandleGlobalCmd(GM_GLOBAL_CONFIG_SET, []string{name, val})
	if err != nil {
		return s.GenRet(err.Error())
	}

	return []byte("success")
}

func (s *IDIPServer) GetConfCenter(key string) []byte {
	if key == "" {
		// 调用全服命令处理
		retData, err := s.HandleGlobalCmd(GM_GLOBAL_CONFIG_MAP, []string{})
		if err != nil {
			return s.GenRet(err.Error())
		}
		return []byte(retData)
	} else {
		retData, err := s.HandleGlobalCmd(GM_GLOBAL_CONFIG_GET, []string{key})
		if err != nil {
			return s.GenRet(err.Error())
		}
		return []byte(retData)
	}
}

func (s *IDIPServer) GetMailList(uaid string) []byte {
	var data []byte
	// 获取db的邮件数据
	userMail, err := s.GetMongoGame(db.KeyUserMail(uaid), nil)
	if err != nil {
		return s.GenRet(err.Error())
	}
	info := &pb.PUserMailInfo{}
	if err = base.UnmarshalData(userMail.Data, info); err != nil {
		return s.GenRet(err.Error())
	}
	// 可视化转换
	ret := s.ConvertMail(info.UserMail)

	// 返回
	data, err = json.Marshal(ret)
	if err != nil {
		return s.GenRet(err.Error())
	}

	return data
}

func (s *IDIPServer) GetSysMail() []byte {
	var data []byte
	info := &pb.PSystemMailInfo{}
	err := s.GetSystemMail(info)
	if err != nil {
		return s.GenRet(err.Error())
	}

	ret := s.ConvertSysMail(info.SystemMail)
	// 返回
	data, err = json.Marshal(ret)
	if err != nil {
		return s.GenRet(err.Error())
	}
	return data
}

func (s *IDIPServer) DelSysMail(mailId int64) []byte {
	logger.Infof("开始删除系统邮件 id: %v", mailId)
	// 1.获取db数据
	info := &pb.PSystemMailInfo{}
	err := s.GetSystemMail(info)
	if err != nil {
		return s.GenRet(err.Error())
	}

	// 2.删除
	delete(info.SystemMail, mailId)

	// 3.保存db
	err = s.SaveSystemMail(info)
	if err != nil {
		return s.GenRet(err.Error())
	}

	// 通知刷新mgr的内存数据
	if _, err = s.NotifyCenterActor(99, nil, nil); err != nil {
		logger.Errorf("notifyCenterActor err:%+v", err)
	}
	logger.Infof("删除系统邮件完成 data: %+v", info)

	// 撤回系统邮件埋点
	taptap.GlobalMailDel(s.AppId, global.APP_VERSION, "", global.ROLLING_VERSION, "idipserver", mailId)

	return []byte("success")
}

func (s *IDIPServer) CheckItem(items []*pb.CommonItem) []byte {
	ret := struct {
		NotFound []string `json:"not_found"`
		NumLimit []string `json:"num_limit"`
	}{}
	for _, item := range items {
		// id, err := strconv.Atoi(item.ItemId)
		// if err != nil {
		// 	ret.NotFound = append(ret.NotFound, item.ItemId)
		// 	continue
		// }
		// 不存在
		var cfg *meta.ItemPkgItemMeta
		// cfg := excel.GetItemMgr().GetById(int32(id))
		if cfg == nil {
			ret.NotFound = append(ret.NotFound, item.ItemId)
			continue
		}
		// 数量超过上限
		if cfg.NumLimit < int64(item.ItemCount) {
			ret.NumLimit = append(ret.NumLimit, item.ItemId)
		}
	}

	if len(ret.NotFound) > 0 || len(ret.NumLimit) > 0 {
		data, err := json.Marshal(ret)
		if err != nil {
			return s.GenRet(err.Error())
		}
		return data
	}
	return []byte("")
}

// UseGMCommand 区分命令调用
func (s *IDIPServer) UseGMCommand(uid, uaid string, command string, params []string) []byte {
	if IsValidCmd(command) {
		return s.GenRet("command is valid")
	}
	logger.Debug("UseGMCommand ", uid, uaid, command, params)
	req := &pb.C2LS_UseGameCommandReq{
		TargetId: 0,
		Cmd:      command,
		Param:    params,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		return s.GenRet(err.Error())
	}

	// fixme 临时执行命令
	if req.Cmd == "get.user.mail" {
		failed := make([]string, 0)       // 处理失败的玩家
		received := make([]string, 0)     // 已经领取的玩家
		repeat := make(map[string]string) // 重复过滤

		// 打开文件
		file, err := os.Open("./output/res/rolelogin-0924.txt")
		if err != nil {
			logger.Error(err)
			return s.GenRet(err.Error())
		}
		defer file.Close()

		// 逐行读取文件内容
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			tempUid := scanner.Text()
			tempUid = strings.TrimSuffix(tempUid, "\r")
			// 过滤重复玩家
			if _, ok := repeat[tempUid]; ok {
				continue
			}
			repeat[tempUid] = ""

			// 逻辑处理
			tempUaid := s.GetUAID(tempUid, 0)
			if tempUaid == "" {
				logger.Warnf("get uaid got err %v", err)
				failed = append(failed, tempUid)
				continue
			}
			kvTable, err := s.GetCache(service.MongoDbType_MongoGame, db.KeyUserMail(tempUaid), server.ICache(s))
			if err != nil {
				logger.Errorf("get cache got err %v", err)
				failed = append(failed, tempUid)
				continue
			}

			pb := &pb.PUserMailInfo{}
			err = db.ParseKvTable(kvTable, pb)
			if err != nil {
				logger.Errorf("parse proto got err %v", err)
				failed = append(failed, tempUid)
				continue
			}
			// 判定是否领取10号邮件
			for _, tempMail := range pb.UserMail {
				if tempMail.Id == 10 {
					received = append(received, tempUid)
					break
				}
			}
		}
		// 打印出结果
		logger.Infof("已经领取的玩家 %+v", received)
		logger.Infof("处理失败的玩家 %+v", failed)
		return s.GenRet("success")
	}

	if IsGlobalCmd(command) {
		retData, err := s.HandleGlobalCmd(command, params)
		if err != nil {
			return s.GenRet(err.Error())
		}
		return s.GenRet(retData)
	}

	// 调用userInvoke
	in := &base.ProtoMsg{}
	in.MsgId = int32(pb.Protocols_PC2LS_UseGameCommandReq)
	in.Data = data
	in.UserId = uid
	in.AppId = s.AppId
	in.RoleId = 0
	in.UAID = uaid
	// GUID:    utils.GenIntUUID(),
	in.ReqIdx = 0
	// 临时构造stub,必须调用ImpActorStub进行方法构建
	userStub := stub.NewUserStub(uaid)
	logger.Debugf("--------->>>> 调用user actor: %s", uaid)
	s.ImpActorStub(userStub)
	_, err = userStub.UserInvoke(context.Background(), in)
	if err != nil {
		return s.GenRet(err.Error())
	}

	return s.GenRet("success")
}

func (s *IDIPServer) GetServerList() []byte {
	data, err := proto.Marshal(&pb.S2S_SvcStatusReq{})
	if err != nil {
		return s.GenRet("")
	}
	bytes := s.CenterSrvInvoke(int32(pb.Protocols_PS2S_SvcStatusReq), data)
	res := &pb.S2S_SvcStatusRes{}
	if err = proto.Unmarshal(bytes, res); err != nil {
		return s.GenRet("")
	}
	data, err = json.Marshal(res)
	if err != nil {
		return s.GenRet(err.Error())
	}
	logger.Debugf("获取服务状态数据：%s", string(data))
	return data
}

func (s *IDIPServer) GetExcelConfig(sheetName string) []byte {
	reqMsg := &pb.S2S_GetExcelConfigReq{SheetName: sheetName}
	rsp, err := s.SvcInvoke(global.ACTOR_SVC, "", 0, "", reqMsg)
	if err != nil {
		return []byte(err.Error())
	}
	return rsp
}

func (s *IDIPServer) GetUserInfo(uaid string, typ pb.GMT_UserData) []byte {
	// 容错处理
	if uaid == "" {
		return s.GenRet("data not found")
	}

	uid, _ := s.ConvUAID(uaid)
	var data []byte
	switch typ {
	case pb.GMT_UserData_Data_All:
		info, _ := s.GetAllUserInfo(uaid, nil)
		return info
	case pb.GMT_UserData_Data_Account:
		info, err := s.GetAccount(db.KeyAccountInfo(uid))
		if err != nil {
			return s.GenRet(err.Error())
		}
		data, err = json.Marshal(info)
		if err != nil {
			return s.GenRet(err.Error())
		}
	case pb.GMT_UserData_Data_Base:
		table, err := s.GetMongoGame(db.KeyUserBaseInfo(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error())
		}
		info := &pb.PServerRoleBaseInfo{}
		if err = base.UnmarshalData(table.Data, info); err != nil {
			return s.GenRet(err.Error())
		}
		data, err = json.Marshal(info)
		if err != nil {
			return s.GenRet(err.Error())
		}
	case pb.GMT_UserData_Data_Card:
		table, err := s.GetMongoGame(db.KeyUserCard(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error())
		}
		info := &pb.PCardData{}
		if err = base.UnmarshalData(table.Data, info); err != nil {
			return s.GenRet(err.Error())
		}
		data, err = json.Marshal(info)
		if err != nil {
			return s.GenRet(err.Error())
		}
	case pb.GMT_UserData_Data_Troops:
		table, err := s.GetMongoGame(db.KeyUserCardTroop(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error())
		}
		info := &pb.PCardTroopsInfo{}
		if err = base.UnmarshalData(table.Data, info); err != nil {
			return s.GenRet(err.Error())
		}
		data, err = json.Marshal(info)
		if err != nil {
			return s.GenRet(err.Error())
		}
	case pb.GMT_UserData_Data_Item:
		table, err := s.GetMongoGame(db.KeyUserItems(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error())
		}
		info := &pb.PCommonItemInfos{}
		if err = base.UnmarshalData(table.Data, info); err != nil {
			return s.GenRet(err.Error())
		}
		data, err = json.Marshal(info)
		if err != nil {
			return s.GenRet(err.Error())
		}
	case pb.GMT_UserData_Data_Camp:
		table, err := s.GetMongoGame(db.KeyUserCamp(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error())
		}
		info := &pb.PPlayerCampBlob{}
		if err = base.UnmarshalData(table.Data, info); err != nil {
			return s.GenRet(err.Error())
		}
		CampRectify(info)
		data, err = json.Marshal(info)
		if err != nil {
			return s.GenRet(err.Error())
		}
	case pb.GMT_UserData_Data_CardPool:
		table, err := s.GetMongoGame(db.KeyUserCardPool(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error())
		}
		info := &pb.PServerCardPoolInfos{}
		if err = base.UnmarshalData(table.Data, info); err != nil {
			return s.GenRet(err.Error())
		}
		data, err = json.Marshal(info)
		if err != nil {
			return s.GenRet(err.Error())
		}
	case pb.GMT_UserData_Data_HandBook:
		table, err := s.GetMongoGame(db.KeyUserHandBook(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error())
		}
		info := &pb.PHandbookInfo{}
		if err = base.UnmarshalData(table.Data, info); err != nil {
			return s.GenRet(err.Error())
		}
		data, err = json.Marshal(info)
		if err != nil {
			return s.GenRet(err.Error())
		}
	case pb.GMT_UserData_Data_Equip:
		table, err := s.GetMongoGame(db.KeyUserEquipInfo(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error())
		}
		info := &pb.PEquipData{}
		if err = base.UnmarshalData(table.Data, info); err != nil {
			return s.GenRet(err.Error())
		}
		data, err = json.Marshal(info)
		if err != nil {
			return s.GenRet(err.Error())
		}
	case pb.GMT_UserData_Data_Duty:
		table, err := s.GetMongoGame(db.KeyUserDutyInfo(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error())
		}
		info := &pb.PDutyData{}
		if err = base.UnmarshalData(table.Data, info); err != nil {
			return s.GenRet(err.Error())
		}
		data, err = json.Marshal(info)
		if err != nil {
			return s.GenRet(err.Error())
		}
	case pb.GMT_UserData_Data_BeginnerTutorial:
		table, err := s.GetMongoGame(db.KeyUserTutorial(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error())
		}
		info := &pb.PPlayerBeginnerTutorialBlob{}
		if err = base.UnmarshalData(table.Data, info); err != nil {
			return s.GenRet(err.Error())
		}
		data, err = json.Marshal(info)
		if err != nil {
			return s.GenRet(err.Error())
		}
	case pb.GMT_UserData_Data_Quest:
		table, err := s.GetMongoGame(db.KeyUserQuestInfo(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error())
		}
		info := &pb.PQuestData{}
		if err = base.UnmarshalData(table.Data, info); err != nil {
			return s.GenRet(err.Error())
		}
		data, err = json.Marshal(info)
		if err != nil {
			return s.GenRet(err.Error())
		}
	case pb.GMT_UserData_Data_Skin:
		table, err := s.GetMongoGame(db.KeyUserCardSkin(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error())
		}
		info := &pb.PSkinData{}
		if err = base.UnmarshalData(table.Data, info); err != nil {
			return s.GenRet(err.Error())
		}
		data, err = json.Marshal(info)
		if err != nil {
			return s.GenRet(err.Error())
		}
	case pb.GMT_UserData_Data_Currency:
		table, err := s.GetMongoGame(db.KeyUserCurrency(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error())
		}
		info := &pb.PCurrencyInfo{}
		if err = base.UnmarshalData(table.Data, info); err != nil {
			return s.GenRet(err.Error())
		}
		data, err = json.Marshal(info)
		if err != nil {
			return s.GenRet(err.Error())
		}
	case pb.GMT_UserData_Data_Campaign:
		table, err := s.GetMongoGame(db.KeyCampaign(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error())
		}
		info := &pb.PPlayerGeneralCampaign{}
		if err = base.UnmarshalData(table.Data, info); err != nil {
			return s.GenRet(err.Error())
		}
		data, err = json.Marshal(info)
		if err != nil {
			return s.GenRet(err.Error())
		}
	case pb.GMT_UserData_Data_Level:
		table, err := s.GetMongoGame(db.KeyUserLevelInfo(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error())
		}
		info := &pb.LS2DB_LevelInfos{}
		if err = base.UnmarshalData(table.Data, info); err != nil {
			return s.GenRet(err.Error())
		}
		data, err = json.Marshal(info)
		if err != nil {
			return s.GenRet(err.Error())
		}
	case pb.GMT_UserData_Data_Shop:
		table, err := s.GetMongoGame(db.KeyUserShopInfo(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error())
		}
		info := &pb.LS2DB_ShopData{}
		if err = base.UnmarshalData(table.Data, info); err != nil {
			return s.GenRet(err.Error())
		}
		data, err = json.Marshal(info)
		if err != nil {
			return s.GenRet(err.Error())
		}
	case pb.GMT_UserData_Data_StoryFlag:
		table, err := s.GetMongoGame(db.KeyUserStoryFlag(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error())
		}
		info := &pb.LS2DB_StoryFlagData{}
		if err = base.UnmarshalData(table.Data, info); err != nil {
			return s.GenRet(err.Error())
		}
		data, err = json.Marshal(info)
		if err != nil {
			return s.GenRet(err.Error())
		}
	case pb.GMT_UserData_Data_Sign:
		table, err := s.GetMongoGame(db.KeyUserSign(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error())
		}
		info := &pb.PSignData{}
		if err = base.UnmarshalData(table.Data, info); err != nil {
			return s.GenRet(err.Error())
		}
		data, err = json.Marshal(info)
		if err != nil {
			return s.GenRet(err.Error())
		}
	case pb.GMT_UserData_Data_PlayerLevel:
		table, err := s.GetMongoGame(db.KeyUserLevelData(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error())
		}
		info := &pb.PPlayerLevelInfo{}
		if err = base.UnmarshalData(table.Data, info); err != nil {
			return s.GenRet(err.Error())
		}
		data, err = json.Marshal(info)
		if err != nil {
			return s.GenRet(err.Error())
		}
	case pb.GMT_UserData_Data_Achievement:
		table, err := s.GetMongoGame(db.KeyUserAchieve(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error())
		}
		info := &pb.PUserAchieves{}
		if err = base.UnmarshalData(table.Data, info); err != nil {
			return s.GenRet(err.Error())
		}
		data, err = json.Marshal(info)
		if err != nil {
			return s.GenRet(err.Error())
		}
	case pb.GMT_UserData_Data_Trial:
		table, err := s.GetMongoGame(db.KeyUserTrial(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error())
		}
		info := &pb.PUserTrial{}
		if err = base.UnmarshalData(table.Data, info); err != nil {
			return s.GenRet(err.Error())
		}
		data, err = json.Marshal(info)
		if err != nil {
			return s.GenRet(err.Error())
		}
	case pb.GMT_UserData_Data_BlockWayEvent:
		table, err := s.GetMongoGame(db.KeyUserBlockWay(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error())
		}
		info := &pb.PBlockWay{}
		if err = base.UnmarshalData(table.Data, info); err != nil {
			return s.GenRet(err.Error())
		}
		data, err = json.Marshal(info)
		if err != nil {
			return s.GenRet(err.Error())
		}
	case pb.GMT_UserData_Data_Friend:
		table, err := s.GetMongoGame(db.KeyUserFriend(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error())
		}
		info := &pb.PFriendData{}
		if err = base.UnmarshalData(table.Data, info); err != nil {
			return s.GenRet(err.Error())
		}
		data, err = json.Marshal(info)
		if err != nil {
			return s.GenRet(err.Error())
		}
	case pb.GMT_UserData_Data_CampPool:
		table, err := s.GetMongoGame(db.KeyUserCampPool(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error())
		}
		info := &pb.PServerCampPoolInfos{}
		if err = base.UnmarshalData(table.Data, info); err != nil {
			return s.GenRet(err.Error())
		}
		data, err = json.Marshal(info)
		if err != nil {
			return s.GenRet(err.Error())
		}
	case pb.GMT_UserData_Data_UseLimit:
		table, err := s.GetMongoGame(db.KeyUseLimit(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error())
		}
		info := &pb.PUseLimitInfo{}
		if err = base.UnmarshalData(table.Data, info); err != nil {
			return s.GenRet(err.Error())
		}
		data, err = json.Marshal(info)
		if err != nil {
			return s.GenRet(err.Error())
		}
	case pb.GMT_UserData_Data_OfflineEvent:
		table, err := s.GetMongoGame(db.KeyOfflineEvent(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error())
		}
		info := &pb.POfflineEventData{}
		if err = base.UnmarshalData(table.Data, info); err != nil {
			return s.GenRet(err.Error())
		}
		data, err = json.Marshal(info)
		if err != nil {
			return s.GenRet(err.Error())
		}
	case pb.GMT_UserData_Data_Relation:
		table, err := s.GetMongoGame(db.KeyUserRelation(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error())
		}
		info := &pb.PUserRelationData{}
		if err = base.UnmarshalData(table.Data, info); err != nil {
			return s.GenRet(err.Error())
		}
		data, err = json.Marshal(info)
		if err != nil {
			return s.GenRet(err.Error())
		}
	case pb.GMT_UserData_Data_UserAlliance:
		table, err := s.GetMongoGame(db.KeyUserAlliance(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error())
		}
		info := &pb.PUserAllianceData{}
		if err = base.UnmarshalData(table.Data, info); err != nil {
			return s.GenRet(err.Error())
		}
		data, err = json.Marshal(info)
		if err != nil {
			return s.GenRet(err.Error())
		}
	case pb.GMT_UserData_Data_GuideTask:
		table, err := s.GetMongoGame(db.KeyUserGuideTask(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error())
		}
		info := &pb.PGuideTaskData{}
		if err = base.UnmarshalData(table.Data, info); err != nil {
			return s.GenRet(err.Error())
		}
		data, err = json.Marshal(info)
		if err != nil {
			return s.GenRet(err.Error())
		}
	case pb.GMT_UserData_Data_Travel_Level:
		table, err := s.GetMongoGame(db.KeyUserTravelLevel(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error())
		}
		info := &pb.PUserTravelLevelData{}
		if err = base.UnmarshalData(table.Data, info); err != nil {
			return s.GenRet(err.Error())
		}
		data, err = json.Marshal(info)
		if err != nil {
			return s.GenRet(err.Error())
		}
	case pb.GMT_UserData_Data_Activity:
		table, err := s.GetMongoGame(db.KeyUserActivity(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error())
		}
		info := &pb.PServerUserActivity{}
		if err = base.UnmarshalData(table.Data, info); err != nil {
			return s.GenRet(err.Error())
		}
		data, err = json.Marshal(info)
		if err != nil {
			return s.GenRet(err.Error())
		}
	}
	return data
}

func (s *IDIPServer) CopyUserInfo(uaid string, start, copyNum int32) []byte {
	// uaid = pc_1_wjjxxx1168_60470121
	// 容错处理
	if uaid == "" {
		return s.GenRet("account not found")
	}

	if copyNum <= 0 || start <= 0 {
		return s.GenRet(fmt.Sprintf("param illegal start:%d copyNum:%d", start, copyNum))
	}

	uid, _ := s.ConvUAID(uaid)
	var ret []byte
	for i := start; i < (start + copyNum); i++ {
		suffix := fmt.Sprintf("copy%d", i)
		copyUid := fmt.Sprintf("%s%s", uid, suffix) // pc_1_wjjxxx1168copy1
		copyRoleId := s.GenRoleId()
		copyUaid := s.UAID(copyUid, copyRoleId)
		// 账号数据修改
		ret = s.copyAccountData(uid, copyUid, copyRoleId, suffix)
		// base数据修改
		ret = s.copyBaseData(uaid, copyUaid, copyRoleId, suffix)
		// 业务数据拷贝
		ret = s.handleCopyGameData(uaid, copyUaid)
		if ret != nil {
			break
		}
		logger.Debugf("copy account %s success", copyUaid)
	}
	return ret
}

func GetChannelUid(uid, channel string) string {
	switch channel {
	case "lilith":
		uidx, err := strconv.Atoi(uid)
		if err != nil {
			return ""
		}
		return sdkconstant.GenLilithUid(uidx)
	case "pc":
		return sdkconstant.GenPCUid(uid)
	}
	return ""
}

func (s *IDIPServer) ExportUserInfo(uids, chanInfo string) []byte {
	// uaid = pc_1_wjjxxx1168_60470121
	// 容错处理
	if uids == "" {
		return s.GenRet("param error")
	}

	if chanInfo == "" {
		return s.GenRet("param error")
	}

	var temp = make([]*UserJsonExport, 0)
	uidArr := strings.Split(uids, ";")
	for _, uid := range uidArr {
		// 获取userId
		userId := GetChannelUid(uid, chanInfo)
		if userId == "" {
			return s.GenRet("userid is nil")
		}
		uaid := s.GetUAID(userId, 0)
		if uaid == "" {
			return s.GenRet("uaid is nil")
		}
		t := &UserJsonExport{
			Account: nil,
			Game:    make(map[string]*state.KvTable),
		}
		// accountdata
		info, err := s.GetAccount(db.KeyAccountInfo(uid))
		if err != nil {
			return s.GenRet(err.Error())
		}

		// playerdata
		ret, err := s.GetAllUserInfo(uaid, t.Game)
		if err != nil {
			return ret
		}
		t.Account = info
		temp = append(temp, t)
	}
	bytes, err := json.Marshal(temp)
	if err != nil {
		return s.GenRet(err.Error())
	}
	logger.Debugf("导出玩家数据 %v", string(bytes))
	return bytes
}

func (s *IDIPServer) ImportUserInfo(files map[string]string) []byte {
	// 容错处理
	if len(files) == 0 {
		return s.GenRet("param error")
	}
	ret := make([]string, 0)
	for filename, file := range files {
		var temp = make([]*UserJsonExport, 0)
		err := json.Unmarshal([]byte(file), &temp)
		if err != nil {
			ret = append(ret, filename)
			continue
		}

		for _, t := range temp {
			// accountdata
			err = s.SaveAccount(t.Account)
			if err != nil {
				ret = append(ret, filename)
				continue
			}

			// playerdata
			for k, v := range t.Game {
				err = s.SaveMongoGame(k, v, nil)
				if err != nil {
					ret = append(ret, filename)
					break
				}
			}
		}
	}
	if len(ret) == 0 {
		ret = append(ret, "success")
	}
	bytes, err := json.Marshal(ret)
	if err != nil {
		return s.GenRet(err.Error())
	}
	return bytes
}

func (s *IDIPServer) GenRoleId() uint64 {
	id := uint64(guid.GenIntUuid())
	var roleId uint64
	if id > 0 {
		roleId = id + myCommon.USER_ID_BASE
	}
	return roleId
}

func (s *IDIPServer) copyAccountData(uid, copyUid string, copyRoleId uint64, suffix string) []byte {
	// copyUid pc_1_wjjxxx1168copy1
	info, err := s.GetAccount(db.KeyAccountInfo(uid))
	if err != nil {
		return s.GenRet(err.Error())
	}
	// 修正数据
	// _, _, copyOpenId, err := myUtils.ScanfUID(copyUid)
	// if err != nil {
	//	return s.GenRet(err.Error())
	// }
	info.Account.OpenId = copyUid // copyOpenId
	info.Account.Uid = copyUid
	info.Account.Nickname += suffix
	info.PlayerList.PlayerId = copyRoleId
	info.PlayerList.Players[1].Id = copyRoleId

	// 覆写
	err = s.SaveAccount(info)
	if err != nil {
		return s.GenRet(err.Error())
	}
	return nil
}

func (s *IDIPServer) copyBaseData(uaid string, copyUaid string, roleId uint64, suffix string) []byte {
	table, err := s.GetMongoGame(db.KeyUserBaseInfo(uaid), nil)
	if err != nil {
		return s.GenRet(err.Error())
	}
	info := &pb.PServerRoleBaseInfo{}
	if err = base.UnmarshalData(table.Data, info); err != nil {
		return s.GenRet(err.Error())
	}
	// 修正数据
	info.Common.RoleId = roleId
	info.Common.RoleName += suffix
	table.Data, err = proto.Marshal(info)
	if err != nil {
		return s.GenRet(err.Error())
	}
	// 覆写
	err = s.SaveMongoGame(db.KeyUserBaseInfo(copyUaid), table, nil)
	if err != nil {
		return s.GenRet(err.Error())
	}
	return nil
}

func (s *IDIPServer) copyGameData(key, copyKey string) []byte {
	table, err := s.GetMongoGame(key, nil)
	if err != nil {
		return s.GenRet(err.Error())
	}
	err = s.SaveMongoGame(copyKey, table, nil)
	if err != nil {
		return s.GenRet(err.Error())
	}
	return nil
}

func (s *IDIPServer) GetAllUserInfo(uaid string, m map[string]*state.KvTable) ([]byte, error) {

	playerData := &pb.PlayerData{
		Base:             &pb.PServerRoleBaseInfo{},
		Cards:            &pb.PCardData{},
		Troops:           &pb.PCardTroopsInfo{},
		ItemData:         &pb.PCommonItemInfos{},
		Camp:             &pb.PPlayerCampBlob{},
		Pools:            &pb.PServerCardPoolInfos{},
		Handbooks:        &pb.PHandbookInfo{},
		EquipData:        &pb.PEquipData{},
		DutyData:         &pb.PDutyData{},
		Tutorial:         &pb.PPlayerBeginnerTutorialBlob{},
		QuestData:        &pb.PQuestData{},
		SkinData:         &pb.PSkinData{},
		Currency:         &pb.PCurrencyInfo{},
		CampaignInfo:     &pb.PPlayerGeneralCampaign{},
		LevelsData:       &pb.LS2DB_LevelInfos{},
		ShopData:         &pb.LS2DB_ShopData{},
		StoryFlagData:    &pb.LS2DB_StoryFlagData{},
		Sign:             &pb.PSignData{},
		PlayerLevelData:  &pb.PPlayerLevelInfo{},
		UserMail:         &pb.PUserMailInfo{},
		AchieveData:      &pb.PUserAchieves{},
		TrialData:        &pb.PUserTrial{},
		BlockWayData:     &pb.PBlockWay{},
		FriendData:       &pb.PFriendData{},
		CampPools:        &pb.PServerCampPoolInfos{},
		UseLimit:         &pb.PUseLimitInfo{},
		OfflineEventData: &pb.POfflineEventData{},
		RelationData:     &pb.PUserRelationData{},
		UserAlliance:     &pb.PUserAllianceData{},
		GuideTaskData:    &pb.PGuideTaskData{},
		TravelLevelData:  &pb.PUserTravelLevelData{},
		ActivityData:     &pb.PServerUserActivity{},
	}
	{
		table, err := s.GetMongoGame(db.KeyUserBaseInfo(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error()), err
		}
		if err = base.UnmarshalData(table.Data, playerData.Base); err != nil {
			return s.GenRet(err.Error()), err
		}
		if m != nil {
			m[db.KeyUserBaseInfo(uaid)] = table
		}
	}

	{
		table, err := s.GetMongoGame(db.KeyUserCard(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error()), err
		}
		if err = base.UnmarshalData(table.Data, playerData.Cards); err != nil {
			return s.GenRet(err.Error()), err
		}
		if m != nil {
			m[db.KeyUserCard(uaid)] = table
		}
	}

	{
		table, err := s.GetMongoGame(db.KeyUserCardTroop(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error()), err
		}
		if err = base.UnmarshalData(table.Data, playerData.Troops); err != nil {
			return s.GenRet(err.Error()), err
		}
		if m != nil {
			m[db.KeyUserCardTroop(uaid)] = table
		}
	}

	{
		table, err := s.GetMongoGame(db.KeyUserItems(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error()), err
		}
		if err = base.UnmarshalData(table.Data, playerData.ItemData); err != nil {
			return s.GenRet(err.Error()), err
		}
		if m != nil {
			m[db.KeyUserItems(uaid)] = table
		}
	}

	{
		// 营地
		table, err := s.GetMongoGame(db.KeyUserCamp(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error()), err
		}
		if err = base.UnmarshalData(table.Data, playerData.Camp); err != nil {
			return s.GenRet(err.Error()), err
		}
		// 补充数据
		CampRectify(playerData.Camp)
		if m != nil {
			m[db.KeyUserCamp(uaid)] = table
		}
	}

	{
		table, err := s.GetMongoGame(db.KeyUserCardPool(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error()), err
		}
		if err = base.UnmarshalData(table.Data, playerData.Pools); err != nil {
			return s.GenRet(err.Error()), err
		}
		if m != nil {
			m[db.KeyUserCardPool(uaid)] = table
		}
	}

	{
		table, err := s.GetMongoGame(db.KeyUserHandBook(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error()), err
		}
		if err = base.UnmarshalData(table.Data, playerData.Handbooks); err != nil {
			return s.GenRet(err.Error()), err
		}
		if m != nil {
			m[db.KeyUserHandBook(uaid)] = table
		}
	}

	{
		table, err := s.GetMongoGame(db.KeyUserEquipInfo(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error()), err
		}
		if err = base.UnmarshalData(table.Data, playerData.EquipData); err != nil {
			return s.GenRet(err.Error()), err
		}
		if m != nil {
			m[db.KeyUserEquipInfo(uaid)] = table
		}
	}

	{
		table, err := s.GetMongoGame(db.KeyUserDutyInfo(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error()), err
		}
		if err = base.UnmarshalData(table.Data, playerData.DutyData); err != nil {
			return s.GenRet(err.Error()), err
		}
		if m != nil {
			m[db.KeyUserDutyInfo(uaid)] = table
		}
	}

	{
		table, err := s.GetMongoGame(db.KeyUserTutorial(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error()), err
		}
		if err = base.UnmarshalData(table.Data, playerData.Tutorial); err != nil {
			return s.GenRet(err.Error()), err
		}
		if m != nil {
			m[db.KeyUserTutorial(uaid)] = table
		}
	}

	{
		table, err := s.GetMongoGame(db.KeyUserQuestInfo(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error()), err
		}
		if err = base.UnmarshalData(table.Data, playerData.QuestData); err != nil {
			return s.GenRet(err.Error()), err
		}
		if m != nil {
			m[db.KeyUserQuestInfo(uaid)] = table
		}
	}

	{
		table, err := s.GetMongoGame(db.KeyUserCardSkin(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error()), err
		}
		if err = base.UnmarshalData(table.Data, playerData.SkinData); err != nil {
			return s.GenRet(err.Error()), err
		}
		if m != nil {
			m[db.KeyUserCardSkin(uaid)] = table
		}
	}

	{
		table, err := s.GetMongoGame(db.KeyUserCurrency(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error()), err
		}
		if err = base.UnmarshalData(table.Data, playerData.Currency); err != nil {
			return s.GenRet(err.Error()), err
		}
		if m != nil {
			m[db.KeyUserCurrency(uaid)] = table
		}
	}

	{
		table, err := s.GetMongoGame(db.KeyCampaign(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error()), err
		}
		if err = base.UnmarshalData(table.Data, playerData.CampaignInfo); err != nil {
			return s.GenRet(err.Error()), err
		}
		if m != nil {
			m[db.KeyCampaign(uaid)] = table
		}
	}

	{
		table, err := s.GetMongoGame(db.KeyUserLevelInfo(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error()), err
		}
		if err = base.UnmarshalData(table.Data, playerData.LevelsData); err != nil {
			return s.GenRet(err.Error()), err
		}
		if m != nil {
			m[db.KeyUserLevelInfo(uaid)] = table
		}
	}

	{
		table, err := s.GetMongoGame(db.KeyUserShopInfo(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error()), err
		}
		if err = base.UnmarshalData(table.Data, playerData.ShopData); err != nil {
			return s.GenRet(err.Error()), err
		}
		if m != nil {
			m[db.KeyUserShopInfo(uaid)] = table
		}
	}

	{
		table, err := s.GetMongoGame(db.KeyUserStoryFlag(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error()), err
		}
		if err = base.UnmarshalData(table.Data, playerData.StoryFlagData); err != nil {
			return s.GenRet(err.Error()), err
		}
		if m != nil {
			m[db.KeyUserStoryFlag(uaid)] = table
		}
	}

	{
		table, err := s.GetMongoGame(db.KeyUserSign(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error()), err
		}
		if err = base.UnmarshalData(table.Data, playerData.Sign); err != nil {
			return s.GenRet(err.Error()), err
		}
		if m != nil {
			m[db.KeyUserSign(uaid)] = table
		}
	}

	{
		table, err := s.GetMongoGame(db.KeyUserLevelData(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error()), err
		}
		if err = base.UnmarshalData(table.Data, playerData.PlayerLevelData); err != nil {
			return s.GenRet(err.Error()), err
		}
		if m != nil {
			m[db.KeyUserLevelData(uaid)] = table
		}
	}

	{
		table, err := s.GetMongoGame(db.KeyUserMail(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error()), err
		}
		if err = base.UnmarshalData(table.Data, playerData.UserMail); err != nil {
			return s.GenRet(err.Error()), err
		}
		if m != nil {
			m[db.KeyUserMail(uaid)] = table
		}
	}
	{
		// 成就数据
		table, err := s.GetMongoGame(db.KeyUserAchieve(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error()), err
		}
		if err = base.UnmarshalData(table.Data, playerData.AchieveData); err != nil {
			return s.GenRet(err.Error()), err
		}
		if m != nil {
			m[db.KeyUserAchieve(uaid)] = table
		}
	}
	{
		// 试炼数据
		table, err := s.GetMongoGame(db.KeyUserTrial(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error()), err
		}
		if err = base.UnmarshalData(table.Data, playerData.TrialData); err != nil {
			return s.GenRet(err.Error()), err
		}
		if m != nil {
			m[db.KeyUserTrial(uaid)] = table
		}
	}
	{
		// 公路事件
		table, err := s.GetMongoGame(db.KeyUserBlockWay(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error()), err
		}
		if err = base.UnmarshalData(table.Data, playerData.BlockWayData); err != nil {
			return s.GenRet(err.Error()), err
		}
		if m != nil {
			m[db.KeyUserBlockWay(uaid)] = table
		}
	}
	{
		// 好友数据
		table, err := s.GetMongoGame(db.KeyUserFriend(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error()), err
		}
		if err = base.UnmarshalData(table.Data, playerData.FriendData); err != nil {
			return s.GenRet(err.Error()), err
		}
		if m != nil {
			m[db.KeyUserFriend(uaid)] = table
		}
	}
	{
		// 家具抽卡数据
		table, err := s.GetMongoGame(db.KeyUserCampPool(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error()), err
		}
		if err = base.UnmarshalData(table.Data, playerData.CampPools); err != nil {
			return s.GenRet(err.Error()), err
		}
		if m != nil {
			m[db.KeyUserCampPool(uaid)] = table
		}
	}
	{
		// 付费计数
		table, err := s.GetMongoGame(db.KeyUseLimit(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error()), err
		}
		if err = base.UnmarshalData(table.Data, playerData.UseLimit); err != nil {
			return s.GenRet(err.Error()), err
		}
		if m != nil {
			m[db.KeyUseLimit(uaid)] = table
		}
	}
	{
		// 离线事件
		table, err := s.GetMongoGame(db.KeyOfflineEvent(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error()), err
		}
		if err = base.UnmarshalData(table.Data, playerData.OfflineEventData); err != nil {
			return s.GenRet(err.Error()), err
		}
		if m != nil {
			m[db.KeyOfflineEvent(uaid)] = table
		}
	}
	{
		// 关系系统
		table, err := s.GetMongoGame(db.KeyUserRelation(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error()), err
		}
		if err = base.UnmarshalData(table.Data, playerData.RelationData); err != nil {
			return s.GenRet(err.Error()), err
		}
		if m != nil {
			m[db.KeyUserRelation(uaid)] = table
		}
	}
	{
		// 联盟数据
		table, err := s.GetMongoGame(db.KeyUserAlliance(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error()), err
		}
		if err = base.UnmarshalData(table.Data, playerData.UserAlliance); err != nil {
			return s.GenRet(err.Error()), err
		}
		if m != nil {
			m[db.KeyUserAlliance(uaid)] = table
		}
	}
	{
		// 引导任务
		table, err := s.GetMongoGame(db.KeyUserGuideTask(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error()), err
		}
		if err = base.UnmarshalData(table.Data, playerData.GuideTaskData); err != nil {
			return s.GenRet(err.Error()), err
		}
		if m != nil {
			m[db.KeyUserGuideTask(uaid)] = table
		}
	}
	{
		// 旅途关卡
		table, err := s.GetMongoGame(db.KeyUserTravelLevel(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error()), err
		}
		if err = base.UnmarshalData(table.Data, playerData.TravelLevelData); err != nil {
			return s.GenRet(err.Error()), err
		}
		if m != nil {
			m[db.KeyUserTravelLevel(uaid)] = table
		}
	}
	{
		// 活动
		table, err := s.GetMongoGame(db.KeyUserActivity(uaid), nil)
		if err != nil {
			return s.GenRet(err.Error()), err
		}
		if err = base.UnmarshalData(table.Data, playerData.ActivityData); err != nil {
			return s.GenRet(err.Error()), err
		}
		if m != nil {
			m[db.KeyUserActivity(uaid)] = table
		}
	}
	data, err := json.Marshal(playerData)
	if err != nil {
		return s.GenRet(err.Error()), err
	}
	return data, nil
}

func CampRectify(camp *pb.PPlayerCampBlob) {

}

func (s *IDIPServer) SrvPushExcel(files []*pb.ExcelFile) []byte {
	ret := &pb.GMTPushExcelRet{
		Code: 2,
	}
	cont := context.Background()
	//cn:pob:aniwar:0.0.1259:build_main.data
	pipline := s.Server.RedisCenter.Pipeline()
	srcMD5 := make(map[string]string, len(files))
	for _, v := range files {
		temp := strings.Split(v.Name, ":")
		temp[3] = global.ROLLING_VERSION
		key := strings.Join(temp, ":")
		hashKey := strings.Join(temp[:3], ":") + ":" + global.ROLLING_VERSION + ":" + "versionList"
		md5 := utils.MD5(v.Data)
		pipline.HSet(cont, hashKey, temp[4], strings.Join([]string{v.Name, md5}, "|"))
		srcMD5[key] = md5
		pipline.Set(cont, key, string(v.Data), -1)
	}
	res, err := pipline.Exec(cont)
	if err != nil {
		logger.Warn("SrvPushExcel  pipline excel err:", err, res)
		ret.Code = 1 // 失败
		return s.GenJsonRet(ret)
	}
	// 做校验
	// newMd5 := make(map[string]string, len(files))
	for k, _ := range srcMD5 {
		pipline.Get(cont, k)
	}
	res, err = pipline.Exec(cont)
	if err != nil {
		ret.Code = 1 // 失败
		for _, v := range res {
			if v.Err() != nil {
				logger.Infof("SrvPushExcel  update redis file:%s", v.Args()[1].(string))
			}
		}
		return s.GenJsonRet(ret)
	}
	for _, item := range res {
		cmdRes, _ := item.(*redis.StringCmd)
		key := item.Args()[1].(string)
		if m, ok := srcMD5[key]; ok {
			tempByte, _ := cmdRes.Bytes()
			if utils.MD5(tempByte) != m {
				ret.Code = 1 // 失败
				logger.Infof("SrvPushExcel  update redis file:%s", key)
			}
		} else {
			ret.Code = 1 // 失败
		}

	}
	return s.GenJsonRet(ret)
}

func (s *IDIPServer) SrvHotReload(req *pb.GMTSrvHotReloadReq) []byte {
	ret := &pb.GMTSrvHotReloadRet{}
	files := make([]string, 0)
	for _, v := range req.Files {
		temp := strings.Split(v, ":")
		files = append(files, temp[len(temp)-1])
	}
	data, err := proto.Marshal(&pb.S2S_HotReloadReq{
		Type:     req.Typ,
		Files:    files,
		Services: req.Services,
	})
	if err != nil {
		logger.Warn("proto.Marshal  got err:", err)
		ret.Code = 1
		return s.GenJsonRet(ret)
	}
	bytes := s.CenterSrvInvoke(int32(pb.Protocols_PS2S_HotReloadReq), data)
	res := &pb.S2S_HotReloadRes{}
	if err = proto.Unmarshal(bytes, res); err != nil {
		logger.Warn("proto unmarshal err:", err)
		ret.Code = 1
		return s.GenJsonRet(ret)
	}
	ret.Progress = res.GetProgress()
	return s.GenJsonRet(ret)
}

func (s *IDIPServer) SrvRestart(req *pb.GMTSrvRestartReq) []byte {
	ret := &pb.GMTSrvRestartRet{}
	data, err := proto.Marshal(&pb.S2S_SvcRestartReq{
		Type:     req.Typ,
		Services: req.Services,
	})
	if err != nil {
		logger.Warn("proto.Marshal  got err:", err)
		ret.Code = 1
		return s.GenJsonRet(ret)
	}
	bytes := s.CenterSrvInvoke(int32(pb.Protocols_PS2S_SvcRestartReq), data)
	res := &pb.S2S_SvcRestartRes{}
	if err = proto.Unmarshal(bytes, res); err != nil {
		logger.Warn("proto unmarshal err:", err)
		ret.Code = 1
		return s.GenJsonRet(ret)
	}
	ret.Progress = res.Progress
	return s.GenJsonRet(ret)
}

func (s *IDIPServer) NotifyDownloadPkg(req *pb.GMTNotifyDownloadPkgReq) []byte {
	ret := &pb.GMTNotifyDownloadPkgRet{}
	// 按包名称下载
	err := OssGetObjectToLocalFile(conf.OSS().VersionBucket, req.PkgName, conf.OSS().DownPath+req.PkgName)
	if err != nil {
		logger.Warn("oss get object failed, err: %s", err.Error())
		ret.Code = 1
	}
	return s.GenJsonRet(ret)
}

func (s *IDIPServer) CopyTapUserInfo(oldAccount, newAccount string) []byte {
	// 容错处理
	if oldAccount == "" || newAccount == "" {
		return s.GenRet("param error")
	}
	// 找老帐号数据
	oldUid, err := s.GetTaptapUid(oldAccount) // oldAccount: xsJMKpCc3L65q+WwMjmiQw==  oldUid: 10101
	if err != nil {
		return s.GenRet(err.Error())
	}
	uid := sdkconstant.GenTaptapUid(oldUid) // uid: taptap_10101
	oldRoleId, err := s.GetPlayerId(uid)    // oldRoleId: 564842
	if err != nil {
		return s.GenRet(err.Error())
	}
	uaid := s.UAID(uid, oldRoleId) // uaid: taptap_10101_564842

	// 遍历拷贝
	newAccountList := strings.Split(newAccount, ",")
	var ret []byte
	for i, taptapUid := range newAccountList {
		suffix := fmt.Sprintf("copy%d", i+1)
		tempUid, err := s.UpdateTaptapUidCache(taptapUid) // tempUid: 10102
		if err != nil {
			return s.GenRet(err.Error())
		}
		copyUid := sdkconstant.GenTaptapUid(tempUid) // taptap渠道uid copyUid: taptap_10102
		copyRoleId := s.GenRoleId()                  // copyRoleId: 564843
		copyUaid := s.UAID(copyUid, copyRoleId)
		// 账号数据修改
		ret = s.copyAccountData(uid, copyUid, copyRoleId, suffix)
		// base数据修改
		ret = s.copyBaseData(uaid, copyUaid, copyRoleId, suffix)
		// 业务数据拷贝
		ret = s.handleCopyGameData(uaid, copyUaid)
		if ret != nil {
			break
		}
		logger.Debugf("copy account %s success", copyUaid)
	}
	return s.GenRet("success")
}

func (s *IDIPServer) DelTapUser(uids string) []byte {
	if uids == "" {
		return s.GenRet("param error")
	}
	uidArr := strings.Split(uids, ",")
	var sb strings.Builder
	for _, uid := range uidArr {
		// 重新绑定一个roleId，不做清空处理
		_, err := s.UpdateTaptapUidCache(uid)
		if err != nil {
			sb.WriteString(uid + ",")
		}
	}
	failed := sb.String()
	if failed != "" {
		return s.GenRet("failed : " + failed)
	}
	return s.GenRet("success")
}

func (s *IDIPServer) SetMinVersion(key, version string) []byte {
	// 获取jenkins 版本号对应的真正版本
	// s.Server.RedisCenter.Set(context.Background(), key, version, -1)
	if err := s.Server.SaveToConfigCenter(key, version); err != nil {
		return nil
	}
	return s.GenRet("success")
}

func (s *IDIPServer) handleCopyGameData(uaid, copyUaid string) []byte {
	var ret []byte
	ret = s.copyGameData(db.KeyUserCard(uaid), db.KeyUserCard(copyUaid))           // 卡片
	ret = s.copyGameData(db.KeyUserCardTroop(uaid), db.KeyUserCardTroop(copyUaid)) // TroopHandler 编队
	ret = s.copyGameData(db.KeyUserItems(uaid), db.KeyUserItems(copyUaid))         // 道具
	ret = s.copyGameData(db.KeyUserCamp(uaid), db.KeyUserCamp(copyUaid))           // 营地
	ret = s.copyGameData(db.KeyUserCardPool(uaid), db.KeyUserCardPool(copyUaid))   // 抽卡
	ret = s.copyGameData(db.KeyUserHandBook(uaid), db.KeyUserHandBook(copyUaid))   // 图鉴
	ret = s.copyGameData(db.KeyUserEquipInfo(uaid), db.KeyUserEquipInfo(copyUaid)) // 装备
	ret = s.copyGameData(db.KeyUserDutyInfo(uaid), db.KeyUserDutyInfo(copyUaid))   // 日值
	ret = s.copyGameData(db.KeyUserTutorial(uaid), db.KeyUserTutorial(copyUaid))   // Tutorial
	ret = s.copyGameData(db.KeyUserQuestInfo(uaid), db.KeyUserQuestInfo(copyUaid)) // QuestHandler
	ret = s.copyGameData(db.KeyUserCardSkin(uaid), db.KeyUserCardSkin(copyUaid))   // 皮肤
	ret = s.copyGameData(db.KeyUserCurrency(uaid), db.KeyUserCurrency(copyUaid))   // 货币
	ret = s.copyGameData(db.KeyCampaign(uaid), db.KeyCampaign(copyUaid))           // Campaign 运动竞赛
	ret = s.copyGameData(db.KeyUserLevelInfo(uaid), db.KeyUserLevelInfo(copyUaid)) // chapter userLevelInfo
	ret = s.copyGameData(db.KeyUserShopInfo(uaid), db.KeyUserShopInfo(copyUaid))   // 商店
	ret = s.copyGameData(db.KeyUserStoryFlag(uaid), db.KeyUserStoryFlag(copyUaid))
	ret = s.copyGameData(db.KeyUserSign(uaid), db.KeyUserSign(copyUaid))               // 签到
	ret = s.copyGameData(db.KeyUserLevelData(uaid), db.KeyUserLevelData(copyUaid))     // 用户等级
	ret = s.copyGameData(db.KeyUserMail(uaid), db.KeyUserMail(copyUaid))               // 邮件
	ret = s.copyGameData(db.KeyUserTrial(uaid), db.KeyUserTrial(copyUaid))             // 试炼
	ret = s.copyGameData(db.KeyUserBlockWay(uaid), db.KeyUserBlockWay(copyUaid))       // 公路事件
	ret = s.copyGameData(db.KeyUserCampPool(uaid), db.KeyUserCampPool(copyUaid))       // 家具抽卡
	ret = s.copyGameData(db.KeyUseLimit(uaid), db.KeyUseLimit(copyUaid))               // 付费计数
	ret = s.copyGameData(db.KeyUserAchieve(uaid), db.KeyUserAchieve(copyUaid))         // 成就
	ret = s.copyGameData(db.KeyUserRelation(uaid), db.KeyUserRelation(copyUaid))       // 关系系统
	ret = s.copyGameData(db.KeyUserGuideTask(uaid), db.KeyUserGuideTask(copyUaid))     // 引导任务
	ret = s.copyGameData(db.KeyUserTravelLevel(uaid), db.KeyUserTravelLevel(copyUaid)) // 旅途关卡
	return ret
}

func (s *IDIPServer) GetRollingVersion() []byte {
	res := GetRollingVersionRes{
		Version: global.ROLLING_VERSION,
	}
	byte_, _ := json.Marshal(res)
	return byte_
}
