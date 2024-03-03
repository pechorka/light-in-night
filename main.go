package main

import (
	"embed"
	"math"
	"math/rand"
	"slices"
	"strconv"
	"strings"

	"github.com/pechorka/illuminate-game-jam/internal/consumables/flare"
	"github.com/pechorka/illuminate-game-jam/internal/consumables/grenade"
	"github.com/pechorka/illuminate-game-jam/internal/enemies/basic"
	"github.com/pechorka/illuminate-game-jam/internal/enemies/fast"
	"github.com/pechorka/illuminate-game-jam/internal/enemies/tank"
	"github.com/pechorka/illuminate-game-jam/internal/projectile"
	"github.com/pechorka/illuminate-game-jam/internal/soldier"
	"github.com/pechorka/illuminate-game-jam/pkg/data_structures/quadtree"
	"github.com/pechorka/illuminate-game-jam/pkg/rlutils"

	rl "github.com/gen2brain/raylib-go/raylib"
)

//go:embed assets
var assets embed.FS

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
	initialFlareCount   = 50
	initialGrenadeCount = 5
)

var (
	enemySpawnRate = float32(1)
)

func main() {

	gb := &gameBoundaries{
		screenWidth:  800,
		screenHeight: 600,
	}
	gb.updateBoundaries()

	rl.SetConfigFlags(rl.FlagWindowResizable)
	rl.InitWindow(int32(gb.screenWidth), int32(gb.screenHeight), "Click to survive!")
	rl.SetTargetFPS(60)

	gs := &gameState{
		boundaries: gb,
		assets: &gameAssets{
			soldier: loadTextureFromImage("assets/soldier.png"),
			enemy: &enemyAssets{
				// basic: loadTextureFromImage("assets/enemy/basic.png"),
				// fast:  loadTextureFromImage("assets/enemy/fast.png"),
				// tank:  loadTextureFromImage("assets/enemy/tank.png"),
				basic: loadTextureFromImage("assets/enemy.png"),
				fast:  loadTextureFromImage("assets/enemy.png"),
				tank:  loadTextureFromImage("assets/enemy.png"),
			},
		},
		prevQuadtree: quadtree.NewQuadtree(gb.arenaBoundaries, quadtreeCapacity),
		quadtree:     quadtree.NewQuadtree(gb.arenaBoundaries, quadtreeCapacity),

		itemStorage: &itemStorage{
			flareCount:   initialFlareCount,
			grenadeCount: initialGrenadeCount,
		},
		selectedConsumable: flares,

		gameScreen: gameScreenGame,
		paused:     true,
	}

	gs.soldiers = append(gs.soldiers, soldier.FromPos(rl.Vector2{
		X: gb.arenaBoundaries.Width / 2,
		Y: gb.arenaBoundaries.Height / 2,
	}, gs.assets.soldier))

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
	texture := rl.LoadTextureFromImage(img)

	return texture
}

type gameAssets struct {
	soldier rl.Texture2D

	enemy *enemyAssets
}

type enemyAssets struct {
	basic rl.Texture2D
	fast  rl.Texture2D
	tank  rl.Texture2D
}

type gameScreen int

const (
	gameScreenMainMenu gameScreen = iota
	gameScreenGame
	gameScreenOver
	// TODO: add leaderboard screen
)

type gameBoundaries struct {
	screenWidth  int
	screenHeight int

	screenBoundaries rl.Rectangle
	arenaBoundaries  rl.Rectangle
	headerBoundaries rl.Rectangle
	footerBoundaries rl.Rectangle
}

func (gb *gameBoundaries) update() bool {
	newWidth := rl.GetScreenWidth()
	newHeight := rl.GetScreenHeight()
	if newWidth == gb.screenWidth && newHeight == gb.screenHeight {
		return false
	}

	gb.screenWidth = newWidth
	gb.screenHeight = newHeight
	gb.updateBoundaries()

	return true
}

func (gb *gameBoundaries) updateBoundaries() {
	gb.screenBoundaries = rl.Rectangle{
		X:      0,
		Y:      0,
		Width:  float32(gb.screenWidth),
		Height: float32(gb.screenHeight),
	}

	gb.headerBoundaries = rl.Rectangle{
		X:      0,
		Y:      0,
		Width:  gb.screenBoundaries.Width,
		Height: gb.screenBoundaries.Height * 0.05,
	}

	gb.footerBoundaries = rl.Rectangle{
		X:      0,
		Y:      gb.screenBoundaries.Height * 0.85,
		Width:  gb.screenBoundaries.Width,
		Height: gb.screenBoundaries.Height * 0.15,
	}

	arenaHeight := gb.screenBoundaries.Height - gb.headerBoundaries.Height - gb.footerBoundaries.Height
	arenaWidth := gb.screenBoundaries.Width

	gb.arenaBoundaries = rl.Rectangle{
		X:      gb.headerBoundaries.X,
		Y:      gb.headerBoundaries.Height,
		Width:  arenaWidth,
		Height: arenaHeight,
	}
}

type enemy interface {
	GetID() int
	IsDead() bool
	Reward() int
	MoveTowards(rl.Vector2) rl.Vector2
	MoveAway(rl.Vector2) rl.Vector2
	GetPos() rl.Vector2
	UpdatePosition(rl.Vector2)
	Draw()
	DealDamage() float32
	TakeDamage(float32)
	Boundaries() rl.Rectangle
}

type gameState struct {
	boundaries   *gameBoundaries
	assets       *gameAssets
	prevQuadtree *quadtree.Quadtree
	quadtree     *quadtree.Quadtree
	flares       []*flare.Flare
	grenades     []*grenade.Grenade
	soldiers     []*soldier.Soldier
	enemies      []enemy
	projectiles  []*projectile.Projectile

	itemStorage        *itemStorage
	selectedConsumable consumable

	gameScreen      gameScreen
	paused          bool
	enemeSpawnedAgo float32
	// TODO: add money
	score int

	gameTime float32

	// draggingSoldier *soldier.Soldier
}

type consumable int

const (
	flares consumable = iota + 1
	grenades
)

type itemStorage struct {
	flareCount   int
	grenadeCount int
}

func (gs *gameState) renderFrame() {
	if rl.IsKeyPressed(rl.KeySpace) {
		gs.paused = !gs.paused
	}

	if gs.boundaries.update() {
		gs.prevQuadtree = quadtree.NewQuadtree(gs.boundaries.arenaBoundaries, quadtreeCapacity)
		gs.quadtree = quadtree.NewQuadtree(gs.boundaries.arenaBoundaries, quadtreeCapacity)
	}

	gs.prevQuadtree, gs.quadtree = gs.quadtree, gs.prevQuadtree
	gs.quadtree.Clear()
	rl.ClearBackground(rl.Black)

	switch gs.gameScreen {
	case gameScreenGame:
		gs.renderGame()
	case gameScreenOver:
		gs.renderGameOver()
	}
}

func (gs *gameState) renderGame() {
	if len(gs.soldiers) == 0 {
		gs.gameScreen = gameScreenOver
		return
	}

	if !gs.paused {
		gs.gameTime += rl.GetFrameTime()

		gs.useConsumable()
		gs.processFlares()
		gs.processGrenades()

		gs.processProjectiles()

		gs.cleanupDeadEnemies()
		gs.spawnEnemies()
		flaredEnemies := gs.processEnemies()

		gs.cleanupDeadSoldiers()
		gs.processSoldiers(flaredEnemies)
	}

	gs.renderHeader()
	gs.renderFooter()
	gs.selectConsumables()
	gs.renderConsumables()
	// gs.renderSoldierStorage()
	// gs.renderItemSelector()
	// gs.placeDraggedSoldier()
	// gs.renderDraggingSoldier()
	gs.renderFlares()
	gs.renderGrenades()
	gs.renderProjectiles()
	gs.renderEnemies()
	gs.renderSoldiers()
}

func (gs *gameState) renderHeader() {
	headerBoundaries := gs.boundaries.headerBoundaries
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
	footerBoundaries := gs.boundaries.footerBoundaries
	rl.DrawRectangleRec(footerBoundaries, rl.Gray)
	// footer should have shop items in squares
	// each square should have price and icon
	// when hovered - show description
	type shopItem struct {
		price       int
		count       int
		name        string
		description string
		icon        rl.Texture2D
		ctype       consumable
	}

	// TODO: shop can have items to increase difficulty, but with higher rewards
	// for example: princess that needs to be protected for some time, but gives a lot of money
	mockItems := []shopItem{
		{price: 10, count: 10, name: "Flare", description: "Reveals enemies", icon: gs.assets.soldier, ctype: flares},
		{price: 20, count: 1, name: "Grenade", description: "Deals damage", icon: gs.assets.soldier, ctype: grenades},
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
		// draw price in top left corner
		priceText := strconv.Itoa(item.price) + "$"
		priceTextPos := rl.Vector2{
			X: itemBoundaries.X + 10,
			Y: itemBoundaries.Y + 10,
		}
		rl.DrawText(priceText, int32(priceTextPos.X), int32(priceTextPos.Y), 20, rl.White)

		// draw count in top right corner
		countText := strconv.Itoa(item.count)
		countTextPos := rl.Vector2{
			X: itemBoundaries.X + itemBoundaries.Width - 10 - float32(rl.MeasureText(countText, 20)),
			Y: itemBoundaries.Y + 10,
		}
		rl.DrawText(countText, int32(countTextPos.X), int32(countTextPos.Y), 20, rl.White)

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
			gs.drawDescription(item.description)

			if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
				switch item.ctype {
				case flares:
					gs.itemStorage.flareCount += item.count
				case grenades:
					gs.itemStorage.grenadeCount += item.count
				}
			}
		}
	}
}

func (gs *gameState) selectConsumables() {
	if rl.IsKeyPressed(rl.KeyOne) {
		gs.selectedConsumable = flares
	}
	if rl.IsKeyPressed(rl.KeyTwo) {
		gs.selectedConsumable = grenades
	}
}

func (gs *gameState) renderConsumables() {
	// in right top corner bellow header
	var widths []int32
	flareText := "1 -> Flares: " + strconv.Itoa(gs.itemStorage.flareCount)
	widths = append(widths, rl.MeasureText(flareText, 20))

	grenadeText := "2 -> Grenades: " + strconv.Itoa(gs.itemStorage.grenadeCount)
	widths = append(widths, rl.MeasureText(grenadeText, 20))

	color := func(selected consumable) rl.Color {
		if gs.selectedConsumable == selected {
			return rl.Green
		}
		return rl.White
	}

	x := int32(gs.boundaries.screenWidth) - 10 - slices.Max(widths)
	y := int32(gs.boundaries.headerBoundaries.Y+gs.boundaries.headerBoundaries.Height) + 10
	rl.DrawText(flareText, x, y, 20, color(flares))
	y += 30
	rl.DrawText(grenadeText, x, y, 20, color(grenades))
}

// func (gs *gameState) renderSoldierStorage() {
// 	rl.DrawRectangleRec(soldierStorageBoundaries, rl.Blue)

// 	type soldierStorageItem struct {
// 		name string
// 		icon rl.Texture2D
// 		// TODO: add constructor for soldier
// 		count       int
// 		description string
// 	}

// 	mockItems := []*soldierStorageItem{
// 		{name: "Soldier", icon: gs.assets.soldier, count: 10, description: "Basic soldier"},
// 		{name: "Commander", icon: gs.assets.soldier, count: 5, description: "Increases damage"},
// 		{name: "Medic", icon: gs.assets.soldier, count: 3, description: "Heals soldiers"},
// 	}

// 	itemWidth := soldierStorageBoundaries.Width - 20 // 10px margin on each side
// 	itemHeight := float32(100)
// 	for i, item := range mockItems {
// 		itemBoundaries := rl.Rectangle{
// 			X:      soldierStorageBoundaries.X + 10,
// 			Y:      soldierStorageBoundaries.Y + float32(i)*(itemHeight+10),
// 			Width:  itemWidth,
// 			Height: itemHeight,
// 		}
// 		rl.DrawRectangleLinesEx(itemBoundaries, 2, rl.White)
// 		// draw count in top left corner and icon in center
// 		countText := strconv.Itoa(item.count)
// 		countTextPos := rl.Vector2{
// 			X: itemBoundaries.X + 10,
// 			Y: itemBoundaries.Y + 10,
// 		}
// 		rl.DrawText(countText, int32(countTextPos.X), int32(countTextPos.Y), 20, rl.White)

// 		iconPos := rl.Vector2{
// 			X: itemBoundaries.X + itemBoundaries.Width/2 - float32(item.icon.Width/2),
// 			Y: itemBoundaries.Y + itemBoundaries.Height/2 - float32(item.icon.Height/2),
// 		}
// 		rl.DrawTextureV(item.icon, iconPos, rl.White)

// 		// draw name in botton left corner of the item
// 		namePos := rl.Vector2{
// 			X: itemBoundaries.X + 5,
// 			Y: itemBoundaries.Y + itemBoundaries.Height - 25,
// 		}
// 		// TODO: adjust font size based on name length
// 		rl.DrawText(item.name, int32(namePos.X), int32(namePos.Y), 20, rl.White)

// 		// display description when hovered
// 		mousePos := rl.GetMousePosition()
// 		if rl.CheckCollisionPointRec(mousePos, itemBoundaries) {
// 			drawDescription(item.description)

// 			// TODO: move to separate function
// 			if rl.IsMouseButtonPressed(rl.MouseLeftButton) && item.count > 0 {
// 				gs.draggingSoldier = soldier.FromPos(mousePos, item.icon, item.icon)
// 				item.count--
// 			}
// 		}
// 	}
// }

// func (gs *gameState) renderItemSelector() {
// 	rl.DrawRectangleRec(itemStorageBoundaries, rl.Beige)

// 	type itemStorageItem struct {
// 		name string
// 		icon rl.Texture2D
// 		// TODO: add constructor for soldier
// 		count       int
// 		description string
// 		consumable  consumable
// 	}

// 	items := []*itemStorageItem{
// 		{name: "Flare", icon: gs.assets.soldier, count: gs.itemStorage.flareCount, description: "Reveals enemies", consumable: flares},
// 		{name: "Grenade", icon: gs.assets.soldier, count: gs.itemStorage.grenadeCount, description: "Deals damage", consumable: grenades},
// 	}

// 	itemWidth := soldierStorageBoundaries.Width - 20 // 10px margin on each side
// 	itemHeight := float32(100)
// 	for i, item := range items {

// 		itemBoundaries := rl.Rectangle{
// 			X:      itemStorageBoundaries.X + 10,
// 			Y:      itemStorageBoundaries.Y + float32(i)*(itemHeight+10),
// 			Width:  itemWidth,
// 			Height: itemHeight,
// 		}
// 		mousePos := rl.GetMousePosition()
// 		if rl.CheckCollisionPointRec(mousePos, itemBoundaries) {
// 			drawDescription(item.description)
// 			if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
// 				gs.selectedConsumable = item.consumable
// 			}
// 		}

// 		color := rl.White
// 		if gs.selectedConsumable == item.consumable {
// 			color = rl.Green
// 		}

// 		rl.DrawRectangleLinesEx(itemBoundaries, 2, color)
// 		// draw count in top left corner and icon in center
// 		countText := strconv.Itoa(item.count)
// 		countTextPos := rl.Vector2{
// 			X: itemBoundaries.X + 10,
// 			Y: itemBoundaries.Y + 10,
// 		}
// 		rl.DrawText(countText, int32(countTextPos.X), int32(countTextPos.Y), 20, rl.White)

// 		iconPos := rl.Vector2{
// 			X: itemBoundaries.X + itemBoundaries.Width/2 - float32(item.icon.Width/2),
// 			Y: itemBoundaries.Y + itemBoundaries.Height/2 - float32(item.icon.Height/2),
// 		}
// 		rl.DrawTextureV(item.icon, iconPos, rl.White)

// 		// draw name in botton left corner of the item
// 		namePos := rl.Vector2{
// 			X: itemBoundaries.X + 5,
// 			Y: itemBoundaries.Y + itemBoundaries.Height - 25,
// 		}
// 		// TODO: adjust font size based on name length
// 		rl.DrawText(item.name, int32(namePos.X), int32(namePos.Y), 20, rl.White)
// 	}
// }

// func (gs *gameState) placeDraggedSoldier() {
// 	if gs.draggingSoldier == nil {
// 		return
// 	}
// 	if !rl.IsMouseButtonPressed(rl.MouseLeftButton) {
// 		return
// 	}
// 	mousePos := rl.GetMousePosition()
// 	if !rl.CheckCollisionPointRec(mousePos, arenaBoundaries) {
// 		return
// 	}
// 	gs.draggingSoldier.Pos = mousePos
// 	gs.soldiers = append(gs.soldiers, gs.draggingSoldier)
// 	gs.prevQuadtree.Insert(gs.draggingSoldier.ID, rlutils.TextureBoundaries(gs.draggingSoldier.Walking, gs.draggingSoldier.Pos), gs.draggingSoldier)
// 	gs.draggingSoldier = nil
// }

// func (gs *gameState) renderDraggingSoldier() {
// 	if gs.draggingSoldier == nil {
// 		return
// 	}
// 	gs.draggingSoldier.Pos = rl.GetMousePosition()
// 	gs.draggingSoldier.Draw()
// }

func (gs *gameState) useConsumable() {
	switch gs.selectedConsumable {
	case flares:
		gs.useFlare()
	case grenades:
		gs.useGrenade()
	}
}

func (gs *gameState) useFlare() {
	if rl.IsMouseButtonPressed(rl.MouseLeftButton) &&
		// gs.draggingSoldier == nil &&
		gs.itemStorage.flareCount > 0 {
		mousePos := rl.GetMousePosition()
		if !rl.CheckCollisionPointRec(mousePos, gs.boundaries.arenaBoundaries) {
			return
		}
		newFlare := flare.FromPos(mousePos)
		gs.flares = append(gs.flares, newFlare)
		gs.itemStorage.flareCount--
	}
}

func (gs *gameState) processFlares() {
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

func (gs *gameState) renderFlares() {
	for _, f := range gs.flares {
		f.Draw()
	}
}

func (gs *gameState) useGrenade() {
	if rl.IsMouseButtonPressed(rl.MouseLeftButton) &&
		// gs.draggingSoldier == nil &&
		gs.itemStorage.grenadeCount > 0 {
		mousePos := rl.GetMousePosition()
		if !rl.CheckCollisionPointRec(mousePos, gs.boundaries.arenaBoundaries) {
			return
		}
		newGrenade := grenade.FromPos(mousePos)
		gs.grenades = append(gs.grenades, newGrenade)
		gs.itemStorage.grenadeCount--
	}
}

func (gs *gameState) processGrenades() {
	activeGrenades := gs.grenades[:0]
	for _, g := range gs.grenades {
		g.ProgressTime(rl.GetFrameTime())
		if !g.Active() {
			continue
		}
		activeGrenades = append(activeGrenades, g)
		gs.quadtree.Insert(g.ID, g.Boundaries(), g)
	}
	gs.grenades = activeGrenades
}

func (gs *gameState) renderGrenades() {
	for _, g := range gs.grenades {
		g.Draw()
	}
}

func (gs *gameState) processProjectiles() {
	activeProjectiles := gs.projectiles[:0]
	for _, p := range gs.projectiles {
		p.Move()
		if !rl.CheckCollisionPointRec(p.Pos, gs.boundaries.arenaBoundaries) {
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
		if e.IsDead() {
			gs.score += e.Reward()
			continue
		}
		aliveEnemies = append(aliveEnemies, e)
	}
	gs.enemies = aliveEnemies
}

func newEnemy(pos rl.Vector2, assets *gameAssets) enemy {
	// select enemy type
	// 0 - basic
	// 1 - fast
	// 2 - tank

	switch rand.Intn(3) {
	case 0:
		return basic.FromPos(pos, assets.enemy.basic)
	case 1:
		return fast.FromPos(pos, assets.enemy.fast)
	case 2:
		return tank.FromPos(pos, assets.enemy.tank)
	default:
		return basic.FromPos(pos, assets.enemy.basic)
	}
}

func (gs *gameState) spawnEnemies() {
	gs.enemeSpawnedAgo += rl.GetFrameTime()
	if gs.enemeSpawnedAgo < enemySpawnRate {
		return
	}

	gs.enemeSpawnedAgo = 0

	newEnemy := gs.spawnEnemy()
	gs.enemies = append(gs.enemies, newEnemy)
}

func (gs *gameState) spawnEnemy() enemy {
	arenaBoundaries := gs.boundaries.arenaBoundaries
	for {

		// should be spawned in arena boundaries
		pos := rl.Vector2{
			// between arena X and X + Width
			X: float32(rand.Intn(int(arenaBoundaries.Width))) + arenaBoundaries.X,
			// between arena Y and Y + Height
			Y: float32(rand.Intn(int(arenaBoundaries.Height))) + arenaBoundaries.Y,
		}

		newEnemy := newEnemy(pos, gs.assets)

		collissions := gs.prevQuadtree.Query(newEnemy.Boundaries())
		if len(collissions) == 0 {
			return newEnemy
		}
	}
}

func (gs *gameState) processEnemies() []enemy {
	flaredEnemies := make([]enemy, 0, len(gs.enemies)/3)
	for _, e := range gs.enemies {
		nearestSoldier := findNearest(gs.soldiers, e.GetPos())
		newPosition := e.MoveTowards(nearestSoldier.Pos)

		// soldiers didn't move yet, so we can use previous quadtree
		soldierCollissions := gs.prevQuadtree.Query(e.Boundaries())

		for _, c := range soldierCollissions {
			if soldier, ok := c.Value.(*soldier.Soldier); ok {
				soldier.Health -= e.DealDamage()
				newPosition = e.GetPos() // don't move if collission
			}
		}

		collissions := gs.quadtree.Query(e.Boundaries())
		for _, c := range collissions {
			switch val := c.Value.(type) {
			case *flare.Flare:
				flaredEnemies = append(flaredEnemies, e)
				// Try to move away from flare
				newPosition = e.MoveAway(val.Pos)
			case *projectile.Projectile:
				e.TakeDamage(val.Damage)
			case *grenade.Grenade:
				e.TakeDamage(val.Damage)
			}
		}

		e.UpdatePosition(newPosition)
		gs.quadtree.Insert(e.GetID(), e.Boundaries(), e)
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

func (gs *gameState) processSoldiers(flaredEnemies []enemy) {
	for _, s := range gs.soldiers {
		s.ProgressTime(rl.GetFrameTime())

		s.State = soldier.Standing

		soldierBoundaries := rlutils.TextureBoundaries(s.Walking, s.Pos)
		collissions := gs.quadtree.Query(soldierBoundaries)

		for _, c := range collissions {
			// TODO: if soldier is inside of flare - blind him
			switch val := c.Value.(type) {
			case enemy:
				s.State = soldier.Melee
				val.TakeDamage(s.Damage)
			case *grenade.Grenade:
				s.Health -= val.Damage
			}
		}

		if s.State == soldier.Standing {
			// try to find shooting target
			nearestEnemy := findNearest(flaredEnemies, s.Pos)
			if nearestEnemy != nil &&
				s.WithinShootingRange(nearestEnemy.GetPos()) &&
				s.CanShoot() {
				s.Shoot()
				s.State = soldier.Shooting
				// spawn projectile
				projectileVelocity := rl.Vector2Subtract(nearestEnemy.GetPos(), s.Pos)
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
		gs.gameScreen = gameScreenGame
		gs.soldiers = nil
		gs.enemies = nil
		gs.flares = nil
		gs.grenades = nil
		gs.projectiles = nil
		return
	}

	renderHelpLabels(
		"Press enter to restart",
	)

	arenaBoundaries := gs.boundaries.arenaBoundaries
	rlutils.DrawTextAtCenterOfRectangle("Game Over", arenaBoundaries, centerLabelFontSize, centerLabelColor)
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

func (gs *gameState) drawDescription(description string) {
	arenaBoundaries := gs.boundaries.arenaBoundaries
	// description should be displayed in the center of the arena in a rectangle lines
	// with white color
	descriptionBoundaries := rl.Rectangle{
		X:      arenaBoundaries.X + 10,
		Y:      arenaBoundaries.Y + 10,
		Width:  arenaBoundaries.Width - 20,
		Height: arenaBoundaries.Height / 3,
	}

	rl.DrawRectangleLinesEx(descriptionBoundaries, 2, rl.White)

	rlutils.DrawTextAtCenterOfRectangle(description, descriptionBoundaries, 20, rl.White)
}
