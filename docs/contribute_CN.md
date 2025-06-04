# 当前架构详解

## 设备抽象层 (device 包)

目标：统一不同型号设备的操作接口，屏蔽底层硬件差异（主要体现在指令到 RawMessage 的转换和设备特定功能的实现上）。

核心接口与结构体：

**Device 接口 (device/device.go): 代表一个可控制的设备单元。**

```go
type Device interface {
    GetID() string
    GetModel() string
    GetHandType() define.HandType
    SetHandType(handType define.HandType) error
    ExecuteCommand(cmd Command) error
    ReadSensorData(sensorID string) (SensorData, error)
    GetComponents(componentType ComponentType) []Component
    GetStatus() (DeviceStatus, error)
    Connect() error
    Disconnect() error

    PoseExecutor // 嵌入 PoseExecutor 接口
    GetAnimationEngine() *AnimationEngine

    GetSupportedPresets() []string
    ExecutePreset(presetName string) error
    GetPresetDescription(presetName string) string
    GetPresetDetails(presetName string) (PresetPose, bool)
}
```

**PoseExecutor 接口 (device/pose_executor.go): 定义了执行基本姿态指令的能力。**

```go
type PoseExecutor interface {
    SetFingerPose(pose []byte) error
    SetPalmPose(pose []byte) error
    ResetPose() error
    GetHandType() define.HandType
}
```

**Command 接口 (device/device.go): 代表一个发送给设备的指令。**

```go
type Command interface {
    Type() string
    Payload() []byte
    TargetComponent() string // 目标组件 ID
}
```

具体指令实现位于 device/commands.go，如 FingerPoseCommand, PalmPoseCommand, GenericCommand。

**SensorData 接口 (device/device.go): 代表从传感器读取的数据。**

```go
type SensorData interface {
    Timestamp() time.Time
    Values() map[string]any
    SensorID() string
}
```

**ComponentType (device/device.go): 定义组件类型。**

```go
const (
    SensorComponent   ComponentType = "sensor"
    SkinComponent     ComponentType = "skin"     // 示例，可扩展
    ActuatorComponent ComponentType = "actuator" // 示例，可扩展
)
```

**Component 接口 (device/device.go): 代表设备的一个可插拔组件。**

```go
type Component interface {
    GetID() string
    GetType() ComponentType
    GetConfiguration() map[string]interface{}
    IsActive() bool
}
```

**具体设备型号实现 (如 device/models/l10.go 中的 L10Hand):**

1. 实现 Device 和 PoseExecutor 接口。
2. 管理内部的 AnimationEngine 和 PresetManager。
3. 包含将通用 Command 转换为发送给 can-bridge 的 RawMessage 的逻辑 (如 commandToRawMessage 方法)。
4. 管理其配备的传感器等组件 (initializeComponents 方法)。

**DeviceManager (device/manager.go): 用于注册、发现和管理可用的设备实例。**

```go
type DeviceManager struct { /* ... */ }
func NewDeviceManager() *DeviceManager { /* ... */ }
func (m *DeviceManager) RegisterDevice(dev Device) error { /* ... */ }
func (m *DeviceManager) GetDevice(id string) (Device, error) { /* ... */ }
```

## 组件化设计 (component 包)

目标：将“皮肤”、“传感器”等视为可配置、可替换的组件。

核心接口与结构体：

**传感器组件 (Sensor):**

component/sensor.go 中定义了通用的 Sensor 接口 (嵌入了 device.Component)。

```go
type Sensor interface {
    device.Component
    ReadData() (device.SensorData, error)
    GetDataType() string
    GetSamplingRate() int
    SetSamplingRate(rate int) error
}
```

具体的传感器实现，如 component/component.go 中的 PressureSensor，实现了 Sensor 接口。

传感器数据的实际获取方式（模拟、通过 can-bridge 的特定端点，或完全独立的数据源）在具体的 Sensor 组件实现中处理。

SensorDataImpl (component/sensor.go) 是 device.SensorData 的一个具体实现。

皮肤组件 (Skin) 及其他组件：

如果“皮肤”影响设备的物理特性或参数范围，可以将其抽象为一个 Skin 组件，实现 device.Component 接口。

设备可以关联多个不同类型的组件，并在其 initializeComponents 方法中进行初始化。

## 动画与姿态控制

目标：提供灵活的动画播放和直接的姿态控制能力，与具体设备和通信方式解耦。

**AnimationEngine (device/engine.go):**

每个设备实例拥有一个 AnimationEngine。

负责注册、启动、停止和管理动画的生命周期。

**使用 PoseExecutor 来执行动画中的姿态变化。**

```go
type AnimationEngine struct { /* ... */ }
func NewAnimationEngine(executor PoseExecutor) *AnimationEngine { /* ... */ }
func (e *AnimationEngine) Register(anim Animation) { /* ... */ }
func (e *AnimationEngine) Start(name string, speedMs int) error { /* ... */ }
func (e *AnimationEngine) Stop() error { /* ... */ }
```

Animation 接口 (device/animation.go): 定义了动画的行为。

```go
type Animation interface {
    Run(executor PoseExecutor, stop <-chan struct{}, speedMs int) error
    Name() string
}
```

具体的动画实现与设备型号绑定，例如 device/models/l10_animation.go 中的 L10WaveAnimation。

直接姿态控制：

通过设备实例直接调用其实现的 PoseExecutor 接口方法 (SetFingerPose, SetPalmPose, ResetPose)。

或者通过构造 FingerPoseCommand 或 PalmPoseCommand，然后调用 device.ExecuteCommand()。

预设姿势 (PresetManager - device/preset.go):

每个设备实例拥有一个 PresetManager。

负责注册和管理预设姿势 (PresetPose 结构体)。

Device 接口提供了 GetSupportedPresets, ExecutePreset, GetPresetDescription 方法与预设姿势交互。

## 通信层抽象 (communication 包)

目标：将与 can-bridge Web 服务的 HTTP 通信细节封装起来，对上层透明。

RawMessage 结构体 (communication/communicator.go): 匹配 can-bridge 服务期望的 JSON 格式。

```go
type RawMessage struct {
    Interface string `json:"interface"`
    ID        uint32 `json:"id"`
    Data      []byte `json:"data"`
}
```

Communicator 接口 (communication/communicator.go): 定义了与 can-bridge Web 服务进行通信的接口。

```go
type Communicator interface {
    SendMessage(ctx context.Context, msg RawMessage) error
    GetInterfaceStatus(ifName string) (isActive bool, err error)
    GetAllInterfaceStatuses() (statuses map[string]bool, err error)
    SetServiceURL(url string)
    IsConnected() bool
}
```

**CanBridgeClient (communication/communicator.go): Communicator 接口的实现。**

1. 内部使用标准的 net/http 包与 can-bridge 服务交互。
2. 负责构造 HTTP 请求 (POST 到 /api/can 用于发送，GET 到 /api/status/* 用于状态检查)。
3. 处理 JSON 序列化/反序列化以及 HTTP 错误。
4. 需要配置 can-bridge 服务的 URL。

具体设备实现 (如 L10Hand) 依赖此 Communicator 接口来发送指令。

## 指令生成与解析

指令生成：上层逻辑（如动画、直接控制）创建 device.Command 类型的对象 (如 NewFingerPoseCommand(...))。

设备的 ExecuteCommand 方法接收此 Command。

设备内部的 commandToRawMessage (或类似) 方法将通用的 Command 转换为特定于该型号的 RawMessage（包含正确的 Interface, ID, Data）。

传感器数据解析：

L10Hand 的 ReadSensorData 方法委托给相应的 Sensor 组件。

Sensor 组件的 ReadData 方法负责获取原始数据（如果通过 CAN，则可能需要 Communicator 支持读取功能，目前 can-bridge 主要用于发送）并将其解析为高层可理解的 SensorData。当前实现中，PressureSensor 是模拟数据。

## 配置与注册

设备工厂 (device/factory.go):

使用 DeviceFactory (defaultFactory) 来创建不同型号的 Device 实例。

RegisterDeviceType(modelName string, constructor func(config map[string]any) (Device, error)): 注册新的设备型号及其构造函数。

CreateDevice(modelName string, config map[string]any) (Device, error): 根据型号和配置创建设备实例。

设备构造函数 (如 NewL10Hand) 接收一个 map[string]any 类型的配置参数。

动画和预设姿势注册：

动画通过 AnimationEngine.Register() 在设备实例化时注册。

预设姿势通过 PresetManager.RegisterPreset() 在设备实例化时注册。

## 如何添加新的设备实现

要添加对新型号设备（例如 "L20"）的支持，请遵循以下步骤：

### 创建设备模型文件：

在 device/models/ 目录下为新设备创建一个 Go 文件，例如 l20.go。

如果需要设备特定的动画，创建 l20_animation.go。

如果需要设备特定的预设姿势，创建 l20_presets.go。

定义设备结构体 (l20.go):

```go
package models

import (
    "context"
    "fmt"
    "log"
    "sync"
    "time"
    // ... 其他必要的 import
    "hands/communication"
    "hands/component" // 如果需要自定义组件或使用现有组件
    "hands/define"
    "hands/device"
)

type L20Hand struct {
    id              string
    model           string
    handType        define.HandType
    communicator    communication.Communicator
    components      map[device.ComponentType][]device.Component
    status          device.DeviceStatus
    mutex           sync.RWMutex
    canInterface    string
    animationEngine *device.AnimationEngine
    presetManager   *device.PresetManager
    // ... L20 特有的字段
}
```

实现构造函数 (NewL20Hand):

```go
func NewL20Hand(config map[string]any) (device.Device, error) {
    // 1. 解析配置 (id, can_service_url, can_interface, hand_type 等)
    // ...

    // 2. 创建 communicator
    comm := communication.NewCanBridgeClient(serviceURL) // serviceURL from config

    hand := &L20Hand{
        id:           id, // from config
        model:        "L20",
        handType:     handType, // from config or default
        communicator: comm,
        components:   make(map[device.ComponentType][]device.Component),
        canInterface: canInterface, // from config or default
        status:       device.DeviceStatus{ /* initial status */ },
        // ... 初始化 L20 特有字段
    }

    // 3. 初始化 AnimationEngine
    hand.animationEngine = device.NewAnimationEngine(hand) // hand 实现了 PoseExecutor
    // 注册 L20 特定的动画 (见步骤 6)
    // hand.animationEngine.Register(NewL20WaveAnimation()) // 示例

    // 4. 初始化 PresetManager
    hand.presetManager = device.NewPresetManager()
    // 注册 L20 特定的预设姿势 (见步骤 7)
    // for _, preset := range GetL20Presets() { hand.presetManager.RegisterPreset(preset) } // 示例

    // 5. 初始化组件
    if err := hand.initializeComponents(config); err != nil {
        return nil, fmt.Errorf("L20 初始化组件失败：%w", err)
    }

    log.Printf("✅ 设备 L20 (%s, %s) 创建成功", hand.id, hand.handType.String())
    return hand, nil
}
```

**实现 device.Device 和 device.PoseExecutor 接口：**

基本方法：GetID(), GetModel(), GetHandType(), SetHandType(), GetStatus(), Connect(), Disconnect()。这些通常比较直接。

**PoseExecutor 方法：**

1. SetFingerPose(pose []byte) error
2. SetPalmPose(pose []byte) error
3. ResetPose() error

这些方法内部会调用 ExecuteCommand，或者直接构造 RawMessage 发送（如果 L20 的姿态设置非常特殊）。通常建议通过 ExecuteCommand。

```go
ExecuteCommand(cmd device.Command) error:

func (h *L20Hand) ExecuteCommand(cmd device.Command) error {
    h.mutex.Lock()
    defer h.mutex.Unlock()
    // 1. 检查设备状态
    // 2. 调用 h.commandToRawMessage(cmd) 将通用指令转换为 L20 特定的 RawMessage
    // 3. 使用 h.communicator.SendMessage(ctx, rawMsg) 发送
    // 4. 更新设备状态和日志
    return nil // or error
}
```

commandToRawMessage(cmd device.Command) (communication.RawMessage, error): 这个辅助方法是设备差异化的关键。它需要根据 L20 的 CAN 协议，将 cmd.Type() 和 cmd.Payload() 转换为正确的 RawMessage.ID 和 RawMessage.Data。

组件和传感器方法：ReadSensorData(), GetComponents()。

动画和预设方法：GetAnimationEngine(), GetSupportedPresets(), ExecutePreset(), GetPresetDescription()。这些通常直接委托给内部的 animationEngine 和 presetManager。

实现设备特定逻辑：initializeComponents(config map[string]any) error: 根据 L20 的硬件配置，创建并注册其传感器、执行器等组件到 h.components。

```go
func (h *L20Hand) initializeComponents(config map[string]any) error {
    // 示例：添加一个 L20 特有的传感器
    // l20Sensor := component.NewL20SpecificSensor("l20_sensor_1", nil)
    // h.components[device.SensorComponent] = append(h.components[device.SensorComponent], l20Sensor)
    return nil
}
```

添加设备特定动画 (l20_animation.go):

定义实现 device.Animation 接口的动画结构体，如 L20WaveAnimation。

在 NewL20Hand 中，使用 hand.animationEngine.Register(NewL20WaveAnimation()) 注册它们。

添加设备特定预设姿势 (l20_presets.go):

定义一个函数如 GetL20Presets() []device.PresetPose，返回 L20 的预设姿势列表。

在 NewL20Hand 中，遍历这些预设并使用 hand.presetManager.RegisterPreset(preset) 注册它们。

注册设备类型：

在 device/models/init.go 的 RegisterDeviceTypes() 函数中添加一行：

device.RegisterDeviceType("L20", NewL20Hand)

## 如何添加新的动画/预设姿势

这里主要指实现项目已定义的 Go 接口，如 device.Animation 或 component.Sensor。

### 添加新的动画 (实现 device.Animation)

定义动画结构体：在设备模型相关的动画文件内 (例如，若为 L10 添加新动画，则在 device/models/l10_animation.go 中)，或为通用动画创建新文件。

示例：

```go
// device/models/l10_animation.go
type L10GreetingAnimation struct{}

func NewL10GreetingAnimation() *L10GreetingAnimation { return &L10GreetingAnimation{} }
```

实现 device.Animation 接口：

```go
func (a *L10GreetingAnimation) Name() string { return "greeting" }

func (a *L10GreetingAnimation) Run(executor device.PoseExecutor, stop <-chan struct{}, speedMs int) error {
    log.Printf("Running %s animation on %s", a.Name(), executor.GetHandType())
    delay := time.Duration(speedMs) * time.Millisecond

    // 示例：挥手动作
    poses := [][]byte{
        {192, 192, 192, 192, 192, 192}, // 张开
        {160, 160, 160, 160, 160, 160}, // 稍弯曲
    }
    palmPoses := [][]byte{
        {100, 128, 128, 128}, // 手掌姿态 1
        {150, 128, 128, 128}, // 手掌姿态 2
    }

    for i := 0; i < 3; i++ { // 重复几次
        for j, pose := range poses {
            if err := executor.SetFingerPose(pose); err != nil { return err }
            if err := executor.SetPalmPose(palmPoses[j%len(palmPoses)]); err != nil { return err } // 循环使用手掌姿态

            select {
            case <-stop:
                log.Printf("%s animation stopped.", a.Name())
                return nil
            case <-time.After(delay):
                // continue
            }
        }
    }
    return nil
}
```

注册动画：在对应设备的构造函数中 (例如 NewL10Hand)，获取 AnimationEngine 实例并注册新动画：

```go
// 在 NewL10Hand 中：
hand.animationEngine.Register(NewL10GreetingAnimation())
```

###  添加新的传感器类型 (实现 component.Sensor 和 device.Component)

定义传感器结构体：

在 component/ 目录下创建新文件，例如 temperature_sensor.go。

定义结构体：

```go
// component/temperature_sensor.go
package component

import (
    "hands/device"
    "math/rand/v2"
    "time"
    "fmt"
)

type TemperatureSensor struct {
    id           string
    config       map[string]any
    isActive     bool
    samplingRate int // Hz
}

func NewTemperatureSensor(id string, config map[string]any) Sensor { // 返回 Sensor 接口
    return &TemperatureSensor{
        id:           id,
        config:       config,
        isActive:     true,
        samplingRate: 1, // 默认 1Hz
    }
}
```

实现 device.Component 接口：

```go
func (ts *TemperatureSensor) GetID() string                { return ts.id }
func (ts *TemperatureSensor) GetType() device.ComponentType  { return device.SensorComponent }
func (ts *TemperatureSensor) GetConfiguration() map[string]any { return ts.config }
func (ts *TemperatureSensor) IsActive() bool               { return ts.isActive }
```

实现 component.Sensor 接口：

```go
func (ts *TemperatureSensor) ReadData() (device.SensorData, error) {
    if !ts.isActive {
        return nil, fmt.Errorf("sensor %s is not active", ts.id)
    }
    // 模拟读取温度数据
    tempValue := 20.0 + rand.Float64()*15.0 // 20-35 度
    values := map[string]any{
        "temperature": tempValue,
        "unit":        "Celsius",
    }
    return NewSensorData(ts.id, values), nil // 使用 component.NewSensorData
}

func (ts *TemperatureSensor) GetDataType() string { return "temperature" }

func (ts *TemperatureSensor) GetSamplingRate() int { return ts.samplingRate }

func (ts *TemperatureSensor) SetSamplingRate(rate int) error {
    if rate <= 0 {
        return fmt.Errorf("sampling rate must be positive")
    }
    ts.samplingRate = rate
    return nil
}
```

集成到设备：在具体设备模型 (如 L10Hand 或 L20Hand) 的 initializeComponents 方法中，创建并添加此传感器的实例：

```go
// 在 L10Hand.initializeComponents 中：
tempSensor1 := component.NewTemperatureSensor("temp_palm", map[string]any{"location": "palm"})
h.components[device.SensorComponent] = append(h.components[device.SensorComponent], tempSensor1)
```

### 如何添加新的 Component

添加一个新的通用组件（非特指传感器）与添加传感器类似，主要区别在于它可能不会实现 component.Sensor 接口，而是直接实现 device.Component 以及任何该组件特有的接口。

定义组件类型 (如果需要新的 ComponentType): 在 device/device.go 中为新的组件类型添加一个常量：

```go
const (
    // ...
    MyCustomComponentType ComponentType = "my_custom_type"
)
```

定义组件特定接口：如果该组件有特定行为，可以在 component/ 目录下或与组件实现同文件中定义一个接口：

```go
// component/my_custom_component.go
package component

import "hands/device"

type MyCustomFunctionality interface {
    PerformAction(param string) (string, error)
}
```

定义组件结构体：在 component/ 目录下创建新文件，例如 my_custom_component.go。

定义结构体：

```go
type MyCustomComponent struct {
    id       string
    config   map[string]any
    isActive bool
    // ... 其他字段
}

func NewMyCustomComponent(id string, config map[string]any) device.Component { // 返回 device.Component
    return &MyCustomComponent{
        id:       id,
        config:   config,
        isActive: true,
    }
}
```

实现 device.Component 接口：

```go
func (mcc *MyCustomComponent) GetID() string                { return mcc.id }
func (mcc *MyCustomComponent) GetType() device.ComponentType  { return MyCustomComponentType } // 使用新定义的类型
func (mcc *MyCustomComponent) GetConfiguration() map[string]any { return mcc.config }
func (mcc *MyCustomComponent) IsActive() bool               { return mcc.isActive }
```

实现组件特定接口：

```go
// 确保 MyCustomComponent 也实现了 MyCustomFunctionality
func (mcc *MyCustomComponent) PerformAction(param string) (string, error) {
    // 实现特定功能
    return "Action performed with " + param, nil
}
```

在这种情况下，NewMyCustomComponent 的返回类型可能需要同时满足 device.Component 和 MyCustomFunctionality，或者在使用时进行类型断言。一个常见的做法是返回具体类型指针 *MyCustomComponent，它自然实现了所有嵌入或直接定义的方法。或者，如果希望返回接口，可以返回 device.Component，然后在需要特定功能时进行类型断言。

集成到设备：在具体设备模型的 initializeComponents 方法中，创建并添加此组件的实例：

```go
// 在 L10Hand.initializeComponents 中：
customComp := component.NewMyCustomComponent("custom_1", map[string]any{"setting": "value"})
h.components[component.MyCustomComponentType] = append(h.components[component.MyCustomComponentType], customComp)
```

设备代码可能需要通过 GetComponents(component.MyCustomComponentType) 获取这些组件，并进行类型断言以调用其特定方法：

```go
comps := h.GetComponents(component.MyCustomComponentType)
for _, comp := range comps {
    if customComp, ok := comp.(component.MyCustomFunctionality); ok { // 或 *component.MyCustomComponent
        result, err := customComp.PerformAction("test")
        // ...
    }
}
```
