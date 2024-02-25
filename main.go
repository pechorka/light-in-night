package main

import (
	"math"
	"math/rand"

	"github.com/pechorka/illuminate-game-jam/internal/enemy"
	"github.com/pechorka/illuminate-game-jam/internal/flare"
	"github.com/pechorka/illuminate-game-jam/internal/soldier"
	"github.com/pechorka/illuminate-game-jam/pkg/data_structures/quadtree"
	"github.com/pechorka/illuminate-game-jam/pkg/rlutils"

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
			enemy:   loadTextureFromImage("assets/enemy.png"),
		},
		prevQuadtree: quadtree.NewQuadtree(quadtreeBounds, quadtreeCapacity),
		quadtree:     quadtree.NewQuadtree(quadtreeBounds, quadtreeCapacity),

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
	enemy   rl.Texture2D
}

type gameScreen int

const (
	gameScreenMainMenu gameScreen = iota
	gameScreenSpawnSoldier
	gameScreenGame
)

type gameState struct {
	assets       *gameAssets
	prevQuadtree *quadtree.Quadtree
	quadtree     *quadtree.Quadtree
	flares       []*flare.Flare
	soldiers     []*soldier.Soldier
	enemies      []*enemy.Enemy

	gameScreen      gameScreen
	paused          bool
	enemeSpawnedAgo float32
}

func (gs *gameState) renderFrame() {
	if rl.IsKeyPressed(rl.KeySpace) {
		gs.paused = !gs.paused
	}

	if gs.paused {
		return
	}

	gs.prevQuadtree, gs.quadtree = gs.quadtree, gs.prevQuadtree
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

	gs.spawnEnemies()
	gs.moveEnemies()
	gs.renderEnemies()

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

// TODO: replace with KD-tree
func (gs *gameState) findNearestFlare(pos rl.Vector2) *flare.Flare {
	var nearestFlare *flare.Flare
	var nearestDist float32 = math.MaxFloat32
	for _, f := range gs.flares {
		dist := rl.Vector2Distance(pos, f.Pos)
		if dist < nearestDist {
			nearestFlare = f
			nearestDist = dist
		}
	}
	return nearestFlare
}

func (gs *gameState) spawnEnemies() {
	gs.enemeSpawnedAgo += rl.GetFrameTime()
	if gs.enemeSpawnedAgo < 1 {
		return
	}

	gs.enemeSpawnedAgo = 0

	newEnemyPos := gs.findPositionForEnemy()
	newEnemy := enemy.FromPos(newEnemyPos, gs.assets.enemy)
	gs.enemies = append(gs.enemies, newEnemy)
}

func (gs *gameState) findPositionForEnemy() rl.Vector2 {
	for {
		pos := rl.Vector2{
			X: float32(rand.Intn(screenWidth)),
			Y: float32(rand.Intn(screenHeight)),
		}

		boundaries := rlutils.TextureBoundaries(gs.assets.enemy, pos)

		collissions := gs.prevQuadtree.Query(boundaries)
		if len(collissions) == 0 {
			return pos
		}
	}
}

func (gs *gameState) moveEnemies() {
	for _, e := range gs.enemies {
		nearestSoldier := gs.findNearestSoldier(e.Pos)
		newPosition := e.MoveTowards(nearestSoldier.Pos)
		enemyBoundaries := rlutils.TextureBoundaries(e.Texture, newPosition)
		// TODO: if found collission with flare in new quadtree, try to find another soldier
		e.MoveTo(newPosition)
		gs.quadtree.Insert(e.ID, enemyBoundaries, e)
	}
}

func (gs *gameState) renderEnemies() {
	for _, e := range gs.enemies {
		e.Draw()
	}
}

func (gs *gameState) moveSoldiers() {
	if len(gs.flares) == 0 {
		return
	}

	for _, s := range gs.soldiers {
		nearestFlare := gs.findNearestFlare(s.Pos)
		newPosition := s.TryMoveTowards(nearestFlare.Pos)

		soldierBoundaries := rlutils.TextureBoundaries(s.Texture, newPosition)
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

// TODO: replace with KD-tree
func (gs *gameState) findNearestSoldier(pos rl.Vector2) *soldier.Soldier {
	var nearestSoldier *soldier.Soldier
	var nearestDist float32 = math.MaxFloat32
	for _, s := range gs.soldiers {
		dist := rl.Vector2Distance(pos, s.Pos)
		if dist < nearestDist {
			nearestSoldier = s
			nearestDist = dist
		}
	}
	return nearestSoldier
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
