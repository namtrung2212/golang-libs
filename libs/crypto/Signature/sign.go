package Signature

import (
	"crypto"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// UnsignSHA256 unsign with SHA256
func UnsignSHA256(publicKey interface{}, toSign, signed []byte) error {
	return unsign(publicKey, toSign, signed, crypto.SHA256)
}

// unsign ...
func unsign(publicKey interface{}, toSign, signed []byte, _hash crypto.Hash) error {
	if publicKey == nil {
		return fmt.Errorf("PublicKey is nil")
	}

	//
	d := hash(_hash, toSign)
	if d == nil {
		return fmt.Errorf("Hashing Algorithm not supported")
	}

	//
	switch publicKey.(type) {
	case *rsa.PublicKey:
		return rsa.VerifyPKCS1v15(publicKey.(*rsa.PublicKey), _hash, d, signed)
	}

	return fmt.Errorf("Public key format not supported")
}

// SignSHA1 sign with sha1
func SignSHA1(privateKey interface{}, data []byte) (signed []byte, err error) {
	return sign(privateKey, data, crypto.SHA1)
}

// SignSHA1Base64String sign with sha1 then base64 string
func SignSHA1Base64String(privateKey interface{}, data []byte) (res string, err error) {
	signed, err := SignSHA1(privateKey, data)
	if err != nil {
		return
	}

	res = base64.StdEncoding.EncodeToString(signed)
	return
}

// SignSHA256 sign with sha256
func SignSHA256(privateKey interface{}, data []byte) (signed []byte, err error) {
	return sign(privateKey, data, crypto.SHA256)
}

// SignMD5 sign with md5
func SignMD5(privateKey interface{}, data []byte) (signed []byte, err error) {
	return sign(privateKey, data, crypto.MD5)
}

// SignSHA256Base64String sign with sha256 then base64 string
func SignSHA256Base64String(privateKey interface{}, data []byte) (res string, err error) {
	signed, err := SignSHA256(privateKey, data)
	if err != nil {
		return
	}

	res = base64.StdEncoding.EncodeToString(signed)
	return
}

// SignMD5Base64String sign with md5 then base64 string
func SignMD5Base64String(privateKey interface{}, data []byte) (res string, err error) {
	signed, err := SignMD5(privateKey, data)
	if err != nil {
		return
	}

	res = base64.StdEncoding.EncodeToString(signed)
	return
}

// sign using custom signature algorithm
func sign(privateKey interface{}, data []byte, _hash crypto.Hash) (signed []byte, err error) {
	if privateKey == nil {
		err = fmt.Errorf("Private key is nil")
		return
	}

	// hashing message first
	d := hash(_hash, data)
	if d == nil {
		err = fmt.Errorf("Hashing Algorithm not supported")
		return
	}

	switch privateKey.(type) {
	// case *ecdsa.PrivateKey:
	//   return privateKey.(*ecdsa.PrivateKey).Sign(rand.Reader, d, _hash)
	case *rsa.PrivateKey:
		return privateKey.(*rsa.PrivateKey).Sign(rand.Reader, d, _hash)
	}

	err = fmt.Errorf("Private key format not supported")
	return
}

func hash(hash crypto.Hash, toHash []byte) []byte {
	var d []byte
	switch hash {
	case crypto.SHA1:
		h := sha1.New()
		h.Write(toHash)
		d = h.Sum(nil)
	case crypto.SHA256:
		h := sha256.New()
		h.Write(toHash)
		d = h.Sum(nil)
	case crypto.MD5:
		h := md5.New()
		h.Write(toHash)
		d = h.Sum(nil)
	}

	return d
}
