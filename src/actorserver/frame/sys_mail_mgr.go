package frame

import (
	"fmt"
	"gitlab.musadisca-games.com/wangxw/musae/framework/global"
	"strings"
	"time"

	"github.com/dapr/go-sdk/service/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/datalog/taptap"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"
)

type SysMailMgr struct {
	Data *cmd.PSystemMailInfo
	Srv  *ActorServer
}

func NewSysMailMgr(srv *ActorServer) *SysMailMgr {
	return &SysMailMgr{
		Data: &cmd.PSystemMailInfo{
			SystemMail: make(map[int64]*cmd.PSysMailInfo),
			Max:        0,
		},
		Srv: srv,
	}
}

func (m *SysMailMgr) AddSystemMailReq(in *base.ProtoMsg) (*common.Content, error) {
	req := &cmd.S2S_SendGMAddMailReq{}
	if err := in.UnmarshalData(req); err != nil {
		return nil, err
	}
	logger.Infof("新增系统邮件 req: %+v", req)
	// 新邮件
	addMail := req.AddMail

	// 1.拉取db数据
	sysMails := &cmd.PSystemMailInfo{}
	err := m.Srv.GetSystemMail(sysMails)
	if err != nil {
		return nil, err
	}
	// 重复id容错
	if _, ok := sysMails.SystemMail[addMail.Id]; ok {
		return nil, fmt.Errorf("系统邮件id重复 id: %v", addMail.Id)
	}

	// 2.记录新邮件
	sysMails.SystemMail[addMail.Id] = addMail
	sysMails.Max = addMail.Id

	// 3.保存db
	if err = m.Srv.SaveSystemMail(sysMails); err != nil {
		return nil, err
	}

	// 4.更新当前mgr数据
	m.Data = sysMails

	if _, err = m.Srv.NotifyCenterActor(99, nil, nil); err != nil {
		logger.Errorf("notifyCenterActor err:%+v", err)
	}

	// 新增系统邮件埋点
	taptap.GlobalMailAdd(m.Srv.AppId, global.APP_VERSION, "", global.ROLLING_VERSION, "actorserver", addMail.Id)

	logger.Infof("新增系统邮件成功 sysMails: %+v", sysMails)
	return m.Srv.sendPacket(in, int32(cmd.Protocols_PS2S_SendGMAddMailRes), &cmd.S2S_SendGMAddMailRes{})
}

func (m *SysMailMgr) CheckSystemMailReq(req *cmd.AS2LS_CheckSystemMailReq) (*cmd.AS2LS_CheckSystemMailRes, error) {

	logger.Infof("尝试拉取系统邮件 req: %+v", req)

	sysMails := m.Data
	add := make([]*cmd.PSysMailInfo, 0)
	var max int64
	now := time.Now().Unix()
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
			// 判断是否在指定玩家列表中
			if mail.ReceiveUserIds != nil { // 有指定玩家才做判断
				if _, ok := mail.ReceiveUserIds[req.Uid]; !ok {
					// 不在指定玩家id中
					continue
				}
			}
			// 邮件是否过期
			if mail.ExpireTime > 0 && mail.ExpireTime < now {
				continue
			}
			add = append(add, mail)
		}
		max = sysMails.Max
	}

	return &cmd.AS2LS_CheckSystemMailRes{AddMails: add, MaxValue: max}, nil
}
