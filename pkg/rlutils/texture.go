package rlutils

import rl "github.com/gen2brain/raylib-go/raylib"

func TextureBoundaries(texture rl.Texture2D, pos rl.Vector2) rl.Rectangle {
	return rl.Rectangle{
		X:      pos.X,
		Y:      pos.Y,
		Width:  float32(texture.Width),
		Height: float32(texture.Height),
	}
}
