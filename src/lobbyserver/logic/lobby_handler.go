package logic

import (
	"fmt"
	"github.com/dapr/go-sdk/service/common"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/db"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/pb"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"
	"strings"
)

func (s *LobbyServer) AddSystemMail(msg *base.ProtoMsg) (*common.Content, error) {
	messageID, uid, roleId, uaid := msg.MsgId, msg.UserId, msg.RoleId, msg.UAID
	req := &pb.S2S_SendGMAddMailReq{}
	if err := msg.UnmarshalData(req); err != nil {
		logger.Debug("proto.Unmarshal error: msgId:", pb.Protocols(messageID), messageID)
		return nil, err
	}
	if err := s.addSystemMail(req.AddMail); err != nil {
		return nil, err
	}

	return s.sendPacket(uid, roleId, uaid, 0, req)
}

func (s *LobbyServer) CheckSystemMailReq(msg *base.ProtoMsg) (*common.Content, error) {

	messageID, uid, roleId, uaid := msg.MsgId, msg.UserId, msg.RoleId, msg.UAID

	var req pb.AS2LS_CheckSystemMailReq
	if err := msg.UnmarshalData(&req); err != nil {
		logger.Debug("proto.Unmarshal error: msgId:", pb.Protocols(messageID), messageID)
		return nil, err
	}

	sysMails := &pb.PSystemMailInfo{}
	err := s.LoadDB(db.KeySystemMail(), sysMails)
	if err != nil {
		return nil, fmt.Errorf("mail get redis err. %v", err)
	}

	// check
	add := make([]*pb.PSysMailInfo, 0)
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

	res := &pb.AS2LS_CheckSystemMailRes{AddMails: add, MaxValue: max}
	return s.sendPacket(uid, roleId, uaid, int32(pb.Protocols_PAS2LS_CheckSystemMailRes), res)
}

// 新增系统邮件
func (s *LobbyServer) addSystemMail(mail *pb.PSysMailInfo) error {
	sysMails := &pb.PSystemMailInfo{}
	err := s.LoadDB(db.KeySystemMail(), sysMails)
	if err != nil {
		return fmt.Errorf("mail get redis err")
	}

	// 容错处理
	if sysMails.SystemMail == nil {
		sysMails.SystemMail = make(map[int64]*pb.PSysMailInfo)
	}

	// 初始化邮件
	sysMails.SystemMail[mail.Id] = mail
	sysMails.Max = mail.Id

	err = s.SaveDB(db.KeySystemMail(), sysMails)
	if err != nil {
		return err
	}

	logger.Debug("新增系统邮件 %+v", mail)
	return nil
}
