package utils

import (
	"math/rand"
	"time"
)

/*
collection of random generating functions
Taken from https://www.calhoun.io/creating-random-strings-in-go/
*/

const (
	// Letters are the letters of an alphabet used to generate a random number
	Letters = "abcdefghijklmnopqrstuvwxyz"
)

// RandomUtils is our helper struct for random related utilities
type RandomUtils struct {
	Seed *rand.Rand
}

// NewRandomUtils is used to generate our random utils struct
func NewRandomUtils() *RandomUtils {
	seed := generateSeed()
	return &RandomUtils{Seed: seed}
}

// generateSeed is used to generate a random seed
func generateSeed() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}

// ReSeed is used to reseed our RNG
func (u *RandomUtils) ReSeed() {
	preSeed := u.Seed.Int63()
	u.Seed = rand.New(rand.NewSource(time.Now().UnixNano() + ((preSeed / 3) % 2)))
}

// GenerateString is used to generate a fixed length random string
// from the specified charset, using a fresh seed each time
func (u *RandomUtils) GenerateString(length int, charset string) string {
	u.ReSeed()
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[u.Seed.Intn(len(charset))]
	}
	return string(b)
}

// NewRandomString is a helper function to generate rando mstrings
func NewRandomString(length int) string {
	rutils := NewRandomUtils()
	return rutils.GenerateString(length, Letters)
}
