package main

import (
	"embed"
	"math"
	"math/rand"
	"strconv"
	"strings"

	"github.com/pechorka/illuminate-game-jam/internal/enemy"
	"github.com/pechorka/illuminate-game-jam/internal/flare"
	"github.com/pechorka/illuminate-game-jam/internal/projectile"
	"github.com/pechorka/illuminate-game-jam/internal/soldier"
	"github.com/pechorka/illuminate-game-jam/pkg/data_structures/quadtree"
	"github.com/pechorka/illuminate-game-jam/pkg/rlutils"

	rl "github.com/gen2brain/raylib-go/raylib"
)

//go:embed assets
var assets embed.FS

const (
	screenWidth  = 800
	screenHeight = 600
)

var (
	headerBoundaries = rl.Rectangle{
		X:      0,
		Y:      0,
		Width:  screenWidth,
		Height: screenHeight * 0.05,
	}
	footerBoundaries = rl.Rectangle{
		X:      0,
		Y:      screenHeight * 0.85,
		Width:  screenWidth,
		Height: screenHeight * 0.15,
	}
)

var (
	arenaWidth  float32 = screenWidth
	arenaHeight float32 = screenHeight - headerBoundaries.Height - footerBoundaries.Height
)

var arenaBoundaries = rl.Rectangle{
	X:      0,
	Y:      headerBoundaries.Height,
	Width:  arenaWidth,
	Height: arenaHeight,
}

const (
	quadtreeCapacity = 10
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

var (
	enemySpawnRate = float32(1)
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
		prevQuadtree: quadtree.NewQuadtree(arenaBoundaries, quadtreeCapacity),
		quadtree:     quadtree.NewQuadtree(arenaBoundaries, quadtreeCapacity),

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
	file, err := assets.ReadFile(imgPath)
	if err != nil {
		panic(err)
	}
	fileExtention := imgPath[strings.LastIndexByte(imgPath, '.'):]
	img := rl.LoadImageFromMemory(fileExtention, file, int32(len(file)))
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
	// TODO: add leaderboard screen
)

type gameState struct {
	assets       *gameAssets
	prevQuadtree *quadtree.Quadtree
	quadtree     *quadtree.Quadtree
	flares       []*flare.Flare
	soldiers     []*soldier.Soldier
	enemies      []*enemy.Enemy
	projectiles  []*projectile.Projectile

	gameScreen      gameScreen
	paused          bool
	enemeSpawnedAgo float32
	// TODO: add money
	score int

	gameTime float32
}

func (gs *gameState) renderFrame() {
	if rl.IsKeyPressed(rl.KeySpace) {
		gs.paused = !gs.paused
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
	// TODO: soldiers should be bought in shop
	// TODO: should have initial amount of money
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

	if !gs.paused {
		gs.gameTime += rl.GetFrameTime()

		gs.addFlare()
		gs.dimFlares()

		gs.moveProjectiles()

		gs.cleanupDeadEnemies()
		gs.spawnEnemies()
		flaredEnemies := gs.moveEnemies()

		gs.cleanupDeadSoldiers()
		gs.moveSoldiers(flaredEnemies)
	}

	gs.renderHeader()
	gs.renderFooter()
	gs.renderFlares()
	gs.renderProjectiles()
	gs.renderEnemies()
	gs.renderSoldiers()
}

func (gs *gameState) renderHeader() {
	rl.DrawRectangleRec(headerBoundaries, rl.Gray)
	// Render header data. Example:
	// Time: 1:15 Score: 100 Money: 100

	time := int(gs.gameTime)
	minutes := time / 60
	seconds := time % 60
	timeText := "Time: " + strconv.Itoa(minutes) + ":" + strconv.Itoa(seconds)
	scoreText := "Score: " + strconv.Itoa(gs.score)

	timeTextWidth := rl.MeasureText(timeText, 20)

	widthOffset := int32(10)
	rl.DrawText(timeText, widthOffset, 10, 20, rl.White)
	widthOffset += timeTextWidth + 10
	// TODO: score is flickering depending on the time
	rl.DrawText(scoreText, widthOffset, 10, 20, rl.White)
	// TODO: add money

	if gs.paused {
		rl.DrawText("Paused", int32(headerBoundaries.Width-100), 10, 20, rl.White)
	}
}

func (gs *gameState) renderFooter() {
	rl.DrawRectangleRec(footerBoundaries, rl.Gray)
	// footer should have shop items in squares
	// each square should have price and icon
	// when hovered - show description
	type shopItem struct {
		price       int
		name        string
		description string
		icon        rl.Texture2D
	}

	// TODO: shop can have items to increase difficulty, but with higher rewards
	// for example: princess that needs to be protected for some time, but gives a lot of money
	mockItems := []shopItem{
		{price: 10, name: "Soldier", icon: gs.assets.soldier, description: "Basic soldier"},
		{price: 20, name: "Flare", icon: gs.assets.soldier, description: "Reveals enemies"},
		{price: 30, name: "Turret", icon: gs.assets.soldier, description: "Shoots enemies"},
		{price: 40, name: "Princess", icon: gs.assets.soldier, description: "Needs to be protected"},
		{price: 50, name: "Dragon", icon: gs.assets.soldier, description: "Does a lot of damage"},
	}
	itemWidth := float32(100)
	itemHeight := footerBoundaries.Height - 20 // 10px margin on each side
	for i, item := range mockItems {
		itemBoundaries := rl.Rectangle{
			X:      footerBoundaries.X + 10 + float32(i)*(itemWidth+10),
			Y:      footerBoundaries.Y + 10,
			Width:  itemWidth,
			Height: itemHeight,
		}
		rl.DrawRectangleLinesEx(itemBoundaries, 2, rl.White)
		// draw price in top left corner and icon in center
		priceText := strconv.Itoa(item.price)
		priceTextPos := rl.Vector2{
			X: itemBoundaries.X + 10,
			Y: itemBoundaries.Y + 10,
		}
		rl.DrawText(priceText, int32(priceTextPos.X), int32(priceTextPos.Y), 20, rl.White)

		iconPos := rl.Vector2{
			X: itemBoundaries.X + itemBoundaries.Width/2 - float32(item.icon.Width/2),
			Y: itemBoundaries.Y + itemBoundaries.Height/2 - float32(item.icon.Height/2),
		}
		rl.DrawTextureV(item.icon, iconPos, rl.White)

		// draw name in botton left corner of the item
		namePos := rl.Vector2{
			X: itemBoundaries.X + 5,
			Y: itemBoundaries.Y + itemBoundaries.Height - 25,
		}
		rl.DrawText(item.name, int32(namePos.X), int32(namePos.Y), 20, rl.White)

		// display description when hovered
		if rl.CheckCollisionPointRec(rl.GetMousePosition(), itemBoundaries) {
			// description should be displayed above the item with border
			descriptionBoundaries := rl.Rectangle{
				X:      footerBoundaries.X + 10,
				Y:      footerBoundaries.Y - 10 - 100,
				Width:  screenWidth - 20,
				Height: 100,
			}
			rl.DrawRectangleLinesEx(descriptionBoundaries, 2, rl.White)
			rl.DrawText(item.description, int32(descriptionBoundaries.X+5), int32(descriptionBoundaries.Y+5), 20, rl.White)
		}
	}
}

func (gs *gameState) addFlare() {
	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		mousePos := rl.GetMousePosition()
		if !rl.CheckCollisionPointRec(mousePos, arenaBoundaries) {
			return
		}
		newFlare := flare.FromPos(mousePos, flareRadius, flareCenterColor, flareEdgeColor)
		gs.flares = append(gs.flares, newFlare)
	}
}

func (gs *gameState) renderFlares() {
	for _, f := range gs.flares {
		f.Draw()
	}
}

func (gs *gameState) dimFlares() {
	wentOutCount := 0
	for _, f := range gs.flares {
		f.Dim()
		if f.WentOut() {
			wentOutCount++
			continue
		}
		gs.quadtree.Insert(f.ID, f.Boundaries(), f)
	}

	if wentOutCount > 0 {
		gs.flares = gs.flares[wentOutCount:]
	}
}

func (gs *gameState) moveProjectiles() {
	activeProjectiles := gs.projectiles[:0]
	for _, p := range gs.projectiles {
		p.Move()
		if p.Pos.X < 0 || p.Pos.X > arenaWidth || p.Pos.Y < 0 || p.Pos.Y > arenaHeight {
			continue
		}
		gs.quadtree.Insert(p.ID, p.Boundaries(), p)
		activeProjectiles = append(activeProjectiles, p)
	}
	gs.projectiles = activeProjectiles
}

func (gs *gameState) renderProjectiles() {
	for _, p := range gs.projectiles {
		p.Draw()
	}
}

func (gs *gameState) cleanupDeadEnemies() {
	aliveEnemies := gs.enemies[:0]
	for _, e := range gs.enemies {
		if e.Health == 0 {
			gs.score += e.Reward
			continue
		}
		aliveEnemies = append(aliveEnemies, e)
	}
	gs.enemies = aliveEnemies
}

func (gs *gameState) spawnEnemies() {
	gs.enemeSpawnedAgo += rl.GetFrameTime()
	if gs.enemeSpawnedAgo < enemySpawnRate {
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
			X: float32(rand.Intn(int(arenaWidth))),
			Y: float32(rand.Intn(int(arenaHeight))),
		}

		boundaries := rlutils.TextureBoundaries(gs.assets.enemy, pos)

		collissions := gs.prevQuadtree.Query(boundaries)
		if len(collissions) == 0 {
			return pos
		}
	}
}

func (gs *gameState) moveEnemies() []*enemy.Enemy {
	flaredEnemies := make([]*enemy.Enemy, 0, len(gs.enemies)/3)
	for _, e := range gs.enemies {
		nearestSoldier := findNearest(gs.soldiers, e.Pos)
		newPosition := e.MoveTowards(nearestSoldier.Pos)
		e.State = enemy.Walking

		enemyBoundaries := rlutils.TextureBoundaries(e.Texture, newPosition)
		// soldiers didn't move yet, so we can use previous quadtree
		soldierCollissions := gs.prevQuadtree.Query(enemyBoundaries)

		for _, c := range soldierCollissions {
			if soldier, ok := c.Value.(*soldier.Soldier); ok {
				soldier.Health -= e.Damage
				e.State = enemy.Melee
				newPosition = e.Pos // don't move if collission
			}
		}

		collissions := gs.quadtree.Query(enemyBoundaries)
		for _, c := range collissions {
			switch val := c.Value.(type) {
			case *flare.Flare:
				flaredEnemies = append(flaredEnemies, e)
				// Try to move away from flare
				e.State = enemy.Walking
				newPosition = e.MoveAway(val.Pos)
			case *projectile.Projectile:
				e.Health -= val.Damage
			}
		}

		e.Pos = newPosition
		gs.quadtree.Insert(e.ID, enemyBoundaries, e)
	}

	return flaredEnemies
}

func (gs *gameState) renderEnemies() {
	for _, e := range gs.enemies {
		e.Draw()
	}
}

func (gs *gameState) cleanupDeadSoldiers() {
	aliveSoldiers := gs.soldiers[:0]
	for _, s := range gs.soldiers {
		if s.Health > 0 {
			aliveSoldiers = append(aliveSoldiers, s)
		}
	}
	gs.soldiers = aliveSoldiers
}

func (gs *gameState) moveSoldiers(flaredEnemies []*enemy.Enemy) {
	for _, s := range gs.soldiers {
		s.ProgressTime(rl.GetFrameTime())

		s.State = soldier.Standing

		soldierBoundaries := rlutils.TextureBoundaries(s.Walking, s.Pos)
		collissions := gs.quadtree.Query(soldierBoundaries)

		for _, c := range collissions {
			// TODO: if soldier is inside of flare - blind him
			switch val := c.Value.(type) {
			case *enemy.Enemy:
				s.State = soldier.Melee
				val.Health -= s.Damage
			}
		}

		if s.State == soldier.Standing {
			// try to find shooting target
			nearestEnemy := findNearest(flaredEnemies, s.Pos)
			if nearestEnemy != nil &&
				s.WithinShootingRange(nearestEnemy.Pos) &&
				s.CanShoot() {
				s.Shoot()
				s.State = soldier.Shooting
				// spawn projectile
				projectileVelocity := rl.Vector2Subtract(nearestEnemy.Pos, s.Pos)
				newProjectile := projectile.FromPos(s.Pos, projectileVelocity)
				gs.projectiles = append(gs.projectiles, newProjectile)
			}
		}

		gs.quadtree.Insert(s.ID, soldierBoundaries, s)
	}
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
		gs.projectiles = nil
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

type positionable interface {
	GetPos() rl.Vector2
}

// TODO: replace with KD-tree
func findNearest[T positionable](items []T, pos rl.Vector2) T {
	var nearest T
	var nearestDist float32 = math.MaxFloat32
	for _, i := range items {
		dist := rl.Vector2Distance(pos, i.GetPos())
		if dist < nearestDist {
			nearest = i
			nearestDist = dist
		}
	}
	return nearest
}
