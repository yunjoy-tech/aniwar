package client

import (
	"bytes"
	"context"
	"fmt"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/errorx"
	"gitee.com/aniwar2/musae/tcpx"
	"gitee.com/aniwar2/musae/utils"
	"gitee.com/aniwar2/robot/conf"
	"google.golang.org/protobuf/proto"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

const (
	none_state             = "none_state"
	wait_resp_state        = "wait_resp_state"
	connerr_state          = "connerr_state"
	restart_state          = "restart_state"
	offline_state          = "offline_state"
	login_srv_state        = "login_srv_state" // 请求登陆login服
	relogin_state          = "relogin_state"
	version_state          = "version_state"
	connect_game_srv_state = "connect_game_srv_state" // 请求连接game服
	enter_gate_state       = "enter_gate_state"
	enter_game_state       = "enter_game_state"
	rsa_state              = "rsa_state"
	auth_state             = "auth_state"
	gate_state             = "gate_state"
	lobby_state            = "lobby_state"
	room_state             = "room_state"
	mini_game_state        = "mini_game_state"
	scene_state            = "scene_state"
)

type FProtoMsgHandler = func(data []byte)

type Client struct {
	HttpAddr    string // http方式地址
	TcpAddr     string // tcp方式地址
	Account     string // 玩家账号
	NewAccount  string // 生成的账号
	Token       string // 登录token
	SessionId   uint64 // 会话Id
	GatewayIp   string // 网关IP
	GatewayPort uint32 // 网关端口
	Uid         uint64 // 角色Id
	Name        string // 角色名
	CreateTs    int64

	UseRsa int32  // 是否使用加密协议
	rsaVal string // 加密秘钥
	cliKey string // 客户端随机码
	pack   *tcpx.Packx
	conn   net.Conn

	state           atomic.Value               // 状态
	ticker          *time.Ticker               // 随机逻辑执行间隔
	waitRespTimeOut int64                      // 请求超时时间
	MsgFunc         map[int32]FProtoMsgHandler // 回包处理注册
	randomTask      []*RandomTask              // 随机协议配置库
	send            chan []byte                // 消息发送
	recv            chan []byte                // 消息接收
	ctx             context.Context
	stop            context.CancelFunc

	isTcp         bool
	hadTcpConnect bool

	LoginHandler *LoginHandler
	GmHandler    *GmHandler
}

func NewClient(account string) *Client {
	c := &Client{Account: account, NewAccount: account}
	c.pack = tcpx.NewPackx(tcpx.ProtobufMarshaller{})
	c.send = make(chan []byte, 128)
	c.recv = make(chan []byte, 128)

	c.MsgFunc = make(map[int32]FProtoMsgHandler)
	if conf.GetConf().NetType == "tcp" {
		c.isTcp = true
	}

	c.HttpAddr = conf.GetConf().HttpAddr
	c.TcpAddr = conf.GetConf().TCPAddr
	c.initHandler()

	// c.ticker = time.NewTicker(time.Duration(conf.GetConf().Ticker) * time.Second)
	c.Debug("创建客户端成功:", account)
	return c
}

func (c *Client) initHandler() {
	c.LoginHandler = NewLoginHandler(c)
	c.GmHandler = NewGmHandler(c)
}

func (c *Client) String() string {
	return fmt.Sprintf("机器人状态输出 Account: %s,State: %v", c.Account, c.GetState())
}

func (c *Client) GetLoginHttpApi() string {
	return fmt.Sprintf("%s:21001/api", c.HttpAddr)
}

func (c *Client) GetGateHttpApi() string {
	return fmt.Sprintf("%s:22001/api", c.HttpAddr)
}

func (c *Client) SetState(state string) {
	if wait_resp_state == state {
		c.waitRespTimeOut = time.Now().Unix() + 10
	}
	c.state.Store(state)
	c.Debugf("状态切换 Client[%s] -- state[%s]", c.Account, state)
}

func (c *Client) GetState() string {
	state := c.state.Load()
	ret, ok := state.(string)
	if ok {
		return ret
	}
	return "none"
}

func (c *Client) RegisterProtoHandler(messageId int32, handler FProtoMsgHandler) {
	if _, ok := c.MsgFunc[messageId]; !ok {
		c.MsgFunc[messageId] = handler
		c.Debugf("register messageId: %d", messageId)
	} else if ok {
		c.Errorf("Duplicate messageId are registered: %d", messageId)
	}
}

func (c *Client) Start() {
	c.SetState(offline_state)
	c.InitRandomTask()
	utils.GoSafeRun(c.Tick, nil)
	utils.GoSafeRun(c.DoHandler, nil)
}

func (c *Client) Restart() {
	c.Info("Client Restart ...", c.Account)
	c.HttpAddr = conf.GetConf().HttpAddr
	c.TcpAddr = conf.GetConf().TCPAddr
	c.Disconnect()
	time.Sleep(time.Second * 2)
	// c.Start()
	c.SetState(offline_state)
}

// Connect 链接服务
func (c *Client) Connect(isLogin bool) {
	if c.hadTcpConnect {
		c.Debugf("长链接已经建立了 %s, %v", c.Account, isLogin)
		return
	}

	c.Debugf("开始长链接 Connect account_id %s, %v", c.Account, isLogin)
	// c.SrvAddr = srvAddr

	// if c.isTcp {
	var err error
	c.conn, err = net.Dial("tcp", fmt.Sprintf("%s:13001", c.TcpAddr))
	if err != nil {
		c.Warn("Client connect error", c.TcpAddr, err)
		c.SetState(connerr_state)
		return
	}

	// 标识已经建立长链接
	c.hadTcpConnect = true

	c.ctx, c.stop = context.WithCancel(context.Background())
	utils.GoSafeRun(c.DoSend, nil)
	utils.GoSafeRun(c.DoRecv, nil)
	// }

	if conf.GetConf().LoginTime > 0 {
		// t, _ := utils.RandomInt(0, conf.GetConf().LoginTime)
		// time.Sleep(time.Second * time.Duration(t))
	}

	c.SetState(wait_resp_state)
}

func (c *Client) SendMsg2Server(isUseTcp bool, url string, buf []byte, async bool) {
	if isUseTcp {
		c.TcpSend(buf)
	} else {
		c.HttpSend(url, buf, async)
	}
}

func (c *Client) TcpSend(buf []byte) {
	c.send <- buf
}

func (c *Client) HttpSend(url string, buf []byte, async bool) ([]byte, error) {
	if !strings.HasPrefix(url, "http") {
		url = fmt.Sprintf("http://%s", url)
	}

	c.Debugf("HttpSend: %s", url)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(buf))
	if err != nil {
		c.Debugf("NewRequest err: %v", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("auth-token", c.Token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.Debugf("HttpPost error: %s", url)
		return nil, err
	}
	defer resp.Body.Close()
	// 判定是否返回http错误
	if resp.StatusCode != http.StatusOK {
		c.Warnf("HttpPost error status: %v", resp.Status)
		return nil, fmt.Errorf("HttpPost error status: %v", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.Debugf("HttpPost io.readAll error: %s", url)
		return nil, err
	}
	c.Debug("===>>>HTTP收到数据 :", len(body), body)
	if async {
		if err == nil {
			utils.GoSafeRun(func() {
				c.MsgHandler(body)
			}, nil)
		} else {
			c.Debugf("ReqLogin error,req: %+v,err:%v", req, err)
		}
	}
	return body, nil
}

// Disconnect 断开链接
func (c *Client) Disconnect() {
	if c.isTcp {
		c.ctx.Done()

		c.stop()
		if c.conn != nil {
			c.conn.Close()
		}
	}

	// 标识断开长链接
	c.hadTcpConnect = false

	// time.Sleep(1 * time.Second)
	c.Debugf("Client connect closed")
}

// Send 消息包发送
func (c *Client) DoSend() {
	c.Debug("DoSend start")
	for {
		select {
		case <-c.ctx.Done():
			c.Info("DoSend stop")
			return
		case buf := <-c.send:
			if c.conn == nil {
				c.Warnf("DoSend conn is nil state: %v", c.String())
				if c.GetState() == lobby_state {
					// c.SetState(offline_state)
					return
				}
			} else {
				_, err := c.conn.Write(buf)
				if err != nil {
					c.Warnf("DoSend Write error state: %v,error:", c.String(), err)
					if c.GetState() == lobby_state {
						// c.SetState(offline_state)
						return
					}
				}
				c.Warnf("[%v] DoSend msg,len:%v, %v", c.String(), len(buf), buf)
			}
		}
	}
}

// Recv 消息包接受
func (c *Client) DoRecv() {
	c.Debug("DoRecv start")
	for {
		select {
		case <-c.ctx.Done():
			c.Info("DoRecv stop")
			return
		default:
			if c.conn == nil {
				c.Warnf("DoRecv conn is nil: %v", c.String())
				if c.GetState() == lobby_state {
					c.SetState(offline_state)
					return
				}
			} else {
				buf, err := tcpx.FirstBlockOfLimitMaxByte(c.conn, int32(16777216))
				// h.client.Warnf(("FirstBlockOfLimitMaxByte ,err:%v, %d, buf:%v", err, len(buf), buf)
				if err != nil {
					if err == io.EOF {
						continue
					}
					c.Warnf("DoRecv:%v ,err:%v", c.String(), err)
					if c.GetState() == lobby_state {
						c.SetState(offline_state)
					}
					return
				}
				if len(buf) > 0 {
					c.recv <- buf
				}
				// h.client.Warnf(("[%v] DoRecv ,len: %v, %v", c.String(), len(buf), buf)
				time.Sleep(time.Millisecond * 200)
			}

		}
	}
}

func (c *Client) Pack(cmd pb.Protocols, src interface{}) ([]byte, error) {
	return c.pack.Pack(int32(cmd), 0, src, c.rsaVal)
}

func (c *Client) Unpack(allData []byte, dest proto.Message) error {
	body, err := tcpx.BodyBytesOf(allData)
	if err != nil {
		c.Warn("Unpack BodyBytesOf", errorx.Wrap(err, "").Error())
	}
	err = proto.Unmarshal(body, dest)
	if err != nil {
		c.Warn("Unpack Unmarshal", errorx.Wrap(err, "").Error())
	}

	return nil
}

// Handler 尝试执行动作
func (c *Client) DoHandler() {
	c.Debug("DoHandler start")
	for {
		select {
		case buf := <-c.recv:
			c.MsgHandler(buf)
		}
	}
}

func (c *Client) Tick() {
	var _logRate = 0
	for {
		now := time.Now().Unix()
		if _logRate += 1; _logRate%10 == 0 { // 每10输出一次日志
			c.Debug(fmt.Sprintf("--->>>%s", c.String()))
			_logRate = 0
		}

		switch c.GetState() {
		case connerr_state:
			time.Sleep(time.Second * 5)
			c.SetState(restart_state)
		case restart_state:
			c.Restart()
		case offline_state:
			c.Connect(true)
			c.SetState(version_state)
		case version_state:
			err := c.LoginHandler.ReqVersionInfo() // GuideServer 请求版本
			if err == nil {
				c.SetState(relogin_state)
			}
		case connect_game_srv_state:
			c.Connect(false)

			// 登录服返回是否加密标识
			if c.UseRsa == 0 {
				// 不使用加密通信
				c.SetState(enter_gate_state)
			} else {
				// 加密通信
				c.SetState(rsa_state)
			}
		case enter_gate_state:
			c.Debugf("步骤:%s,登录gate", enter_gate_state)
			c.LoginHandler.LoginGameReq()
		case enter_game_state:
			c.LoginHandler.LoginGameReq()

		case rsa_state:
			// c.LoginHandler.RsaReq()
		case relogin_state:
			c.LoginHandler.ReqLogin() // LoginServer 请求登录
		case login_srv_state:
			// c.Debug("login loading...")
			time.Sleep(time.Millisecond * 100)
		case gate_state:
			c.LoginHandler.LoginGameReq()
		case lobby_state:
			random := rand.Int() % 100
			if random <= conf.GetConf().Restart {
				c.Restart()
				// c.SetState(room_state)
			} else {
				c.sendRandomReq()
			}
		case room_state:
			// c.RoomHandler.start()
		case mini_game_state:
			// do nothing...
			c.Debugf("mini_game_state 当前在小游戏中....")
		case wait_resp_state:
			time.Sleep(time.Millisecond * 200)
			if c.waitRespTimeOut > 0 && now > c.waitRespTimeOut {
				c.SetState(restart_state)
			}
		}
		// time.Sleep(time.Millisecond * 1000)
		time.Sleep(time.Millisecond * 200)
	}
}

// MsgHandler 数据回包处理逻辑
func (c *Client) MsgHandler(bytes []byte) {
	c.Debug("MsgHandler处理数据 :", len(bytes))
	allData, err := tcpx.Decrypt(bytes, c.rsaVal)
	if err != nil {
		c.Warn("Unpack Decrypt", errorx.Wrap(err, "").Error())
		return
	}

	messageId, err := tcpx.MessageIDOf(allData)
	c.Debugf("MsgHandler: msg=%s msgId=%d", pb.Protocols(messageId), messageId)
	if err != nil {
		c.Error("MsgHandler parse messageId failed:", err)
		return
	}

	// c.Debug("收到数据包 :", cmd.Protocols(messageId), messageId, len(bytes))

	if handler, ok := c.MsgFunc[messageId]; ok {
		handler(allData)
		return
	}
	c.Warn("MsgHandler unknown msg:", messageId, len(allData))
}
