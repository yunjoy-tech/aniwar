package logic

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/dapr/go-sdk/service/common"
	myCommon "github.com/yunjoy-tech/aniwar/src/common"
	"github.com/yunjoy-tech/aniwar/src/common/conf"
	"github.com/yunjoy-tech/aniwar/src/common/db"
	"github.com/yunjoy-tech/aniwar/src/meta"
	"github.com/yunjoy-tech/aniwar/src/proto/pb"
	"github.com/yunjoy-tech/musae/logger"
	"github.com/yunjoy-tech/musae/service"
	"github.com/yunjoy-tech/musae/state"
	"github.com/yunjoy-tech/musae/utils"
	netutil "github.com/yunjoy-tech/musae/utils/net"
	timeutil "github.com/yunjoy-tech/musae/utils/time"
	"google.golang.org/protobuf/proto"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

func (s *IDIPServer) PreHandle(in *common.InvocationEvent, secretKey string) ([]byte, pb.ErrorCode, string) {
	// IP校验
	logger.Debugf("remote addr: %s", in.Request.RemoteAddr)
	if conf.GMT().IsIpWhite {
		ip, err := netutil.GetClientIP(in.Request)
		if err != nil {
			return nil, pb.ErrorCode_IpLimit, IP_LIMIT
		}
		if !CheckIp(conf.GMT().IpWhiteList, ip) {
			return nil, pb.ErrorCode_IpLimit, IP_LIMIT
		}
	}

	// 验签
	reqJson, b := CheckSign(string(in.Data), secretKey)
	if !b {
		logger.Debugf("request sign failed.")
		return nil, pb.ErrorCode_SignCheckError, Sign_Check_Error
	}

	// 前置处理结束了，返回请求json数据进行逻辑处理
	return reqJson, pb.ErrorCode_Success, ""
}

// CheckSign
//
//	@Description: 校验gm请求的签名
//	@param reqData 请求字符串
//	@param secretKey 签名密钥
//	@return []byte
//	@return bool 验签通过返回true，否则返回false
func CheckSign(reqData, secretKey string) ([]byte, bool) {
	// 验签和解析数据规则:
	// 1. 截取数据前32个字符作为签名
	// 2. 将数据剩余部分与APISecret拼接后算出md5值，与步骤1获取的签名做对比
	// 3. 如果签名验证通过，则将步骤2中数据剩余部分用base64解码得到json格式的查询数据

	logger.Debugf("check sign reqData: %s", reqData)
	signStr := string([]rune(reqData)[:32])
	dataStr := string([]rune(reqData)[32:])

	tempStr := fmt.Sprintf("%s%s", dataStr, secretKey)
	md5Str := utils.MD5Str(tempStr)
	logger.Debugf("signStr: %s  dataStr: %s  md5Str: %s ", signStr, dataStr, md5Str)
	if md5Str == signStr {
		// 解析数据
		retStr, err := base64.StdEncoding.DecodeString(dataStr)
		if err != nil {
			return nil, false
		}
		return retStr, true
	}

	return nil, false
}

// CheckIp
//
//	@Description: 请求IP校验
//	@param ip 请求来源IP
//	@return bool 合法返回true，否则返回false
func CheckIp(whiteIps []string, ip string) bool {

	logger.Debugf("check ip: %v, ---> %s", whiteIps, ip)
	// whiteList := conf.GMT().IpWhiteList
	for _, e := range whiteIps {
		if e == ip {
			return true
		}
	}

	return false
}

// RecordOperation
//
//	@Description: api调用请求记录
//	@param key 存档的key
//	@param reqData 请求的json数据
//	@return error 返回遇到的错误
func (s *IDIPServer) RecordOperation(key string, reqData []byte) error {

	logger.Debugf("record operation data: %s", string(reqData))
	now := time.Now().Unix()
	kvTable := &state.KvTable{
		Id:      0,
		UID:     "0", // 无法获取到，lilith gmt会记录
		Data:    reqData,
		UpSecTS: now,
		InSecTS: now,
	}

	return s.SaveMongoGmt(key, kvTable, nil)
}

func lilithKey() string {
	return db.KeyGmtLilith(strconv.FormatInt(time.Now().Unix(), 10))
}

func aniwarKey() string {
	return db.KeyGmtAniwar(strconv.FormatInt(time.Now().Unix(), 10))
}

// RetCommonMsg
//
//	@Description: 通用的返回数据处理
//	@param c
//	@param httpCode 处理失败需返回500,处理请求成功则返回200
//	@param ret 错误码，全部或者部分处理成功返回0，失败为非0值
//	@param info 处理失败为错误信息字符串，处理成功为返回的json数据
func RetCommonMsg(out *common.Content, httpCode int, ret int32, info interface{}) {
	// 构建数据
	retMsg := CommonRet{
		Ret:  ret,
		Info: info,
	}

	// 返回resp
	data, err := json.Marshal(retMsg)
	if err != nil {
		logger.Error(err)
		return
	}
	out.Data = data
	logger.Debugf("RetCommonMsg httpCode: %d, errCode: %d, info: %+v", httpCode, ret, info)
}

func buildRetItem(svrId, uid, ret int32, info string) *RetItems {
	return &RetItems{
		SvrId:  svrId,
		UserId: uid,
		Ret:    ret,
		Info:   info,
	}
}

// 查询玩家的游戏模块数据
func (s *IDIPServer) getUserGameData(key string, data proto.Message) error {
	table, err := s.GetMongoGame(key, nil)
	if err != nil {
		return err
	}

	if err = proto.Unmarshal(table.Data, data); err != nil {
		return err
	}
	return nil
}

// 查询玩家的账号数据
func (s *IDIPServer) getUserAccountData(uid string, account proto.Message) error {
	table, err := s.GetMongoAccount(db.KeyAccountInfo(uid), nil)
	if err != nil {
		return err
	}

	if table != nil {
		if err = proto.Unmarshal(table.Data, account); err != nil {
			return err
		}
	}

	return nil
}

func (s *IDIPServer) getSysMailData(key string, value proto.Message) error {
	if kvTable, err := s.GetMongoMail(key, nil); err != nil {
		if errors.Is(err, service.DB_ERROR_NOT_EXIST) {
			return nil // db中没有数据
		} else {
			return err
		}
	} else {
		err = proto.Unmarshal(kvTable.Data, value)
		if err != nil {
			return err
		}
	}

	return nil
}

// IsValidCmd 合法cmd校验
func IsValidCmd(cmd string) bool {
	// if !conf.Base().IsDebug {
	//	return strings.HasPrefix(cmd, "user.test")
	// }

	return false
}

func IsGlobalCmd(cmd string) bool {
	return strings.HasPrefix(cmd, "global.")
}

func errCheck(err error, id int, items []*RetItems) {
	if err != nil {
		items = append(items, &RetItems{
			SvrId:  0,
			UserId: int32(id),
			Ret:    int32(pb.ErrorCode_InternalError),
			Info:   err.Error(),
		})
		logger.Error("add error", err)
	}
}

type TempMail struct {
	Id              int64  `json:"id,omitempty"`          // 唯一id
	Title           string `json:"title,omitempty"`       // 标题
	Content         string `json:"content,omitempty"`     // 内容
	Sender          string `json:"sender,omitempty"`      // 发件人
	MailType        int32  `json:"mail_type,omitempty"`   // 类型
	CreateTime      string `json:"create_time,omitempty"` // 创建时间
	ExpireTime      string `json:"expire_time,omitempty"` // 过期时间
	IsReceived      string `json:"is_received,omitempty"` // 领取状态（1=未领取，2=已领取）
	Attachments     string `json:"attachments,omitempty"` // 附件
	ReceiveUserList string `json:"receive_user_list"`     // 领取白名单
}

// ConvertMail 邮件可视化转换
func (s *IDIPServer) ConvertMail(mails map[int64]*pb.PMailInfo) []*TempMail {
	ret := make([]*TempMail, 0, len(mails))
	for _, v := range mails {
		t := &TempMail{
			Id:          v.Id,
			Title:       s.GetLocalizedStr(v.Title),
			Content:     s.GetLocalizedStr(v.Content),
			Sender:      s.GetLocalizedStr(v.Sender),
			MailType:    v.MailType,
			CreateTime:  timeutil.FormatStr(v.CreateTime),
			ExpireTime:  timeutil.FormatStr(v.ExpireTime),
			Attachments: s.ConvertItem(v.Attachments),
		}
		if v.IsReceived == myCommon.MAIL_STATUS_UNRECEIVE {
			t.IsReceived = "未领取"
		} else if v.IsReceived == myCommon.MAIL_STATUS_RECEIVED {
			t.IsReceived = "已领取"
		} else {
			t.IsReceived = "不可领取"
		}

		ret = append(ret, t)
	}

	return ret
}

// ConvertSysMail 系统邮件可视化处理
func (s *IDIPServer) ConvertSysMail(mails map[int64]*pb.PSysMailInfo) []*TempMail {
	ret := make([]*TempMail, 0, len(mails))
	for _, v := range mails {
		t := &TempMail{
			Id:              v.Id,
			Title:           s.GetLocalizedStr(v.Title),
			Content:         s.GetLocalizedStr(v.Content),
			Sender:          s.GetLocalizedStr(v.Sender),
			MailType:        v.MailType,
			Attachments:     s.ConvertItem(v.Attachments),
			ReceiveUserList: ConvertItem2(v.ReceiveUserIds),
		}
		if v.CreateTime != 0 {
			t.CreateTime = timeutil.FormatStr(v.CreateTime)
		}
		if v.ExpireTime != 0 {
			t.ExpireTime = timeutil.FormatStr(v.ExpireTime)
		}

		ret = append(ret, t)
	}

	return ret
}

func ConvertItem2(uids map[string]int32) string {
	if len(uids) == 0 {
		return ""
	}
	var sb strings.Builder
	for uid := range uids {
		sb.WriteString(uid)
		sb.WriteString(",")
	}
	return strings.TrimSuffix(sb.String(), ",")
}

func (s *IDIPServer) ConvertItem(items []*pb.ItemReward) string {
	var sb strings.Builder
	for _, v := range items {
		var cfg *meta.ItemPkgItemMeta
		// cfg := excel.GetItemMgr().GetById(int32(v.ItemId))
		if cfg == nil {
			continue
		}
		sb.WriteString(fmt.Sprintf("%s:%d;", s.GetLocalizedStr(""), v.Num))
	}
	return sb.String()
}

// GetLocalizedStr 获取国际化配置中文字符串，如果找不到则返回原key
func (s *IDIPServer) GetLocalizedStr(key string) string {
	ret, ok := s.LocalizedStr[key]
	if !ok {
		return key
	}
	return ret
}

type TempItem struct {
	ItemId              int32  `json:"item_id"`              // id
	Num                 int32  `json:"num"`                  // 数量
	ExpirationTimestamp string `json:"expiration_timestamp"` // 过期时间
	Name                string `json:"name"`                 //  名字
}

func (s *IDIPServer) ConvertBagItem(items map[uint64]*pb.PCommonItemInfo) []*TempItem {
	ret := make([]*TempItem, 0, len(items))
	for _, v := range items {
		var cfg *meta.ItemPkgItemMeta
		// cfg := excel.GetItemMgr().GetById(int32(v.GetBaseId()))
		if cfg == nil {
			continue
		}
		timestamp := ""
		if v.GetExpirationTimestamp() > 0 {
			timestamp = time.Unix(v.GetExpirationTimestamp(), 0).Format("2006-01-02 15:04:05")
		}
		ret = append(ret, &TempItem{
			ItemId:              int32(v.GetBaseId()),
			Num:                 int32(v.GetItemNum()),
			ExpirationTimestamp: timestamp,
			Name:                s.GetLocalizedStr(""),
		})
	}
	return ret
}

type TempCard struct {
	BaseId          int32        `json:"base_id"`
	Name            string       `json:"name"`
	CardLevel       int32        `json:"card_level"`
	CardExp         int32        `json:"card_exp"`
	Hp              int32        `json:"hp"`
	AddNum          int32        `json:"add_num"`
	AwakenLevel     int32        `json:"awaken_level"` // 突破等级
	CreateTimestamp string       `json:"create_timestamp"`
	EquipId         []*EquipInfo `json:"equip_id"`
	FavoriteLevel   int32        `json:"favorite_level"`
	FavoriteExp     int32        `json:"favorite_exp"`
	Character       []int32      `json:"character"`
	CurCharacter    int32        `json:"cur_character"`
	OldMaxHp        int32        `json:"old_max_hp"`
	SkillCfgId      []*SkillInfo `json:"skill_cfg_id"`
	SkinId          *SkinInfo    `json:"skin_id"`
}

type SkillInfo struct {
	Pos     int32  `json:"pos"`
	SkillId int32  `json:"skill_id"`
	Level   int32  `json:"level"`
	Name    string `json:"name"`
}

func (s *IDIPServer) NewSkillInfo(pos, skillId int32) *SkillInfo {
	// cfg := excel.GetSkillMgr().GetById(skillId)
	// if cfg == nil {
	// 	return nil
	// }
	return &SkillInfo{
		Pos:     pos,
		SkillId: skillId,
		Level:   0,
		Name:    s.GetLocalizedStr(""),
	}
}

type EquipInfo struct {
	Pos      int32  `json:"pos"`
	EquipId  int32  `json:"equip_id"` // 装备Id
	ConfigId int32  `json:"config_id"`
	Name     string `json:"name"`
	Level    int32  `json:"level"`
}

func (s *IDIPServer) NewEquipInfo(pos, configId, equipId int32) *EquipInfo {
	// cfg := excel.GetEquipmentMgr().GetById(configId)
	// if cfg == nil {
	// 	return nil
	// }

	return &EquipInfo{
		Pos:      pos,
		EquipId:  equipId,
		ConfigId: configId,
		Name:     s.GetLocalizedStr(""),
		Level:    0,
	}
}

type SkinInfo struct {
	SkinId int32  `json:"skin_id"`
	Name   string `json:"name"`
}

func (s *IDIPServer) NewSkin(skinId int32) *SkinInfo {
	// cfg := excel.GetSkinMgr().GetById(skinId)
	// if cfg == nil {
	// 	return nil
	// }
	// return &SkinInfo{
	// 	SkinId: skinId,
	// 	Name:   s.GetLocalizedStr(cfg.SkinName),
	// }
	return nil
}

func (s *IDIPServer) ConvertCard(cards map[uint32]*pb.CardData, equip *pb.PEquipData) []*TempCard {
	ret := make([]*TempCard, 0, len(cards))
	for _, v := range cards {
		// cfg := excel.GetBeastarMgr().GetById(int32(v.GetBaseId()))
		// if cfg == nil {
		// 	continue
		// }
		timestamp := ""
		if v.GetCreateTimestamp() > 0 {
			timestamp = time.Unix(v.GetCreateTimestamp(), 0).Format("2006-01-02 15:04:05")
		}
		// 技能
		skillMap := make([]*SkillInfo, 0)
		for k, v := range v.GetSkillCfgId() {
			skillMap = append(skillMap, s.NewSkillInfo(int32(k), int32(v)))
		}
		// 装备
		tmpEquip := make([]*EquipInfo, 0)
		for pos, id := range v.GetEquipId() {
			if tmp, ok := equip.Equips[id]; ok {
				e := s.NewEquipInfo(int32(pos), tmp.ConfigId, int32(id))
				e.Level = tmp.Level
				tmpEquip = append(tmpEquip, e)
			}
		}
		ret = append(ret, &TempCard{
			BaseId:          int32(v.GetBaseId()),
			Name:            s.GetLocalizedStr(""),
			CardLevel:       int32(v.GetCardLevel()),
			CardExp:         int32(v.GetCardExp()),
			Hp:              int32(v.GetHp()),
			AddNum:          int32(v.GetAddNum()),
			AwakenLevel:     int32(v.GetAwakenLevel()),
			CreateTimestamp: timestamp,
			FavoriteLevel:   int32(v.GetFavoriteLevel()),
			FavoriteExp:     int32(v.GetFavoriteExp()),
			SkinId:          s.NewSkin(int32(v.GetSkinId())),
			SkillCfgId:      skillMap,
			Character:       v.GetCharacter(),
			CurCharacter:    int32(v.GetCurCharacter()),
			EquipId:         tmpEquip,
			OldMaxHp:        int32(v.GetOldMaxHp()),
		})
	}
	return ret
}

// 下载oss文件到本地指定文件中
func OssGetObjectToLocalFile(bucketName, objName, localFS string) error {
	client, err := oss.New(conf.OSS().Endpoint, conf.OSS().AccessKey, conf.OSS().AccessSecret)
	if err != nil {
		return err
	}

	bucket, err := client.Bucket(bucketName)
	if err != nil {
		return err
	}

	body, err := bucket.GetObject(objName)
	if err != nil {
		return err
	}
	defer body.Close()

	fd, err := os.OpenFile(localFS, os.O_WRONLY|os.O_CREATE, 0660)
	if err != nil {
		return err
	}
	defer fd.Close()

	_, err = io.Copy(fd, body)
	return err
}
