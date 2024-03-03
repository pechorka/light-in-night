package grenade

import (
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var (
	centerColor             = rl.Color{R: 0, G: 255, B: 0, A: 127}
	edgeColor               = rl.Color{R: 0, G: 255, B: 127, A: 127}
	initialRadius   float32 = 50
	initialDiameter float32 = initialRadius * 2
)

const (
	duration float32 = 1 // seconds
	damage   float32 = 100
)

type Grenade struct {
	ID     int
	Pos    rl.Vector2
	Damage float32

	activeFor float32
}

func FromPos(pos rl.Vector2) *Grenade {
	return &Grenade{
		ID:     rand.Int(),
		Pos:    pos,
		Damage: damage,
	}
}

func (f *Grenade) Draw() {
	rl.DrawCircleGradient(
		int32(f.Pos.X), int32(f.Pos.Y), initialRadius,
		centerColor, edgeColor,
	)
}

func (f *Grenade) ProgressTime(dt float32) {
	f.activeFor += dt
}

func (f *Grenade) Active() bool {
	return f.activeFor < duration
}

func (f *Grenade) Boundaries() rl.Rectangle {
	return rl.Rectangle{
		X:      f.Pos.X - initialRadius,
		Y:      f.Pos.Y - initialRadius,
		Width:  initialDiameter,
		Height: initialDiameter,
	}
}

func (f *Grenade) GetPos() rl.Vector2 {
	return f.Pos
}
