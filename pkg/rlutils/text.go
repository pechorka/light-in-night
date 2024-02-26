package rlutils

import rl "github.com/gen2brain/raylib-go/raylib"

func DrawTextAtCenterOfScreen(text string, width, height, fontSize int32, color rl.Color) {
	textWidth := rl.MeasureText(text, fontSize)
	x := width/2 - textWidth/2
	y := height / 2
	rl.DrawText(text, x, y, fontSize, color)
}
