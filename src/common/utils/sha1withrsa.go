package utils

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

const (
	// 废弃字段，先不删
	serverprivatekey = "-----BEGIN RSA PRIVATE KEY-----\n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\n-----END RSA PRIVATE KEY-----\n"
	// sdk解密公钥
	serverpublickey = "-----BEGIN PUBLIC KEY-----\nMIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEA5jqhz24dZQERezDd0aul\ngSGwasQtLdWX8CAwXdZFJA/BQ/aOpGqd+aDO+b+x3ZFZIWGiisFN3kJsphk8yzu0\nLqa7YXhiSxM2v8NctdseGwo53qzEeEwq4WMomc5cZz5e1KuMF0LpNazdsYL9Il+S\nPzaG4IjkD9tY1nCVwwoOzwVYPlk1Hl40aUHZjLJWP0ufpeYmWfCrh0aB1PheqeKv\ntyQ6+8uGQMtLpyyXl1U1OgAkE/jKgNzZ2w1/lk+WR8Ah7TTJEnaaXbxETN539Qyu\ngWV/kx85zhJDRGH0m9GcUcWvRMOyexGVE1cRTh4xiO9CSnAzHQLWPAMm5LszMtsx\nsjSOMe43BLKkPhkzHXW1Le1bLPQaD759erSCFFVLfm0TRA48+k0StqrtzQXcc++S\ntQ8eEgdiJ3CAuUFK2uk19WefFmDn+5qA2L+EOvLVFOwEpXiqehzLL7JdrMjLdSIt\nd/8bb6oRuvP2LH+RiDvhOiEqiel/z3K64rn9M90jeXcX8475DnspC7Ce+xqi2XKR\n0b+X9ZaizDWu+1RjwYOAXAxgcPSQjUJm8Ug3XH96gH4R6FDmUniGlxesVtv4Wz1y\n86EpafnW9SuE1qxOw/+qyN7i40s6sDSAg/ovJy2k3w0dOK89mRY39BLIWBcLZSJ3\n07vr40/UmV574IsCtX+p5k8CAwEAAQ==\n-----END PUBLIC KEY-----"
)

var (
	sdkPublicKey  *rsa.PublicKey
	srvPrivateKey *rsa.PrivateKey
)

func init() {
	UpdateSrvPrivateKey(serverprivatekey, serverpublickey)
}

func UpdateSrvPrivateKey(priKey, pubKey string) bool {
	block, _ := pem.Decode([]byte(priKey))
	if block == nil {
		return false
	}
	var err error
	srvPrivateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return false
	}

	block, _ = pem.Decode([]byte(pubKey))
	if block == nil {
		return false
	}
	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return false
	}
	sdkPublicKey = publicKey.(*rsa.PublicKey)
	return true
}

//rsa key generate
func genRsaKey() ([]byte, []byte, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}
	derStream := x509.MarshalPKCS1PrivateKey(privateKey)
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: derStream,
	}
	prvkey := pem.EncodeToMemory(block)
	publicKey := &privateKey.PublicKey
	derPkix, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return nil, nil, err
	}
	block = &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: derPkix,
	}
	pubkey := pem.EncodeToMemory(block)
	return prvkey, pubkey, nil
}

// rsa sign with sha1 by serverprivatekey
func rsaSignWithSha1(data []byte) ([]byte, error) {
	if srvPrivateKey == nil {
		return nil, errors.New("private key error")
	}
	h := sha1.New()
	h.Write(data)
	hashed := h.Sum(nil)
	signature, err := rsa.SignPKCS1v15(rand.Reader, srvPrivateKey, crypto.SHA1, hashed)
	if err != nil {
		return nil, errors.New("ParsePKCS1PrivateKey error")
	}
	return signature, nil
}

// rsa sign with sha1 by serverpublickey
func RsaVerifySignWithSha1(data, signData []byte) error {
	if sdkPublicKey == nil {
		return errors.New("public key error")
	}
	hashed := sha1.Sum(data)
	err := rsa.VerifyPKCS1v15(sdkPublicKey, crypto.SHA1, hashed[:], signData)
	if err != nil {
		return err
	}
	return nil
}

// rsa sdkrsa
func rsaEncrypt(data []byte) ([]byte, error) {
	block, _ := pem.Decode([]byte(serverpublickey))
	if block == nil {
		return nil, errors.New("public key error")
	}
	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pub := pubInterface.(*rsa.PublicKey)
	ciphertext, err := rsa.EncryptPKCS1v15(rand.Reader, pub, data)
	if err != nil {
		return nil, err
	}
	return ciphertext, nil
}

// rsa decrypt
func rsaDecrypt(ciphertext []byte) ([]byte, error) {
	block, _ := pem.Decode([]byte(serverprivatekey))
	if block == nil {
		return nil, errors.New("private key error")
	}
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	data, err := rsa.DecryptPKCS1v15(rand.Reader, priv, ciphertext)
	if err != nil {
		return nil, err
	}
	return data, nil
}
