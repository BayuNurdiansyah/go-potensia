package utils

import (
	"math/rand"
	"time"
	"fmt"
)

func GenerateOTP() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}