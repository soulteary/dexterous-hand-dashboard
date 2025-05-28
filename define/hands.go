package define

type HandType int

const (
	HAND_TYPE_LEFT  HandType = 0x28
	HAND_TYPE_RIGHT HandType = 0x27
)

func (ht HandType) String() string {
	if ht == HAND_TYPE_LEFT {
		return "左手"
	}
	return "右手"
}
