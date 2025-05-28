package models

import "hands/device"

// GetL10Presets 获取 L10 设备的所有预设姿势
func GetL10Presets() []device.PresetPose {
	return []device.PresetPose{
		// 基础姿势
		{
			Name:        "fist",
			Description: "握拳姿势",
			FingerPose:  []byte{64, 64, 64, 64, 64, 64},
		},
		{
			Name:        "open",
			Description: "完全张开姿势",
			FingerPose:  []byte{192, 192, 192, 192, 192, 192},
		},
		{
			Name:        "pinch",
			Description: "捏取姿势",
			FingerPose:  []byte{120, 120, 64, 64, 64, 64},
		},
		{
			Name:        "thumbsup",
			Description: "竖起大拇指姿势",
			FingerPose:  []byte{64, 192, 192, 192, 192, 64},
		},
		{
			Name:        "point",
			Description: "食指指点姿势",
			FingerPose:  []byte{192, 64, 192, 192, 192, 64},
		},

		// 数字手势
		{
			Name:        "1",
			Description: "数字 1 手势",
			FingerPose:  []byte{192, 64, 192, 192, 192, 64},
		},
		{
			Name:        "2",
			Description: "数字 2 手势",
			FingerPose:  []byte{192, 64, 64, 192, 192, 64},
		},
		{
			Name:        "3",
			Description: "数字 3 手势",
			FingerPose:  []byte{192, 64, 64, 64, 192, 64},
		},
		{
			Name:        "4",
			Description: "数字 4 手势",
			FingerPose:  []byte{192, 64, 64, 64, 64, 64},
		},
		{
			Name:        "5",
			Description: "数字 5 手势",
			FingerPose:  []byte{192, 192, 192, 192, 192, 192},
		},
		{
			Name:        "6",
			Description: "数字 6 手势",
			FingerPose:  []byte{64, 192, 192, 192, 192, 64},
		},
		{
			Name:        "7",
			Description: "数字 7 手势",
			FingerPose:  []byte{64, 64, 192, 192, 192, 64},
		},
		{
			Name:        "8",
			Description: "数字 8 手势",
			FingerPose:  []byte{64, 64, 64, 192, 192, 64},
		},
		{
			Name:        "9",
			Description: "数字 9 手势",
			FingerPose:  []byte{64, 64, 64, 64, 192, 64},
		},
	}
}
