package enemy

import (
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	initialSpeed  = 1
	initialHealth = 10
	initialAttack = 10
	initialState  = Walking
)

type State int

const (
	Walking State = iota + 1
	Melee
)

type Enemy struct {
	ID    int
	Pos   rl.Vector2
	State State

	Speed  float32
	Health int
	Attack int

	Texture rl.Texture2D
}

func FromPos(pos rl.Vector2, texture rl.Texture2D) *Enemy {
	return &Enemy{
		ID:    rand.Int(),
		Pos:   pos,
		State: initialState,

		Speed:  initialSpeed,
		Health: initialHealth,
		Attack: initialAttack,

		Texture: texture,
	}
}

func (e *Enemy) Draw() {
	// TODO: draw different texture based on state
	rl.DrawTexture(e.Texture, int32(e.Pos.X), int32(e.Pos.Y), rl.White)
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
