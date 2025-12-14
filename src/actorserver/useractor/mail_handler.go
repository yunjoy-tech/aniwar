package useractor

import (
	"context"
	"fmt"
	"gitee.com/aniwar2/aniwar/src/meta"
	"math"
	"sort"
	"time"

	"gitee.com/aniwar2/aniwar/src/common/conf"
	"gitee.com/aniwar2/aniwar/src/common/datalog/taptap"
	"gitee.com/aniwar2/musae/framework/baseconf"
	"gitee.com/aniwar2/musae/framework/global"
	"gitee.com/aniwar2/musae/framework/threading"

	"gitee.com/aniwar2/aniwar/src/common/clidto"
	"gitee.com/aniwar2/aniwar/src/common/db"
	"gitee.com/aniwar2/musae/framework/service"

	"gitee.com/aniwar2/aniwar/src/common"
	myUtils "gitee.com/aniwar2/aniwar/src/common/utils"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/framework/base"
	"gitee.com/aniwar2/musae/framework/guid"
	"google.golang.org/protobuf/proto"
)

type MailHandler struct {
	*UABaseHandler
}

func NewMailHandler(actor *UserActor) *MailHandler {
	h := &MailHandler{UABaseHandler: NewUABaseHandler(actor, "MailHandler")}
	h.ChildHandler = h

	actor.RegisterProtoHandler(int32(pb.Protocols_PC2MS_MailListInfosReq), h.MailListInfosReq)
	actor.RegisterProtoHandler(int32(pb.Protocols_PC2MS_MailReceiveReq), h.MailReceiveReq) // 领取邮件
	actor.RegisterProtoHandler(int32(pb.Protocols_PC2MS_MailReadReq), h.MailReadReq)
	actor.RegisterProtoHandler(int32(pb.Protocols_PC2MS_MailDeleteReq), h.MailDeleteReq)

	return h
}

func (h *MailHandler) Init() error {
	h.actor.Data.UserMail = &pb.PUserMailInfo{
		Createtime: time.Now().Unix(),
		UserMail:   make(map[int64]*pb.PMailInfo),
		CheckValue: 0,
	}

	// 保存
	if err := h.SaveDB(true); err != nil {
		return err
	}

	h.Debug("init user mail data success. player: %s", h.actor.ID())
	return nil
}

func (h *MailHandler) EnterGame() error {
	return nil
}

func (h *MailHandler) DailyRefresh() error {
	return nil
}

func (h *MailHandler) SetDBData(dbData proto.Message) error {
	if dbVal, ok := dbData.(*pb.PUserMailInfo); ok {
		h.actor.Data.UserMail = dbVal
	} else {
		return fmt.Errorf("SetDBData, 数据类型错误! %v", dbData)
	}

	return nil
}

func (h *MailHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoGame, db.KeyUserMail(h.actor.ID()), h.actor.Data.UserMail
}

// 获取新邮件数量
func (h *MailHandler) getNewMailsCount() int32 {
	var counter int32
	// check一下全局邮件
	userMails, _, err := h.checkSystemMail()
	if err != nil {
		h.Error(err)
		return counter
	}
	for _, mail := range userMails.UserMail {
		// 未读邮件
		if mail.IsRead == common.MAIL_STATUS_UNREAD {
			counter++
			continue
		}
		// 奖励未领取
		if mail.IsReceived == common.MAIL_STATUS_UNRECEIVE {
			counter++
		}
	}
	return counter
}

func (h *MailHandler) MailListInfosReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req pb.C2MS_MailListInfosReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	// check一下全局邮件
	userMails, delMails, err := h.checkSystemMail()
	if err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	// 返回消息
	mails := make([]*pb.PMailInfo, 0)
	for _, v := range userMails.UserMail {
		mails = append(mails, v)
	}
	res := &pb.MS2C_MailListInfosRes{Mails: mails, DelMails: delMails}
	return res, nil, 0
}

func (h *MailHandler) MailReceiveReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {

	var req pb.C2MS_MailReceiveReq
	if err := in.UnmarshalData(&req); err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	// check
	if req.ReceiveType != common.MAIL_RECEIVE_ALL && req.ReceiveType != common.MAIL_RECEIVE_ONE {
		return nil, fmt.Errorf("param check failed %d", req.ReceiveType), int32(pb.ErrorCode_ParamError)
	}

	usermail := h.actor.GetUserMailData()

	now := time.Now().Unix()
	target := make([]*pb.PMailInfo, 0)
	if req.ReceiveType == common.MAIL_RECEIVE_ALL {
		for _, v := range usermail.UserMail {
			// 取可以领取的邮件
			if v.IsReceived == common.MAIL_STATUS_RECEIVED || v.IsReceived == common.MAIL_STATUS_NOT_RECEIVE {
				continue
			}
			if v.ExpireTime > 0 && now >= v.ExpireTime {
				continue
			}
			target = append(target, v)
		}
	} else {
		t := usermail.UserMail[req.MailIds]
		if t != nil && t.IsReceived == common.MAIL_STATUS_UNRECEIVE && (t.ExpireTime == 0 || now < t.ExpireTime) {
			target = append(target, t)
		}
	}

	h.Debugf("MailReceiveReq  target: %d", len(target))
	if len(target) > 0 {
		err, code, rewards, mailIds := h.receiveMailAttachment(target, h.actor.comData)
		if code != pb.ErrorCode_Success {
			return nil, err, int32(code)
		}
		delIds := make([]int64, 0)
		for _, id := range mailIds {
			t := usermail.UserMail[id]
			t.IsRead = common.MAIL_STATUS_READ
			t.IsReceived = common.MAIL_STATUS_RECEIVED
			// 道具溢出类型，领取后自动删除邮件
			if t.MailType == common.MAIL_TYPE_3 {
				delIds = append(delIds, id)
				delete(usermail.UserMail, id)
			}
		}

		if err = h.SaveDB(); err != nil {
			return nil, err, int32(pb.ErrorCode_InternalError)
		}

		// 埋点log
		tempIds := make([]int64, 0)
		for _, v := range target {
			tempIds = append(tempIds, v.Id)
		}

		// threading.RunSafe(func() {
		//	lilith.WriteDataLog(&lilith.MailReceive{
		//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_MailReceive, h.actor.uid, h.actor.Account.CliDeviceInfo),
		//		MailIds:        lilith.ConvertList2Str(tempIds),
		//		Reward:         lilith.ConvertListStruct2Str(rewards),
		//	})
		// })
		threading.RunSafe(func() {
			e := &taptap.MailReceive{
				PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
				MailIds:           taptap.ConvertList2Str(tempIds),
				Reward:            taptap.ConvertListStruct2Str(rewards),
			}
			taptap.WriteDataLog(taptap.LogType_MailReceive, h.actor.uid, h.actor.Account.TapUserInfo, e)
		})

		// 返回消息
		res := &pb.MS2C_MailReceiveRes{
			MailIds:    mailIds,
			Rewards:    rewards,
			CommonData: h.actor.comData.FixDownComData(),
			Result:     len(mailIds) == len(target),
			DelIds:     delIds,
		}
		return res, nil, 0
	}

	return &pb.MS2C_MailReceiveRes{}, nil, 0
}

func (h *MailHandler) MailReadReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req pb.C2MS_MailReadReq
	if err := in.UnmarshalData(&req); err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	// check
	if req.ReadType != common.MAIL_READ_ALL && req.ReadType != common.MAIL_READ_ONE {
		return nil, fmt.Errorf("param check failed %d", req.ReadType), int32(pb.ErrorCode_ParamError)
	}

	mail := h.actor.GetUserMailData()

	mails := make([]int64, 0)
	change := false
	if req.ReadType == common.MAIL_READ_ALL {
		for _, info := range mail.UserMail {
			if info.IsRead == common.MAIL_STATUS_UNREAD {
				info.IsRead = common.MAIL_STATUS_READ
				// 特殊处理
				if info.IsReceived == common.MAIL_STATUS_NOT_RECEIVE {
					info.IsReceived = common.MAIL_STATUS_RECEIVED
				}
				change = true
				mails = append(mails, info.Id)
			}
		}
	} else {
		target := mail.UserMail[req.MailIds]
		if target != nil && target.IsRead == common.MAIL_STATUS_UNREAD {
			target.IsRead = common.MAIL_STATUS_READ
			if target.IsReceived == common.MAIL_STATUS_NOT_RECEIVE {
				target.IsReceived = common.MAIL_STATUS_RECEIVED
			}
			change = true
			mails = append(mails, req.MailIds)
		}
	}

	if change {
		if err := h.SaveDB(); err != nil {
			return nil, err, int32(pb.ErrorCode_InternalError)
		}
	}

	// 返回消息
	res := &pb.MS2C_MailReadRes{MailIds: mails}
	return res, nil, 0
}

func (h *MailHandler) MailDeleteReq(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {

	var req pb.C2MS_MailDeleteReq
	if err := in.UnmarshalData(&req); err != nil {
		return nil, err, int32(pb.ErrorCode_DeSerializeError)
	}

	mailData := h.actor.GetUserMailData()

	mailIds := make([]int64, 0)
	if req.DelType == common.MAIL_DELETE_ALL {
		for _, info := range mailData.UserMail {
			if info.IsRead == common.MAIL_STATUS_UNREAD {
				continue
			}
			if info.IsRead == common.MAIL_STATUS_READ && info.IsReceived == common.MAIL_STATUS_UNRECEIVE {
				continue
			}

			delete(mailData.UserMail, info.Id)
			mailIds = append(mailIds, info.Id)
		}
	} else {
		t := mailData.UserMail[req.MailIds]
		if t != nil && t.IsReceived == common.MAIL_STATUS_RECEIVED {
			delete(mailData.UserMail, req.MailIds)
			mailIds = append(mailIds, t.Id)
		}
	}

	if err := h.SaveDB(); err != nil {
		return nil, err, int32(pb.ErrorCode_InternalError)
	}

	// 埋点log
	// threading.RunSafe(func() {
	//	lilith.WriteDataLog(&lilith.MailDelete{
	//		CustomHeadInfo: lilith.BuildCustomHeadInfo(lilith.LogType_MailDelete, h.actor.uid, h.actor.Account.CliDeviceInfo),
	//		MailIds:        lilith.ConvertList2Str(mailIds),
	//	})
	// })
	threading.RunSafe(func() {
		e := &taptap.MailDelete{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			MailIds:           taptap.ConvertList2Str(mailIds),
		}
		taptap.WriteDataLog(taptap.LogType_MailDelete, h.actor.uid, h.actor.Account.TapUserInfo, e)
	})

	res := &pb.MS2C_MailDeleteRes{DelMails: mailIds}
	return res, nil, 0
}

// receiveMailAttachment
//
//	@Description: 领取邮件的附件
//	@receiver h
//	@param mails 待领取的列表
//	@param commonData
//	@return pb.ErrorCode 返回错误码
//	@return []*pb.ItemReward 领取到的奖励列表
//	@return []int64 领取完成的邮件id列表
func (h *MailHandler) receiveMailAttachment(mails []*pb.PMailInfo, commonData *clidto.Comdata) (error, pb.ErrorCode, []*pb.ItemReward, []int64) {
	// 数量上限检查
	total := make([]*pb.ItemReward, 0)
	received := make([]int64, 0)
	limitMap := make(map[int32]int32)      // 道具计数
	campItems := make([]*pb.ItemReward, 0) // 营地道具特殊处理
	for _, v := range mails {
		items := myUtils.ConvertItem2(v.Attachments)
		// 构造新道具数量
		newItems := make(map[int32]int32)
		for id, num := range items {
			// 排除家具类型道具
			var cfg *meta.ItemPkgItemMeta
			// cfg := excel.GetItemMgr().GetById(id)
			if cfg == nil || cfg.Type == int32(pb.ItemType_PlayerCamp) {
				continue
			}
			newItems[id] = limitMap[id] + num
		}
		b := GetDropMgr(h.actor).CheckMapLimit(newItems)
		if len(mails) == 1 {
			// 单邮件处理
			if b {
				return fmt.Errorf("package is full"), pb.ErrorCode_PackageIsFull, nil, nil
			}
		} else {
			// 多邮件处理
			if b {
				continue
			}
		}
		// 剔除过期
		target, campItem := fixExpiredItem(v)
		total = append(total, target...)
		campItems = append(campItems, campItem...)

		received = append(received, v.Id)

		// 累计道具计数
		for id, num := range items {
			limitMap[id] += num
		}
	}

	// 家具下发
	var retItems []*pb.ItemReward
	// retItems, err := h.actor.CampHandler.HandleDropMailItem(campItems)
	// if err != nil {
	// 	return fmt.Errorf("internal error"), pb.ErrorCode_InternalError, nil, nil
	// }

	// 附件下发
	rewards, err := GetDropMgr(h.actor).DropList2(myUtils.ConvertItem2(total), true, nil, commonData, common.CR_Mail_Attachment)
	if err != nil {
		return fmt.Errorf("internal error"), pb.ErrorCode_InternalError, nil, nil
	}
	retItems = append(retItems, rewards.Items...)
	return nil, pb.ErrorCode_Success, retItems, received
}

// 排除过期道具,返回家具和非家具
func fixExpiredItem(mails *pb.PMailInfo) ([]*pb.ItemReward, []*pb.ItemReward) {
	items := make([]*pb.ItemReward, 0)
	campItems := make([]*pb.ItemReward, 0)
	// now := time.Now().Unix()
	for _, attachment := range mails.Attachments {
		var cfg *meta.ItemPkgItemMeta
		// cfg := excel.GetItemMgr().GetById(int32(attachment.ItemId))
		if cfg == nil {
			continue
		}

		// if cfg.GetTimeLimit() > 0 && mails.CreateTime+int64(cfg.GetTimeLimit()) < now {
		// 	continue
		// }

		if cfg.Type == int32(pb.ItemType_PlayerCamp) {
			campItems = append(campItems, attachment)
		} else {
			items = append(items, attachment)
		}
	}

	return items, campItems
}

func (h *MailHandler) GMTAddUserMail(req *pb.S2S_SendGMAddUserMailReq, commonData *clidto.Comdata) error {
	mails := h.actor.GetUserMailData()

	newMail := req.AddMail
	// 处理邮件状态
	if len(newMail.Attachments) == 0 {
		newMail.IsReceived = common.MAIL_STATUS_NOT_RECEIVE
	}

	lang := h.actor.Data.Base.Common.Language
	// 多语言处理
	if newMail.Title == "" {
		if language, ok := req.LangMap[lang]; ok {
			newMail.Title = language.Content["title"]
			newMail.Content = language.Content["context"]
			newMail.Sender = language.Content["sender_name"]
		} else {
			return fmt.Errorf("lang not found %s", lang)
		}
	}

	// 问卷处理
	if req.QuestionId != "" {
		newMail.QuestionUrl = h.genQuestionUrl(h.actor.uid, h.actor.Data.Base.Common.RoleId, req.QuestionType, req.QuestionId, lang)
		// 问卷数据处理
		// err := h.actor.QuestionHandler.AddQuestion(req.QuestionId, newMail.Attachments)
		// if err != nil {
		// 	return err
		// }
		// 该邮件奖励废弃
		newMail.Attachments = nil
		newMail.IsReceived = common.MAIL_STATUS_NOT_RECEIVE
	}

	// 新增
	mails.UserMail[newMail.Id] = newMail
	commonData.Data.Mails = append(commonData.Data.Mails, newMail)
	// 上限判定，删除最旧的邮件
	// if int32(len(mails.UserMail)) > excel.GetConfigMgr().GetCfg().MAIL_NUM_LIMIT {
	// 	h.tryDelMail(mails.UserMail)
	// }

	if err := h.SaveDB(); err != nil {
		return err
	}

	return nil
}

// AddUserMail 发送模板邮件
func (h *MailHandler) AddUserMail(mailId int32, reward map[int32]int32, commonData *clidto.Comdata) error {
	// 邮件配置
	var mailCfg *meta.MailPkgMailMeta
	// mailCfg := excel.GetMailMgr().GetById(mailId)
	if mailCfg == nil {
		return fmt.Errorf("mail config not found: %d", mailId)
	}

	// 附件奖励处理,超出上限的道具发送多封邮件
	attachment := make(map[int32]int32)
	// 透传
	if reward != nil {
		attachment = reward
	}
	// 模板配置
	for _, r := range mailCfg.GiftsContents {
		attachment[r.ItemId] += r.Num
	}
	// 计算奖励发送次数
	max := 0
	for k, v := range attachment {
		// 道具上限
		var cfg *meta.ItemPkgItemMeta
		// cfg := excel.GetItemMgr().GetById(k)
		if cfg == nil {
			return fmt.Errorf("item config not found %d", k)
		}

		need := int(math.Ceil(float64(v) / float64(cfg.NumLimit)))

		if need > max {
			max = need
		}
	}
	// 无附件邮件容错
	if max == 0 {
		max = 1
	}
	mails := h.actor.GetUserMailData()

	// 发送邮件
	for i := 0; i < max; i++ {
		// 附件处理
		r := make([]*pb.ItemReward, 0)
		for k, v := range attachment {
			// 道具上限
			var cfg *meta.ItemPkgItemMeta
			// cfg := excel.GetItemMgr().GetById(k)

			num := uint32(0)
			if int64(v) > cfg.NumLimit {
				attachment[k] -= int32(cfg.NumLimit)
				num = uint32(cfg.NumLimit)
			} else {
				delete(attachment, k)
				num = uint32(v)
			}

			if num > 0 {
				r = append(r, &pb.ItemReward{
					ItemId: uint32(k),
					Num:    num,
				})
			}
		}

		mail := h.createMail(mailCfg, r)
		mails.UserMail[mail.Id] = mail
		commonData.Data.Mails = append(commonData.Data.Mails, mail)
	}

	// 上限判定，删除最旧的邮件
	// if int32(len(mails.UserMail)) > excel.GetConfigMgr().GetCfg().MAIL_NUM_LIMIT {
	// 	h.tryDelMail(mails.UserMail)
	// }

	if err := h.SaveDB(); err != nil {
		return err
	}
	return nil
}

// 根据邮件模板创建一封邮件
func (h *MailHandler) createMail(mailCfg *meta.MailPkgMailMeta, attachment []*pb.ItemReward) *pb.PMailInfo {
	now := time.Now()
	expireTime := int64(0)
	if mailCfg.ExpireTime > 0 {
		expireTime = now.Add(time.Hour * time.Duration(24*mailCfg.ExpireTime)).Unix()
	}

	isReceived := common.MAIL_STATUS_UNRECEIVE
	if len(attachment) == 0 {
		isReceived = common.MAIL_STATUS_NOT_RECEIVE
	}
	mail := &pb.PMailInfo{
		Id:          int64(h.actor.Srv.GenGUID(guid.GUID_MAIL)),
		Title:       mailCfg.Title,
		Content:     mailCfg.Content,
		Sender:      mailCfg.Addresser,
		SendType:    common.MAIL_SEND_TYPE_SYSTEM,
		MailType:    mailCfg.MailType,
		CreateTime:  now.Unix(),
		ExpireTime:  expireTime,
		IsRead:      common.MAIL_STATUS_UNREAD,
		IsReceived:  int32(isReceived),
		Attachments: attachment,
		GiftsType:   mailCfg.GiftsType,
		QuestionUrl: "",
	}

	return mail
}

// 将系统邮件转换成个人邮件
func (h *MailHandler) convertMail(uid string, roleId uint64, lang string, mail *pb.PSysMailInfo) (*pb.PMailInfo, error) {
	isReceived := common.MAIL_STATUS_UNRECEIVE
	if len(mail.Attachments) == 0 {
		isReceived = common.MAIL_STATUS_NOT_RECEIVE
	}

	// 多语言处理
	if mail.Title == "" {
		if language, ok := mail.LangMap[lang]; ok {
			mail.Title = language.Content["title"]
			mail.Content = language.Content["context"]
			mail.Sender = language.Content["sender_name"]
		} else {
			return nil, fmt.Errorf("system mail lang not found")
		}
	} else {
		if mail.Lang != lang {
			return nil, fmt.Errorf("system mail lang not match")
		}
	}

	// 问卷处理
	var questionUrl string
	if mail.QuestionId != "" {
		questionUrl = h.genQuestionUrl(uid, roleId, mail.QuestionType, mail.QuestionId, lang)
		// 问卷数据处理
		// err := h.actor.QuestionHandler.AddQuestion(mail.QuestionId, mail.Attachments)
		// if err != nil {
		// 	return nil, err
		// }
		// 该邮件奖励废弃
		mail.Attachments = nil
		isReceived = common.MAIL_STATUS_NOT_RECEIVE
	}

	return &pb.PMailInfo{
		Id:          mail.Id,
		Title:       mail.Title,
		Content:     mail.Content,
		Sender:      mail.Sender,
		SendType:    mail.SendType,
		MailType:    mail.MailType,
		CreateTime:  mail.CreateTime,
		ExpireTime:  mail.ExpireTime,
		IsRead:      common.MAIL_STATUS_UNREAD,
		IsReceived:  int32(isReceived),
		Attachments: mail.Attachments,
		GiftsType:   common.MAIL_REWARD_TYPE_OTHER,
		QuestionUrl: questionUrl,
	}, nil
}

// 尝试check系统邮件
func (h *MailHandler) checkSystemMail() (*pb.PUserMailInfo, []int64, error) {

	// 取个人邮件
	userMails := h.actor.GetUserMailData()

	// 从ActorServer中获取全局邮件
	reqMsg := &pb.AS2LS_CheckSystemMailReq{
		CurValue: userMails.CheckValue,
		Level:    int32(h.actor.LoginHandler.getRoleLevel()),
		VipLevel: 0, // todo 暂时填0
		Plat:     h.actor.LoginHandler.GetRoleChannel(),
		Os:       h.actor.LoginHandler.GetRoleOS(),
		Uid:      h.actor.uid,
	}

	resMsg, err := h.actor.Srv.SysMailMgr.CheckSystemMailReq(reqMsg)
	if err != nil {
		return nil, nil, err
	}

	h.Infof("拉取到新的系统邮件 res: %+v", resMsg)

	// 填充到个人邮件中
	addIds := make([]int64, 0)
	if resMsg.MaxValue > 0 {
		userMails.CheckValue = resMsg.MaxValue
	}
	for _, m := range resMsg.AddMails {
		addIds = append(addIds, m.Id)
		mail, err := h.convertMail(h.actor.uid, h.actor.Data.Base.Common.RoleId, h.actor.Data.Base.Common.Language, m)
		if err != nil {
			h.Warnf("转换系统邮件失败 mailId: %v, err: %v", m.Id, err)
			continue
		}
		userMails.UserMail[m.Id] = mail
	}

	// 玩家领取到系统邮件的埋点
	threading.RunSafe(func() {
		e := &taptap.GlobalToPersonalMail{
			PropertyFieldInfo: taptap.BuildPropertyFieldInfo(h.actor.Account.CliDeviceInfo),
			MailIds:           taptap.ConvertList2Str(addIds), // 新增邮件id列表
		}
		taptap.WriteDataLog(taptap.LogType_GloToPersonalMail, h.actor.GetUID(), h.actor.Account.TapUserInfo, e)
	})

	del := h.tryDelMail(userMails.UserMail)

	if err = h.SaveDB(); err != nil {
		return nil, nil, err
	}
	return userMails, del, nil
}

// 根据规则尝试删除邮件,返回删除的邮件id集合
func (h *MailHandler) tryDelMail(mails map[int64]*pb.PMailInfo) []int64 {
	now := time.Now().Unix()
	keySet := make([]int, 0)
	target := make([]int64, 0)
	// 优先删除过期
	for k, v := range mails {
		if v.ExpireTime > 0 && now >= v.ExpireTime {
			delete(mails, k)
			target = append(target, k)
			continue
		}

		// 没过期\没有过期限制
		keySet = append(keySet, int(k))
	}

	// 到达上限
	// limit := int(excel.GetConfigMgr().GetCfg().MAIL_NUM_LIMIT)
	limit := 0
	if len(keySet) > limit {
		sort.Ints(keySet)          // 增序排列
		sub := len(keySet) - limit //  需要删除的数量
		keySet2 := make([]int, 0)

		// 先删除无附件或者附件已经领取的邮件
		for _, k := range keySet {
			if sub <= 0 {
				break
			}

			// 取出mail
			mail := mails[int64(k)]
			if mail == nil || mail.IsReceived == common.MAIL_STATUS_UNRECEIVE {
				keySet2 = append(keySet2, k)
				continue
			}

			delete(mails, int64(k))
			target = append(target, int64(k))
			sub--
		}

		// 再按照从旧到新删除,直到小于上限为止
		for _, k := range keySet2 {
			if sub <= 0 {
				break
			}

			// 取出mail
			mail := mails[int64(k)]
			if mail == nil {
				continue
			}

			delete(mails, int64(k))
			target = append(target, int64(k))
			sub--
		}
	}

	return target
}

// genQuestionUrl
//
//	@Description: 生成问卷完整url
//	@param uid 玩家id
//	@param questionType 问卷类型 1=单语言 2=多语言
//	@param questionId 问卷id
//	@param lang
//	@return string 返回问卷url
func (h *MailHandler) genQuestionUrl(uid string, roleId uint64, questionType int32, questionId string, lang string) string {

	// url生成规则:
	// 基础url由问卷系统生成，可自行创建或询问用研获取；
	// 签名url由三个部分组成：基础url + 信息参数(参数顺序固定，如下) + clientKey，基础url包含协议+域名+路径；
	// 问卷url由三个部分组成：基础url + 信息参数(参数无顺序要求) + 签名，基础url包含协议+域名+路径。
	// 其中问卷url中的签名是由签名url进行MD5（32位小写）加密得来
	// 注: 单语言path: deliver 多语言path: temporary

	// role_key="open_id:user_id" ID格式应与数仓ID格式保持一致：问卷user_id = 数仓role_id，问卷open_id = 数仓open_id

	var url string
	var baseUrl = conf.GConf().Question.BaseUrl
	var region = conf.GConf().Sdk.ServerRegion
	var clientKey = conf.GConf().Question.ClientKey
	var roleKey = fmt.Sprintf("%s;%d", uid /* fixme 废弃 lilith.GetOpenId(uid)*/, roleId)

	if questionType == common.QUESTION_LANG_TYPE_SINGLE {
		signUrl := fmt.Sprintf("%s%s?sid=%s&region=%s&role_key=%s&key=%s", baseUrl, "deliver", questionId, region, roleKey, clientKey)
		sign := myUtils.Md5Str(signUrl)
		url = fmt.Sprintf("%s%s?sid=%s&region=%s&role_key=%s&sign=%s", baseUrl, "deliver", questionId, region, roleKey, sign)
		h.Debugf("signUrl: %s , url: %s", signUrl, url)
	} else if questionType == common.QUESTION_LANG_TYPE_MULTI {
		signUrl := fmt.Sprintf("%s%s/%s?region=%s&role_key=%s&value_url_key=%s&key=%s", baseUrl, "temporary", questionId, region, roleKey, lang, clientKey)
		sign := myUtils.Md5Str(signUrl)
		url = fmt.Sprintf("%s%s/%s?region=%s&role_key=%s&value_url_key=%s&sign=%s", baseUrl, "temporary", questionId, region, roleKey, lang, sign)
		h.Debugf("signUrl: %s , url: %s", signUrl, url)
	} else {
		h.Warnf("unrealized question type %d", questionType)
	}

	return url
}

// 随机一个MailActor进行调用
func randMailActorId() string {
	min := baseconf.GetBaseConf().MailActorMin
	percent := baseconf.GetBaseConf().MailActorPercent
	need := myUtils.CalTenThousand(global.UserActorCount, percent)
	if need < min {
		need = min
	}
	// 随机一个id
	id, err := myUtils.RandomInt(0, need)
	if err != nil {
		id = 0
	}
	return fmt.Sprint("mailactor:", id)
}
