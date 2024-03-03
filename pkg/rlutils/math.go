package rlutils

import (
	"math/rand"

	"golang.org/x/exp/constraints"
)

func ScaleValueToSize[T constraints.Float | constraints.Integer](
	value T,
	min, max T,
	newMin, newMax T,
) T {
	return newMin + (value-min)*(newMax-newMin)/(max-min)
}

func RandomFloat(from, to float32) float32 {
	return from + (to-from)*rand.Float32()
}
