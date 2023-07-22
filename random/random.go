package random

import (
	"math/rand"
	"time"
)

// Random is a source of random numbers. It is not thread-safe.
type Random struct {
	rnd *rand.Rand
}

const (
	alphanumeric = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
)

// New returns a new Random.
func New() *Random {
	src := rand.NewSource(time.Now().UnixNano())
	rnd := rand.New(src)
	return &Random{rnd: rnd}
}

// Bytes returns a random alphanumeric byte slice of given length.
func (r *Random) Bytes(length uint8) []byte {
	b := make([]byte, length)
	r.rnd.Read(b)
	return b
}

// String returns a random alphanumeric string of given length.
func (r *Random) String(length uint8) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = alphanumeric[rand.Int63()%int64(len(alphanumeric))]
	}
	return string(b)
}

// Int returns a non-negative pseudo-random int.
func (r *Random) Int() int {
	return r.rnd.Int()
}

// Intn returns, as an int, a non-negative pseudo-random number in [0,n).
// It panics if n <= 0.
func (r *Random) Intn(n int) int {
	return r.rnd.Intn(n)
}

// Int32 returns a non-negative pseudo-random 31-bit integer as an int32.
func (r *Random) Int32() int32 {
	return r.rnd.Int31()
}

// Int32n returns, as an int32, a non-negative pseudo-random number in [0,n).
// It panics if n <= 0.
func (r *Random) Int32n(n int32) int32 {
	return r.rnd.Int31n(n)
}

// Int64 returns a non-negative pseudo-random 63-bit integer as an int64.
func (r *Random) Int64() int64 {
	return r.rnd.Int63()
}

// Int64n returns, as an int64, a non-negative pseudo-random number in [0,n).
// It panics if n <= 0.
func (r *Random) Int64n(n int64) int64 {
	return r.rnd.Int63n(n)
}

// Uint32 returns a pseudo-random 32-bit value as a uint32.
func (r *Random) Uint32() uint32 {
	return r.rnd.Uint32()
}

// Uint32n returns, as an uint32, a non-negative pseudo-random number in [0,n).
// It panics if n <= 0.
func (r *Random) Uint32n(n uint32) uint32 {
	return uint32(r.rnd.Int31n(int32(n)))
}

// Uint64 returns a pseudo-random 64-bit value as a uint64.
func (r *Random) Uint64() uint64 {
	return r.rnd.Uint64()
}

// Uint64n returns, as an uint64, a non-negative pseudo-random number in [0,n).
// It panics if n <= 0.
func (r *Random) Uint64n(n uint64) uint64 {
	return uint64(r.rnd.Int63n(int64(n)))
}

// Float32 returns, as a float32, a pseudo-random number in [0.0,1.0).
func (r *Random) Float32() float32 {
	return r.rnd.Float32()
}

// Float32n returns, as a float32, a pseudo-random number in [0.0,n) if n >= 0,
// (-n,0.0] otherwise.
func (r *Random) Float32n(n float32) float32 {
	return r.rnd.Float32() * n
}

// Float64 returns, as a float64, a pseudo-random number in [0.0,1.0).
func (r *Random) Float64() float64 {
	return r.rnd.Float64()
}

// Float64n returns, as a float64, a pseudo-random number in [0.0,n) if n >= 0,
// (-n,0.0] otherwise.
func (r *Random) Float64n(n float64) float64 {
	return r.rnd.Float64() * n
}

// Perm returns, as a slice of n ints, a pseudo-random permutation of the
// integers [0,n).
func (r *Random) Perm(n int) []int {
	return r.rnd.Perm(n)
}

// Shuffle pseudo-randomizes the order of elements.
// n is the number of elements. Shuffle panics if n < 0.
// swap swaps the elements with indexes i and j.
func (r *Random) Shuffle(n int, swap func(i, j int)) {
	r.rnd.Shuffle(n, swap)
}
