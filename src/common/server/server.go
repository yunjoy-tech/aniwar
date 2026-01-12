package server

import (
	"context"
	"errors"
	"fmt"
	"gitee.com/aniwar2/aniwar/src/common"
	"gitee.com/aniwar2/aniwar/src/common/actor/stub"
	"gitee.com/aniwar2/aniwar/src/common/conf"
	"gitee.com/aniwar2/aniwar/src/common/db"
	"gitee.com/aniwar2/aniwar/src/common/sdkconstant"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/base"
	"gitee.com/aniwar2/musae/gamelib/guid"
	"gitee.com/aniwar2/musae/global"
	"gitee.com/aniwar2/musae/logger"
	"gitee.com/aniwar2/musae/metrics"
	"gitee.com/aniwar2/musae/service"
	svc "gitee.com/aniwar2/musae/service"
	"gitee.com/aniwar2/musae/state"
	"gitee.com/aniwar2/musae/tcpx"
	dapr "github.com/dapr/go-sdk/client"
	"github.com/xuri/excelize/v2"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"os"
	"strconv"
	"strings"
	"time"
)

// 一个通用的微服务实例结构，对创建微服务需要的能力进行封装
type Server struct {
	svc.Service
	pack         *tcpx.Packx       // TODO 理论上这个只有gateServer需要
	version      *VersionSupport   // TODO 这个只有loginServer需要，当有不兼容更新的时候，需要 停服维护 / T人维护
	Args         map[string]string // 运行参数列表
	LiveTime     int64             // 生存时间戳
	NeedExcel    map[string]int    // TODO musae提供支持，根据srv类型进行加载 需要加载的策划excel表
	LocalizedStr map[string]string // TODO 废弃 国际化文本，只取中文，后台可视化展示使用
}

// IsValidAppId check appid
func IsValidAppId(appid string) bool {
	switch appid {
	case GUIDE_SVC, LOGIN_SVC, GATE_SVC, IDIP_SVC, BILL_SVC,
		ACTOR_SVC, CENTER_SVC, LOBBY_SVC:
		return true
	default:
		return false
	}
}

func RpcErr(err error, code pb.ErrorCode) base.RpcError {
	if err == nil {

	}
	return base.RpcError{Err: err, Code: int32(code)}
}

func (s *Server) GetUAIDByRoleId(roleId uint64) (string, error) {
	kvt, err := s.GetCache(service.MongoDbType_MongoGame, db.KeyPlayerUAID(roleId), ICache(s))
	if err == nil && kvt != nil {
		return s.UAID(kvt.UID, kvt.Id), nil
	}
	return "", err
}

func (s *Server) UAID(accountId string, roleId uint64) string {
	return fmt.Sprintf("%s_%v", accountId, roleId)
}

func (s *Server) ConvUAID(uaid string) (string, uint64) {
	if uaid == "" {
		return "", 0
	}
	idx := strings.LastIndex(uaid, "_")
	if idx <= 0 || idx >= len(uaid)-1 {
		return "", 0
	}
	roleId, err := strconv.ParseUint(uaid[idx+1:], 10, 64)
	if err != nil {
		return "", 0
	}
	accountId := uaid[0:idx]
	return accountId, roleId
}

func (s *Server) OnUpdateStatus() {
	req := &pb.S2S_SvcStatusReq{}
	if global.AppID == ACTOR_SVC {
		req.Actor = map[string]*pb.ActorStatus{}
		res, err := s.Daprc.GrpcClient().GetMetadata(context.Background(), &emptypb.Empty{})
		if err != nil {
			logger.Warn("OnUpdateActorCount err:", err)
			return
		}
		status := &pb.ActorStatus{Counts: []*pb.ActorCount{}}
		for _, actor := range res.ActiveActorsCount {
			status.Counts = append(status.Counts, &pb.ActorCount{
				Type:  actor.Type,
				Count: actor.Count,
			})
			if actor.Type == "UserActor" {
				metrics.GaugeSet(metrics.UserActorCount, int64(actor.Count))
			}
			if actor.Type == "RoomActor" {
				metrics.GaugeSet(metrics.RoomActorCount, int64(actor.Count))
			}
			if actor.Type == "AllianceActor" {
				metrics.GaugeSet(metrics.AllianceActorCount, int64(actor.Count))
			}
		}

		// 更新用户列表
		var count int32
		var del []string
		s.OnlinePlayers.Range(func(key, value any) bool {
			if time.Now().Unix() > value.(int64)+int64(conf.Base().ServerHeartbeatTimout) {
				del = append(del, key.(string))
			} else {
				count += 1
			}
			return true
		})
		for _, v := range del {
			s.OnlinePlayers.Delete(v)
		}
		status.Counts = append(status.Counts, &pb.ActorCount{
			Type:  global.PlayerCountType,
			Count: count,
		})
		metrics.GaugeSet(metrics.UserCount, int64(count))
		req.Actor[global.HostName] = status
	}

	req.Service = &pb.ServiceData{
		Name:           s.PrivateTopicID(),
		StartTime:      global.StartTime,
		ReportTS:       time.Now().Unix(),
		AppVersion:     global.APP_VERSION,
		RollingVersion: global.ROLLING_VERSION,
	}

	// logger.Debugf("SvcStatusReq: %+v", req)
	data, err := proto.Marshal(req)
	if err != nil {
		logger.Warn("OnUpdateActorCount proto.Marshal  err:", err)
		return
	}
	msg, err := s.ActorInvoke(stub.CenterActorType, global.CenterActorID, &base.ProtoMsg{
		AppId:   ACTOR_SVC,
		MsgId:   int32(pb.Protocols_PS2S_SvcStatusReq),
		UserId:  "",
		RoleId:  0,
		UAID:    global.CenterActorID,
		Data:    data,
		ErrCode: 0,
		ReqIdx:  guid.GenIntUuid(),
		Topic:   "",
		Uids:    nil,
	})
	if err != nil {
		logger.Warn("ActorInvoke err:", err)
		return
	}
	res := &pb.S2S_SvcStatusRes{}
	proto.Unmarshal(msg.Data, res)

	for _, actor := range res.Counts {
		switch actor.Type {
		case global.PlayerCountType:
			global.TotalPlayerCount = actor.Count
		case stub.UserActorType:
			global.UserActorCount = actor.Count
		case stub.RoomActorType:
			global.RoomActorCount = actor.Count
		case stub.AllianceActorType:
			global.AllianceActorCount = actor.Count
		}
	}

	global.GateServices = []string{}
	for _, srv := range res.Services {
		if IsGate(srv.Name) {
			global.GateServices = append(global.GateServices, srv.Name)
		}
	}
}

func (s *Server) OnTimerEventCB(cb base.TimerEventCB) {
	defer func() {
		if err := recover(); err != nil {
			logger.Error("[OnTimerEventCB] recover, err:", err)
		}
	}()
	err := cb()
	if err != nil {
		logger.Error(err)
	}
}

// PlayerIsOnline 是否在线
func (s *Server) PlayerIsOnline(uid string) (bool, *pb.UserSession) {
	session, _, _ := s.GetUserSession(uid)
	if session == nil {
		return false, session
	}
	if time.Now().Unix() > session.LastHeartbeatTs+
		int64(conf.Base().HeartbeatTimout) {
		return false, session
	}
	return true, session
}

func (s *Server) SaveMongoAndRedisDB(mongoDbType service.MongoDbType, dbKey string, msg proto.Message, meta map[string]string, so ...dapr.StateOption) {
	kvTable, err := db.BuildKvTable(msg, dbKey)
	if err != nil {
		logger.Errorf("sever SaveMongoAndRedisDB got error:%v", err.Error())
		return
	}

	s.SaveMongoAndRedisDBByKvTable(mongoDbType, dbKey, kvTable, meta, so...)
}

func (s *Server) SaveMongoAndRedisDBByKvTable(mongoDbType service.MongoDbType, dbKey string, kvTable *state.KvTable, meta map[string]string, so ...dapr.StateOption) {
	if err := s.SaveMongo(mongoDbType, dbKey, kvTable, meta, so...); err != nil {
		logger.Errorf("sever SaveMongoAccount got error:%v", err.Error())
		return
	}

	if err := s.SaveGlobalRedis(dbKey, kvTable, meta, so...); err != nil {
		logger.Errorf("sever SaveGlobalRedis got error:%v", err.Error())
		return
	}
}

// SetFromDbIfNotCacheHandler 指定缓存数据的落地库
func (s *Server) SetFromDbIfNotCacheHandler(mongoDbName service.MongoDbType, key string, kvTable *state.KvTable) error {
	var (
		err       error
		startTime = time.Now()
	)

	err = s.SaveDbByKvTable(mongoDbName, key, kvTable)
	logger.WarnDelayf(time.Since(startTime).Milliseconds(), "SetFromDbIfNotCacheHandler: db:%s, key:%v", mongoDbName, key)

	return err
}

// GetFromDbIfNotCacheHandler 指定缓存数据的加载源
func (s *Server) GetFromDbIfNotCacheHandler(mongoDbName service.MongoDbType, key string) (*state.KvTable, error) {
	var (
		err       error
		kvTable   *state.KvTable
		startTime = time.Now()
	)

	kvTable, err = s.LoadMongoDB(mongoDbName, key, nil)
	logger.WarnDelayf(time.Since(startTime).Milliseconds(), "")

	return kvTable, err
}

func (s *Server) SaveDbByKvTable(mongoDbName service.MongoDbType, key string, kvTable *state.KvTable) error {
	var (
		err error
	)

	switch mongoDbName {
	case service.MongoDbType_MongoAccount:
		err = s.SaveMongoAccount(key, kvTable, nil)
		if err != nil {
			return err
		}

	case service.MongoDbType_MongoGame:
		// 保存
		err = s.SaveMongoGame(key, kvTable, nil)
		if err != nil {
			return err
		}

	case service.MongoDbType_MongoMail:
		logger.Errorf("还未支持该数据库, dbName=%s", mongoDbName)
	default:
		logger.Errorf("还未支持该数据库, dbName=%s", mongoDbName)
	}

	logger.Debugf("UserActor SaveDB,%s, %s, %s", mongoDbName, key, kvTable.ToString())
	return nil
}

func (s *Server) LoadMongoDB(mongoDbName service.MongoDbType, key string, value proto.Message) (*state.KvTable, error) {
	var (
		err     error
		kvTable *state.KvTable
	)

	switch mongoDbName {
	case service.MongoDbType_MongoAccount:
		kvTable, err = s.GetMongoAccount(key, nil)

	case service.MongoDbType_MongoGame:
		kvTable, err = s.GetMongoGame(key, nil)

	case service.MongoDbType_MongoMail:
		logger.Errorf("还未支持该数据库, dbName=%s", mongoDbName)
	default:
		logger.Errorf("还未支持该数据库, dbName=%s", mongoDbName)
	}

	if err != nil {
		return nil, err // if data NOT exist, err is service.DB_ERROR_NOT_EXIST
	}

	if kvTable == nil {
		return kvTable, nil
	}

	// parse kvTable to protoObj
	err = db.ParseKvTable(kvTable, value)
	if err != nil {
		return nil, err
	}

	return kvTable, nil
}

func (s *Server) UpdateUAIDCache(uid string, playerId uint64, savedb ...bool) error {
	logger.Debugf("玩家UAID更新 uid:%s, roleId:%d", uid, playerId)
	now := time.Now().Unix()
	table := &state.KvTable{Id: playerId, UID: uid, Data: []byte(s.UAID(uid, playerId)), UpSecTS: now, InSecTS: now, DataSrc: s.UAID(uid, playerId)}
	kvTableMap := make(map[string]*state.KvTable, 2)
	kvTableMap[db.KeyPlayerUAID(playerId)] = table
	kvTableMap[db.KeyAccountUAID(uid)] = table
	meta := map[string]string{"ttlInSeconds": strconv.Itoa(conf.Base().AccTokenTTL)} // 过期时间

	// 持久化roleId-uaid映射
	var f bool
	if len(savedb) > 0 {
		f = savedb[0]
	}
	if f {
		err := s.SaveMongoAndRedis(service.MongoDbType_MongoGame, db.KeyPlayerUAID(playerId), table, nil, ICache(s))
		if err != nil {
			return err
		}
	}

	return s.UpsertRedisTableTransaction(service.RedisCache, meta, kvTableMap)
}

func (s *Server) GetPlayerId(uid string) (uint64, error) {
	// 查询db中的数据
	var playerId uint64
	// 先查询redis
	if kvTable, err := s.GetCacheRedis(db.KeyAccountUAID(uid), nil); err == nil &&
		kvTable != nil && kvTable.Id > 0 {
		playerId = kvTable.Id
		return playerId, nil
	}
	if kvTable, err := s.GetMongoAccount(db.KeyAccountInfo(uid), nil); err != nil {
		return 0, err
	} else {
		// 角色存在
		account := &pb.UserData{}
		err = proto.Unmarshal(kvTable.Data, account)
		if err != nil {
			logger.Warn("proto unmarshal err: ", err)
			return 0, err
		}
		// var sync bool
		if account.PlayerList != nil && account.PlayerList.Players != nil {
			player, ok := account.PlayerList.Players[1]
			if ok && player != nil {
				playerId = player.Id
				// uaid缓存失效,更新缓存
				s.UpdateUAIDCache(uid, playerId, true)
			} else {
				if account.PlayerList.PlayerId > 0 {
					playerId = account.PlayerList.PlayerId
				} else {
					id := guid.GenIntUuid()
					if id == 0 {
						return 0, fmt.Errorf("guid error")
					}
					playerId = uint64(id + common.USER_ID_BASE)
				}
			}
		}
		if playerId > 0 {
			return playerId, nil
		}
	}
	return 0, fmt.Errorf("playerId error")
}

func (s *Server) KickOutUser(uid string) error {
	// err := s.SaveToken(uid, "")
	userSession, err, _ := s.GetUserSession(uid)
	if err != nil {
		return err
	}

	userSession.Token = ""
	userSession.LastToken = ""
	err = s.SaveUserSession(userSession)
	if err != nil {
		return err
	}
	logger.Info("kickout user:", uid)
	return nil
}

// LoadNeedExcel 加载server所需的策划配置
// @param assign 表示本次指定更新的配置文件
func (s *Server) LoadNeedExcel(assign []string) error {
	temp := make(map[string]int)
	for _, v := range assign {
		temp[v] = 0
	}

	files := make([]string, 0)
	for f := range s.NeedExcel {
		if _, ok := temp[f]; !ok && len(temp) > 0 {
			continue
		}
		files = append(files, f)
	}
	// return data.LoadByFileNames(s.MetaDir, files, s.AppId, s.AppId)
	return nil
}

// LoadLocalizedStr 加载需要的国际化配置文件
func (s *Server) LoadLocalizedStr() error {
	s.LocalizedStr = make(map[string]string)
	var fpath = "./output/res/localization/"
	files, err := os.ReadDir(fpath)
	if err != nil {
		return err
	}
	for _, finfo := range files {
		if finfo.IsDir() {
			continue
		}
		// 读取文件内容
		f, err := excelize.OpenFile(fpath + finfo.Name())
		if err != nil {
			continue
		}
		defer func() {
			// Close the spreadsheet.
			if err = f.Close(); err != nil {
				logger.Warn(err)
			}
		}()

		// 遍历sheet
		for _, sheet := range f.GetSheetList() {
			rows, err := f.GetRows(sheet)
			if err != nil {
				continue
			}

			for i, row := range rows {
				if i <= 1 || len(row) < 3 {
					continue
				}
				s.LocalizedStr[row[1]] = row[2]
			}
		}
	}
	return nil
}

func (s *Server) ErrorPack(errCode pb.ErrorCode) []byte {
	ret := &pb.S2C_ErrorCodeNtf{ErrorCode: uint32(errCode)}
	data, err := s.Pack(pb.Protocols_PS2C_ErrorCodeNtf, errCode, ret, "") // 错误消息不走加密
	if err != nil {
		data = []byte(fmt.Sprintf("error pack:%s", err.Error()))
	}
	return data
}

func (s *Server) CheckToken(session *pb.UserSession, token string) (error, pb.ErrorCode) {
	if session == nil {
		return errors.New("无效的session"), pb.ErrorCode_TokenInvalid
	}

	// currToken, err := s.GetToken(session.Uid)
	// if err != nil {
	//	logger.Errorf("GetToken err:%+v", err)
	//	if s.IsLastToken(session, token) {
	//		return errors.New("上次的token"), pb.ErrorCode_KnockedOff
	//	} else {
	//		return errors.New("获取token报错"), pb.ErrorCode_TokenInvalid
	//	}
	// }
	currToken := session.Token

	if token == "" || token != currToken {
		if s.IsLastToken(session, token) {
			return errors.New("上次的token"), pb.ErrorCode_KnockedOff
		} else {
			return errors.New("获取token报错"), pb.ErrorCode_TokenInvalid
		}
	}

	// 判断当前token是否超时失效
	nowSec := time.Now().Unix()
	if conf.DDos().TimeInterval > 0 && // 配置过期时间为0, 表示不做过期时间检查
		nowSec > session.LimitTs {
		return errors.New("token失效了"), pb.ErrorCode_TokenTimeout
	}

	if token == currToken {
		return nil, pb.ErrorCode_Success
	}

	return errors.New("无效的token"), pb.ErrorCode_TokenInvalid
}

func (s *Server) IsLastToken(session *pb.UserSession, token string) bool {
	if session == nil {
		return false
	}
	if token == "" {
		return false
	}

	// lastToken, err := s.GetToken(session.Uid)
	// if err != nil {
	//	logger.Errorf("IsLastToken GetToken err:%+v", err)
	//	return false
	// }

	if session.LastToken == token {
		return true
	}

	return false
}

func (s *Server) IsPCChannel(channel string) bool {
	if channel == sdkconstant.GenPCChannel() {
		return true
	}

	return false
}

func (s *Server) IsTapChannel(channel string) bool {
	if channel == sdkconstant.GenTaptapChannel() {
		return true
	}

	return false
}

// 查询taptap 映射的uid
func (s *Server) GetTaptapUid(unionId string) (int, error) {
	kvt, err := s.GetCache(service.MongoDbType_MongoGame, db.KeyTaptapOpenId(unionId), ICache(s))
	if err != nil || kvt == nil {
		return 0, err
	}
	return int(kvt.Id), nil
}

// 建立taptap openId-roleId映射
func (s *Server) UpdateTaptapUidCache(unionId string) (int, error) {
	now := time.Now().Unix()
	uid := uint64(guid.GenIntUuid())
	if uid == 0 {
		return 0, fmt.Errorf("uid generate failed")
	}
	uid += common.USER_ID_BASE
	table := &state.KvTable{Id: uid, UID: strconv.Itoa(int(uid)), Data: []byte(unionId), UpSecTS: now, InSecTS: now, DataSrc: unionId}
	meta := map[string]string{"ttlInSeconds": strconv.Itoa(conf.Base().AccTokenTTL)} // 过期时间

	err := s.SaveMongoAndRedis(service.MongoDbType_MongoGame, db.KeyTaptapOpenId(unionId), table, meta, ICache(s))
	if err != nil {
		return 0, err
	}

	return int(uid), nil
}

// 通知center热更
func (s *Server) NotifyCenterActor(ntfType int32, files, services []string) (*pb.S2S_HotReloadRes, error) {

	logger.Infof("通知center进行热更 type: %v, files: %v, services: %v", ntfType, files, services)

	reqData, err := proto.Marshal(&pb.S2S_HotReloadReq{
		Type:     ntfType,
		Files:    files,
		Services: services,
	})
	if err != nil {
		return nil, err
	}
	bytes := s.CenterSrvInvoke(int32(pb.Protocols_PS2S_HotReloadReq), reqData)
	res := &pb.S2S_HotReloadRes{}
	if err = proto.Unmarshal(bytes, res); err != nil {
		return nil, err
	}
	return res, nil
}

func GetExcelData(sheetName string) []byte {
	if !strings.Contains(sheetName, ".data") {
		sheetName = sheetName + ".data"
	}
	// excelData, err := data.GetDataByFileName(sheetName)
	// if err != nil {
	// 	return []byte(err.Error())
	// }
	// return []byte(excelData)
	return nil
}
