package projectile

import (
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	initialSpeed  = 1
	initialRadius = 5
	initialDamage = 10
)

type Projectile struct {
	ID       int
	Pos      rl.Vector2
	Velocity rl.Vector2

	Radius float32
	Damage int
}

func FromPos(pos, velocity rl.Vector2) *Projectile {
	velocity = rl.Vector2Normalize(velocity)
	velocity = rl.Vector2Scale(velocity, initialSpeed)
	return &Projectile{
		ID:       rand.Int(),
		Pos:      pos,
		Velocity: velocity,

		Radius: initialRadius,
		Damage: initialDamage,
	}
}

func (p *Projectile) Move() {
	p.Pos = rl.Vector2Add(p.Pos, p.Velocity)
}

func (p *Projectile) Draw() {
	rl.DrawCircle(int32(p.Pos.X), int32(p.Pos.Y), p.Radius, rl.Red)
}

func (p *Projectile) Boundaries() rl.Rectangle {
	return rl.Rectangle{
		X:      p.Pos.X - p.Radius,
		Y:      p.Pos.Y - p.Radius,
		Width:  p.Radius * 2,
		Height: p.Radius * 2,
	}
}

func (p *Projectile) GetPos() rl.Vector2 {
	return p.Pos
}
