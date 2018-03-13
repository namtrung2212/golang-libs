package PPKeyTransform

import (
	"bytes"
	"crypto/dsa"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"
	"encoding/gob"
	"errors"
	"math/big"
)

var blankBytes = []byte{}

type DsaKeyFormat struct {
	Version       int
	P, Q, G, Y, X *big.Int
}

// ToByte_ECDSA_PrivateKey convert ecdsa private key to byte
func ToByte_ECDSA_PrivateKey(key *ecdsa.PrivateKey) ([]byte, error) {
	return x509.MarshalECPrivateKey(key)
}

// ToByte_RSA_PrivateKey convert ecdsa private key to byte
func ToByte_RSA_PrivateKey(key *rsa.PrivateKey) ([]byte, error) {
	return x509.MarshalPKCS1PrivateKey(key), nil
}

// ToByte_DSA_PrivateKey convert ecdsa private key to byte
func ToByte_DSA_PrivateKey(key *dsa.PrivateKey) ([]byte, error) {
	return asn1.Marshal(key)
}

// ToByte_ECDSA_PublicKey convert ecdsa public key to byte
func ToByte_ECDSA_PublicKey(key *ecdsa.PublicKey) ([]byte, error) {
	return x509.MarshalPKIXPublicKey(key)
}

// ToByte_DSA_PublicKey convert ecdsa public key to byte
func ToByte_DSA_PublicKey(key *dsa.PublicKey) (res []byte, err error) {
	buf := new(bytes.Buffer)
	encoder := gob.NewEncoder(buf)

	err = encoder.Encode(key)
	res = buf.Bytes()

	return
}

// ToByte_RSA_PublicKey convert rsa public key to byte
func ToByte_RSA_PublicKey(key *rsa.PublicKey) ([]byte, error) {
	return x509.MarshalPKIXPublicKey(key)
}

// GenericPublicKey get parsed public key from der format
func GenericPublicKey(data []byte) (pub interface{}, err error) {
	pub, err = x509.ParsePKIXPublicKey(data)
	if err == nil {
		return
	}

	var publickey dsa.PublicKey

	// Write to buffer
	buf := new(bytes.Buffer)
	buf.Write(data)

	// Get decoded
	decoder := gob.NewDecoder(buf)
	err = decoder.Decode(&publickey)
	pub = &publickey

	return
}

// ToByte_PublicKey convert dsa/rsa/ecdsa public key to byte[]
func ToByte_PublicKey(pub interface{}) ([]byte, error) {
	if pub == nil {
		return blankBytes, errors.New("Input is nil")
	}

	switch pub.(type) {
	case *ecdsa.PublicKey:
		return ToByte_ECDSA_PublicKey(pub.(*ecdsa.PublicKey))
	case *rsa.PublicKey:
		return ToByte_RSA_PublicKey(pub.(*rsa.PublicKey))
	case *dsa.PublicKey:
		return ToByte_DSA_PublicKey(pub.(*dsa.PublicKey))
	}

	return blankBytes, errors.New("Input is nil")
}

// ECDSA_PublicKey binary to public key
func ECDSA_PublicKey(data []byte) (*ecdsa.PublicKey, error) {
	public_key, err := GenericPublicKey(data)
	if err != nil {
		return nil, err
	}

	switch public_key := public_key.(type) {
	case *ecdsa.PublicKey:
		return public_key, nil
	default:
		return nil, errors.New("Wrong key format")
	}
}

// DSA_PublicKey binary to public key
func DSA_PublicKey(data []byte) (*dsa.PublicKey, error) {
	public_key, err := GenericPublicKey(data)
	if err != nil {
		return nil, err
	}

	switch public_key := public_key.(type) {
	case *dsa.PublicKey:
		return public_key, nil
	default:
		return nil, errors.New("Wrong key format")
	}
}

// RSA_PublicKey binary to public key
func RSA_PublicKey(data []byte) (*rsa.PublicKey, error) {
	public_key, err := GenericPublicKey(data)
	if err != nil {
		return nil, err
	}

	switch public_key := public_key.(type) {
	case *rsa.PublicKey:
		return public_key, nil
	default:
		return nil, errors.New("Wrong key format")
	}
}

// ECDSA_PrivateKey binary to private key
func ECDSA_PrivateKey(data []byte) (*ecdsa.PrivateKey, error) {
	return x509.ParseECPrivateKey(data)
}

// RSA_PrivateKey binary to private key from pkcs1 format
func RSA_PrivateKey(data []byte) (*rsa.PrivateKey, error) {
	return x509.ParsePKCS1PrivateKey(data)
}

// DSA_PrivateKey binary to private key from pkcs1 format
func DSA_PrivateKey(data []byte) (*dsa.PrivateKey, error) {
	val := new(DsaKeyFormat)

	_, err := asn1.Unmarshal(data, val)
	if err != nil {
		return nil, err
	}

	key := &dsa.PrivateKey{
		PublicKey: dsa.PublicKey{
			Parameters: dsa.Parameters{
				P: val.P,
				Q: val.Q,
				G: val.G,
			},
			Y: val.Y,
		},
		X: val.X,
	}
	return key, nil
}
