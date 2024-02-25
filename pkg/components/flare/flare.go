package flare

import rl "github.com/gen2brain/raylib-go/raylib"

const (
	DimSpd           = 0.99
	WentOutThreshold = 0.5
)

type Flare struct {
	X, Y                   int32
	Radius                 float32
	CenterColor, EdgeColor rl.Color
}

func FromPos(pos rl.Vector2, radius float32, centerColor, edgeColor rl.Color) *Flare {
	return &Flare{
		X:           int32(pos.X),
		Y:           int32(pos.Y),
		Radius:      radius,
		CenterColor: centerColor,
		EdgeColor:   edgeColor,
	}
}

func (f *Flare) Draw() {
	rl.DrawCircleGradient(
		f.X, f.Y, f.Radius,
		f.CenterColor, f.EdgeColor,
	)
}

func (f *Flare) Dim() {
	f.Radius *= DimSpd
}

func (f *Flare) WentOut() bool {
	return f.Radius < WentOutThreshold
}
