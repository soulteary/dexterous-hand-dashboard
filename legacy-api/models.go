package api

type FingerPoseRequest struct {
	Interface string `json:"interface,omitempty"`
	Pose      []byte `json:"pose" binding:"required,len=6"`
	HandType  string `json:"handType,omitempty"` // 新增：手型类型
	HandId    uint32 `json:"handId,omitempty"`   // 新增：CAN ID
}

type PalmPoseRequest struct {
	Interface string `json:"interface,omitempty"`
	Pose      []byte `json:"pose" binding:"required,len=4"`
	HandType  string `json:"handType,omitempty"` // 新增：手型类型
	HandId    uint32 `json:"handId,omitempty"`   // 新增：CAN ID
}

type AnimationRequest struct {
	Interface string `json:"interface,omitempty"`
	Type      string `json:"type" binding:"required,oneof=wave sway stop"`
	Speed     int    `json:"speed" binding:"min=0,max=2000"`
	HandType  string `json:"handType,omitempty"` // 新增：手型类型
	HandId    uint32 `json:"handId,omitempty"`   // 新增：CAN ID
}

type HandTypeRequest struct {
	Interface string `json:"interface" binding:"required"`
	HandType  string `json:"handType" binding:"required,oneof=left right"`
	HandId    uint32 `json:"handId" binding:"required"`
}
