package soldier

import (
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	Speed = float32(1)
)

type Soldier struct {
	ID      int
	Pos     rl.Vector2
	Texture rl.Texture2D
}

func FromPos(pos rl.Vector2, texture rl.Texture2D) *Soldier {
	return &Soldier{
		ID:      rand.Int(),
		Pos:     pos,
		Texture: texture,
	}
}

func (s *Soldier) Draw() {
	rl.DrawTexture(s.Texture, int32(s.Pos.X), int32(s.Pos.Y), rl.White)
}

func (s *Soldier) TryMoveTowards(pos rl.Vector2) rl.Vector2 {
	dir := rl.Vector2Subtract(pos, s.Pos)
	dir = rl.Vector2Normalize(dir)
	dir = rl.Vector2Scale(dir, Speed)

	return rl.Vector2Add(s.Pos, dir)
}

func (s *Soldier) MoveTo(pos rl.Vector2) {
	s.Pos = pos
}
