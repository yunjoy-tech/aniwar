package common

import (
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/pb"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"
	"google.golang.org/protobuf/proto"
)

// ParserReq 解析请求数据接口
func ParserReq(in *base.ProtoMsg, req proto.Message) (int32, string, error, pb.ErrorCode) {
	var (
		err     error
		msgId   int32
		uid     string
		reqData []byte
	)

	msgId, uid, reqData = in.MsgId, in.UserId, in.Data
	// err = proto.Unmarshal(reqData, req)
	err = base.UnmarshalData(reqData, req)
	if err != nil {
		logger.Errorf("解析请求参数出错: in:%v, err:%+v", in, err)
		return msgId, uid, err, pb.ErrorCode_DeSerializeError
	}
	// logger.Debugf("解析请求参数: in:%v, req:%T{%v}", in, req, req)

	return msgId, uid, nil, pb.ErrorCode_Success
}
