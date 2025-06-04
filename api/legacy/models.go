package legacy

// FingerPoseRequest 手指姿态设置请求
type FingerPoseRequest struct {
	Interface string `json:"interface,omitempty"`
	Pose      []byte `json:"pose" binding:"required,len=6"`
	HandType  string `json:"handType,omitempty"` // 新增：手型类型
	HandId    uint32 `json:"handId,omitempty"`   // 新增：CAN ID
}

// PalmPoseRequest 掌部姿态设置请求
type PalmPoseRequest struct {
	Interface string `json:"interface,omitempty"`
	Pose      []byte `json:"pose" binding:"required,len=4"`
	HandType  string `json:"handType,omitempty"` // 新增：手型类型
	HandId    uint32 `json:"handId,omitempty"`   // 新增：CAN ID
}

// AnimationRequest 动画控制请求
type AnimationRequest struct {
	Interface string `json:"interface,omitempty"`
	Type      string `json:"type" binding:"required,oneof=wave sway stop"`
	Speed     int    `json:"speed" binding:"min=0,max=2000"`
	HandType  string `json:"handType,omitempty"` // 新增：手型类型
	HandId    uint32 `json:"handId,omitempty"`   // 新增：CAN ID
}

// HandTypeRequest 手型设置请求
type HandTypeRequest struct {
	Interface string `json:"interface" binding:"required"`
	HandType  string `json:"handType" binding:"required,oneof=left right"`
	HandId    uint32 `json:"handId" binding:"required"`
}
