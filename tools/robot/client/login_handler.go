package client

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/forgoer/openssl"
	"github.com/yunjoy-tech/aniwar/src/common/server"
	"github.com/yunjoy-tech/aniwar/src/common/tls"
	"github.com/yunjoy-tech/aniwar/src/proto/pb"
	randutil "github.com/yunjoy-tech/musae/utils/rand"
	"robot/conf"
	"time"
)

type LoginHandler struct {
	client *Client
}

func NewLoginHandler(c *Client) *LoginHandler {
	l := &LoginHandler{client: c}
	c.RegisterProtoHandler(int32(pb.Protocols_PLS2C_LoginRes), l.LoginRes)
	// c.RegisterProtoHandler(int32(pb.Protocols_PDB2C_SessionAuthRes), l.SessionAuthRes)
	// c.RegisterProtoHandler(int32(pb.Protocols_PDB2C_QueryRoleListRes), l.QueryRoleListRes)
	// c.RegisterProtoHandler(int32(pb.Protocols_PDB2C_CreateRoleRes), l.CreateRoleRes)
	c.RegisterProtoHandler(int32(pb.Protocols_PLS2GWS_PlayerEnterGameNtf), l.EnterGameNtf)
	// c.RegisterProtoHandler(int32(pb.Protocols_PLS2C_RsaServerRandomRes), l.RsaRes)
	c.RegisterProtoHandler(int32(pb.Protocols_PG2C_LoginGameRes), l.LoginGameRes)
	c.RegisterProtoHandler(int32(pb.Protocols_PS2C_ErrorCodeNtf), l.handleErrorMsg)

	return l
}

func (h *LoginHandler) ReqVersionInfo() error {
	req := &server.VerReq{AccountId: h.client.Account, IsHttp: true}
	buf, err := json.Marshal(req)
	if err != nil {
		h.client.Error("json Marshal failed")
	}

	url := fmt.Sprintf("%s:20001/api/version", conf.GetConf().HttpAddr)
	buf, err = h.client.HttpSend(url, buf, false)
	if err != nil {
		h.client.Errorf("HttpPost err, url:[%s], err: %v", url, err)
		return err
	}

	version := &server.Version{}
	if err = json.Unmarshal(buf, version); err != nil {
		return err
	}
	// h.client.SrvAddr = version.ServerAddr[0]
	h.client.Debugf("版本信息请求 /api/version: %+v", version)
	return nil
}

func (h *LoginHandler) ReqLogin() {
	req := &pb.C2LS_LoginReq{
		AccountId:     h.client.Account,
		AccountPasswd: "111",
		CliRandomSeed: h.RsaReq(),
		CliDeviceInfo: &pb.CliDeviceInfo{AndroidId: ""},
	}
	buf, err := h.client.Pack(pb.Protocols_PC2LS_LoginReq, req)
	if err != nil {
		h.client.Warn("ReqLogin Pack error:", err)
	}

	h.client.SetState(wait_resp_state)
	// if h.client.isTcp {
	//	h.client.send <- buf
	// } else {
	//	h.client.HttpSend(fmt.Sprintf("%s:21001/api", h.client.SrvAddr), buf, true)
	// }
	h.client.SendMsg2Server(h.client.isTcp, h.client.GetLoginHttpApi(), buf, true)
	h.client.Debugf("登录login请求 C2LS_LoginReq: %+v", req)
}

func (h *LoginHandler) LoginRes(data []byte) {
	res := &pb.LS2C_LoginRes{}
	err := h.client.Unpack(data, res)
	if err != nil {
		h.client.Debug("MsgHandler Unpack error: ", err)
		return
	}
	h.client.Debugf("loginRes回包: %+v", res)

	h.client.SessionId = res.SessionId
	h.client.GatewayIp = res.GatewayIp
	h.client.GatewayPort = res.GatewayPort
	h.client.UseRsa = res.UseRsa
	h.client.NewAccount = res.AccountId
	h.client.Token = res.Token
	h.client.Debugf("是否使用加密通信: %+v", res.UseRsa)

	if h.client.UseRsa == 1 {
		h.RsaRes(res.SrvRandomSeed)
	}

	h.client.Disconnect()
	time.Sleep(1 * time.Second) // 等待login服务的send、recv协程退出
	h.client.SetState(enter_gate_state)
}

func (h *LoginHandler) LoginGateReq() {
	req := &pb.C2G_LoginGateReq{AccountId: h.client.NewAccount}
	buf, err := h.client.Pack(pb.Protocols_PC2G_LoginGateReq, req)

	if err != nil {
		h.client.Warn("LoginGateReq Pack error:", err)
	}

	// if h.client.isTcp {
	//	h.client.send <- buf
	// } else {
	//	h.client.HttpSend(h.client.GetGateHttpApi(), buf, true)
	// }
	h.client.SendMsg2Server(h.client.isTcp, h.client.GetGateHttpApi(), buf, true)

	h.client.SetState(wait_resp_state)
	h.client.Debugf("LoginGateReq: %+v", req)
}

func (h *LoginHandler) LoginGateRes(data []byte) {
	res := &pb.G2C_LoginGateRes{}
	h.client.Unpack(data, res)
	if res.Err_Code != int32(pb.ErrorCode_Success) {
		h.client.Debugf("LoginGateRes: %+v", res)
		time.Sleep(time.Second * 1)
		h.client.SetState(connerr_state)
		return
	}
	h.client.SetState(enter_game_state)
	h.client.Debugf("LoginGateRes: %+v", res)
}

func (h *LoginHandler) LoginGameReq() {
	req := &pb.C2G_LoginGameReq{AccountId: h.client.NewAccount, Token: h.client.Token, DeviceId: "PC"}
	buf, err := h.client.Pack(pb.Protocols_PC2G_LoginGameReq, req)

	if err != nil {
		h.client.Warn("LoginGameReq Pack error:", err)
	}

	// if h.client.isTcp {
	//	h.client.send <- buf
	// } else {
	//	h.client.HttpSend(h.client.GetGateHttpApi(), buf, true)
	// }
	h.client.SendMsg2Server(h.client.isTcp, h.client.GetGateHttpApi(), buf, true)

	h.client.SetState(wait_resp_state)
	h.client.Debugf("LoginGameReq: %+v", req)
}

func (h *LoginHandler) LoginGameRes(data []byte) {
	res := &pb.G2C_LoginGameRes{}
	h.client.Unpack(data, res)
	if res.Err_Code != int32(pb.ErrorCode_Success) {
		h.client.Debugf("LoginGameRes: %+v", res)
		time.Sleep(time.Second * 1)
		h.client.SetState(connerr_state)
		return
	}
	h.client.SetState(lobby_state)
	h.client.Debugf("LoginGameRes: %+v", res)

	// for _, info := range res.CommonData.Card {
	// h.client.CardHandler.data[info.Common.CardId] = info.Common
	// }
	// l.client.CardHandler.data = res.CommonData.Card
	// h.client.EquipHandler.data = res.CommonData.Equip
}

func (h *LoginHandler) RsaReq() string {
	h.client.cliKey = randutil.RandomStr(32, true, true, true)
	h.client.Debugf("客户端随机值: %+v", h.client.cliKey)
	encrypt, err := tls.RsaEncrypt([]byte(h.client.cliKey))
	if err != nil {
		h.client.Warn("RsaReq RsaEncrypt error:", err)
	}
	enStr := base64.StdEncoding.EncodeToString(encrypt)
	h.client.Debugf("客户端随机值-发送的数据: %+v", enStr)
	return enStr
	// req := &pb.C2LS_RsaClientRandomReq{CliRandomSeed: enStr}
	// buf, err := l.client.Pack(pb.Protocols_PC2LS_RsaClientRandomReq, req)
	//
	// if err != nil {
	//	h.client.Warn("RsaReq Pack error:", err)
	// }
	//
	// if l.client.isTcp {
	//	l.client.send <- buf
	// } else {
	//	l.client.HttpSend(fmt.Sprintf("%s:21001/api", l.client.SrvAddr), buf, true)
	// }
	// l.client.SetState(wait_resp_state)
	// h.client.Debugf("RsaReq: %+v", req)
}

func (h *LoginHandler) RsaRes(srvRandomSeed string) {
	// res := &pb.LS2C_RsaServerRandomRes{}
	// l.client.Unpack(data, res)

	deStr, err := base64.StdEncoding.DecodeString(srvRandomSeed)
	if err != nil {
		h.client.Warn("RsaRes DecodeString error:", err)
		// return
	}

	// 服务器随机码 AES(cbc)加密
	srvKeyEncrypt, err := openssl.AesCBCDecrypt(deStr, []byte(h.client.cliKey), make([]byte, 16), openssl.PKCS7_PADDING)

	h.client.rsaVal = tls.RsaVal(h.client.cliKey, string(srvKeyEncrypt))
	// l.client.SetState(enter_game_state)
	h.client.Debugf("最终通信秘钥: %+v", h.client.rsaVal)
}

func (h *LoginHandler) ReqSessionAuth() {
	req := &pb.C2DB_SessionAuthReq{SessionId: h.client.SessionId, AccountId: h.client.Account, SessionType: 1, DeviceId: "PC"}
	buf, err := h.client.Pack(pb.Protocols_PC2DB_SessionAuthReq, req)

	if err != nil {
		h.client.Warn("ReqLogin Pack error:", err)
	}

	// if h.client.isTcp {
	//	h.client.send <- buf
	// } else {
	//	h.client.HttpSend(h.client.GetGateHttpApi(), buf, true)
	// }
	h.client.SendMsg2Server(h.client.isTcp, h.client.GetGateHttpApi(), buf, true)

	h.client.SetState(wait_resp_state)
	h.client.Debugf("ReqSessionAuth: %+v", req)
}

// func (l *LoginHandler) SessionAuthRes(data []byte) {
//	res := &pb.DB2C_SessionAuthRes{}
//	l.client.Unpack(data, res)
//
//	h.client.Debugf("authRes回包: %+v", res)
//	l.ReqQueryRoleList()
// }
//
// func (l *LoginHandler) ReqQueryRoleList() {
//	req := &pb.C2DB_QueryRoleListReq{AccountId: l.client.Account}
//	buf, err := l.client.Pack(pb.Protocols_PC2DB_QueryRoleListReq, req)
//
//	if err != nil {
//		h.client.Warn("ReqLogin Pack error:", err)
//	}
//
//	l.client.send <- buf
//	h.client.Debugf("ReqQueryRoleList: %+v", req)
//
//	l.client.SetState(wait_resp_state)
// }
//
// func (l *LoginHandler) QueryRoleListRes(data []byte) {
//	res := &pb.DB2C_QueryRoleListRes{}
//	l.client.Unpack(data, res)
//	l.client.Uid = res.RoleId
//
//	h.client.Debugf("QueryRoleListRes回包: %+v", res)
//	if l.client.Uid == 0 {
//		l.CreateRoleReq()
//	} else {
//		l.ReqSelectRole()
//	}
// }

func (h *LoginHandler) CreateRoleReq() {
	req := &pb.C2DB_CreateRoleReq{AccountId: h.client.Account, RoleName: h.client.Account, RoleSex: 1}
	buf, err := h.client.Pack(pb.Protocols_PC2DB_CreateRoleReq, req)

	if err != nil {
		h.client.Warn("CreateRoleReq Pack error:", err)
	}

	// if h.client.isTcp {
	//	h.client.send <- buf
	// } else {
	//	h.client.HttpSend(h.client.GetGateHttpApi(), buf, true)
	// }
	h.client.SendMsg2Server(h.client.isTcp, h.client.GetGateHttpApi(), buf, true)

	h.client.Debugf("CreateRoleReq: %+v", req)

	h.client.SetState(wait_resp_state)
}

// CreateRoleRes 创建角色列表回包
func (h *LoginHandler) CreateRoleRes(data []byte) {
	res := &pb.DB2C_CreateRoleRes{}
	h.client.Unpack(data, res)
	h.client.Uid = res.RoleId
	h.client.Name = res.RoleName
	h.client.CreateTs = res.CreateTimestamp

	h.client.Debugf("CreateRoleRes回包: %+v", res)
	if h.client.Uid > 0 {
		h.ReqSelectRole()
	}
}

func (h *LoginHandler) ReqSelectRole() {
	req := &pb.C2DB_SelectRoleReq{AccountId: h.client.Account, RoleId: h.client.Uid}
	buf, err := h.client.Pack(pb.Protocols_PC2DB_SelectRoleReq, req)

	if err != nil {
		h.client.Warn("ReqSelectRele Pack error:", err)
	}

	// if h.client.isTcp {
	//	h.client.send <- buf
	// } else {
	//	h.client.HttpSend(h.client.GetGateHttpApi(), buf, true)
	// }
	h.client.SendMsg2Server(h.client.isTcp, h.client.GetGateHttpApi(), buf, true)

	h.client.Debugf("ReqSelectRele: %+v", req)

	h.client.SetState(wait_resp_state)

}

func (h *LoginHandler) EnterGameNtf(data []byte) {
	h.client.Debugf("进入游戏: %s", h.client.Account)
	h.client.SetState(lobby_state)
	h.client.GmHandler.ReqGmCommand("additemall", []string{})
}

func (h *LoginHandler) Heartbeat() {
	req := &pb.C2LS_HeartBeatReq{}
	buf, err := h.client.Pack(pb.Protocols_PC2LS_HeartBeatReq, req)

	if err != nil {
		h.client.Warn("HeartbeatReq Pack error:", err)
	}

	// if h.client.isTcp {
	//	h.client.send <- buf
	// } else {
	//	h.client.HttpSend(h.client.GetGateHttpApi(), buf, true)
	// }
	h.client.SendMsg2Server(h.client.isTcp, h.client.GetGateHttpApi(), buf, true)

	h.client.Debugf("heart beat send...")
}

func (h *LoginHandler) TryLogout() {
	if h.client.GetState() == lobby_state {
		h.client.Debug("玩家下线 ", h.client.Account)
		h.client.Disconnect()
	}
}

func (h *LoginHandler) handleErrorMsg(data []byte) {
	res := &pb.S2C_ErrorCodeNtf{}
	err := h.client.Unpack(data, res)
	if err != nil {
		h.client.Debug("MsgHandler Unpack error: ", err)
		h.client.SetState(lobby_state)
		return
	}
	if res.ErrorCode != 0 {
		h.client.Debugf("收到错误码: errCode=%d", res.ErrorCode)
	}
}
