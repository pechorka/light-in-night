package soldier

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

type Soldier struct {
	X, Y    int32
	Texture rl.Texture2D
}

func FromPos(pos rl.Vector2, texture rl.Texture2D) *Soldier {
	return &Soldier{
		X:       int32(pos.X),
		Y:       int32(pos.Y),
		Texture: texture,
	}
}

func (s *Soldier) Draw() {
	rl.DrawTexture(s.Texture, s.X, s.Y, rl.White)
}
