package flare

import (
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	DimSpd           = 0.99
	WentOutThreshold = 10 // percent
)

var (
	centerColor           = rl.Color{R: 255, G: 0, B: 0, A: 127}
	edgeColor             = rl.Color{R: 255, G: 127, B: 0, A: 127}
	initialRadius float32 = 50
)

var (
	wentOutRadius float32 = initialRadius * WentOutThreshold / 100
)

type Flare struct {
	ID     int
	Pos    rl.Vector2
	Radius float32
}

func FromPos(pos rl.Vector2) *Flare {
	return &Flare{
		ID:     rand.Int(),
		Pos:    pos,
		Radius: initialRadius,
	}
}

func (f *Flare) Draw() {
	rl.DrawCircleGradient(
		int32(f.Pos.X), int32(f.Pos.Y), f.Radius,
		centerColor, edgeColor,
	)
}

func (f *Flare) Dim() {
	f.Radius *= DimSpd
}

func (f *Flare) WentOut() bool {
	return f.Radius < wentOutRadius
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
