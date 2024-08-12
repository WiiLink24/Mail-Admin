package main

import (
	"fmt"
	"math/rand"
	"time"
)

func generateBoundary() string {
	source := rand.NewSource(time.Now().Unix())
	val := rand.New(source)
	return fmt.Sprintf("%s/%d", time.Now().Format("200601021504"), val.Intn(8999999)+1000000)
}