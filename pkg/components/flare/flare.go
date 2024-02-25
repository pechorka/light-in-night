package flare

import (
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	DimSpd           = 0.99
	WentOutThreshold = 1
)

type Flare struct {
	ID                     int
	Pos                    rl.Vector2
	Radius                 float32
	CenterColor, EdgeColor rl.Color

	Boundaries rl.Rectangle
}

func FromPos(pos rl.Vector2, radius float32, centerColor, edgeColor rl.Color) *Flare {
	return &Flare{
		ID:          rand.Int(),
		Pos:         pos,
		Radius:      radius,
		CenterColor: centerColor,
		EdgeColor:   edgeColor,

		Boundaries: rl.Rectangle{
			X:      pos.X - radius,
			Y:      pos.Y - radius,
			Width:  radius * 2,
			Height: radius * 2,
		},
	}
}

func (f *Flare) Draw() {
	rl.DrawCircleGradient(
		int32(f.Pos.X), int32(f.Pos.Y), f.Radius,
		f.CenterColor, f.EdgeColor,
	)
}

func (f *Flare) Dim() {
	f.Radius *= DimSpd
}

func (f *Flare) WentOut() bool {
	return f.Radius < WentOutThreshold
}
