package Bytes

import (
	"crypto/rand"

	msgpack "gopkg.in/vmihailenco/msgpack.v2"
)

// GenerateRandomBytes returns securely generated random bytes.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}

	return b, nil
}

func Encode(data interface{}) ([]byte, error) {
	return msgpack.Marshal(data)

	// var b bytes.Buffer
	// err := gob.NewEncoder(&b).Encode(data)
	// return b.Bytes(), err

}

func Decode(data []byte, v interface{}) error {
	return msgpack.Unmarshal(data, v)

	//return gob.NewDecoder(bytes.NewReader(data)).Decode(v)
}
