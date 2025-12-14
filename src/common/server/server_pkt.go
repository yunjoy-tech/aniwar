package server

import (
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/framework/errorx"
	"gitee.com/aniwar2/musae/framework/logger"
	"gitee.com/aniwar2/musae/framework/tcpx"
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
		logger.Warn("Unpack BodyBytesOf", errorx.Wrap(err).Error())
	}
	err = proto.Unmarshal(body, dest)
	if err != nil {
		logger.Warn("Unpack Unmarshal", errorx.Wrap(err).Error())
	}

	return nil
}
