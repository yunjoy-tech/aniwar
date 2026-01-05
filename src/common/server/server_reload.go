package server

import (
	"context"
	"encoding/json"
	"fmt"
	comn "gitee.com/aniwar2/aniwar/src/common"
	"gitee.com/aniwar2/aniwar/src/common/actor/stub"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/base"
	"gitee.com/aniwar2/musae/gamelib/guid"
	"gitee.com/aniwar2/musae/global"
	"gitee.com/aniwar2/musae/logger"
	"github.com/dapr/go-sdk/service/common"
	"google.golang.org/protobuf/proto"
	"strings"
	"time"
)

func (s *Server) HotReload(ctx context.Context, in *common.InvocationEvent) (out *common.Content, err error) {
	defer func() {
		if err := recover(); err != any(nil) {
			logger.Error("hotreload failed, err: ", err)
		}
	}()

	if in == nil {
		err = fmt.Errorf("nil invocation parameter")
		logger.Warn("hotreload nil invocation parameter")
		return out, err
	}

	out = &common.Content{
		ContentType: in.ContentType,
		DataTypeURL: in.DataTypeURL,
	}

	logger.Debugf("hotreload - ContentType:%s, Verb:%s, QueryString:%s, Data:%v", in.ContentType, in.Verb, in.QueryString, in.Data)

	param := &comn.ReloadParam{}
	err = json.Unmarshal(in.Data, param)
	if err != nil {
		logger.Warn("hotreload ReloadParam error")
		return out, err
	}

	switch param.Type {
	case "conf":
		err = s.LoadConf()
		if err != nil {
			out.Data = []byte(err.Error())
		} else {
			out.Data = []byte("SUCCESS")
		}
	case "excel":
		if strings.Compare(param.Files, "all") == 0 {
			if s.AppId == "actor" {
				err = s.LoadExcel()
			} else {
				err = s.LoadNeedExcel(nil) // 非actorserver都调用这个加载方法
			}
		} else {
			files := strings.Split(param.Files, "|")
			if s.AppId == "actor" {
				// TODO 后面完善
				// err = data.LoadByFileNames(s.MetaDir, files, s.AppId, "actorserver")
			} else {
				err = s.LoadNeedExcel(files) // 非actorserver都调用这个加载方法
			}
		}
		if err != nil {
			out.Data = []byte(err.Error())
		} else {
			out.Data = []byte("SUCCESS")
		}
	case "dirtyword": // 静态屏蔽词更新
		err = s.LoadWordCfg()
		if err != nil {
			out.Data = []byte(err.Error())
		} else {
			out.Data = []byte("SUCCESS")
		}
	default:
		out.Data = []byte("invalid param")
	}
	logger.Infof("hotreload param:[%+v], out: %s", param, string(out.Data))
	return out, nil
}

func (s *Server) HandlerHotEvent(in *base.ProtoMsg) (err error) {
	req := &pb.S2S_HotReloadReq{}
	now := time.Now().Unix()
	in.UnmarshalData(req)
	notify := &pb.S2S_HotReloadNotifyReq{}
	logger.Infof("HandlerHotEvent Begin files:%+v", req.Files)
	if len(req.Files) > 0 {
		tag := req.Files[0]
		if tag == "all" {
			err = s.LoadExcelData()
		} else if tag == "server.conf" {
			err = s.ReloadConf()
		} else {
			err = s.LoadExcelDataByFiles(req.Files)
		}
		if err != nil {
			notify.Service = s.PrivateTopicID()
			notify.Ts = -1
			logger.Errorf("HandlerHotEvent fail err:%v files:%+v", err, req.Files)
		} else {
			notify.Service = s.PrivateTopicID()
			notify.Ts = now
			logger.Infof("HandlerHotEvent success notify:%+v files:%+v", notify, req.Files)
		}
	}

	reqData, err := proto.Marshal(notify)
	if err == nil {
		_, _ = s.ActorInvoke(stub.CenterActorType, global.CenterActorID, &base.ProtoMsg{
			AppId:   global.ACTOR_SVC,
			MsgId:   int32(pb.Protocols_PS2S_HotReloadNotifyReq),
			UserId:  "",
			RoleId:  0,
			UAID:    global.CenterActorID,
			Data:    reqData,
			ErrCode: 0,
			ReqIdx:  guid.GenIntUuid(),
			Topic:   "",
			Uids:    nil,
		})
	}
	return err
}
