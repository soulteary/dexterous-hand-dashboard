package device

// FingerPoseCommand 手指姿态指令
type FingerPoseCommand struct{ poseData []byte }

func NewFingerPoseCommand(fingerID string, poseData []byte) *FingerPoseCommand {
	return &FingerPoseCommand{poseData: poseData}
}

func (c *FingerPoseCommand) Type() string { return "SetFingerPose" }

func (c *FingerPoseCommand) Payload() []byte { return c.poseData }

// PalmPoseCommand 手掌姿态指令
type PalmPoseCommand struct{ poseData []byte }

func NewPalmPoseCommand(poseData []byte) *PalmPoseCommand {
	return &PalmPoseCommand{poseData: poseData}
}

func (c *PalmPoseCommand) Type() string { return "SetPalmPose" }

func (c *PalmPoseCommand) Payload() []byte { return c.poseData }

// GenericCommand 通用指令
type GenericCommand struct {
	cmdType string
	payload []byte
}

func NewGenericCommand(cmdType string, payload []byte, targetComp string) *GenericCommand {
	return &GenericCommand{
		cmdType: cmdType,
		payload: payload,
	}
}

func (c *GenericCommand) Type() string { return c.cmdType }

func (c *GenericCommand) Payload() []byte { return c.payload }
