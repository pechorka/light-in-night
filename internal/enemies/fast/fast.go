package fast

import (
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/pechorka/illuminate-game-jam/internal/enemies"
	"github.com/pechorka/illuminate-game-jam/pkg/rlutils"
)

const (
	initialSpeed  = 3
	initialHealth = 10
	initialDamage = 5
)

type Enemy struct {
	ID  int
	Pos rl.Vector2

	Speed  float32
	Health float32
	Damage float32

	Texture rl.Texture2D

	initialHealth float32
	initialSpeed  float32
}

func FromPos(pos rl.Vector2, texture rl.Texture2D) *Enemy {
	return &Enemy{
		ID:  rand.Int(),
		Pos: pos,

		Speed:  initialSpeed,
		Health: initialHealth,
		Damage: initialDamage,

		Texture: texture,

		initialHealth: initialHealth,
		initialSpeed:  initialSpeed,
	}
}

func (e *Enemy) GetID() int {
	return e.ID
}

func (e *Enemy) IsDead() bool {
	return e.Health <= 0
}

func (e *Enemy) Reward() int {
	return enemies.Reward(e.initialHealth, e.initialSpeed)
}

func (e *Enemy) MoveTowards(pos rl.Vector2) rl.Vector2 {
	dir := rl.Vector2Subtract(pos, e.Pos)
	dir = rl.Vector2Normalize(dir)
	dir = rl.Vector2Scale(dir, e.Speed)

	return rl.Vector2Add(e.Pos, dir)
}

func (e *Enemy) MoveAway(pos rl.Vector2) rl.Vector2 {
	dir := rl.Vector2Subtract(e.Pos, pos)
	dir = rl.Vector2Normalize(dir)
	dir = rl.Vector2Scale(dir, e.Speed)

	return rl.Vector2Add(e.Pos, dir)
}

func (e *Enemy) GetPos() rl.Vector2 {
	return e.Pos
}

func (e *Enemy) UpdatePosition(pos rl.Vector2) {
	e.Pos = pos
}

func (e *Enemy) Draw() {
	rl.DrawTexture(e.Texture, int32(e.Pos.X), int32(e.Pos.Y), rl.White)

	name := "fast"
	rl.DrawText(name, int32(e.Pos.X), int32(e.Pos.Y-10), 10, rl.Red)
}

func (e *Enemy) DealDamage() float32 {
	return e.Damage
}

func (e *Enemy) TakeDamage(damage float32) {
	e.Health -= damage
}

func (e *Enemy) Boundaries() rl.Rectangle {
	return rlutils.TextureBoundaries(e.Texture, e.Pos)
}
