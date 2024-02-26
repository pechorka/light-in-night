package soldier

import (
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	initialSpeed  = 2
	initialHealth = 100
	initialAttack = 10
	initialState  = Walking
)

type State int

const (
	Walking State = iota + 1
	Melee
)

type Soldier struct {
	ID    int
	Pos   rl.Vector2
	State State

	Speed  float32
	Health int
	Attack int

	Walking rl.Texture2D
	Melee   rl.Texture2D
}

func FromPos(pos rl.Vector2, walking, melee rl.Texture2D) *Soldier {
	return &Soldier{
		ID:    rand.Int(),
		Pos:   pos,
		State: initialState,

		Speed:  initialSpeed,
		Health: initialHealth,
		Attack: initialAttack,

		Walking: walking,
		Melee:   melee,
	}
}

func (s *Soldier) Draw() {
	texture := s.Walking
	switch s.State {
	case Walking:
		texture = s.Walking
	case Melee:
		texture = s.Melee
	}
	// draw health bar above soldier
	// +2 and +1 are to make the health bar look better
	rl.DrawRectangleLines(int32(s.Pos.X), int32(s.Pos.Y-10), initialHealth+2, 10, rl.White)
	rl.DrawRectangle(int32(s.Pos.X+1), int32(s.Pos.Y-9), int32(s.Health), 8, rl.Green)

	rl.DrawTexture(texture, int32(s.Pos.X), int32(s.Pos.Y), rl.White)
}

func (s *Soldier) TryMoveTowards(pos rl.Vector2) rl.Vector2 {
	dir := rl.Vector2Subtract(pos, s.Pos)
	dir = rl.Vector2Normalize(dir)
	dir = rl.Vector2Scale(dir, s.Speed)

	return rl.Vector2Add(s.Pos, dir)
}
