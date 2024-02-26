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
	centerLabelFontSize = int32(50)
	centerLabelColor    = rl.White
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
			soldier:      loadTextureFromImage("assets/soldier.png"),
			soldierMelee: loadTextureFromImage("assets/soldier_melee.png"),
			enemy:        loadTextureFromImage("assets/enemy.png"),
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
	soldier      rl.Texture2D
	soldierMelee rl.Texture2D

	enemy rl.Texture2D
}

type gameScreen int

const (
	gameScreenMainMenu gameScreen = iota
	gameScreenSpawnSoldier
	gameScreenGame
	gameScreenOver
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
		rlutils.DrawTextAtCenterOfScreen("Paused", screenWidth, screenHeight, centerLabelFontSize, centerLabelColor)
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
	case gameScreenOver:
		gs.renderGameOver()
	}
}

func (gs *gameState) renderSoldierSpawn() {
	if rl.IsKeyPressed(rl.KeyEnter) {
		gs.gameScreen = gameScreenGame
		return
	}

	renderHelpLabels(
		"Spawn soldiers by clicking",
		"Press enter to start game",
	)

	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		mousePos := rl.GetMousePosition()
		newSoldier := soldier.FromPos(mousePos, gs.assets.soldier, gs.assets.soldierMelee)
		gs.soldiers = append(gs.soldiers, newSoldier)
	}

	gs.renderSoldiers()
}

func (gs *gameState) renderGame() {
	if len(gs.soldiers) == 0 {
		gs.gameScreen = gameScreenOver
		return
	}

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

	gs.cleanupDeadEnemies()
	gs.spawnEnemies()
	gs.moveEnemies()
	gs.renderEnemies()

	gs.cleanupDeadSoldiers()
	gs.moveSoldiers()
	gs.renderSoldiers()

	gs.dimFlares()
}

func (gs *gameState) renderFlares() {
	for _, f := range gs.flares {
		f.Draw()
		gs.quadtree.Insert(f.ID, f.Boundaries(), f)
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

func (gs *gameState) cleanupDeadEnemies() {
	aliveEnemies := make([]*enemy.Enemy, 0, len(gs.enemies))
	for _, e := range gs.enemies {
		if e.Health > 0 {
			aliveEnemies = append(aliveEnemies, e)
		}
	}
	gs.enemies = aliveEnemies
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
		e.State = enemy.Walking

		enemyBoundaries := rlutils.TextureBoundaries(e.Texture, newPosition)
		// soldiers didn't move yet, so we can use previous quadtree
		collissions := gs.prevQuadtree.Query(enemyBoundaries)

		for _, c := range collissions {
			if soldier, ok := c.Value.(*soldier.Soldier); ok {
				soldier.Health -= e.Attack
				e.State = enemy.Melee
				newPosition = e.Pos // don't move if collission
			}
		}
		// TODO: if found collission with flare in new quadtree, try to find another soldier
		e.Pos = newPosition
		gs.quadtree.Insert(e.ID, enemyBoundaries, e)
	}
}

func (gs *gameState) renderEnemies() {
	for _, e := range gs.enemies {
		e.Draw()
	}
}

func (gs *gameState) cleanupDeadSoldiers() {
	aliveSoldiers := make([]*soldier.Soldier, 0, len(gs.soldiers))
	for _, s := range gs.soldiers {
		if s.Health > 0 {
			aliveSoldiers = append(aliveSoldiers, s)
		}
	}
	gs.soldiers = aliveSoldiers
}

func (gs *gameState) moveSoldiers() {
	for _, s := range gs.soldiers {
		nearestFlare := gs.findNearestFlare(s.Pos)
		newPosition := s.Pos
		if nearestFlare != nil {
			newPosition = s.TryMoveTowards(nearestFlare.Pos)
		}
		s.State = soldier.Walking

		soldierBoundaries := rlutils.TextureBoundaries(s.Walking, newPosition)
		collissions := gs.quadtree.Query(soldierBoundaries)

		for _, c := range collissions {
			switch val := c.Value.(type) {
			case *soldier.Soldier:
				newPosition = s.Pos // don't move if collission
				// TODO: check if soldier is dead
			case *enemy.Enemy:
				newPosition = s.Pos // don't move if collission
				s.State = soldier.Melee
				val.Health -= s.Attack
			}
		}
		s.Pos = newPosition

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

func (gs *gameState) renderGameOver() {
	if rl.IsKeyPressed(rl.KeyEnter) {
		gs.gameScreen = gameScreenSpawnSoldier
		gs.soldiers = nil
		gs.enemies = nil
		gs.flares = nil
		return
	}

	renderHelpLabels(
		"Press enter to restart",
	)

	rlutils.DrawTextAtCenterOfScreen("Game Over", screenWidth, screenHeight, centerLabelFontSize, centerLabelColor)
}

func renderHelpLabels(labels ...string) {
	x := int32(helpLabelInitialPos.X)
	y := int32(helpLabelInitialPos.Y)

	for i := int32(0); i < int32(len(labels)); i++ {
		spacing := helpLabelSpacing * i
		rl.DrawText(labels[i], x, y+spacing, helpLabelFontSize, helpLabelColor)
	}
}
