package mailactor

import (
	"context"
	"fmt"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/db"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/server"
	"strings"

	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"
	"google.golang.org/protobuf/proto"
)

type MailHandler struct {
	*UMBaseHandler
}

func NewMailHandler(actor *MailActor) *MailHandler {
	h := &MailHandler{UMBaseHandler: NewUMBaseHandler(actor, "MailHandler")}
	h.ChildHandler = h

	actor.RegisterProtoHandler(int32(cmd.Protocols_PAS2LS_CheckSystemMailReq), h.CheckSystemMailReq) // 拉取全局邮件
	actor.RegisterProtoHandler(int32(cmd.Protocols_PS2S_SendGMAddMailReq), h.AddSystemMailReq)       // 新增全局邮件

	return h
}

// Init 初始化模块数据
func (h *MailHandler) Init() error {
	h.actor.Data = &cmd.PSystemMailInfo{
		SystemMail: make(map[int64]*cmd.PSysMailInfo),
		Max:        0,
	}
	h.Debug("初始化全局邮件数据成功")
	return nil
}

func (h *MailHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*cmd.PSystemMailInfo); ok {
		h.actor.Data = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}
	h.Debugf("加载全局邮件数据: %+v", dbData)
	return nil
}

func (h *MailHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeySystemMail(), h.actor.MailData.Data
}

func (h *MailHandler) EnterGame() error {
	// implement me
	return nil
}

func (h *MailHandler) DailyRefresh() error {
	// implement me
	return nil
}

func (h *MailHandler) AddSystemMailReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {

	req := &cmd.S2S_SendGMAddMailReq{}
	if err := in.UnmarshalData(req); err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	sysMails := h.actor.MailData.Data
	if sysMails.SystemMail == nil {
		sysMails.SystemMail = make(map[int64]*cmd.PSysMailInfo)
	}
	mail := req.AddMail

	// 初始化邮件
	sysMails.SystemMail[mail.Id] = mail
	sysMails.Max = mail.Id

	key := db.KeySystemMail()
	kvTable, err := db.BuildKvTable(sysMails, key)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InternalError)
	}
	err = h.actor.Srv.SaveMongoAndRedis(service.MongoDbType_MongoGame, key, kvTable, nil, server.ICache(h.actor.Srv))
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_SaveDBError)
	}

	// 通过配置中心通知其他mailactor更新内存数据
	//err = h.actor.Srv.SaveToConfigCenter(db.KeyCfgGlobalSysMail, "sysmail")
	//if err != nil {
	//	return nil, err, int32(cmd.ErrorCode_InternalError)
	//}

	h.Infof("新增系统邮件 %+v", mail)
	return &cmd.S2S_SendGMAddMailRes{}, nil, 0
}

func (h *MailHandler) CheckSystemMailReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {

	var req cmd.AS2LS_CheckSystemMailReq
	if err := in.UnmarshalData(&req); err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	sysMails := h.actor.MailData.Data

	add := make([]*cmd.PSysMailInfo, 0)
	var max int64
	if sysMails.SystemMail != nil {
		if req.CurValue < sysMails.Max {
			for _, mail := range sysMails.SystemMail {
				if mail.Id <= req.CurValue {
					continue
				}
				// 条件判定
				if mail.Plat != "" && mail.Plat != req.Plat {
					continue
				}
				if mail.Level > req.Level {
					continue
				}
				if mail.MinVip > 0 && mail.MinVip > req.VipLevel {
					continue
				}
				if mail.MaxVip > 0 && req.VipLevel > mail.MaxVip {
					continue
				}
				if mail.Os != "" {
					osArr := strings.Split(mail.Os, ",")
					f := false
					for _, v := range osArr {
						if req.Os == v {
							f = true
							break
						}
					}
					if !f {
						continue
					}
				}
				add = append(add, mail)
			}
			max = sysMails.Max
		}
	}

	return &cmd.AS2LS_CheckSystemMailRes{AddMails: add, MaxValue: max}, nil, 0
}
