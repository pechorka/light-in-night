package rlutils

import (
	"testing"
)

func TestScaleValueToSize(t *testing.T) {
	t.Run("800 to 0-100 -> 80", func(t *testing.T) {
		got := ScaleValueToSize(800, 0, 1000, 0, 100)
		want := 80
		if got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	})
}
