package main

import (
	"math/rand"
	"time"
)

// RandomIntValue a simple random number generator
// this can be improved in future
func RandomIntValue(r *rand.Rand) int {
	return r.Int() % 1000000
}

// RandomInt64Value a simple random number generator
// based on the needs
func RandomInt64Value(r *rand.Rand) int64 {
	return r.Int63n(10000000000000000)
}

// RandomTimeValue genrates a random value in two minutes
// window (one minute on each side) from the current time
func RandomTimeValue(r *rand.Rand) int64 {
	d := r.Int() % 120
	d -= 60
	return time.Now().Unix() + int64(d)

}
