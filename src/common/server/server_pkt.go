package server

import (
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/errorx"
	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"
	"gitlab.musadisca-games.com/wangxw/musae/framework/tcpx"
	"google.golang.org/protobuf/proto"
)

func (s *Server) Pack(cmdId cmd.Protocols, errCode cmd.ErrorCode, src interface{}, key string) ([]byte, error) {
	if cmd.ErrorCode_Success != errCode {
		key = "" // errCode不走加密
	}

	return s.pack.Pack(int32(cmdId), int32(errCode), src, key)
}

func (s *Server) PackWithBody(cmdId cmd.Protocols, errCode cmd.ErrorCode, body []byte, key string) ([]byte, error) {
	if cmd.ErrorCode_Success != errCode {
		key = "" // errCode不走加密
	}

	return s.pack.PackWithBody(int32(cmdId), int32(errCode), body, key)
}

func (s *Server) Unpack(allData []byte, dest proto.Message) error {
	body, err := tcpx.BodyBytesOf(allData)
	if err != nil {
		logger.Warn("Unpack BodyBytesOf", errorx.Wrap(err).Error())
	}
	err = proto.Unmarshal(body, dest)
	if err != nil {
		logger.Warn("Unpack Unmarshal", errorx.Wrap(err).Error())
	}

	return nil
}
