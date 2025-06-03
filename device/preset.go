package device

// PresetPose 定义预设姿势的结构
type PresetPose struct {
	Name        string // 姿势名称
	Description string // 姿势描述
	FingerPose  []byte // 手指姿态数据
	PalmPose    []byte // 手掌姿态数据（可选）
}

// PresetManager 预设姿势管理器
type PresetManager struct{ presets map[string]PresetPose }

// NewPresetManager 创建新的预设姿势管理器
func NewPresetManager() *PresetManager {
	return &PresetManager{
		presets: make(map[string]PresetPose),
	}
}

// RegisterPreset 注册一个预设姿势
func (pm *PresetManager) RegisterPreset(preset PresetPose) { pm.presets[preset.Name] = preset }

// GetPreset 获取指定名称的预设姿势
func (pm *PresetManager) GetPreset(name string) (PresetPose, bool) {
	preset, exists := pm.presets[name]
	return preset, exists
}

// GetSupportedPresets 获取所有支持的预设姿势名称列表
func (pm *PresetManager) GetSupportedPresets() []string {
	presets := make([]string, 0, len(pm.presets))
	for name := range pm.presets {
		presets = append(presets, name)
	}
	return presets
}

// GetPresetDescription 获取预设姿势的描述
func (pm *PresetManager) GetPresetDescription(name string) string {
	if preset, exists := pm.presets[name]; exists {
		return preset.Description
	}
	return ""
}
