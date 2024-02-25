package flare

import rl "github.com/gen2brain/raylib-go/raylib"

const (
	DimSpd           = 0.99
	WentOutThreshold = 1
)

type Flare struct {
	Pos                    rl.Vector2
	Radius                 float32
	CenterColor, EdgeColor rl.Color
}

func FromPos(pos rl.Vector2, radius float32, centerColor, edgeColor rl.Color) *Flare {
	return &Flare{
		Pos:         pos,
		Radius:      radius,
		CenterColor: centerColor,
		EdgeColor:   edgeColor,
	}
}

func (f *Flare) Draw() {
	rl.DrawCircleGradient(
		int32(f.Pos.X), int32(f.Pos.Y), f.Radius,
		f.CenterColor, f.EdgeColor,
	)
}

func (f *Flare) Dim() {
	f.Radius *= DimSpd
}

func (f *Flare) WentOut() bool {
	return f.Radius < WentOutThreshold
}
