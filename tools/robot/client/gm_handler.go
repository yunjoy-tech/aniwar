package client

import "github.com/yunjoy-tech/aniwar/src/proto/pb"

type GmHandler struct {
	client *Client
}

func NewGmHandler(c *Client) *GmHandler {
	h := &GmHandler{client: c}

	return h
}

func (h *GmHandler) ReqGmCommand(command string, param []string) {
	req := &pb.C2LS_UseGameCommandReq{
		TargetId: 0,
		Cmd:      command,
		Param:    param,
	}
	buf, err := h.client.Pack(pb.Protocols_PC2LS_UseGameCommandReq, req)

	if err != nil {
		h.client.Warn("ReqGmCommand Pack error:", err)
	}

	// if h.client.isTcp {
	//	h.client.send <- buf
	// } else {
	//	h.client.HttpSend(h.client.GetGateHttpApi(), buf, true)
	// }
	h.client.SendMsg2Server(h.client.isTcp, h.client.GetGateHttpApi(), buf, true)

	h.client.Debugf("ReqGmCommand: %+v", req)
}

func (h *GmHandler) ReqTestUgcCheck(param string) {
	h.ReqGmCommand("user.test.ugc", []string{param, "1"})
}
