package cdkeyUtil

import (
	"errors"
	"github.com/speps/go-hashids"
	"strings"
)

type CdKeyMgr struct {
	hashID   *hashids.HashID
	cdKeyLen int
}

const (
	SECRET   = "aniwar.1024"
	CHAR_SET = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	KEY_LEN  = 14
)

func Init(secret, charSetStr string, cdKeyLen int) (CdKeyMgr, error) {

	upStr := strings.ToUpper(charSetStr)
	if strings.Compare(upStr, charSetStr) != 0 {
		return CdKeyMgr{}, errors.New("CDKEY必须使用大写字母：" + charSetStr)
	}

	mateData := &hashids.HashIDData{
		Alphabet:  charSetStr,
		MinLength: cdKeyLen,
		Salt:      secret,
	}

	HashId, err := hashids.NewWithData(mateData)
	if err != nil {
		return CdKeyMgr{}, err
	}

	return CdKeyMgr{hashID: HashId, cdKeyLen: cdKeyLen}, err
}

// Generate 生成指定数量的cdKey，同参数生成的cdKey相同
// 例如：如针对礼包1生成100个，则使用 Generate(1,1,100)。生成礼包2，之前已经生成了100个，想继续生成100个，则 Generate(2,101,100)
func (c CdKeyMgr) Generate(contentId, numFrom int64, needCount int) (cdKeyList []string, err error) {
	if numFrom < 1 || needCount < 1 {
		err = errors.New("错误的生成调用")
		return
	}
	cdKeyList = make([]string, 0, needCount)
	for i := 0; i < needCount; i++ {
		s, e := c.hashID.EncodeInt64([]int64{contentId, numFrom})
		if e != nil {
			err = e
			return
		}
		cdKeyList = append(cdKeyList, s)
		numFrom++
	}
	return
}

// Decode 解析cdKey，返回生成时所用的数据
func (c CdKeyMgr) Decode(cdKey string) (contentId, number int64) {
	if len(cdKey) < c.cdKeyLen {
		return
	}
	cdKey = strings.ToUpper(cdKey)
	d, err := c.hashID.DecodeInt64WithError(cdKey)
	if err != nil {
		return
	}
	if len(d) != 2 {
		return
	}
	return d[0], d[1]
}
