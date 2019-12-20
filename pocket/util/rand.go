package util

import (
	"time"
	"math/rand"
)

var (
	seed = rand.New(rand.NewSource(time.Now().UnixNano()))
)

// Rd implement Intn
func Rd (n int) int {
	return seed.Intn(n)
}

// RdRange rand between [min(n, m), max(n, m))
func RdRange (n, m int) int {
	if n == m {
		return n
	}
	if m < n {
		n, m = m, n
	}
	return n + seed.Intn(m - n)
}
