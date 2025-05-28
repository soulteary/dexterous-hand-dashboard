package device

import "hands/define"

// PoseExecutor 定义了执行基本姿态指令的能力
type PoseExecutor interface {
	// SetFingerPose 设置手指姿态
	// pose: 6 字节数据，代表 6 个手指的位置
	SetFingerPose(pose []byte) error

	// SetPalmPose 设置手掌姿态
	// pose: 4 字节数据，代表手掌的 4 个自由度
	SetPalmPose(pose []byte) error

	// ResetPose 重置到默认姿态
	ResetPose() error

	// GetHandType 获取当前手型
	GetHandType() define.HandType
}
