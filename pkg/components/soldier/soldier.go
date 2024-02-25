package soldier

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	Speed = float32(1)
)

type Soldier struct {
	Pos     rl.Vector2
	Texture rl.Texture2D
}

func FromPos(pos rl.Vector2, texture rl.Texture2D) *Soldier {
	return &Soldier{
		Pos:     pos,
		Texture: texture,
	}
}

func (s *Soldier) ToRect() rl.Rectangle {
	return rl.Rectangle{
		X:      s.Pos.X,
		Y:      s.Pos.Y,
		Width:  float32(s.Texture.Width),
		Height: float32(s.Texture.Height),
	}
}

func (s *Soldier) Draw() {
	rl.DrawTexture(s.Texture, int32(s.Pos.X), int32(s.Pos.Y), rl.White)
}

func (s *Soldier) IsColiding(other *Soldier) bool {
	curSoldierRect := s.ToRect()
	otherSoldierRect := other.ToRect()
	return rl.CheckCollisionRecs(curSoldierRect, otherSoldierRect)
}

func (s *Soldier) AvoidCollision(other *Soldier) {
	if !s.IsColiding(other) {
		return
	}

	dir := rl.Vector2Subtract(s.Pos, other.Pos)
	dir = rl.Vector2Normalize(dir)
	dir = rl.Vector2Scale(dir, float32(Speed))

	s.Pos = rl.Vector2Add(s.Pos, dir)
}

func (s *Soldier) MoveTowards(pos rl.Vector2, stopBefore float32) {
	dir := rl.Vector2Subtract(pos, s.Pos)

	distance := rl.Vector2Length(dir)

	if stopBefore >= distance {
		return
	}

	dir = rl.Vector2Normalize(dir)

	// Move the soldier at a speed proportional to how far it is from the stopBefore distance,
	// but do not exceed the soldier's maximum speed
	speed := Speed
	if distance-stopBefore < Speed {
		speed = distance - stopBefore
	}
	dir = rl.Vector2Scale(dir, float32(speed))

	s.Pos = rl.Vector2Add(s.Pos, dir)
}
