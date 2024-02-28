package rlutils

import "golang.org/x/exp/constraints"

func ScaleValueToSize[T constraints.Float | constraints.Integer](
	value T,
	min, max T,
	newMin, newMax T,
) T {
	return newMin + (value-min)*(newMax-newMin)/(max-min)
}
