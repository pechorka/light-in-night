package flare

import (
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	DimSpd           = 0.99
	WentOutThreshold = 10 // percent
)

type Flare struct {
	ID                     int
	Pos                    rl.Vector2
	Radius                 float32
	CenterColor, EdgeColor rl.Color

	originalRadius float32
}

func FromPos(pos rl.Vector2, radius float32, centerColor, edgeColor rl.Color) *Flare {
	return &Flare{
		ID:             rand.Int(),
		Pos:            pos,
		Radius:         radius,
		originalRadius: radius,
		CenterColor:    centerColor,
		EdgeColor:      edgeColor,
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
	return f.Radius < f.originalRadius*WentOutThreshold/100
}

func (f *Flare) Boundaries() rl.Rectangle {
	return rl.Rectangle{
		X:      f.Pos.X - f.Radius,
		Y:      f.Pos.Y - f.Radius,
		Width:  f.Radius * 2,
		Height: f.Radius * 2,
	}
}

func (f *Flare) GetPos() rl.Vector2 {
	return f.Pos
}
