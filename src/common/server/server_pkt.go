package server

import (
	"github.com/yunjoy-tech/aniwar/src/proto/pb"
	"github.com/yunjoy-tech/musae/logger"
	"github.com/yunjoy-tech/musae/tcpx"
	"google.golang.org/protobuf/proto"
)

func (s *Server) Pack(cmdId pb.Protocols, errCode pb.ErrorCode, src interface{}, key string) ([]byte, error) {
	if pb.ErrorCode_Success != errCode {
		key = "" // errCode不走加密
	}

	return s.pack.Pack(int32(cmdId), int32(errCode), src, key)
}

func (s *Server) PackWithBody(cmdId pb.Protocols, errCode pb.ErrorCode, body []byte, key string) ([]byte, error) {
	if pb.ErrorCode_Success != errCode {
		key = "" // errCode不走加密
	}

	return s.pack.PackWithBody(int32(cmdId), int32(errCode), body, key)
}

func (s *Server) Unpack(allData []byte, dest proto.Message) error {
	body, err := tcpx.BodyBytesOf(allData)
	if err != nil {
		logger.Warn("Unpack BodyBytesOf", err.Error())
	}
	err = proto.Unmarshal(body, dest)
	if err != nil {
		logger.Warn("Unpack Unmarshal", err.Error())
	}

	return nil
}
