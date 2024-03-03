package enemies

func Reward(health, speed float32) int {
	return int(health*speed) / 10
}
