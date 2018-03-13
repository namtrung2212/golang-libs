package Number

import (
	"math"
	"crypto/rand"
	"math/big"
	mathRan "math/rand" 
)
	

// Uint64ToBytes ...
func Uint64ToBytes(num uint64) []byte {
	result := make([]byte, 8)
	for i := range result {
		result[i] = byte(num & 255)
		num >>= 8
	}
	return result
}

// Int64ToBytes ...
func Int64ToBytes(num int64) []byte {
	result := make([]byte, 8)
	for i := range result {
		result[i] = byte(num & 255)
		num >>= 8
	}
	return result
}

// Uint32ToBytes ...
func Uint32ToBytes(num uint32) []byte {
	result := make([]byte, 4)
	for i := range result {
		result[i] = byte(num & 255)
		num >>= 8
	}
	return result
}

// Int32ToBytes ...
func Int32ToBytes(num int32) []byte {
	result := make([]byte, 4)
	for i := range result {
		result[i] = byte(num & 255)
		num >>= 8
	}
	return result
}

// Uint16ToBytes ...
func Uint16ToBytes(num uint16) []byte {
	result := make([]byte, 2)
	for i := range result {
		result[i] = byte(num & 255)
		num >>= 8
	}
	return result
}

// Int16ToBytes ...
func Int16ToBytes(num int16) []byte {
	result := make([]byte, 2)
	for i := range result {
		result[i] = byte(num & 255)
		num >>= 8
	}
	return result
}

// Round ...
func Round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

// ToFixed ...
func ToFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(Round(num*output)) / output
}

// GenerateInt generate random int in range
func GenerateInt(low, high int) int64 {
	if low == high {
		return int64(high)
	}

	if low > high {
		high, low = low, high
	}

	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(high-low)))
	if err != nil {
		return int64(mathRan.Intn(high-low)) + int64(low)
	}

	return nBig.Int64() + int64(low)
}