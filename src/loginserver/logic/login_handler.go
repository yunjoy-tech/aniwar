package logic

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"gitee.com/bychannel/aniwar/src/common/datalog/taptap"

	"gitee.com/bychannel/musae/framework/global"

	"gitee.com/bychannel/musae/framework/base"

	"gitee.com/bychannel/aniwar/src/common/auth"
	"gitee.com/bychannel/aniwar/src/common/rsa"
	"gitee.com/bychannel/aniwar/src/common/sdkconstant"

	"gitee.com/bychannel/aniwar/src/common/conf"
	"gitee.com/bychannel/aniwar/src/common/db"
	"gitee.com/bychannel/aniwar/src/proto/pb"
	"gitee.com/bychannel/musae/framework/baseconf"
	"gitee.com/bychannel/musae/framework/logger"
	"gitee.com/bychannel/musae/framework/metrics"
	"gitee.com/bychannel/musae/framework/service"
	"gitee.com/bychannel/musae/framework/threading"
	"gitee.com/bychannel/musae/framework/utils"
	"google.golang.org/protobuf/proto"
)

func (s *LoginServer) pushMsg(msg *Msg) {
	s.ch <- msg
	logger.Debugf("LoginServer pushMsg %v", msg.String())
	/*select {
	case s.ch <- msg:
	default:
		// TODO 向客户端回个服务端繁忙消息?
		logger.Debug("login msg chan full, drop msg:%d", msg.msgId)
	}*/
}

func (s *LoginServer) GrantGateTicket() {
	for i := int32(0); i < conf.GConf().Base.LoginReqRate; i++ {
		select {
		case s.ticket <- struct{}{}:
			s.ticketAddSum++
		default:
			return
		}
	}
}

func (s *LoginServer) doHandleMsg() {
	var err error
	for {
		select {
		case <-s.ticket:
			threading.RunSafe(func() {
				msg := <-s.ch
				s.ticketDecSum++
				res := s.handleLoginReq(msg)
				if res.ErrCode == int32(pb.ErrorCode_Success) {
					err = msg.ctx.Reply(int32(pb.Protocols_PLS2C_LoginRes), res.ErrCode, res)
				} else {
					err = msg.ctx.Reply(int32(pb.Protocols_PS2C_ErrorCodeNtf), res.ErrCode, &pb.S2C_ErrorCodeNtf{ErrorCode: uint32(res.ErrCode), Param: []string{strconv.Itoa(int(res.ErrCode))}})
				}

				if err != nil {
					logger.Error(err.Error())
				}
			})
		}
	}
}

func (s *LoginServer) handleLoginReq(msg *Msg) *pb.LS2C_LoginRes {
	req := &pb.C2LS_LoginReq{}
	err := proto.Unmarshal(msg.Data, req)
	res := &pb.LS2C_LoginRes{}
	if err != nil {
		res.ErrCode = int32(pb.ErrorCode_DeSerializeError)
		logger.Warnf("OnNetMessage err,%v %v %v %v", err, pb.Protocols(msg.msgId), req, res)
		return res
	}

	// 配置了登录开关，判定一下
	if b, err := s.GetConfigKeyForBool(db.KeyCfgLoginSwitch); err == nil && b {
		res.ErrCode = int32(pb.ErrorCode_LoginClose)
		return res
	}

	// 配置了登录IP黑名单，判定一下
	if str, err := s.GetConfigKeyForStr(db.KeyCfgLoginBlackIp); err == nil {
		ipArr := strings.Split(str, ",")
		if len(ipArr) > 0 {
			for _, ip := range ipArr {
				if len(ip) > 0 && ip == msg.ClientIp {
					res.ErrCode = int32(pb.ErrorCode_LoginClose)
					return res
				}
			}
		}
	}

	var (
		base64SrvKey string
		rsaKey       string
	)
	// 交换rsa随机值
	if baseconf.GetBaseConf().UseEncrypt == 1 {
		_, base64SrvKey, rsaKey = rsa.CreateSrvRsaKey(nil, req.CliRandomSeed)
	}

	sessionId := utils.GenIntGUID()
	uid, errCode := s.DoHandleLoginReq(req, res, msg.ClientIp, rsaKey)
	res.ErrCode = int32(errCode)
	if res.ErrCode == int32(pb.ErrorCode_Success) {
		res.AccountId = uid
		res.GatewayIp = global.TcpAddr
		res.GatewayPort = uint32(conf.GConf().Base.GatePort)
		res.SessionId = uint64(sessionId)
		res.UseRsa = baseconf.GetBaseConf().UseEncrypt // Gate上启动加密通信
		res.SrvRandomSeed = base64SrvKey

		/*if !global.IsCloud && conf.Base().AutoGateIp {
			gateIP, err := utils.ExternalIP()
			if err != nil {
				logger.Error("ExternalIP failed")
			} else {
				res.GatewayIp = gateIP
				logger.Debug("ExternalIP: ", gateIP)
			}
		}
		if s.Gateway != "" {
			res.GatewayIp = s.Gateway
		}
		if global.IsCloud {
			res.GatewayUrl = conf.Login().GateUrl
		} else {
			res.GatewayUrl = fmt.Sprintf(conf.Login().GateUrlDev, res.GatewayIp)
		}*/
		metrics.GaugeInc(metrics.LoginSucceedCount)
	} else {
		metrics.GaugeInc(metrics.LoginFailedCount)
	}

	logger.Infof("handleLoginReq [LoginStep] ErrCode:[%v], req=%s, res=%s", pb.ErrorCode(res.ErrCode), utils.PrettyJson(req), utils.PrettyJson(res))
	return res
}

func (s *LoginServer) DoHandleLoginReq(req *pb.C2LS_LoginReq, res *pb.LS2C_LoginRes, clientIp string, rsaKey string) (string, pb.ErrorCode) {
	var (
		err     error
		uid     string
		channel string
		// taptapToken  string
		taptapOpenId string
		tapUserInfo  *pb.TaptapUserInfo
	)

	logger.Infof("请求登陆, LoginChannelType:%s, req:%v", req.LoginChannelType, req)

	switch req.LoginChannelType {
	case pb.LoginChannelType_SS_Game:
		// 还未开通
		return fmt.Sprintf("%v_%v", "ssgame", req.AccountId), pb.ErrorCode_Account_auth_fail
	case pb.LoginChannelType_Lilith_Game:
		appUidStr := req.AccountId
		appToken := req.AccountPasswd

		appUid, err := strconv.Atoi(appUidStr)
		if err != nil {
			logger.Warnf("lilith, 登陆请求验证, 无效的请求参数 appUidStr=%s, err:%v", appUidStr, err.Error())
			return "", pb.ErrorCode_Account_auth_fail
		}

		if _, errCode := s.handleAuthLilith(appUid, appToken); errCode != pb.ErrorCode_Success {
			return "", errCode
		} else {
			uid = sdkconstant.GenLilithUid(appUid)
			channel = sdkconstant.GenLilithChannel()
		}

	case pb.LoginChannelType_Taptap_Game:
		if id, _tapUserInfo, errCode := s.handleAuthTaptap(req.AccountId, req.AccountPasswd, req.Extra); errCode != pb.ErrorCode_Success {
			return "", errCode
		} else {
			uid = sdkconstant.GenTaptapUid(id)
			channel = sdkconstant.GenTaptapChannel()
			tapUserInfo = _tapUserInfo
			taptapOpenId = req.AccountId
		}
	case pb.LoginChannelType_KuaiBao_Game:
		if errCode := s.handleAuthKuaiBao(req.AccountId, req.AccountPasswd); errCode != pb.ErrorCode_Success {
			return "", errCode
		} else {
			uid = sdkconstant.GenKuaiBaoUid(req.AccountId)
			channel = sdkconstant.GenKuaiBaoChannel()
		}

	default:
		// 判断账号合法的字符
		if accountLegalChar(req.AccountId) {
			logger.Warnf("无效账号, account=%s", req.AccountId)
			return "", pb.ErrorCode_AccountInvalid
		}
		if errCode := s.handleAuthPC(req); errCode != pb.ErrorCode_Success {
			return "", errCode
		} else {
			uid = sdkconstant.GenPCUid(req.AccountId)
			channel = sdkconstant.GenPCChannel()
		}
	}
	if uid == "" {
		logger.Warnf("没有生成对应的账号, account=%s", req.AccountId)
		return "", pb.ErrorCode_Account_auth_fail
	}
	// 账号白名单检查
	if b, err := s.GetConfigKeyForBool(db.KeyCfgOpenWhiteList); err == nil && b {
		if !s.checkWhitelist(uid) {
			logger.Warnf("登陆白名单验证失败 account=%s", uid)
			return "", pb.ErrorCode_LoginClose
		}
	}
	// 账号黑名单检查
	if b, err := s.GetConfigKeyForBool(db.KeyCfgOpenBlackList); err == nil && b {
		if s.checkBlacklist(uid) {
			logger.Warnf("登陆黑名单验证限制 account=%s", uid)
			return "", pb.ErrorCode_LoginBlackLimit
		}
	}

	now := time.Now().Unix()

	res.Token, err = auth.EncodeAuthToken(uid, channel, utils.GenStrUUID(), int64(baseconf.GetBaseConf().AccTokenTTL))
	if err != nil {
		res.ErrCode = int32(pb.ErrorCode_InternalError)
		logger.Warnf("create token error, account=%s", req.AccountId)
		return "", pb.ErrorCode_InternalError
	}
	// 验证最后登录时间
	lastToken := ""
	lastSession, _, _ := s.GetUserSession(uid)
	if lastSession != nil {
		lastToken = lastSession.Token
		if now < lastSession.LastLoginTime {
			logger.Warnf("登录请求过于频繁")
			return "", pb.ErrorCode_RepeatMsg
		}
	}

	// 加载account信息
	// 加锁，防止多次创建
	ok, err := s.TryLock(uid, db.KeyAccountLoginLock(uid), service.LOCK_TTL_SEC)
	defer s.UnLock(uid, db.KeyAccountLoginLock(uid))
	if !ok || err != nil {
		logger.Errorf("TryLock err:%s, %s, %s", err.Error(), uid, db.KeyAccountLoginLock(uid))
		return "", pb.ErrorCode_InternalError
	}
	// var isCreate bool
	kvTable, err := s.GetMongoAccount(db.KeyAccountInfo(uid), nil)
	var account *pb.UserData
	if err != nil { // 第一次创建，没有账号数据
		if !errors.Is(err, service.DB_ERROR_NOT_EXIST) {
			logger.Warn("GetMongoAccount err: ", err)
			return "", pb.ErrorCode_InternalError
		} else { // account数据不存在,创建
			// 注册开关判定
			if b, err := s.GetConfigKeyForBool(db.KeyCfgRegisterSwitch); err == nil && b {
				return "", pb.ErrorCode_RegisterClose
			}

			// 注册IP黑名单判定
			if str, err := s.GetConfigKeyForStr(db.KeyCfgRegisterBlackIp); err == nil {
				ipArr := strings.Split(str, ",")
				for _, ip := range ipArr {
					if len(ip) > 0 && ip == clientIp {
						res.ErrCode = int32(pb.ErrorCode_RegisterClose)
						return "", pb.ErrorCode_RegisterClose
					}
				}
			}

			// 判定注册上限，避免创建账号
			if limit, err := s.GetConfigKeyForInt(db.KeyCfgServerRegisterLimit); err == nil && limit > 0 {
				count, err := s.RedisBitCount(context.Background(), db.KeyServerRegisterUsers(), nil)
				if err != nil || count >= int64(limit) {
					return "", pb.ErrorCode_RegisterLimit
				}
			}

			account = s.createAccount(req.AccountId, uid, channel, now)
			// isCreate = true
			logger.Infof("[LoginServer] createAccount %s", uid)
		}
	} else {
		ud := &pb.UserData{}
		err = proto.Unmarshal(kvTable.Data, ud)
		if err != nil {
			logger.Warn("proto unmarshal err: ", err)
			return "", pb.ErrorCode_InternalError
		}
		account = ud
	}

	if account == nil {
		logger.Errorf("get or create account err: ", err)
		return "", pb.ErrorCode_InternalError
	}

	if account.Account.BannedTs > now {
		logger.Infof("账号封禁中, account: %s, sec: %v", account.Account.Uid, account.Account.BannedTs-now)
		return "", pb.ErrorCode_AccountBanned
	}

	// 客户端设备信息
	account.CliDeviceInfo = req.CliDeviceInfo
	if req.LoginChannelType == pb.LoginChannelType_Taptap_Game && req.CliDeviceInfo.Channel == "" {
		account.CliDeviceInfo.Channel = "taptap"
	}

	// tap用户信息
	if tapUserInfo != nil {
		account.TapUserInfo = tapUserInfo
	}

	// 存储账号信息
	err = s.SaveAccount(account)
	if err != nil {
		return "", pb.ErrorCode_SaveDBError
	}

	// 预先拉启UserActor
	if conf.Login().UserActorAhead {
		threading.GoSafe(func() {
			player, ok := account.PlayerList.Players[1]
			if ok && player != nil && player.Id > 0 {
				uaid := s.UAID(uid, player.Id)
				s.UserInvoke(uaid, &base.ProtoMsg{
					AppId:   s.AppId,
					MsgId:   int32(pb.Protocols_PS2S_SvcInvokeReq),
					UserId:  uid,
					RoleId:  player.Id,
					UAID:    uaid,
					Data:    nil,
					ErrCode: 0,
					// GUID:    utils.GenIntUUID(),
					ServerReqIdx: utils.GenIntUUID(),
				})
			}
		})
	}

	// 通知客户端被踢下线
	if isOnline, session := s.PlayerIsOnline(account.Account.Uid); isOnline {
		if session != nil {
			_, err = s.GetGlobalRedis(db.KeyHeartBeat(session.Uid), nil)
			heartBeat := s.GetHeartBeat(session.Uid)

			if heartBeat != nil {
				ret := &pb.S2C_ErrorCodeNtf{ErrorCode: uint32(pb.ErrorCode_KnockedOff)}
				logger.Infof("kickout knocked uid:%s  ", account.Account.Uid)
				// err = s.Send2GateOne2(session.Uid, strconv.Itoa(int(session.PlayerId)), heartBeat.UserInfo, ret)
				err = s.Send2Gate(strconv.Itoa(int(session.PlayerId)), &pb.ActorUserInfo{Uid: session.Uid, GateId: heartBeat.GateTopic}, ret)
				if err != nil {
					logger.Errorf("Send2Gate err:%+v", err)
				}
				taptap.RepeatedLoginComm(session.Uid, tapUserInfo, req.CliDeviceInfo)
			}
		}
	}

	// table := &state.KvTable{Id: 0, UID: accountId, Data: []byte(token), UpSecTS: 0, InSecTS: now}
	// err = s.SaveGlobalRedis(db.KeyAccountToken(accountId), table, map[string]string{"ttlInSeconds": strconv.Itoa(conf.GConf().Base.AccTokenTTL)})
	// lastToken := ""
	// lastSession, _, _ := s.GetUserSession(account.Account.Uid)
	// if lastSession != nil {
	//	lastToken = lastSession.Token
	// }

	session := &pb.UserSession{
		Uid:             account.Account.Uid,
		OpenId:          account.Account.OpenId,
		Channel:         account.Account.Channel,
		Uaid:            "",
		PlayerId:        0,
		Token:           res.Token,
		LastToken:       lastToken,
		LimitTs:         now + int64(conf.GConf().DDos.TimeInterval),
		LimitNum:        0,
		LimitSize:       0,
		LastHeartbeatTs: 0,
		OnlineTs:        now,
		OfflineTs:       0,
		LastRspData:     nil,
		CryptKey:        rsaKey,
		DeviceId:        req.CliDeviceInfo.AndroidId,
		// TaptapToken:     tapUserInfo.Token,
		TaptapOpenId:  taptapOpenId,
		LastLoginTime: time.Now().Unix() + 5, // 相当于login 请求的CD
	}
	if tapUserInfo != nil {
		session.TaptapToken = tapUserInfo.Token
	}

	err = s.SaveUserSession(session)
	if err != nil {
		logger.Error("HandleLogin, token save failed,", req)
	}

	// err = s.SaveToken(uid, session.Token)
	// if err != nil {
	//	logger.Errorf("SaveToken err:%+v,", err)
	// }
	taptap.AccountLoginComm(uid, tapUserInfo, req.CliDeviceInfo, int32(req.LoginChannelType), req.Extra)

	return uid, pb.ErrorCode_Success
}

// 账号合法的字符
func accountLegalChar(s string) bool {
	for _, c := range s {
		// 大小写字母，数字，下划线，其他字符均为非法
		if !unicode.IsNumber(c) && !unicode.IsLetter(c) && c != '_' {
			return true
		}
	}
	return false
}
