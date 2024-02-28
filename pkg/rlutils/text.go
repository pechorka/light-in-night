package rlutils

import rl "github.com/gen2brain/raylib-go/raylib"

func DrawTextAtCenterOfRectangle(text string, rec rl.Rectangle, fontSize int32, color rl.Color) {
	textWidth := rl.MeasureText(text, fontSize)

	textX := rec.X + (rec.Width-float32(textWidth))/2
	textY := rec.Y + (rec.Height-float32(fontSize))/2

	rl.DrawText(text, int32(textX), int32(textY), fontSize, color)
}
