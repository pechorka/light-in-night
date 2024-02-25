package main

import (
	"github.com/pechorka/illuminate-game-jam/pkg/components/flare"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	screenWidth  = 800
	screenHeight = 600
)

var (
	flareCenterColor = rl.Color{R: 255, G: 0, B: 0, A: 127}
	flareEdgeColor   = rl.Color{R: 255, G: 127, B: 0, A: 127}
	flareRadius      = float32(50)
)

func main() {
	rl.InitWindow(screenWidth, screenHeight, "Click to boom!")
	rl.SetTargetFPS(60)

	gs := &gameState{}

	for !rl.WindowShouldClose() {
		rl.BeginDrawing()
		rl.ClearBackground(rl.Black)

		gs.renderFrame()

		rl.EndDrawing()
	}

	rl.CloseWindow()
}

type gameState struct {
	flares []*flare.Flare
	paused bool
}

func (gs *gameState) renderFrame() {
	if rl.IsKeyPressed(rl.KeySpace) {
		gs.paused = !gs.paused
	}

	if gs.paused {
		return
	}

	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		mousePos := rl.GetMousePosition()
		newFlare := flare.FromPos(mousePos, flareRadius, flareCenterColor, flareEdgeColor)
		gs.flares = append(gs.flares, newFlare)
	}

	gs.renderFlares()
}

func (gs *gameState) renderFlares() {
	wentOutCount := 0
	for _, f := range gs.flares {
		f.Draw()
		f.Dim()
		if f.WentOut() {
			wentOutCount++
		}
	}

	if wentOutCount > 0 {
		gs.flares = gs.flares[wentOutCount:]
	}
}
