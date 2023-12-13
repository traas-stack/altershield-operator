package utils

import (
	"math/rand"
	"time"
)

func Int32IndirectOrZero(ptr *int32) int32 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

const (
	CharSet = "abcdefghijklmnopqrstuvwxyz"
)
func RandomString(length int) (str string) {
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = CharSet[seededRand.Intn(len(CharSet))]
	}
	return string(b)
}