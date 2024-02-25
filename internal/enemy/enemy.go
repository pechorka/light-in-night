package enemy

import (
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Enemy struct {
	ID      int
	Pos     rl.Vector2
	Texture rl.Texture2D
}

func FromPos(pos rl.Vector2, texture rl.Texture2D) *Enemy {
	return &Enemy{
		ID:      rand.Int(),
		Pos:     pos,
		Texture: texture,
	}
}

func (e *Enemy) Draw() {
	rl.DrawTexture(e.Texture, int32(e.Pos.X), int32(e.Pos.Y), rl.White)
}

func (e *Enemy) MoveTowards(pos rl.Vector2) rl.Vector2 {
	dir := rl.Vector2Subtract(pos, e.Pos)
	dir = rl.Vector2Normalize(dir)
	dir = rl.Vector2Scale(dir, 1)

	return rl.Vector2Add(e.Pos, dir)
}

func (e *Enemy) MoveTo(pos rl.Vector2) {
	e.Pos = pos
}
