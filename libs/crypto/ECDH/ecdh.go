// Package ECDH encrypts and decrypts data using elliptic curve keys. Data
// is encrypted with AES-256-GCM with HMAC-SHA1 message tags using
// ECDHE to generate a shared key. The P384 curve is chosen in
// keeping with the use of AES-256 for encryption.
package ECDH

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"io"
)

var (
	// ErrEncrypt ...
	ErrEncrypt = errors.New("secret: encryption failed")
	// ErrDecrypt ...
	ErrDecrypt = errors.New("secret: decryption failed")
	// Curve ...
	Curve = elliptic.P256
	//
	blankBytes = []byte{}
)

const (
	// KeySize_AES256 ...
	KeySize_AES256 = 32
	// KeySize_AES128 ...
	KeySize_AES128 = 16
)

// Encrypt_ECDHE_ECDSA_AES256_GCM_HMAC_SHA1 encrypt secures and authenticates its input using the public key
// with ECDHE_ECDSA_AES256_GCM_HMAC_SHA1
func Encrypt_ECDHE_ECDSA_AES256_GCM_HMAC_SHA1(pub *ecdsa.PublicKey, in []byte) (out []byte, err error) {
	return encrypt(pub, in, KeySize_AES256)
}

// Decrypt_ECDHE_ECDSA_AES256_GCM_HMAC_SHA1 encrypt secures and authenticates its input using the public key
// with ECDHE_ECDSA_AES256_GCM_HMAC_SHA1
func Decrypt_ECDHE_ECDSA_AES256_GCM_HMAC_SHA1(pub *ecdsa.PrivateKey, in []byte) (out []byte, err error) {
	return decrypt(pub, in, KeySize_AES256)
}

// Encrypt_ECDHE_ECDSA_AES128_GCM_HMAC_SHA1 encrypt secures and authenticates its input using the public key
// with ECDHE_ECDSA_AES128_GCM_HMAC_SHA1
func Encrypt_ECDHE_ECDSA_AES128_GCM_HMAC_SHA1(pub *ecdsa.PublicKey, in []byte) (out []byte, err error) {
	return encrypt(pub, in, KeySize_AES128)
}

// Decrypt_ECDHE_ECDSA_AES128_GCM_HMAC_SHA1 encrypt secures and authenticates its input using the public key
// with ECDHE_ECDSA_AES128_GCM_HMAC_SHA1
func Decrypt_ECDHE_ECDSA_AES128_GCM_HMAC_SHA1(pub *ecdsa.PrivateKey, in []byte) (out []byte, err error) {
	return decrypt(pub, in, KeySize_AES128)
}

// encrypt secures and authenticates its input using the public key
// using ECDHE with AES-256-CBC-HMAC-SHA1.
func encrypt(pub *ecdsa.PublicKey, in []byte, KeySize int) (out []byte, err error) {
	ephemeral, err := ecdsa.GenerateKey(Curve(), rand.Reader)
	if err != nil {
		return
	}

	x, _ := pub.Curve.ScalarMult(pub.X, pub.Y, ephemeral.D.Bytes())
	if x == nil {
		return nil, errors.New("Failed to generate encryption key")
	}

	var shared []byte
	if KeySize == 32 {
		_shared := sha512.Sum384(x.Bytes())
		shared = make([]byte, len(_shared))
		copy(shared[:], _shared[:])
	} else {
		_shared := sha256.Sum256(x.Bytes())
		shared = make([]byte, len(_shared))
		copy(shared[:], _shared[:])
	}

	iv, err := makeRandom(aes.BlockSize)
	if err != nil {
		return
	}

	paddedIn := addPadding(in)
	ct, err := EncryptGCM(paddedIn, iv, shared[:KeySize])
	if err != nil {
		return
	}

	ephPub := elliptic.Marshal(pub.Curve, ephemeral.PublicKey.X, ephemeral.PublicKey.Y)
	out = make([]byte, 1+len(ephPub)+aes.BlockSize)
	out[0] = byte(len(ephPub))
	copy(out[1:], ephPub)
	copy(out[1+len(ephPub):], iv)
	out = append(out, ct...)

	h := hmac.New(sha1.New, shared[KeySize:])
	h.Write(iv)
	h.Write(ct)
	out = h.Sum(out)

	return
}

// decrypt authentications and recovers the original message from
// its input using the private key and the ephemeral key included in
// the message.
func decrypt(priv *ecdsa.PrivateKey, in []byte, KeySize int) (out []byte, err error) {
	ephLen := int(in[0])
	ephPub := in[1 : 1+ephLen]
	ct := in[1+ephLen:]

	if len(ct) < sha1.Size+aes.BlockSize {
		return nil, errors.New("Invalid ciphertext")
	}

	x, y := elliptic.Unmarshal(Curve(), ephPub)
	if x == nil {
		return nil, errors.New("Invalid public key")
	}

	x, _ = priv.Curve.ScalarMult(x, y, priv.D.Bytes())
	if x == nil {
		return nil, errors.New("Failed to generate encryption key")
	}

	var shared []byte
	if KeySize == 32 {
		_shared := sha512.Sum384(x.Bytes())
		shared = make([]byte, len(_shared))
		copy(shared[:], _shared[:])
	} else {
		_shared := sha256.Sum256(x.Bytes())
		shared = make([]byte, len(_shared))
		copy(shared[:], _shared[:])
	}

	tagStart := len(ct) - sha1.Size
	h := hmac.New(sha1.New, shared[KeySize:])
	h.Write(ct[:tagStart])
	mac := h.Sum(nil)
	if !hmac.Equal(mac, ct[tagStart:]) {
		return nil, errors.New("Invalid MAC")
	}

	paddedOut, err := DecryptGCM(ct[aes.BlockSize:tagStart], ct[:aes.BlockSize], shared[:KeySize])
	// paddedOut, err := DecryptCBC(ct[aes.BlockSize:tagStart], ct[:aes.BlockSize], shared[:KeySize])
	if err != nil {
		return
	}

	out, err = removePadding(paddedOut)
	return
}

// removePadding removes padding from data that was added with
// AddPadding
func removePadding(b []byte) ([]byte, error) {
	l := int(b[len(b)-1])
	if l > 16 {
		return nil, errors.New("Padding incorrect")
	}

	return b[:len(b)-l], nil
}

// addPadding adds padding to a block of data
func addPadding(b []byte) []byte {
	l := 16 - len(b)%16
	padding := make([]byte, l)
	padding[l-1] = byte(l)
	return append(b, padding...)
}

// DecryptCBC decrypt bytes using a key and IV with AES in CBC mode.
func DecryptCBC(data, iv, key []byte) (decryptedData []byte, err error) {
	aesCrypt, err := aes.NewCipher(key)
	if err != nil {
		return
	}
	ivBytes := append(blankBytes, iv...)

	decryptedData = make([]byte, len(data))
	aesCBC := cipher.NewCBCDecrypter(aesCrypt, ivBytes)
	aesCBC.CryptBlocks(decryptedData, data)

	return
}

// EncryptCBC encrypt data using a key and IV with AES in CBC mode.
func EncryptCBC(data, iv, key []byte) (encryptedData []byte, err error) {
	aesCrypt, err := aes.NewCipher(key)
	if err != nil {
		return
	}
	ivBytes := append(blankBytes, iv...)

	encryptedData = make([]byte, len(data))
	aesCBC := cipher.NewCBCEncrypter(aesCrypt, ivBytes)
	aesCBC.CryptBlocks(encryptedData, data)

	return
}

// EncryptGCM ...
func EncryptGCM(data, iv, key []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, ErrEncrypt
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, ErrEncrypt
	}

	nonce, err := makeRandom(gcm.NonceSize())
	if err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, data, iv), nil
}

// DecryptGCM ...
func DecryptGCM(data, iv, key []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, ErrEncrypt
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, ErrEncrypt
	}

	NonceSize := gcm.NonceSize()

	return gcm.Open(nil, data[:NonceSize], data[NonceSize:], iv)
}

// makeRandom is a helper that makes a new buffer full of random data.
func makeRandom(length int) ([]byte, error) {
	bytes := make([]byte, length)
	_, err := io.ReadFull(rand.Reader, bytes)
	return bytes, err
}
