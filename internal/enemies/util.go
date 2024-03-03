package enemies

import "github.com/pechorka/illuminate-game-jam/pkg/rlutils"

func Reward(health, speed float32) int {
	return int(health*speed) / 10
}

func ScaledStat(min, max, time float32) float32 {
	multiplier := time/60 + 1 // minutes
	min *= multiplier
	max *= multiplier
	return rlutils.RandomFloat(min, max)
}
