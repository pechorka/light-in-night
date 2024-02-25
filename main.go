package main

import (
	"math"

	"github.com/pechorka/illuminate-game-jam/pkg/components/flare"
	"github.com/pechorka/illuminate-game-jam/pkg/components/soldier"
	"github.com/pechorka/illuminate-game-jam/pkg/data_structures/quadtree"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	screenWidth  = 800
	screenHeight = 600
)

var (
	quadtreeCapacity = 10
	quadtreeBounds   = rl.Rectangle{
		X: 0, Y: 0,
		Width:  float32(screenWidth),
		Height: float32(screenHeight),
	}
)

var (
	helpLabelInitialPos = rl.Vector2{X: 10, Y: 10}
	helpLabelSpacing    = int32(30)
	helpLabelFontSize   = int32(20)
	helpLabelColor      = rl.White
)

var (
	flareCenterColor = rl.Color{R: 255, G: 0, B: 0, A: 127}
	flareEdgeColor   = rl.Color{R: 255, G: 127, B: 0, A: 127}
	flareRadius      = float32(50)
)

func main() {
	rl.InitWindow(screenWidth, screenHeight, "Click to survive!")
	rl.SetTargetFPS(60)

	gs := &gameState{
		assets: &gameAssets{
			soldier: loadTextureFromImage("assets/soldier.png"),
		},
		quadtree: quadtree.NewQuadtree(quadtreeBounds, quadtreeCapacity),

		gameScreen: gameScreenSpawnSoldier,
	}

	for !rl.WindowShouldClose() {
		rl.BeginDrawing()

		gs.renderFrame()

		rl.EndDrawing()
	}

	rl.CloseWindow()
}

func loadTextureFromImage(imgPath string) rl.Texture2D {
	img := rl.LoadImage(imgPath)
	defer rl.UnloadImage(img)
	return rl.LoadTextureFromImage(img)
}

type gameAssets struct {
	soldier rl.Texture2D
}

type gameScreen int

const (
	gameScreenMainMenu gameScreen = iota
	gameScreenSpawnSoldier
	gameScreenGame
)

type gameState struct {
	assets   *gameAssets
	quadtree *quadtree.Quadtree
	flares   []*flare.Flare
	soldiers []*soldier.Soldier

	gameScreen gameScreen
	paused     bool
}

func (gs *gameState) renderFrame() {
	if rl.IsKeyPressed(rl.KeySpace) {
		gs.paused = !gs.paused
	}

	if gs.paused {
		return
	}

	gs.quadtree.Clear()
	rl.ClearBackground(rl.Black)

	switch gs.gameScreen {
	case gameScreenSpawnSoldier:
		gs.renderSoldierSpawn()
	case gameScreenGame:
		gs.renderGame()
	}
}

func (gs *gameState) renderSoldierSpawn() {
	renderHelpLabels(
		"Spawn soldiers by clicking",
		"Press enter to start game",
	)

	if rl.IsKeyPressed(rl.KeyEnter) {
		gs.gameScreen = gameScreenGame
		return
	}

	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		mousePos := rl.GetMousePosition()
		newSoldier := soldier.FromPos(mousePos, gs.assets.soldier)
		gs.soldiers = append(gs.soldiers, newSoldier)
	}

	gs.renderSoldiers()
}

func (gs *gameState) renderGame() {
	renderHelpLabels(
		"Click to place flares",
		"Press space to pause",
	)

	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		mousePos := rl.GetMousePosition()
		newFlare := flare.FromPos(mousePos, flareRadius, flareCenterColor, flareEdgeColor)
		gs.flares = append(gs.flares, newFlare)
	}

	gs.renderFlares()

	gs.moveSoldiers()
	gs.renderSoldiers()

	gs.dimFlares()
}

func (gs *gameState) renderFlares() {
	for _, f := range gs.flares {
		f.Draw()
		gs.quadtree.Insert(f.ID, f.Boundaries, f)
	}
}

func (gs *gameState) dimFlares() {
	wentOutCount := 0
	for _, f := range gs.flares {
		f.Dim()
		if f.WentOut() {
			wentOutCount++
		}
	}

	if wentOutCount > 0 {
		gs.flares = gs.flares[wentOutCount:]
	}
}

func (gs *gameState) moveSoldiers() {
	// TODO: replace with KD-tree
	findNearestFlare := func(s *soldier.Soldier) *flare.Flare {
		var nearestFlare *flare.Flare
		var nearestDist float32 = math.MaxFloat32
		soldierPos := s.Pos
		for _, f := range gs.flares {
			dist := rl.Vector2Distance(soldierPos, f.Pos)
			if dist < nearestDist {
				nearestFlare = f
				nearestDist = dist
			}
		}
		return nearestFlare
	}

	for _, s := range gs.soldiers {
		nearestFlare := findNearestFlare(s)
		newPosition := s.TryMoveTowards(nearestFlare.Pos)

		soldierBoundaries := s.GetBoundariesAt(newPosition)
		collissions := gs.quadtree.Query(soldierBoundaries)

		move := true
		for _, c := range collissions {
			switch c.Value.(type) {
			case *soldier.Soldier:
				move = false
				// TODO: check if soldier is dead
			case *flare.Flare:
				move = false
				// TODO: handle enemy collision
			}
		}

		if move {
			s.MoveTo(newPosition)
		}

		gs.quadtree.Insert(s.ID, soldierBoundaries, s)
	}
}

func (gs *gameState) renderSoldiers() {
	for _, s := range gs.soldiers {
		s.Draw()
	}
}

func renderHelpLabels(labels ...string) {
	x := int32(helpLabelInitialPos.X)
	y := int32(helpLabelInitialPos.Y)

	for i := int32(0); i < int32(len(labels)); i++ {
		spacing := helpLabelSpacing * i
		rl.DrawText(labels[i], x, y+spacing, helpLabelFontSize, helpLabelColor)
	}
}
