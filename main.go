package main

import (
	"embed"
	"fmt"
	"math"
	"math/rand"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/pechorka/illuminate-game-jam/internal/consumables/flare"
	"github.com/pechorka/illuminate-game-jam/internal/consumables/grenade"
	"github.com/pechorka/illuminate-game-jam/internal/db"
	"github.com/pechorka/illuminate-game-jam/internal/enemies/basic"
	"github.com/pechorka/illuminate-game-jam/internal/enemies/fast"
	"github.com/pechorka/illuminate-game-jam/internal/enemies/tank"
	"github.com/pechorka/illuminate-game-jam/internal/projectile"
	"github.com/pechorka/illuminate-game-jam/internal/soldier"
	"github.com/pechorka/illuminate-game-jam/pkg/data_structures/quadtree"
	"github.com/pechorka/illuminate-game-jam/pkg/rlutils"
	"go.etcd.io/bbolt"

	rl "github.com/gen2brain/raylib-go/raylib"
)

//go:embed assets
var assets embed.FS

const (
	quadtreeCapacity = 10
)

const (
	maxNameLength = 10
)

const gameTitle = "Light in Night"

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
	initialSpawnRate    = float32(1)
	spawnRateLimit      = float32(0.1)
)

// TODO: disgusting global variables
var closeWindow = false

var (
	soldierCount int // 1-4
)

func main() {
	boltCli, err := bbolt.Open("light-in-night.db", os.ModePerm, nil)
	if err != nil {
		panic(err)
	}
	defer boltCli.Close()

	db := db.New(boltCli)

	gb := &gameBoundaries{
		screenWidth:  1280,
		screenHeight: 720,
	}
	gb.updateBoundaries()

	rl.SetConfigFlags(rl.FlagWindowResizable)
	rl.InitWindow(int32(gb.screenWidth), int32(gb.screenHeight), gameTitle)
	// rl.InitAudioDevice()
	windowIcon := loadImageFromMemory("assets/app.ico")
	rl.SetWindowIcon(*windowIcon)

	gs := &gameState{
		boundaries: gb,
		assets: &gameAssets{
			// titleMusic: loadMusicStream("assets/sounds/music/title.mp3"),
			soldier: loadTextureFromImage("assets/soldier.png"),
			levelup: loadTextureFromImage("assets/levelup.png"),
			enemy: &enemyAssets{
				basic: loadTextureFromImage("assets/enemy/basic.png"),
				fast:  loadTextureFromImage("assets/enemy/fast.png"),
				tank:  loadTextureFromImage("assets/enemy/tank.png"),
			},
			consumables: &consumableAssets{
				flare:   loadTextureFromImage("assets/consumables/flare.png"),
				grenade: loadTextureFromImage("assets/consumables/grenade.png"),
			},
		},
		prevQuadtree: quadtree.NewQuadtree(gb.arenaBoundaries, quadtreeCapacity),
		quadtree:     quadtree.NewQuadtree(gb.arenaBoundaries, quadtreeCapacity),

		itemStorage: &itemStorage{
			flareCount:   initialFlareCount,
			grenadeCount: initialGrenadeCount,
		},
		selectedConsumable: flares,
		spawnRate:          initialSpawnRate,

		gameScreen: gameScreenMainMenu,

		db: db,
	}

	// rl.PlayMusicStream(gs.assets.titleMusic)

	rl.SetTargetFPS(60)

	for !(rl.WindowShouldClose() || closeWindow) {
		rl.BeginDrawing()

		gs.renderFrame()

		rl.EndDrawing()
	}

	gs.assets.unload()
	rl.UnloadImage(windowIcon)
	// rl.CloseAudioDevice()

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

func loadImageFromMemory(imgPath string) *rl.Image {
	file, err := assets.ReadFile(imgPath)
	if err != nil {
		panic(err)
	}
	fileExtention := imgPath[strings.LastIndexByte(imgPath, '.'):]
	return rl.LoadImageFromMemory(fileExtention, file, int32(len(file)))
}

func loadMusicStream(musicPath string) rl.Music {
	file, err := assets.ReadFile(musicPath)
	if err != nil {
		panic(err)
	}
	fileExtention := musicPath[strings.LastIndexByte(musicPath, '.'):]
	music := rl.LoadMusicStreamFromMemory(fileExtention, file, int32(len(file)))

	return music
}

type gameAssets struct {
	soldier    rl.Texture2D
	levelup    rl.Texture2D
	titleMusic rl.Music

	enemy       *enemyAssets
	consumables *consumableAssets
}

func (ga *gameAssets) unload() {
	rl.UnloadTexture(ga.soldier)
	rl.UnloadMusicStream(ga.titleMusic)
	rl.UnloadTexture(ga.enemy.basic)
	rl.UnloadTexture(ga.enemy.fast)
	rl.UnloadTexture(ga.enemy.tank)
	rl.UnloadTexture(ga.consumables.flare)
	rl.UnloadTexture(ga.consumables.grenade)
}

type enemyAssets struct {
	basic rl.Texture2D
	fast  rl.Texture2D
	tank  rl.Texture2D
}

type consumableAssets struct {
	flare   rl.Texture2D
	grenade rl.Texture2D
}

type gameScreen int

const (
	gameScreenMainMenu gameScreen = iota
	gameScreenSetupGame
	gameScreenGame
	gameScreenOver
	gameScreenLeaderboard
	gameScreenHowToPlay
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
	spawnRate       float32

	score int
	money int

	gameTime float32

	db *db.DB

	nameInput string
	victory   bool

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
	case gameScreenMainMenu:
		gs.renderMainMenu()
	case gameScreenSetupGame:
		gs.renderSetupGameScreen()
	case gameScreenGame:
		gs.renderGame()
	case gameScreenOver:
		gs.renderGameOver()
	case gameScreenLeaderboard:
		gs.renderLeaderboardScreen()
	case gameScreenHowToPlay:
		gs.renderHowToPlayScreen()
	}
}

func (gs *gameState) renderMainMenu() {
	type menuItem struct {
		name   string
		action func()
	}
	actionNextScreen := func(nextScreen gameScreen) func() {
		return func() {
			gs.gameScreen = nextScreen
		}
	}

	items := []menuItem{
		{name: "New game", action: actionNextScreen(gameScreenSetupGame)},
		{name: "Leaderboard", action: actionNextScreen(gameScreenLeaderboard)},
		{name: "How to play", action: actionNextScreen(gameScreenHowToPlay)},
		{name: "Exit", action: func() { closeWindow = true }},
	}

	titleX := int32(gs.boundaries.screenBoundaries.Width / 2)
	titleY := int32(gs.boundaries.screenBoundaries.Y + 10)
	titleFontSize := int32(100)
	titleWidth := rl.MeasureText(gameTitle, titleFontSize)
	rl.DrawText(gameTitle, titleX-titleWidth/2, titleY, titleFontSize, rl.White)

	x := int32(gs.boundaries.screenBoundaries.Width / 2)
	y := int32(gs.boundaries.screenBoundaries.Height / 3)
	spacing := int32(50)
	for i, item := range items {
		textWidth := rl.MeasureText(item.name, centerLabelFontSize)
		textX := x - textWidth/2
		textY := y + int32(i)*spacing

		mousePos := rl.GetMousePosition()
		itemBoundaries := rl.Rectangle{
			X:      float32(textX),
			Y:      float32(textY),
			Width:  float32(textWidth),
			Height: float32(centerLabelFontSize),
		}

		color := rl.White
		if rl.CheckCollisionPointRec(mousePos, itemBoundaries) {
			color = rl.Green
			if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
				item.action()
			}
		}

		rl.DrawText(item.name, textX, textY, centerLabelFontSize, color)
	}
}

func (gs *gameState) renderSetupGameScreen() {
	x := int32(gs.boundaries.screenBoundaries.Width / 2)
	y := int32(gs.boundaries.screenBoundaries.Height / 3)
	spacing := int32(50)
	fontSize := int32(30)

	numberOfSoldiersItem := "Select number of soldiers: "
	numberOfSoldiersItemWidth := rl.MeasureText(numberOfSoldiersItem, fontSize)
	numberOfSoldiersItemX := x - numberOfSoldiersItemWidth/2
	numberOfSoldiersItemY := y

	rl.DrawText(numberOfSoldiersItem, numberOfSoldiersItemX, numberOfSoldiersItemY, fontSize, rl.White)

	optionX := numberOfSoldiersItemX + numberOfSoldiersItemWidth
	optionY := numberOfSoldiersItemY
	optionWidth := rl.MeasureText("4", fontSize)
	for i := range 4 {
		option := strconv.Itoa(i + 1)

		optionX += optionWidth + 10
		optionBoundaries := rl.Rectangle{
			X:      float32(optionX) - 5,
			Y:      float32(optionY) - 5,
			Width:  float32(optionWidth) + 10,
			Height: float32(centerLabelFontSize),
		}

		color := rl.White
		if rl.CheckCollisionPointRec(rl.GetMousePosition(), optionBoundaries) {
			color = rl.Green
			if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
				soldierCount = i + 1
				gs.placeSoldiersOnRandomPositions(soldierCount)
			}
		}
		if i+1 == soldierCount {
			color = rl.Green
		}

		rl.DrawRectangleLinesEx(optionBoundaries, 2, color)
		rl.DrawText(option, optionX, optionY, fontSize, color)
	}

	backToMainMenuItem := "Back to main menu"
	backToMainMenuItemWidth := rl.MeasureText(backToMainMenuItem, fontSize)
	backToMainMenuItemX := x - backToMainMenuItemWidth/2
	y += spacing
	backToMainMenuItemY := y
	backToMainMenuItemBoundaries := rl.Rectangle{
		X:      float32(backToMainMenuItemX),
		Y:      float32(backToMainMenuItemY),
		Width:  float32(backToMainMenuItemWidth),
		Height: float32(centerLabelFontSize),
	}
	color := rl.White
	if rl.CheckCollisionPointRec(rl.GetMousePosition(), backToMainMenuItemBoundaries) {
		color = rl.Green
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			gs.gameScreen = gameScreenMainMenu
		}
	}
	rl.DrawText(backToMainMenuItem, backToMainMenuItemX, backToMainMenuItemY, fontSize, color)

	startGameItem := "Start game"
	startGameItemWidth := rl.MeasureText(startGameItem, fontSize)
	startGameItemX := x - startGameItemWidth/2
	y += spacing
	startGameItemY := y
	startGameItemBoundaries := rl.Rectangle{
		X:      float32(startGameItemX),
		Y:      float32(startGameItemY),
		Width:  float32(startGameItemWidth),
		Height: float32(centerLabelFontSize),
	}

	color = rl.Gray
	if soldierCount > 0 {
		color = rl.White
		if rl.CheckCollisionPointRec(rl.GetMousePosition(), startGameItemBoundaries) {
			color = rl.Green
			if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
				gs.gameScreen = gameScreenGame
			}
		}
	}
	rl.DrawText(startGameItem, startGameItemX, startGameItemY, fontSize, color)
}

func (gs *gameState) placeSoldiersOnRandomPositions(soldierCount int) {
	gs.soldiers = make([]*soldier.Soldier, 0, soldierCount)
	ab := gs.boundaries.arenaBoundaries
	ab = rl.Rectangle{
		X:      ab.X + 100,
		Y:      ab.Y + 100,
		Width:  ab.Width - 100,
		Height: ab.Height - 100,
	}
	quadtree := quadtree.NewQuadtree(ab, quadtreeCapacity)
	for soldierCount > 0 {
		pos := rl.Vector2{
			X: rlutils.RandomFloat(ab.X+100, ab.Width+ab.X-100),
			Y: rlutils.RandomFloat(ab.Y+100, ab.Height+ab.Y-100),
		}
		newSoldier := soldier.FromPos(pos, gs.assets.soldier, gs.assets.levelup)

		collisions := quadtree.Query(newSoldier.Boundaries())
		if len(collisions) > 0 {
			continue
		}
		quadtree.Insert(newSoldier.ID, newSoldier.Boundaries(), newSoldier)
		gs.soldiers = append(gs.soldiers, newSoldier)
		soldierCount--
	}
}

func (gs *gameState) renderTodoScreen() {
	renderHelpLabels(
		"This screen is not implemented yet",
		"Click anywhere to go back to the main menu",
	)

	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		gs.gameScreen = gameScreenMainMenu
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

		gs.spawnEnemies()
		flaredEnemies := gs.processEnemies()
		gs.cleanupDeadEnemies()

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

	timeText := "Time: " + gameTimeToString(gs.gameTime)
	scoreText := "Score: " + strconv.Itoa(gs.score)
	moneyText := "Money: " + strconv.Itoa(gs.money) + "$"

	timeTextWidth := rl.MeasureText(timeText, 20)
	scoreTextWidth := rl.MeasureText(scoreText, 20)

	widthOffset := int32(10)
	rl.DrawText(timeText, widthOffset, 10, 20, rl.White)
	widthOffset += timeTextWidth + 10
	// TODO: score is flickering depending on the time
	rl.DrawText(scoreText, widthOffset, 10, 20, rl.White)
	widthOffset += scoreTextWidth + 10
	rl.DrawText(moneyText, widthOffset, 10, 20, rl.White)

	if gs.paused {
		rl.DrawText("Paused", int32(headerBoundaries.Width-100), 10, 20, rl.White)
	}
}

func gameTimeToString(time float32) string {
	timeInt := int(time)
	minutes := timeInt / 60
	seconds := timeInt % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

type shopItem struct {
	price       int
	count       int
	name        string
	description string
	icon        rl.Texture2D
	ctype       consumable
	quickBuyBtn int32
}

func (gs *gameState) renderFooter() {
	footerBoundaries := gs.boundaries.footerBoundaries
	rl.DrawRectangleRec(footerBoundaries, rl.Gray)
	// TODO: shop can have items to increase difficulty, but with higher rewards
	// for example: princess that needs to be protected for some time, but gives a lot of money
	mockItems := []shopItem{
		{
			price: 10, count: 10,
			name: "Flare", description: "Reveals enemies",
			icon:        gs.assets.consumables.flare,
			ctype:       flares,
			quickBuyBtn: rl.KeyQ,
		},
		{
			price: 20, count: 1,
			name: "Grenade", description: "Deals damage",
			icon:        gs.assets.consumables.grenade,
			ctype:       grenades,
			quickBuyBtn: rl.KeyW,
		},
	}
	itemWidth := float32(100)
	itemHeight := footerBoundaries.Height - 20 // 10px margin on each side
	buyItem := func(item shopItem) {
		if gs.money >= item.price {
			gs.money -= item.price
			switch item.ctype {
			case flares:
				gs.itemStorage.flareCount += item.count
			case grenades:
				gs.itemStorage.grenadeCount += item.count
			}
		}
	}
	for i, item := range mockItems {
		itemBoundaries := rl.Rectangle{
			X:      footerBoundaries.X + 10 + float32(i)*(itemWidth+10),
			Y:      footerBoundaries.Y + 10,
			Width:  itemWidth,
			Height: itemHeight,
		}
		rl.DrawRectangleLinesEx(itemBoundaries, 2, rl.White)

		iconPos := rl.Vector2{
			X: itemBoundaries.X + itemBoundaries.Width/2 - float32(item.icon.Width/2),
			Y: itemBoundaries.Y + itemBoundaries.Height/2 - float32(item.icon.Height/2),
		}
		rl.DrawTextureV(item.icon, iconPos, rl.White)

		// display description when hovered
		if rl.CheckCollisionPointRec(rl.GetMousePosition(), itemBoundaries) {
			// description should be displayed above the item with border
			gs.drawShopItemDescription(item)

			if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
				buyItem(item)
			}
		}

		if rl.IsKeyPressed(item.quickBuyBtn) {
			buyItem(item)
		}
	}

	// in right side render "End run" button
	endRunText := "End run"
	endRunFontSize := int32(50)
	endRunTextWidth := rl.MeasureText(endRunText, endRunFontSize)
	endRunTextX := int32(footerBoundaries.X+footerBoundaries.Width-10) - endRunTextWidth
	endRunTextY := int32(footerBoundaries.Y + 30)
	endRunTextBoundaries := rl.Rectangle{
		X:      float32(endRunTextX),
		Y:      float32(endRunTextY),
		Width:  float32(endRunTextWidth) + 10,
		Height: float32(endRunFontSize) + 10,
	}
	color := rl.White
	if rl.CheckCollisionPointRec(rl.GetMousePosition(), endRunTextBoundaries) {
		color = rl.Green
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			gs.gameScreen = gameScreenOver
		}
	}
	rl.DrawText(endRunText, endRunTextX, endRunTextY, endRunFontSize, color)
}

func (gs *gameState) renderHowToPlayScreen() {
	tutorialText := []string{
		"Start a new game from the main menu and select the number of soldiers.",
		"Use the left mouse button to deploy flares and grenades.",
		"Flares reveal and repel enemies. Soldiers will shoot at enemies in flare range.",
		"Switch between flares and grenades with the 1 and 2 keys.",
		"Use money earned from defeating enemies to buy more flares and grenades.",
		"You can quick buy flares and grenades with the Q and W keys.",
		"Soldiers will automatically attack enemies in their range.",
		"The game ends when all soldiers are defeated.",
		"Pause the game anytime with the spacebar.",
		"The less soldiers you choose, the more money/score you earn.",
		"Don't delete light-in-night.db file, it contains your highscore.",
	}

	x := int32(gs.boundaries.screenBoundaries.Width / 2)
	y := int32(gs.boundaries.screenBoundaries.Y + 10)
	howToPlayTitle := "How to play"
	howToPlayTitleWidth := rl.MeasureText(howToPlayTitle, 50)
	rl.DrawText(howToPlayTitle, x-howToPlayTitleWidth/2, y, 50, rl.White)
	y += 60

	lineX := x - 300

	for _, line := range tutorialText {
		rl.DrawText(line, lineX, y, 20, rl.White)
		y += 30
	}

	// Add a back button to return to the main menu
	backButton := "Click anywhere to return to the main menu"
	backButtonWidth := rl.MeasureText(backButton, 20)
	rl.DrawText(backButton, x-backButtonWidth/2, y+20, 20, rl.Gray) // Draw back button text
	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		gs.gameScreen = gameScreenMainMenu // Return to main menu on click
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
		if !rl.CheckCollisionPointRec(p.Pos, gs.boundaries.arenaBoundaries) || p.Expired {
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

const (
	halfMaxInt    = math.MaxInt / 2
	quorterMaxInt = halfMaxInt / 2
	tenthMaxInt   = math.MaxInt / 10
)

var (
	basicRange = [2]int{0, halfMaxInt}                                        // 0 - 50%
	fastRange  = [2]int{halfMaxInt, halfMaxInt + quorterMaxInt + tenthMaxInt} // 50% - 85%
	// tank is default case (15%)
)

func inRange(x int, r [2]int) bool {
	return x >= r[0] && x < r[1]
}

func newEnemy(pos rl.Vector2, assets *gameAssets, time float32) enemy {
	n := rand.Intn(math.MaxInt)
	switch {
	case inRange(n, basicRange):
		return basic.FromPos(pos, assets.enemy.basic, time)
	case inRange(n, fastRange):
		return fast.FromPos(pos, assets.enemy.fast, time)
	default:
		return tank.FromPos(pos, assets.enemy.tank, time)
	}
}

func (gs *gameState) spawnEnemies() {
	gs.enemeSpawnedAgo += rl.GetFrameTime()

	multiplier := gs.gameTime / 60
	spawnRate := gs.spawnRate * float32(math.Pow(0.90, float64(multiplier)))
	if spawnRate < spawnRateLimit {
		spawnRate = spawnRateLimit
	}
	if gs.enemeSpawnedAgo < spawnRate {
		return
	}

	gs.enemeSpawnedAgo = 0

	newEnemy, ok := gs.spawnEnemy()
	if !ok {
		return
	}
	gs.enemies = append(gs.enemies, newEnemy)
}

func (gs *gameState) spawnEnemy() (enemy, bool) {
	arenaBoundaries := gs.boundaries.arenaBoundaries
	attempt := 0
	for {
		attempt++
		if attempt > 100 {
			gs.victory = true
			gs.gameScreen = gameScreenOver
			return nil, false
		}
		// should be spawned in arena boundaries
		pos := rl.Vector2{
			// between arena X and X + Width
			X: float32(rand.Intn(int(arenaBoundaries.Width))) + arenaBoundaries.X,
			// between arena Y and Y + Height
			Y: float32(rand.Intn(int(arenaBoundaries.Height))) + arenaBoundaries.Y,
		}

		if gs.anySoldierCanShoot(pos) {
			continue
		}

		newEnemy := newEnemy(pos, gs.assets, gs.gameTime)

		collissions := gs.prevQuadtree.Query(newEnemy.Boundaries())
		if len(collissions) == 0 {
			return newEnemy, true
		}
	}
}

func (gs *gameState) anySoldierCanShoot(pos rl.Vector2) bool {
	for _, s := range gs.soldiers {
		if s.WithinShootingRange(pos) {
			return true
		}
	}
	return false
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

		flared := false
		collissions := gs.quadtree.Query(e.Boundaries())
		for _, c := range collissions {
			switch val := c.Value.(type) {
			case *flare.Flare:
				flared = true
				// Try to move away from flare
				newPosition = e.MoveAway(val.Pos)
			case *projectile.Projectile:
				if !val.Expired {
					e.TakeDamage(val.Damage)
					val.Expired = true
				}
				if e.IsDead() {
					val.Shooter.EarnExp(reward(e.Reward()))
				}
			case *grenade.Grenade:
				e.TakeDamage(val.Damage)
			}
		}

		if e.IsDead() {
			continue
		}

		if flared {
			flaredEnemies = append(flaredEnemies, e)
		}

		e.UpdatePosition(newPosition)
		gs.quadtree.Insert(e.GetID(), e.Boundaries(), e)
	}

	return flaredEnemies
}

func (gs *gameState) cleanupDeadEnemies() {
	aliveEnemies := gs.enemies[:0]
	for _, e := range gs.enemies {
		if e.IsDead() {
			gs.score += reward(e.Reward())
			gs.money += reward(e.Reward())
			continue
		}
		aliveEnemies = append(aliveEnemies, e)
	}
	gs.enemies = aliveEnemies
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
			nearestEnemy := findNearest(gs.enemies, s.Pos)
			shootFast := true
			if nearestEnemy == nil || !s.WithinShootingRange(nearestEnemy.GetPos()) {
				nearestEnemy = findNearest(flaredEnemies, s.Pos)
				shootFast = false
			}
			if nearestEnemy != nil &&
				s.CanShoot(shootFast) {
				s.Shoot()
				s.State = soldier.Shooting
				// spawn projectile
				projectileVelocity := rl.Vector2Subtract(nearestEnemy.GetPos(), s.Pos)
				newProjectile := projectile.FromPos(s.Pos, projectileVelocity, s)
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
	x := int32(gs.boundaries.screenBoundaries.Width / 2)
	y := int32(gs.boundaries.screenBoundaries.Height / 3)
	spacing := int32(50)
	fontSize := int32(30)

	gameOver := "Game Over"
	if gs.victory {
		gameOver = "Victory"
	}
	gameOverWidth := rl.MeasureText(gameOver, fontSize)
	gameOverX := x - gameOverWidth/2
	gameOverY := y
	rl.DrawText(gameOver, gameOverX, gameOverY, fontSize, rl.White)

	scoreMultiplier := gs.scoreMultiplierForAliveSoldiers()

	if scoreMultiplier > 1 {
		victoryBonus := fmt.Sprintf("Alive soldiers bonus: x%d", scoreMultiplier)
		victoryBonusWidth := rl.MeasureText(victoryBonus, fontSize)
		victoryBonusX := x - victoryBonusWidth/2
		y += spacing
		victoryBonusY := y
		rl.DrawText(victoryBonus, victoryBonusX, victoryBonusY, fontSize, rl.White)

		score := fmt.Sprintf("Score: %d x %d = %d", gs.score, scoreMultiplier, gs.score*scoreMultiplier)
		scoreWidth := rl.MeasureText(score, fontSize)
		scoreX := x - scoreWidth/2
		y += spacing
		scoreY := y
		rl.DrawText(score, scoreX, scoreY, fontSize, rl.White)
	} else {
		score := fmt.Sprintf("Score: %d", gs.score)
		scoreWidth := rl.MeasureText(score, fontSize)
		scoreX := x - scoreWidth/2
		y += spacing
		scoreY := y
		rl.DrawText(score, scoreX, scoreY, fontSize, rl.White)
	}

	time := "Time: " + gameTimeToString(gs.gameTime)
	timeWidth := rl.MeasureText(time, fontSize)
	timeX := x - timeWidth/2
	y += spacing
	timeY := y
	rl.DrawText(time, timeX, timeY, fontSize, rl.White)

	nameInput := "Enter your name: "
	nameInputWidth := rl.MeasureText(nameInput, fontSize)
	nameInputX := x - nameInputWidth/2
	y += spacing
	nameInputY := y
	rl.DrawText(nameInput, nameInputX, nameInputY, fontSize, rl.White)

	// render input field to the right of nameInput
	inputBoundaries := rl.Rectangle{
		X:      float32(nameInputX + nameInputWidth),
		Y:      float32(nameInputY),
		Width:  200,
		Height: float32(fontSize) + 10,
	}
	rl.DrawRectangleLinesEx(inputBoundaries, 2, rl.White)
	if rl.CheckCollisionPointRec(rl.GetMousePosition(), inputBoundaries) {
		rl.SetMouseCursor(rl.MouseCursorIBeam)
	}

	if len(gs.nameInput) > 0 {
		rl.DrawText(gs.nameInput, int32(inputBoundaries.X+10), int32(inputBoundaries.Y+10), fontSize, rl.White)
	}

	for key := rl.GetKeyPressed(); key > 0; key = rl.GetKeyPressed() {
		if key == rl.KeyBackspace {
			if len(gs.nameInput) > 0 {
				gs.nameInput = gs.nameInput[:len(gs.nameInput)-1]
			}
		} else if key == rl.KeyEnter {
			gs.saveScore()
			gs.reset()
			gs.gameScreen = gameScreenMainMenu
		} else if 32 <= key && key <= 125 && len(gs.nameInput) < maxNameLength {
			gs.nameInput += string(key)
		}
	}

	// render (x/maxNameLength) to show how many characters can be entered
	rl.DrawText(strconv.Itoa(len(gs.nameInput))+"/"+strconv.Itoa(maxNameLength), int32(inputBoundaries.X+inputBoundaries.Width+10), int32(inputBoundaries.Y), fontSize, rl.White)

	backToMainMenuItem := "Save score and back to main menu"
	backToMainMenuItemWidth := rl.MeasureText(backToMainMenuItem, fontSize)
	backToMainMenuItemX := x - backToMainMenuItemWidth/2
	y += spacing
	backToMainMenuItemY := y
	backToMainMenuItemBoundaries := rl.Rectangle{
		X:      float32(backToMainMenuItemX),
		Y:      float32(backToMainMenuItemY),
		Width:  float32(backToMainMenuItemWidth),
		Height: float32(fontSize),
	}
	color := rl.Gray
	if rl.CheckCollisionPointRec(rl.GetMousePosition(), backToMainMenuItemBoundaries) {
		color = rl.Green
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			gs.saveScore()
			gs.reset()
			gs.gameScreen = gameScreenMainMenu
		}
	}
	rl.DrawText(backToMainMenuItem, backToMainMenuItemX, backToMainMenuItemY, fontSize, color)

	backToMainMenuItemNoScore := "Back to main menu without saving score"
	backToMainMenuItemNoScoreWidth := rl.MeasureText(backToMainMenuItemNoScore, fontSize)
	backToMainMenuItemNoScoreX := x - backToMainMenuItemNoScoreWidth/2
	y += spacing
	backToMainMenuItemNoScoreY := y

	backToMainMenuItemNoScoreBoundaries := rl.Rectangle{
		X:      float32(backToMainMenuItemNoScoreX),
		Y:      float32(backToMainMenuItemNoScoreY),
		Width:  float32(backToMainMenuItemNoScoreWidth),
		Height: float32(fontSize),
	}
	color = rl.Gray
	if rl.CheckCollisionPointRec(rl.GetMousePosition(), backToMainMenuItemNoScoreBoundaries) {
		color = rl.Green
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			gs.reset()
			gs.gameScreen = gameScreenMainMenu
		}
	}

	rl.DrawText(backToMainMenuItemNoScore, backToMainMenuItemNoScoreX, backToMainMenuItemNoScoreY, fontSize, color)
}

func (gs *gameState) scoreMultiplierForAliveSoldiers() int {
	return int(math.Pow(2, float64(len(gs.soldiers))))
}

func (gs *gameState) saveScore() {
	if len(gs.nameInput) == 0 {
		return
	}

	score := gs.score * gs.scoreMultiplierForAliveSoldiers()
	rl.TraceLog(rl.LogInfo, "Saving score for %s: %d", gs.nameInput, score)

	err := gs.db.AddHighscore(db.Highscore{
		Name:    gs.nameInput,
		Score:   score,
		Time:    gs.gameTime,
		Victory: gs.victory,
	})
	if err != nil {
		rl.TraceLog(rl.LogError, "Error saving highscore: %v", err)
	}
}

func (gs *gameState) reset() {
	gs.gameTime = 0
	gs.score = 0
	gs.money = 0
	gs.soldiers = nil
	gs.enemies = nil
	gs.flares = nil
	gs.grenades = nil
	gs.projectiles = nil
	soldierCount = 0
	gs.nameInput = ""
	gs.victory = false
	gs.selectedConsumable = flares
	gs.itemStorage.flareCount = initialFlareCount
	gs.itemStorage.grenadeCount = initialGrenadeCount
}

func (gs *gameState) renderLeaderboardScreen() {
	leaderboard, _ := gs.db.GetHighscores()
	x := int32(gs.boundaries.screenBoundaries.Width / 2)
	y := int32(gs.boundaries.screenBoundaries.Y + 10)

	leaderboardTitle := "Leaderboard"
	leaderboardTitleWidth := rl.MeasureText(leaderboardTitle, 50)
	leaderboardTitleX := x - leaderboardTitleWidth/2
	rl.DrawText(leaderboardTitle, leaderboardTitleX, y, 50, rl.White)
	y += 60

	spacing := int32(30)
	fontSize := int32(20)

	nameX := x - 200
	scoreX := x

	for _, score := range leaderboard {
		name := score.Name
		rl.DrawText(name, nameX, y, fontSize, rl.White)
		prefix := "Unsuccessful run"
		if score.Victory {
			prefix = "Victory run"
		}
		score := fmt.Sprintf("%s %d points in %s", prefix, score.Score, gameTimeToString(score.Time))
		// scoreWidth := rl.MeasureText(score, fontSize)
		rl.DrawText(score, scoreX, y, fontSize, rl.White)
		y += spacing
	}

	backButton := "Click anywhere to return to the main menu"
	backButtonWidth := rl.MeasureText(backButton, fontSize)
	backButtonX := x - backButtonWidth/2
	rl.DrawText(backButton, backButtonX, y+20, fontSize, rl.Gray) // Draw back button text
	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		gs.gameScreen = gameScreenMainMenu // Return to main menu on click
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

func (gs *gameState) drawShopItemDescription(item shopItem) {
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

	x := int32(descriptionBoundaries.X + 10)
	y := int32(descriptionBoundaries.Y + 10)
	spacing := int32(30)

	name := "Name: " + item.name
	rl.DrawText(name, x, y, 20, rl.White)
	y += spacing
	price := "Price: " + strconv.Itoa(item.price) + "$"
	rl.DrawText(price, x, y, 20, rl.White)
	y += spacing
	count := "Count: " + strconv.Itoa(item.count)
	rl.DrawText(count, x, y, 20, rl.White)
	y += spacing
	quickBuy := "Quick buy: " + buttonToString(item.quickBuyBtn)
	rl.DrawText(quickBuy, x, y, 20, rl.White)
	y += spacing
	description := "Description: " + item.description
	rl.DrawText(description, x, y, 20, rl.White)
}

func buttonToString(btn int32) string {
	switch btn {
	case rl.KeyQ:
		return "Q"
	case rl.KeyW:
		return "W"
	default:
		return ""
	}

}

func reward(base int) int {
	// multiplier := 4 / soldierCount // Playtest
	multiplier := 1
	return base * multiplier
}
