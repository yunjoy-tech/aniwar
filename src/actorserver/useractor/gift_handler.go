package useractor

import (
	"context"
	"errors"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common"
	cdkeyUtil "gitlab.musadisca-games.com/wangxw/aniwar/src/common/cdk"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/clidto"
	excel "gitlab.musadisca-games.com/wangxw/aniwar/src/excel/data"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/proto/cmd"
	"gitlab.musadisca-games.com/wangxw/musae/framework/base"
	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"
	"gitlab.musadisca-games.com/wangxw/musae/framework/service"
	"google.golang.org/protobuf/proto"
)

type GiftHandler struct {
	*UABaseHandler
	cdkeyUtil.CdKeyMgr
}

// 临时兑换码保存
var CodeMap = make(map[string]int)

func NewGiftHandler(actor *UserActor) *GiftHandler {
	cdkey, err := cdkeyUtil.Init(cdkeyUtil.SECRET, cdkeyUtil.CHAR_SET, cdkeyUtil.KEY_LEN)
	if err != nil {
		return nil
	}

	h := &GiftHandler{UABaseHandler: NewUABaseHandler(actor, "GiftHandler"),
		CdKeyMgr: cdkey}
	h.ChildHandler = h

	actor.RegisterProtoHandler(int32(cmd.Protocols_PC2LS_UseGiftCodeReq), h.UseGiftCode)

	return h
}

func (h *GiftHandler) Init() error {
	return nil
}

func (h *GiftHandler) EnterGame() error {
	return nil
}

func (h *GiftHandler) DailyRefresh() error {
	return nil
}

func (h *GiftHandler) SetDBData(dbData proto.Message) error {
	return nil
}

func (h *GiftHandler) DBTable() (service.MongoDbType, string, proto.Message) {
	return service.MongoDbType_MongoNil, "", nil
}

func (h *GiftHandler) UseGiftCode(ctx context.Context, in *base.ProtoMsg) (proto.Message, error, int32) {
	var req cmd.C2LS_UseGiftCodeReq
	err := in.UnmarshalData(&req)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_DeSerializeError)
	}

	err = h.Redeem(req.Code, h.actor.comData)
	if err != nil {
		return nil, err, int32(cmd.ErrorCode_InvalidParam)
	}

	return &cmd.LS2C_UseGiftCodeRes{CommonData: h.actor.comData.FixDownComData()}, nil, 0
}

// 生成兑换码
func (h *GiftHandler) Generate(packageId int64, from int64, count int) (res []string, err error) {
	if packageId == 0 || from == 0 || count == 0 {
		err = errors.New("无效参数")
		return
	}
	// 数据库中检查该礼包是否存在
	//p, err := h.DB.CdKeyPackageGet(packageId)
	//if err != nil {
	//	return
	//}
	//if p.ID != packageId {
	//	return nil, errors.New("记录未找到")
	//}

	// 调用生成器生成指定数量兑换码
	res, err = h.CdKeyMgr.Generate(packageId, from, count)
	if err != nil {
		return
	}
	logger.Debugf("兑换码生成结果: %v", res)
	for _, v := range res {
		CodeMap[v] = 0
	}

	// 更新数据库中可兑换礼包的数量
	//nowMaxCount := from + int64(count) - 1
	//if nowMaxCount > p.GenerateCount {
	//	err = h.RdsGame.CdKeyPackageUpdateCount(packageId, nowMaxCount)
	//	if err != nil {
	//		return
	//	}
	//}

	return
}

// 兑换处理
func (h *GiftHandler) Redeem(code string, commonData *clidto.Comdata) (err error) {
	// 校验阶段
	//var cdkeyPackage dao.CdKeyPackage
	packageId, number := h.CdKeyMgr.Decode(code)
	if packageId == 0 || number == 0 {
		// 解析失败
		return errors.New("兑换码无效")
	}
	if _, ok := CodeMap[code]; !ok {
		return errors.New("兑换码不存在")
	}

	cfg := excel.GetPackageMgr().GetById(int32(packageId))
	if cfg == nil {
		return errors.New("配置不存在")
	}

	delete(CodeMap, code)
	_, err = GetDropMgr(h.actor).DropList2(cfg.Itemcontain, true, nil, commonData, common.CR_GM)

	// 查询礼包信息
	//cdkeyPackage, err = h.DB.CdKeyPackageGet(packageId)
	//if err != nil {
	//	return err
	//}
	//if cdkeyPackage.ID == 0 {
	//	return errors.New("兑换码无效")
	//}
	//if !cdkeyPackage.Valid {
	//	return errors.New("兑换尚未开启")
	//}
	//if number > cdkeyPackage.GenerateCount {
	//	return errors.New("兑换码无效")
	//}
	//now := time.Now().Unix()
	//if cdkeyPackage.StartTime > now {
	//	return errors.New("兑换尚未开启")
	//}
	//if cdkeyPackage.EndTime < now {
	//	return errors.New("兑换已经结束")
	//}
	//
	//user, err := h.DB.GetUser(uid)
	//if err != nil {
	//	return errors.New("用户不存在")
	//}

	// 实际兑换

	// 直接插入数据库
	//repeatedCode, err := h.DB.CdKeyRecordCreate(uid, packageId, number)
	//if err != nil {
	//	return err
	//}
	//if repeatedCode == 1 {
	//	return errors.New("礼包码无效：您已兑换过该礼包")
	//} else if repeatedCode == 2 {
	//	return errors.New("礼包码无效：该礼包码已被他人使用")
	//}

	// 发送奖励
	//err = h.sendReward(user, cdkeyPackage.MailTemplateId, cdkeyPackage.Reward)

	return err
}
