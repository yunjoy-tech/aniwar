package frame

import (
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/pb"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"
	"gitlab.musadisca-games.com/wangxw/musae/framework/process"
	"gitlab.musadisca-games.com/wangxw/musae/framework/threading"
	"google.golang.org/protobuf/proto"
	"time"
)

func (s *ActorServer) HandlerSubEvent(msg *base.ProtoMsg) (err error) {
	switch msg.MsgId {
	case int32(pb.Protocols_PS2S_HotReloadReq):
		req := &pb.S2S_HotReloadReq{}
		err = msg.UnmarshalData(req)
		if err != nil {
			logger.Errorf("HandlerSubEvent err:%+v msg:s", err, msg.Str())
			return err
		}
		if req.Type == 99 {
			err = s.HandlerSysMailEvent(req)
		} else {
			err = s.HandlerHotEvent(msg)
		}
	case int32(pb.Protocols_PS2S_SvcRestartReq):
		threading.GoSafe(func() {
			process.Exit()
		})
	default:
	}
	if err != nil {
		logger.Errorf("HandlerSubEvent got err: %v", err)
	}
	return err
}

func (s *ActorServer) HandlerSysMailEvent(req *pb.S2S_HotReloadReq) (err error) {
	now := time.Now().Unix()
	notify := &pb.S2S_HotReloadNotifyReq{}
	err = s.GetSystemMail(s.SysMailMgr.Data)
	if err != nil {
		notify.Service = s.PrivateTopicID()
		notify.Ts = -1
		logger.Errorf("HotReloadNotify fail err:%v req:%+v", err, req)
	} else {
		notify.Service = s.PrivateTopicID()
		notify.Ts = now
		logger.Infof("HotReloadNotify success notify:%+v req:%+v", notify, req)
	}

	reqData, err := proto.Marshal(notify)
	if err == nil {
		s.CenterSrvInvoke(int32(pb.Protocols_PS2S_HotReloadNotifyReq), reqData)
	}
	return err
}
