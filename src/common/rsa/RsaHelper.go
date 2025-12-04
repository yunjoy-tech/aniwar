package rsa

import (
	"encoding/base64"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/common/tls"

	"gitlab.musadisca-games.com/wangxw/musae/framework/tcpx"

	"github.com/forgoer/openssl"
	myUtils "gitlab.musadisca-games.com/wangxw/aniwar/src/common/utils"
	"gitlab.musadisca-games.com/wangxw/musae/framework/errorx"
	"gitlab.musadisca-games.com/wangxw/musae/framework/logger"
)

func CreateSrvRsaKey(c *tcpx.Context, base64CliRsaKey string) (string, string, string) {
	var (
		cliKey = ""
	)

	// base64解码
	bytes, err := base64.StdEncoding.DecodeString(base64CliRsaKey)
	if err != nil {
		logger.Warn(errorx.Wrap(err).Error())
		return "", "", ""
	}
	// rsa解密
	if cliKeyBytes, err := tls.RsaDecrypt(bytes); err != nil {
		logger.Warn(errorx.Wrap(err).Error())
		return "", "", ""
	} else {
		cliKey = string(cliKeyBytes)
	}
	logger.Debug("HandleRsa 客户端随机数: ", cliKey)

	// 服务器生成随机码
	srvKey := myUtils.RandomStr(32, true, true, true)
	logger.Debug("HandleRsa 服务器随机数: ", srvKey)

	// 服务器随机码 AES(cbc)加密
	srvKeyEncrypt, err := openssl.AesCBCEncrypt([]byte(srvKey), []byte(cliKey), make([]byte, 16), openssl.PKCS7_PADDING)
	//fmt.Printf(base64.StdEncoding.EncodeToString(srvKeyEncrypt))
	if err != nil {
		logger.Debugf(err.Error())
		return "", "", ""
	}

	logger.Debugf("===>>> RSA\n客户端随机码:%s, 密文:%s\n 服务器随机码:%s, 密文:%s",
		cliKey, cliKey, srvKey, srvKeyEncrypt)

	// 最终密码规则: MD5(客户端随机码+服务器随机码)
	rsaKey := tls.RsaVal(cliKey, srvKey)

	// base64 加码
	baseStr := base64.StdEncoding.EncodeToString(srvKeyEncrypt)

	return srvKey, baseStr, rsaKey
}
