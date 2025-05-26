# Dexterous Hand Dashboard 项目文档

## 项目概述

**Dexterous Hand Dashboard** 是专为 LinkerHand 灵巧手设备开发的控制仪表盘服务。该服务基于 Golang 开发，提供灵活的 RESTful API 接口，可实现手指与掌部姿态控制、预设动作执行及实时传感器数据监控，并支持动态配置手型（左手或右手）及 CAN 接口。

## 功能特性

* **动态手型配置**：支持左手和右手手型的动态切换。
* **灵活接口配置**：支持多种 CAN 接口（如 `can0`, `can1`），可通过命令行参数或环境变量动态设置。
* **手指与掌部姿态控制**：提供手指（6字节）和掌部（4字节）姿态数据发送功能。
* **预设动作执行**：内置丰富的手势动作，如握拳、张开、捏取、点赞、数字手势等。
* **实时动画控制**：支持波浪、横向摆动等动画效果，用户可动态启动和停止。
* **传感器数据实时监控**：提供接口压力数据的实时模拟和更新。
* **健康检查与服务监控**：实时监控 CAN 服务状态和接口活跃情况。

## API 接口

### 手型配置

* `POST /api/hand-type`

设置接口对应手型。

### 手指姿态

* `POST /api/fingers`

发送手指姿态数据。

### 掌部姿态

* `POST /api/palm`

发送掌部姿态数据。

### 预设姿势

* `POST /api/preset/{pose}`

执行预定义的姿势。

### 动画控制

* `POST /api/animation`

启动或停止动画效果。

### 传感器数据

* `GET /api/sensors`

获取指定接口或所有接口的实时传感器数据。

### 系统状态

* `GET /api/status`

查询系统整体运行状态、CAN 服务状态及接口配置信息。

### 可用接口列表

* `GET /api/interfaces`

获取当前可用的 CAN 接口列表。

### 手型配置查询

* `GET /api/hand-configs`

查询所有接口的手型配置。

### 健康检查

* `GET /api/health`

系统健康检查端点。

## 配置选项

通过命令行参数或环境变量进行配置：

* `CAN_SERVICE_URL` 或 `-can-url`：设置 CAN 服务的 URL。
* `WEB_PORT` 或 `-port`：设置 Web 服务端口。
* `DEFAULT_INTERFACE` 或 `-interface`：默认 CAN 接口。
* `CAN_INTERFACES` 或 `-can-interfaces`：配置可用的 CAN 接口列表。

## 使用示例

```bash
./control-service -can-interfaces can0,can1,vcan0
CAN_INTERFACES=can0,can1 ./control-service
```

## 系统运行要求

* Golang 环境 (1.20+)
* CAN 通信服务

## 启动方式

启动控制服务：

```bash
go run main.go -can-url http://localhost:8080 -port 9099
```

## 日志与监控

服务提供详尽的日志输出，包括接口状态、动作发送情况及错误提示，便于快速诊断与排查问题。

## 贡献指南

欢迎社区开发者贡献代码、报告 Bug 或提交功能建议。

## 许可证

本项目使用 GPL-3.0 license 开源许可。
