package tls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
)

var (
	// 公钥私钥生成方式: (两种生成方式)
	// 第一种: 通过 rsa.GenerateKey 生成一个PrivateKey, 其内部持有一个PublicKey
	// rsa.GenerateKey(random io.Reader, bits int) (*rsa.PrivateKey, error)
	// 第二种:通过openssl来生成密钥对的文件
	//openssl genrsa 1024 > private-key.pem  // 1024代表密钥长度（单位bit）
	//openssl rsa -in private-key.pem -pubout -out public-key.pem
	sdkPublicKey  *rsa.PublicKey
	srvPrivateKey *rsa.PrivateKey
)

func GetPublic() *rsa.PublicKey {
	return sdkPublicKey
}

func GetPrivateKey() *rsa.PrivateKey {
	return srvPrivateKey
}

func InitRsaKey() error {
	// 读取公钥
	if publicKey, err := RSAReadPublicKey("./output/res/cert/public-key.pem"); err != nil {
		return err
	} else {
		sdkPublicKey = publicKey
	}

	// 读取私钥
	if privateKey, err := RSAReadPrivateKey("./output/res/cert/private-key.pem"); err != nil {
		return err
	} else {
		srvPrivateKey = privateKey
	}

	return nil
}

// RSAReadPrivateKey 读取私钥
func RSAReadPrivateKey(filePath string) (*rsa.PrivateKey, error) {
	privateKeyData, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	privateKeyBlock, _ := pem.Decode(privateKeyData)

	//var pri *rsa.PrivateKey
	//由BEGIN RSA PRIVATE KEY开头的文件（PKCS#1），用x509.ParsePKCS1PrivateKey
	//由BEGIN PRIVATE KEY开头的文件（PKCS#8），用x509.ParsePKCS8PrivateKey
	pri, parseErr := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)
	if parseErr != nil {
		return nil, parseErr
	}

	return pri, nil
}

// RSAReadPublicKey 读取公钥
func RSAReadPublicKey(filePath string) (*rsa.PublicKey, error) {
	publicKeyData, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	publicKeyBlock, _ := pem.Decode(publicKeyData)

	//var pri *rsa.PublicKey
	pkixPublicKey, err := x509.ParsePKIXPublicKey(publicKeyBlock.Bytes)
	if err != nil {
		return nil, err
	}

	return pkixPublicKey.(*rsa.PublicKey), nil
}

// rsa sdkrsa
func RsaEncrypt(data []byte) ([]byte, error) {
	//block, _ := pem.Decode([]byte(serverpublickey))
	//if block == nil {
	//	return nil, errors.New("public key error")
	//}
	//pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	//if err != nil {
	//	return nil, err
	//}
	//pub := pubInterface.(*rsa.PublicKey)
	if sdkPublicKey == nil {
		err := InitRsaKey()
		if err != nil {
			return nil, err
		}
	}

	ciphertext, err := rsa.EncryptPKCS1v15(rand.Reader, sdkPublicKey, data)
	if err != nil {
		return nil, err
	}
	return ciphertext, nil
}

// rsa decrypt
func RsaDecrypt(ciphertext []byte) ([]byte, error) {
	//block, _ := pem.Decode([]byte(serverprivatekey))
	//if block == nil {
	//	return nil, errors.New("private key error")
	//}
	//priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	//if err != nil {
	//	return nil, err
	//}
	if srvPrivateKey == nil {
		err := InitRsaKey()
		if err != nil {
			return nil, err
		}
	}

	data, err := rsa.DecryptPKCS1v15(rand.Reader, srvPrivateKey, ciphertext)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func RsaVal(cliKey, srvKey string) string {
	return cliKey + srvKey
}
