package device

// FingerPoseCommand 手指姿态指令
type FingerPoseCommand struct {
	fingerID   string
	poseData   []byte
	targetComp string
}

func NewFingerPoseCommand(fingerID string, poseData []byte) *FingerPoseCommand {
	return &FingerPoseCommand{
		fingerID:   fingerID,
		poseData:   poseData,
		targetComp: "finger_" + fingerID,
	}
}

func (c *FingerPoseCommand) Type() string {
	return "SetFingerPose"
}

func (c *FingerPoseCommand) Payload() []byte {
	return c.poseData
}

func (c *FingerPoseCommand) TargetComponent() string {
	return c.targetComp
}

// PalmPoseCommand 手掌姿态指令
type PalmPoseCommand struct {
	poseData   []byte
	targetComp string
}

func NewPalmPoseCommand(poseData []byte) *PalmPoseCommand {
	return &PalmPoseCommand{
		poseData:   poseData,
		targetComp: "palm",
	}
}

func (c *PalmPoseCommand) Type() string {
	return "SetPalmPose"
}

func (c *PalmPoseCommand) Payload() []byte {
	return c.poseData
}

func (c *PalmPoseCommand) TargetComponent() string {
	return c.targetComp
}

// GenericCommand 通用指令
type GenericCommand struct {
	cmdType    string
	payload    []byte
	targetComp string
}

func NewGenericCommand(cmdType string, payload []byte, targetComp string) *GenericCommand {
	return &GenericCommand{
		cmdType:    cmdType,
		payload:    payload,
		targetComp: targetComp,
	}
}

func (c *GenericCommand) Type() string {
	return c.cmdType
}

func (c *GenericCommand) Payload() []byte {
	return c.payload
}

func (c *GenericCommand) TargetComponent() string {
	return c.targetComp
}
