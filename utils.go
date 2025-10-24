package main

import (
	"fmt"
	"math/rand"
	"time"
)

func generateID() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprint(r.Int63())
}
