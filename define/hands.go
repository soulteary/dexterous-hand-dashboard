package define

type HandType int

const (
	HAND_TYPE_LEFT    HandType = 0x28
	HAND_TYPE_RIGHT   HandType = 0x27
	HAND_TYPE_UNKNOWN HandType = 0x00
)

func (ht HandType) String() string {
	if ht == HAND_TYPE_LEFT {
		return "左手"
	}
	return "右手"
}

func HandTypeFromString(s string) HandType {
	switch s {
	case "left":
		return HAND_TYPE_LEFT
	case "right":
		return HAND_TYPE_RIGHT
	default:
		return HAND_TYPE_UNKNOWN
	}
}
