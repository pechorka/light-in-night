package soldier

import (
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/pechorka/illuminate-game-jam/pkg/rlutils"
)

const (
	initialSpeed            = 2
	initialHealth           = 100
	initialDamage           = 10
	initialShootingRange    = 100
	initialShootingRate     = 0.5
	initialState            = Standing
	initialLevelUpThreshold = 30
)

const (
	nextLevelThreshold = 30 // percent more than previous level
	statUp             = 10 // increase by percent when leveling up
)

const (
	healthbarWidth  float32 = 50
	healthbarHeight float32 = 10
)

type State int

const (
	Standing State = iota + 1
	Melee
	Shooting
)

type Soldier struct {
	ID    int
	Pos   rl.Vector2
	State State

	Speed         float32
	MaxHealth     float32
	Health        float32
	Damage        float32
	ShootingRange float32
	ShootingRate  float32

	Walking rl.Texture2D
	Levelup rl.Texture2D

	ShootAgo float32

	exp                  int
	levelUpThreshold     int
	levelupAnimationTime float32
}

func FromPos(pos rl.Vector2, walking, levelup rl.Texture2D) *Soldier {
	return &Soldier{
		ID:    rand.Int(),
		Pos:   pos,
		State: initialState,

		Speed:         initialSpeed,
		Health:        initialHealth,
		MaxHealth:     initialHealth,
		Damage:        initialDamage,
		ShootingRange: initialShootingRange,
		ShootingRate:  initialShootingRate,

		Walking: walking,
		Levelup: levelup,

		ShootAgo:         initialShootingRate,
		levelUpThreshold: initialLevelUpThreshold,
	}
}

func (s *Soldier) EarnExp(exp int) {
	s.exp += exp
	if s.exp >= s.levelUpThreshold {
		s.levelUpThreshold = s.levelUpThreshold + s.levelUpThreshold*nextLevelThreshold/100
		s.levelUp()
	}
}

func (s *Soldier) levelUp() {
	s.MaxHealth += s.MaxHealth * statUp / 100
	s.Health = s.MaxHealth
	s.Damage += s.Damage * statUp / 100
	s.ShootingRange += s.ShootingRange * statUp / 100
	s.ShootingRate -= s.ShootingRate * statUp / 100
	s.levelupAnimationTime = 1
}

func (s *Soldier) Draw() {
	texture := s.Walking
	switch s.State {
	case Standing:
		texture = s.Walking
	case Melee:
		texture = s.Walking // TODO: change to shooting texture
	case Shooting:
		texture = s.Walking // TODO: change to shooting texture
	}
	// draw health bar above soldier
	// +2 and +1 are to make the health bar look better
	scaledDownInitialHealth := scaleToWidth(s.MaxHealth, s.MaxHealth, healthbarWidth) + 4
	healthbarBorder := rl.NewRectangle(s.Pos.X, s.Pos.Y-10, scaledDownInitialHealth, healthbarHeight)
	rl.DrawRectangleLinesEx(healthbarBorder, 2, rl.White)
	scaledDownCurrentHealth := scaleToWidth(s.Health, s.MaxHealth, healthbarWidth)
	healthbar := rl.Rectangle{
		X:      s.Pos.X + 2,
		Y:      s.Pos.Y - 8,
		Width:  scaledDownCurrentHealth,
		Height: healthbarHeight - 4,
	}
	rl.DrawRectangleRec(healthbar, rl.Green)

	if s.levelupAnimationTime > 0 {
		levelUpPosition := rl.Vector2{X: s.Pos.X - 20, Y: s.Pos.Y - 20}
		rl.DrawTexture(s.Levelup, int32(levelUpPosition.X), int32(levelUpPosition.Y), rl.White)
	}

	// draw circle with radius of shooting range
	rl.DrawCircleLines(int32(s.Pos.X), int32(s.Pos.Y), s.ShootingRange, rl.White)

	rl.DrawTexture(texture, int32(s.Pos.X), int32(s.Pos.Y), rl.White)
}

func scaleToWidth(health, max, width float32) float32 {
	return rlutils.ScaleValueToSize(health, 0, max, 0, width)
}

func (s *Soldier) MoveTowards(pos rl.Vector2) rl.Vector2 {
	dir := rl.Vector2Subtract(pos, s.Pos)
	dir = rl.Vector2Normalize(dir)
	dir = rl.Vector2Scale(dir, s.Speed)

	return rl.Vector2Add(s.Pos, dir)
}

func (s *Soldier) WithinShootingRange(pos rl.Vector2) bool {
	return rl.Vector2Distance(s.Pos, pos) < s.ShootingRange
}

func (s *Soldier) CanShoot(shootFast bool) bool {
	rate := s.ShootingRate
	if shootFast {
		rate = rate / 2
	}
	return s.ShootAgo >= rate
}

func (s *Soldier) Shoot() {
	s.ShootAgo = 0
}

func (s *Soldier) ProgressTime(dt float32) {
	if s.ShootAgo < s.ShootingRate {
		s.ShootAgo += dt
	}
	if s.levelupAnimationTime > 0 {
		s.levelupAnimationTime -= dt
	}
}

func (s *Soldier) GetPos() rl.Vector2 {
	return s.Pos
}

func (s *Soldier) Boundaries() rl.Rectangle {
	return rlutils.TextureBoundaries(s.Walking, s.Pos)
}
