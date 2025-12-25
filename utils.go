package main

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"time"
)

func generateBoundary() string {
	return fmt.Sprintf("--%s-%d", time.Now().Format("20060102150405"), rand.Int63())
}

func UTF16ToBytes(uint16s []uint16) []byte {
	byteArray := make([]byte, len(uint16s)*2)
	for i, v := range uint16s {
		binary.BigEndian.PutUint16(byteArray[i*2:], v)
	}

	return byteArray
}
