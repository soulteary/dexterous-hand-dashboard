// 全局变量
let availableInterfaces = [];
let interfaceStatus = {};
let handConfigs = {}; // 存储每个手的配置
const handTypeIds = {
  left: 0x28, // HAND_TYPE_LEFT
  right: 0x27, // HAND_TYPE_RIGHT
};

// 主要控制模块
const LinkerHandController = {
  // 常量定义
  DEFAULTS: {
    FINGER: {
      OPEN: 64, // 完全张开值
      CLOSED: 192, // 完全闭合值
      NEUTRAL: 128, // 中间值
    },
    PALM: {
      NEUTRAL: 128, // 中间值
      LEFT: 48, // 左侧
      RIGHT: 208, // 右侧
    },
    ANIMATION: {
      DEFAULT_SPEED: 500, // 默认动画速度
    },
  },

  // 预设姿势配置
  PRESETS: {
    FIST: [64, 64, 64, 64, 64, 64], // 握拳
    OPEN: [192, 192, 192, 192, 192, 192], // 张开
    THUMBSUP: [255, 255, 0, 0, 0, 0], // 竖起大拇指
    POINT: [0, 0, 255, 0, 0, 0], // 食指指点
    YO: [255, 255, 255, 0, 0, 255], // Yo!
    GUN: [255, 255, 255, 255, 0, 0], // PONG!
    WAVE: [40, 60, 80, 100, 120, 140], // 波浪形
    PALM_LEFT: [48, 48, 48, 48], // 掌部左移
    PALM_RIGHT: [208, 208, 208, 208], // 掌部右移
    PALM_NEUTRAL: [128, 128, 128, 128], // 掌部中立
    PALM_GUN: [0, 0, 0, 128], // 掌部 GUN

    PINCH: [114, 63, 136, 0, 0, 0], // 捏取姿势
    PALM_PINCH: [255, 163, 255, 127],

    OK: [124, 31, 132, 255, 255, 255],
    PALM_OK: [255, 163, 255, 127],

    BIG_FIST: [49, 32, 40, 36, 41, 46], // 大握拳
    PALM_BIG_FIST: [255, 235, 128, 128], // 大握拳掌部

    BIG_OPEN: [255, 255, 255, 255, 255, 255], // 大张开
    PALM_BIG_OPEN: [128, 128, 128, 128], // 大张开掌部

    YEAH: [0, 103, 255, 255, 0, 0], // Yeah!
    PALM_YEAH: [255, 235, 128, 128], // Yeah! 掌部

    // 数字手势预设
    ONE: [0, 57, 255, 0, 0, 0],
    PALM_ONE: [255, 109, 255, 118],
    TWO: [0, 57, 255, 255, 0, 0],
    PALM_TWO: [255, 109, 255, 118],
    THREE: [0, 57, 255, 255, 255, 0],
    PALM_THREE: [255, 109, 255, 118],
    FOUR: [0, 57, 255, 255, 255, 255],
    PALM_FOUR: [255, 109, 255, 118],
    FIVE: [255, 255, 255, 255, 255, 255],
    PALM_FIVE: [255, 109, 255, 118],
    SIX: [255, 255, 0, 0, 0, 255],
    PALM_SIX: [255, 255, 255, 255],
    SEVEN: [110, 137, 130, 109, 0, 0],
    PALM_SEVEN: [255, 200, 199, 76],
    EIGHT: [216, 240, 255, 36, 41, 46],
    PALM_EIGHT: [106, 200, 199, 76],
    NINE: [0, 255, 159, 0, 0, 0],
    PALM_NINE: [255, 38, 195, 51],
  },

  // 防抖函数
  debounce: (func, delay) => {
    let timer;
    return () => {
      clearTimeout(timer);
      timer = setTimeout(func, delay);
    };
  },

  // 初始化滑块显示与实时控制发送（带防抖）
  initSliderDisplays: function () {
    const fingerSliders = Array.from({ length: 6 }, (_, i) =>
      document.getElementById(`finger${i}`),
    );
    const palmSliders = Array.from({ length: 4 }, (_, i) =>
      document.getElementById(`palm${i}`),
    );
    const delayDefault = 30;

    const updateFingerPose = this.debounce(() => {
      const pose = this.getFingerPoseValues();
      this.sendFingerPoseToAll(pose);
    }, delayDefault);

    const updatePalmPose = this.debounce(() => {
      const pose = this.getPalmPoseValues();
      this.sendPalmPoseToAll(pose);
    }, delayDefault);

    // 初始化手指滑块监听器
    fingerSliders.forEach((slider, i) => {
      slider.addEventListener("input", () => {
        document.getElementById(`finger${i}-value`).textContent = slider.value;
        updateFingerPose();
      });
    });

    // 初始化掌部滑块监听器
    palmSliders.forEach((slider, i) => {
      slider.addEventListener("input", () => {
        document.getElementById(`palm${i}-value`).textContent = slider.value;
        updatePalmPose();
      });
    });

    // 动画速度滑块更新
    const animationSlider = document.getElementById("animation-speed");
    animationSlider.addEventListener("input", function () {
      document.getElementById("speed-value").textContent = this.value;
    });
  },

  // 获取手指姿态值
  getFingerPoseValues: () => {
    const pose = [];
    for (let i = 0; i < 6; i++) {
      pose.push(Number.parseInt(document.getElementById(`finger${i}`).value));
    }
    return pose;
  },

  // 获取掌部姿态值
  getPalmPoseValues: () => {
    const pose = [];
    for (let i = 0; i < 4; i++) {
      pose.push(Number.parseInt(document.getElementById(`palm${i}`).value));
    }
    return pose;
  },

  // 设置手指滑块值
  applyFingerPreset: (values) => {
    if (!Array.isArray(values) || values.length !== 6) {
      logMessage("error", "无效的手指预设值");
      return;
    }

    // 设置滑块值
    for (let i = 0; i < 6; i++) {
      const slider = document.getElementById(`finger${i}`);
      slider.value = values[i];
      document.getElementById(`finger${i}-value`).textContent = values[i];
    }

    logMessage("info", "已应用手指预设姿势");
  },

  // 设置掌部滑块值
  applyPalmPreset: (values) => {
    if (!Array.isArray(values) || values.length !== 4) {
      logMessage("error", "无效的掌部预设值");
      return;
    }

    // 设置滑块值
    for (let i = 0; i < 4; i++) {
      const slider = document.getElementById(`palm${i}`);
      slider.value = values[i];
      document.getElementById(`palm${i}-value`).textContent = values[i];
    }

    logMessage("info", "已应用掌部预设姿势");
  },

  // 发送手指姿态到所有启用手部
  sendFingerPoseToAll: (pose) => {
    const enabledHands = getEnabledHands();
    if (enabledHands.length === 0) {
      logMessage("error", "没有启用的手部");
      return;
    }

    logMessage(
      "info",
      `发送手指姿态到 ${enabledHands.length} 个启用的手部：[${pose.join(
        ", ",
      )}]`,
    );

    enabledHands.forEach(async (config) => {
      await sendFingerPoseToHand(config, pose);
    });
  },

  // 发送掌部姿态到所有启用手部
  sendPalmPoseToAll: (pose) => {
    const enabledHands = getEnabledHands();
    if (enabledHands.length === 0) {
      logMessage("error", "没有启用的手部");
      return;
    }

    logMessage(
      "info",
      `发送掌部姿态到 ${enabledHands.length} 个启用的手部：[${pose.join(
        ", ",
      )}]`,
    );

    enabledHands.forEach(async (config) => {
      await sendPalmPoseToHand(config, pose);
    });
  },

  // 启动传感器数据轮询
  startSensorDataPolling: function () {
    // 立即获取一次数据
    this.fetchSensorData();

    // 设置定时获取
    setInterval(() => {
      this.fetchSensorData();
    }, 2000); // 每 2 秒更新一次
  },

  // 获取传感器数据
  fetchSensorData: function () {
    fetch("/api/legacy/sensors")
      .then((response) => response.json())
      .then((data) => {
        if (data.status === "success") {
          this.updateSensorDisplay(data.data);
        }
      })
      .catch((error) => {
        console.error("获取传感器数据失败：", error);
      });
  },

  // 更新传感器显示
  updateSensorDisplay: function (data) {
    const sensorDisplay = document.getElementById("sensor-data");
    if (!sensorDisplay || !data) return;

    // 创建进度条显示
    let html = '<table style="width:100%;">';

    // 手指压力传感器
    html += this.createSensorRow("拇指压力", data.thumb);
    html += this.createSensorRow("食指压力", data.index);
    html += this.createSensorRow("中指压力", data.middle);
    html += this.createSensorRow("无名指压力", data.ring);
    html += this.createSensorRow("小指压力", data.pinky);

    html += "</table>";

    // 更新最后更新时间
    const lastUpdate = new Date(data.lastUpdate).toLocaleTimeString();
    html += `<div style="text-align:right;font-size:0.8em;margin-top:5px;">最后更新：${lastUpdate}</div>`;

    sensorDisplay.innerHTML = html;
  },

  // 创建传感器行
  createSensorRow: (label, value) => {
    if (value === undefined || value === null) value = 0;
    return `<tr>
            <td>${label}</td>
            <td style="filter:blur(10px)"><progress value="${value}" max="100"></progress></td>
            <td style="filter:blur(10px)">${value}%</td>
        </tr>`;
  },
};

// 页面加载时初始化
document.addEventListener("DOMContentLoaded", () => {
  initializeSystem();
  setupEventListeners();
  setupSliderEvents();
  LinkerHandController.initSliderDisplays();
  LinkerHandController.startSensorDataPolling();
  startStatusUpdater();
});

// 初始化系统 - 添加更详细的错误处理和调试
async function initializeSystem() {
  try {
    logMessage("info", "开始初始化系统...");

    // 步骤 1: 加载可用接口
    logMessage("info", "步骤 1/3: 加载可用接口");
    await loadAvailableInterfaces();

    // 验证接口加载是否成功
    if (!availableInterfaces || availableInterfaces.length === 0) {
      throw new Error("未能获取到任何可用接口");
    }

    // 步骤 2: 生成手部配置
    logMessage("info", "步骤 2/3: 生成手部配置");
    generateHandConfigs();

    // 验证手部配置是否成功
    if (!handConfigs || Object.keys(handConfigs).length === 0) {
      throw new Error("未能生成手部配置");
    }

    // 步骤 3: 检查接口状态
    logMessage("info", "步骤 3/3: 检查接口状态");
    await checkAllInterfaceStatus();

    logMessage("success", "系统初始化完成");
  } catch (error) {
    logMessage("error", `系统初始化失败：${error.message}`);
    console.error("InitializeSystem Error:", error);

    // 尝试使用默认配置恢复
    if (!availableInterfaces || availableInterfaces.length === 0) {
      logMessage("info", "尝试使用默认配置恢复...");
      availableInterfaces = ["can0", "can1", "vcan0", "vcan1"];
      generateHandConfigs();
    }
  }
}

// 加载可用接口
async function loadAvailableInterfaces() {
  try {
    logMessage("info", "正在获取可用 CAN 接口...");
    const response = await fetch("/api/legacy/interfaces");

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }

    const data = await response.json();
    if (data.status === "success") {
      availableInterfaces = data.data.availableInterfaces || [];

      logMessage(
        "success",
        `获取到 ${
          availableInterfaces.length
        } 个可用接口：${availableInterfaces.join(", ")}`,
      );
      hideConnectionWarning();
    } else {
      throw new Error(data.error || "获取接口失败");
    }
  } catch (error) {
    logMessage("error", `获取接口失败：${error.message}`);
    showConnectionWarning();
    // 设置默认值
    availableInterfaces = ["can0", "can1", "vcan0", "vcan1"];
  }
}

// 生成手部配置 - 添加调试和错误处理
function generateHandConfigs() {
  const handsGrid = document.getElementById("hands-grid");
  if (!handsGrid) {
    console.error("Hands grid element not found");
    logMessage("error", "无法找到手部配置容器");
    return;
  }

  // 清空现有内容
  handsGrid.innerHTML = "";

  if (!availableInterfaces || availableInterfaces.length === 0) {
    handsGrid.innerHTML =
      '<div style="text-align: center; color: #666; padding: 20px;">没有可用的 CAN 接口</div>';
    logMessage("warning", "没有可用接口，无法生成手部配置");
    return;
  }

  logMessage("info", `为 ${availableInterfaces.length} 个接口生成手部配置...`);

  // 清空现有配置
  handConfigs = {};

  // 为每个接口创建配置项
  availableInterfaces.forEach((iface, index) => {
    const handId = `hand_${iface}`;

    try {
      // 创建默认配置
      handConfigs[handId] = {
        id: handId,
        interface: iface,
        handType: index % 2 === 0 ? "right" : "left", // 交替默认左右手
        enabled: index < 2, // 默认启用前两个
        status: "offline",
      };

      // 创建HTML元素
      const handElement = createHandElement(handConfigs[handId]);
      if (handElement) {
        handsGrid.appendChild(handElement);
      } else {
        throw new Error("创建手部元素失败");
      }
    } catch (error) {
      console.error(`Failed to create hand element for ${iface}:`, error);
      logMessage("error", `创建 ${iface} 的手部配置失败: ${error.message}`);
    }
  });

  // 延迟更新状态，确保DOM完全构建
  setTimeout(() => {
    updateEnabledHandsStatus();
    logMessage(
      "success",
      `成功生成 ${Object.keys(handConfigs).length} 个手部配置`,
    );
  }, 100);
}

// 添加一个安全的 DOM 检查函数
function validateHandElement(handId) {
  const element = document.getElementById(handId);
  if (!element) {
    console.error(`validateHandElement: 找不到元素 ${handId}`);
    return false;
  }

  const requiredElements = [
    `.hand-title`,
    `#${handId}_checkbox`,
    `#${handId}_interface`,
    `#${handId}_handtype`,
    `#${handId}_status_dot`,
    `#${handId}_status_text`,
  ];

  let isValid = true;
  requiredElements.forEach((selector) => {
    const el = selector.startsWith("#")
      ? document.getElementById(selector.slice(1))
      : element.querySelector(selector);

    if (!el) {
      console.error(`validateHandElement: 在 ${handId} 中找不到 ${selector}`);
      isValid = false;
    }
  });

  return isValid;
}

// 增强的错误处理包装器
function safeUpdateHandElement(handId) {
  try {
    if (validateHandElement(handId)) {
      updateHandElement(handId);
    } else {
      logMessage("error", `手部元素 ${handId} 验证失败，跳过更新`);
    }
  } catch (error) {
    console.error(`Error updating hand element ${handId}:`, error);
    logMessage("error", `更新手部元素 ${handId} 时出错：${error.message}`);
  }
}

// 创建手部配置元素
function createHandElement(config) {
  const div = document.createElement("div");
  div.className = `hand-item ${config.enabled ? "enabled" : "disabled"}`;
  div.id = config.id;

  const handEmoji = config.handType === "left" ? "✋" : "🤚";
  const handLabel = config.handType === "left" ? "左手" : "右手";
  const handId = handTypeIds[config.handType];

  // 确保 HTML 结构完整且正确
  div.innerHTML = `
        <div class="hand-header">
            <input type="checkbox" class="hand-checkbox" id="${
              config.id
            }_checkbox" ${config.enabled ? "checked" : ""}>
            <span class="hand-title">${handEmoji} ${
    config.interface
  } - ${handLabel}</span>
        </div>
        <div class="hand-controls">
            <div class="control-group">
                <label class="control-label">CAN 接口</label>
                <select class="hand-select interface-select" id="${
                  config.id
                }_interface">
                    ${availableInterfaces
                      .map(
                        (iface) =>
                          `<option value="${iface}" ${
                            iface === config.interface ? "selected" : ""
                          }>${iface}</option>`,
                      )
                      .join("")}
                </select>
            </div>
            <div class="control-group">
                <label class="control-label">手型 (CAN ID: 0x${handId
                  .toString(16)
                  .toUpperCase()})</label>
                <select class="hand-select hand-type-select" id="${
                  config.id
                }_handtype">
                    <option value="right" ${
                      config.handType === "right" ? "selected" : ""
                    }>🤚 右手 (0x27)</option>
                    <option value="left" ${
                      config.handType === "left" ? "selected" : ""
                    }>✋ 左手 (0x28)</option>
                </select>
            </div>
        </div>
        <div class="hand-status">
            <span class="status-dot loading" id="${
              config.id
            }_status_dot"></span>
            <span id="${config.id}_status_text">检查中...</span>
        </div>
    `;

  // 使用 requestAnimationFrame 确保 DOM 完全渲染后再设置事件监听器
  requestAnimationFrame(() => {
    setTimeout(() => {
      setupHandEventListeners(config.id);
    }, 0);
  });

  return div;
}

// 设置手部事件监听器
function setupHandEventListeners(handId) {
  // 使用更安全的元素获取方式
  const checkbox = document.getElementById(`${handId}_checkbox`);
  const interfaceSelect = document.getElementById(`${handId}_interface`);
  const handTypeSelect = document.getElementById(`${handId}_handtype`);

  // 检查所有必需的元素是否存在
  if (!checkbox) {
    console.error(
      `setupHandEventListeners: 找不到 checkbox - ${handId}_checkbox`,
    );
    return;
  }

  if (!interfaceSelect) {
    console.error(
      `setupHandEventListeners: 找不到 interfaceSelect - ${handId}_interface`,
    );
    return;
  }

  if (!handTypeSelect) {
    console.error(
      `setupHandEventListeners: 找不到 handTypeSelect - ${handId}_handtype`,
    );
    return;
  }

  // 移除现有的事件监听器（如果有的话）
  checkbox.removeEventListener("change", checkbox._changeHandler);
  interfaceSelect.removeEventListener("change", interfaceSelect._changeHandler);
  handTypeSelect.removeEventListener("change", handTypeSelect._changeHandler);

  // 创建新的事件处理器
  checkbox._changeHandler = function () {
    if (handConfigs[handId]) {
      handConfigs[handId].enabled = this.checked;
      updateHandElement(handId);
      updateEnabledHandsStatus();
      logMessage("info", `${handId}: ${this.checked ? "启用" : "禁用"}`);
    }
  };

  interfaceSelect._changeHandler = function () {
    if (handConfigs[handId]) {
      handConfigs[handId].interface = this.value;
      logMessage("info", `${handId}: 接口切换到 ${this.value}`);
      checkSingleInterfaceStatus(handId);
    }
  };

  handTypeSelect._changeHandler = function () {
    if (handConfigs[handId]) {
      handConfigs[handId].handType = this.value;
      updateHandElement(handId);
      const handName = this.value === "left" ? "左手" : "右手";
      const handIdHex = handTypeIds[this.value];
      logMessage(
        "info",
        `${handId}: 切换到${handName}模式 (0x${handIdHex
          .toString(16)
          .toUpperCase()})`,
      );
    }
  };

  // 添加事件监听器
  checkbox.addEventListener("change", checkbox._changeHandler);
  interfaceSelect.addEventListener("change", interfaceSelect._changeHandler);
  handTypeSelect.addEventListener("change", handTypeSelect._changeHandler);
}

// 更新手部元素
function updateHandElement(handId) {
  const config = handConfigs[handId];
  const element = document.getElementById(handId);

  // 添加安全检查
  if (!element || !config) {
    console.warn(`updateHandElement: 找不到元素或配置 - handId: ${handId}`);
    return;
  }

  const handEmoji = config.handType === "left" ? "✋" : "🤚";
  const handLabel = config.handType === "left" ? "左手" : "右手";
  const handIdHex = handTypeIds[config.handType];

  // 更新样式
  element.className = `hand-item ${config.enabled ? "enabled" : "disabled"}`;

  // 安全地更新标题
  const title = element.querySelector(".hand-title");
  if (title) {
    title.textContent = `${handEmoji} ${config.interface} - ${handLabel}`;
  } else {
    console.warn(
      `updateHandElement: 找不到 .hand-title 元素 - handId: ${handId}`,
    );
  }

  // 安全地更新手型标签
  const handTypeLabels = element.querySelectorAll(".control-label");
  if (handTypeLabels.length >= 2) {
    const handTypeLabel = handTypeLabels[1]; // 第二个 label 是手型的
    if (handTypeLabel) {
      handTypeLabel.textContent = `手型 (CAN ID: 0x${handIdHex
        .toString(16)
        .toUpperCase()})`;
    }
  } else {
    console.warn(`updateHandElement: 找不到手型标签 - handId: ${handId}`);
  }

  // 确保选择框的值也同步更新
  const handTypeSelect = document.getElementById(`${handId}_handtype`);
  if (handTypeSelect) {
    handTypeSelect.value = config.handType;
  }

  const interfaceSelect = document.getElementById(`${handId}_interface`);
  if (interfaceSelect) {
    interfaceSelect.value = config.interface;
  }

  const checkbox = document.getElementById(`${handId}_checkbox`);
  if (checkbox) {
    checkbox.checked = config.enabled;
  }
}

// 更新启用手部状态显示
function updateEnabledHandsStatus() {
  const enabledHands = Object.values(handConfigs).filter(
    (config) => config.enabled,
  );
  const statusDiv = document.getElementById("enabled-hands-status");

  if (enabledHands.length === 0) {
    statusDiv.innerHTML =
      '<span style="color: #e74c3c;">❌ 没有启用的手部</span>';
  } else {
    const statusList = enabledHands
      .map((config) => {
        const emoji = config.handType === "left" ? "✋" : "🤚";
        const handName = config.handType === "left" ? "左手" : "右手";
        const statusDot = config.status === "online" ? "🟢" : "🔴";
        return `${statusDot} ${emoji} ${config.interface} (${handName})`;
      })
      .join("<br>");
    statusDiv.innerHTML = statusList;
  }
}

// 检查所有接口状态 - 修复错误处理
async function checkAllInterfaceStatus() {
  try {
    const response = await fetch("/api/legacy/status");

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }

    const data = await response.json();

    if (!data || data.status !== "success") {
      throw new Error(data?.error || "获取状态失败");
    }

    // 安全地获取接口状态
    const responseData = data.data || {};
    interfaceStatus = responseData.interfaces || {};

    updateAllHandStatus();
    hideConnectionWarning();
  } catch (error) {
    logMessage("error", `状态检查失败：${error.message}`);
    console.error("CheckAllInterfaceStatus Error:", error);
    showConnectionWarning();
    setAllHandStatusOffline();
  }
}

// 检查单个接口状态
async function checkSingleInterfaceStatus(handId) {
  await checkAllInterfaceStatus();
}

// 更新所有手部状态
function updateAllHandStatus() {
  Object.keys(handConfigs).forEach((handId) => {
    const config = handConfigs[handId];
    const status = interfaceStatus[config.interface];

    if (status && status.active) {
      config.status = "online";
      updateHandStatusDisplay(handId, "online", "在线");
    } else {
      config.status = "offline";
      updateHandStatusDisplay(handId, "offline", "离线");
    }
  });
  updateEnabledHandsStatus();
}

// 设置所有手部状态为离线
function setAllHandStatusOffline() {
  Object.keys(handConfigs).forEach((handId) => {
    handConfigs[handId].status = "offline";
    updateHandStatusDisplay(handId, "offline", "连接失败");
  });
  updateEnabledHandsStatus();
}

// 更新手部状态显示
function updateHandStatusDisplay(handId, status, text) {
  const statusDot = document.getElementById(`${handId}_status_dot`);
  const statusText = document.getElementById(`${handId}_status_text`);

  if (statusDot && statusText) {
    statusDot.className = `status-dot ${status}`;
    statusText.textContent = text;
  }
}

// 显示连接警告
function showConnectionWarning() {
  document.getElementById("connection-warning").style.display = "block";
}

// 隐藏连接警告
function hideConnectionWarning() {
  document.getElementById("connection-warning").style.display = "none";
}

// 获取启用的手部配置
function getEnabledHands() {
  return Object.values(handConfigs).filter((config) => config.enabled);
}

// 设置事件监听器
function setupEventListeners() {
  const delayDefault = 30;

  // 刷新所有接口按钮
  document.getElementById("refresh-all").addEventListener("click", () => {
    logMessage("info", "手动刷新所有接口...");
    initializeSystem();
  });

  // 全局控制按钮
  document
    .getElementById("send-all-finger-poses")
    .addEventListener("click", sendAllFingerPoses);
  document
    .getElementById("send-all-palm-poses")
    .addEventListener("click", sendAllPalmPoses);
  document
    .getElementById("reset-all-hands")
    .addEventListener("click", resetAllHands);
  document
    .getElementById("stop-all-animations")
    .addEventListener("click", stopAllAnimations);

  // 动画按钮
  document
    .getElementById("start-wave")
    .addEventListener("click", () => startAnimationForAll("wave"));
  document
    .getElementById("start-sway")
    .addEventListener("click", () => startAnimationForAll("sway"));
  document
    .getElementById("stop-animation")
    .addEventListener("click", stopAllAnimations);

  // 预设姿势按钮 - 使用 LinkerHandController 的预设
  setupPresetButtons();

  // 数字手势按钮事件
  setupNumericPresets();

  // Refill core 按钮
  setupRefillCore();
}

// 设置预设按钮
function setupPresetButtons() {
  const delayDefault = 30;

  // 基础预设姿势
  const presets = {
    "pose-fist": { finger: "FIST", palm: null },
    "pose-open": { finger: "OPEN", palm: null },
    "pose-pinch": { finger: "PINCH", palm: "PALM_PINCH" },
    "pose-point": { finger: "POINT", palm: null },
    "pose-thumbs-up": { finger: "THUMBSUP", palm: null },
    "pose-yeah": { finger: "YEAH", palm: "PALM_YEAH" },
    "pose-wave": { finger: "WAVE", palm: null },
    "pose-big-fist": { finger: "BIG_FIST", palm: "PALM_BIG_FIST" },
    "pose-big-open": { finger: "BIG_OPEN", palm: "PALM_BIG_OPEN" },
    "pose-yo": { finger: "YO", palm: null },
    "pose-gun": { finger: "GUN", palm: "PALM_GUN" },
    "pose-ok": { finger: "OK", palm: "PALM_OK" },
  };

  Object.entries(presets).forEach(([id, preset]) => {
    const button = document.getElementById(id);
    if (button) {
      button.addEventListener("click", () => {
        if (preset.palm) {
          LinkerHandController.applyPalmPreset(
            LinkerHandController.PRESETS[preset.palm],
          );
          const palmPose = LinkerHandController.getPalmPoseValues();
          LinkerHandController.sendPalmPoseToAll(palmPose);

          setTimeout(() => {
            LinkerHandController.applyFingerPreset(
              LinkerHandController.PRESETS[preset.finger],
            );
            const fingerPose = LinkerHandController.getFingerPoseValues();
            LinkerHandController.sendFingerPoseToAll(fingerPose);
          }, delayDefault);
        } else {
          LinkerHandController.applyFingerPreset(
            LinkerHandController.PRESETS[preset.finger],
          );
          const fingerPose = LinkerHandController.getFingerPoseValues();
          LinkerHandController.sendFingerPoseToAll(fingerPose);
        }
      });
    }
  });
}

// 设置数字预设
function setupNumericPresets() {
  const delayDefault = 30;

  // 数字 1-9 的预设
  for (let i = 1; i <= 9; i++) {
    const button = document.getElementById(`pose-${i}`);
    if (button) {
      button.addEventListener("click", () => {
        const palmPreset =
          LinkerHandController.PRESETS[`PALM_${getNumberName(i)}`];
        const fingerPreset = LinkerHandController.PRESETS[getNumberName(i)];

        if (palmPreset) {
          LinkerHandController.applyPalmPreset(palmPreset);
          const palmPose = LinkerHandController.getPalmPoseValues();
          LinkerHandController.sendPalmPoseToAll(palmPose);
        }

        setTimeout(() => {
          if (fingerPreset) {
            LinkerHandController.applyFingerPreset(fingerPreset);
            const fingerPose = LinkerHandController.getFingerPoseValues();
            LinkerHandController.sendFingerPoseToAll(fingerPose);
          }
        }, delayDefault);
      });
    }
  }
}

// 获取数字名称
function getNumberName(num) {
  const names = [
    "",
    "ONE",
    "TWO",
    "THREE",
    "FOUR",
    "FIVE",
    "SIX",
    "SEVEN",
    "EIGHT",
    "NINE",
  ];
  return names[num] || "";
}

// 设置 Refill Core 功能
function setupRefillCore() {
  document.getElementById("refill-core").addEventListener("click", () => {
    event.preventDefault();
    event.stopPropagation();

    console.log("refill-core");

    const rukaPoseList = [
      [
        [246, 188, 128, 128],
        [149, 30, 145, 36, 41, 46],
      ], // 食指
      [
        [246, 155, 154, 66],
        [138, 80, 0, 154, 41, 46],
      ], // 中指
      [
        [246, 155, 154, 40],
        [140, 80, 0, 15, 155, 46],
      ], // 无名指
      [
        [246, 155, 154, 25],
        [140, 62, 0, 15, 29, 143],
      ], // 小指
    ];

    const delayTime = 350; // 设定延迟时间为 350ms

    // 创建完整的序列：从第一个到最后一个，再从最后一个回到第二个
    const forwardIndices = [...Array(rukaPoseList.length).keys()]; // [0,1,2,3]
    const backwardIndices = [...forwardIndices].reverse().slice(1); // [3,2,1]
    const sequenceIndices = [...forwardIndices, ...backwardIndices];

    // 遍历序列索引，为每个索引创建两个操作（palm 和 finger）
    sequenceIndices.forEach((index, step) => {
      const targetPose = rukaPoseList[index];

      // 应用 palm 预设
      setTimeout(() => {
        console.log(
          `Step ${step + 1}a: Applying palm preset for pose ${index + 1}`,
        );
        LinkerHandController.applyPalmPreset(targetPose[0]);
        const palmPose = LinkerHandController.getPalmPoseValues();
        LinkerHandController.sendPalmPoseToAll(palmPose);
      }, delayTime * (step * 2)); // 每个完整步骤有两个操作，所以是 step*2

      // 应用 finger 预设
      setTimeout(() => {
        console.log(
          `Step ${step + 1}b: Applying finger preset for pose ${index + 1}`,
        );
        LinkerHandController.applyFingerPreset(targetPose[1]);
        const fingerPose = LinkerHandController.getFingerPoseValues();
        LinkerHandController.sendFingerPoseToAll(fingerPose);
      }, delayTime * (step * 2 + 1)); // 偏移一个 delayTime
    });
  });
}

// 设置滑块事件
function setupSliderEvents() {
  // 手指滑块
  for (let i = 0; i < 6; i++) {
    const slider = document.getElementById(`finger${i}`);
    const valueDisplay = document.getElementById(`finger${i}-value`);
    slider.addEventListener("input", function () {
      valueDisplay.textContent = this.value;
    });
  }

  // 掌部滑块
  for (let i = 0; i < 4; i++) {
    const slider = document.getElementById(`palm${i}`);
    const valueDisplay = document.getElementById(`palm${i}-value`);
    slider.addEventListener("input", function () {
      valueDisplay.textContent = this.value;
    });
  }

  // 速度滑块
  const speedSlider = document.getElementById("animation-speed");
  const speedDisplay = document.getElementById("speed-value");
  speedSlider.addEventListener("input", function () {
    speedDisplay.textContent = this.value;
  });
}

// 发送所有启用手部的手指姿态
async function sendAllFingerPoses() {
  const enabledHands = getEnabledHands();
  if (enabledHands.length === 0) {
    logMessage("error", "没有启用的手部");
    return;
  }

  const pose = [];
  for (let i = 0; i < 6; i++) {
    pose.push(Number.parseInt(document.getElementById(`finger${i}`).value));
  }

  logMessage("info", `发送手指姿态到 ${enabledHands.length} 个启用的手部...`);

  for (const config of enabledHands) {
    await sendFingerPoseToHand(config, pose);
  }
}

// 发送所有启用手部的掌部姿态
async function sendAllPalmPoses() {
  const enabledHands = getEnabledHands();
  if (enabledHands.length === 0) {
    logMessage("error", "没有启用的手部");
    return;
  }

  const pose = [];
  for (let i = 0; i < 4; i++) {
    pose.push(Number.parseInt(document.getElementById(`palm${i}`).value));
  }

  logMessage("info", `发送掌部姿态到 ${enabledHands.length} 个启用的手部...`);

  for (const config of enabledHands) {
    await sendPalmPoseToHand(config, pose);
  }
}

// 发送手指姿态到指定手部
async function sendFingerPoseToHand(config, pose) {
  try {
    const response = await fetch("/api/legacy/fingers", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        interface: config.interface,
        pose: pose,
        handType: config.handType,
        handId: handTypeIds[config.handType],
      }),
    });

    const data = await response.json();
    if (data.status === "success") {
      const handName = config.handType === "left" ? "左手" : "右手";
      logMessage(
        "success",
        `${config.interface} (${handName}): 手指姿态发送成功 [${pose.join(
          ", ",
        )}]`,
      );
    } else {
      logMessage("error", `${config.interface}: ${data.error}`);
    }
  } catch (error) {
    logMessage("error", `${config.interface}: 发送失败 - ${error.message}`);
  }
}

// 发送掌部姿态到指定手部
async function sendPalmPoseToHand(config, pose) {
  try {
    const response = await fetch("/api/legacy/palm", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        interface: config.interface,
        pose: pose,
        handType: config.handType,
        handId: handTypeIds[config.handType],
      }),
    });

    const data = await response.json();
    if (data.status === "success") {
      const handName = config.handType === "left" ? "左手" : "右手";
      logMessage(
        "success",
        `${config.interface} (${handName}): 掌部姿态发送成功 [${pose.join(
          ", ",
        )}]`,
      );
    } else {
      logMessage("error", `${config.interface}: ${data.error}`);
    }
  } catch (error) {
    logMessage("error", `${config.interface}: 发送失败 - ${error.message}`);
  }
}

// 为所有启用手部设置预设姿势
async function setPresetPoseForAll(preset) {
  const enabledHands = getEnabledHands();
  if (enabledHands.length === 0) {
    logMessage("error", "没有启用的手部");
    return;
  }

  logMessage(
    "info",
    `设置预设姿势 "${preset}" 到 ${enabledHands.length} 个启用的手部...`,
  );

  for (const config of enabledHands) {
    await setPresetPoseToHand(config, preset);
  }
}

// 为指定手部设置预设姿势
async function setPresetPoseToHand(config, preset) {
  try {
    const response = await fetch(
      `/api/legacy/preset/${preset}?interface=${config.interface}&handType=${config.handType}`,
      {
        method: "POST",
      },
    );

    const data = await response.json();
    if (data.status === "success") {
      const handName = config.handType === "left" ? "左手" : "右手";
      logMessage(
        "success",
        `${config.interface} (${handName}): ${data.message}`,
      );
    } else {
      logMessage("error", `${config.interface}: ${data.error}`);
    }
  } catch (error) {
    logMessage("error", `${config.interface}: 预设姿势失败 - ${error.message}`);
  }
}

// 为所有启用手部启动动画
async function startAnimationForAll(type) {
  const enabledHands = getEnabledHands();
  if (enabledHands.length === 0) {
    logMessage("error", "没有启用的手部");
    return;
  }

  const speed = Number.parseInt(
    document.getElementById("animation-speed").value,
  );
  logMessage(
    "info",
    `启动 "${type}" 动画到 ${enabledHands.length} 个启用的手部...`,
  );

  for (const config of enabledHands) {
    await startAnimationForHand(config, type, speed);
  }
}

// 为指定手部启动动画
async function startAnimationForHand(config, type, speed) {
  try {
    const response = await fetch("/api/legacy/animation", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        interface: config.interface,
        type: type,
        speed: speed,
        handType: config.handType,
        handId: handTypeIds[config.handType],
      }),
    });

    const data = await response.json();
    if (data.status === "success") {
      const handName = config.handType === "left" ? "左手" : "右手";
      logMessage(
        "success",
        `${config.interface} (${handName}): ${data.message}`,
      );
    } else {
      logMessage("error", `${config.interface}: ${data.error}`);
    }
  } catch (error) {
    logMessage("error", `${config.interface}: 动画启动失败 - ${error.message}`);
  }
}

// 停止所有启用手部的动画
async function stopAllAnimations() {
  const enabledHands = getEnabledHands();
  if (enabledHands.length === 0) {
    logMessage("error", "没有启用的手部");
    return;
  }

  logMessage("info", `停止 ${enabledHands.length} 个启用手部的动画...`);

  for (const config of enabledHands) {
    await stopAnimationForHand(config);
  }
}

// 停止指定手部的动画
async function stopAnimationForHand(config) {
  try {
    const response = await fetch("/api/legacy/animation", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        interface: config.interface,
        type: "stop",
        handType: config.handType,
        handId: handTypeIds[config.handType],
      }),
    });

    const data = await response.json();
    if (data.status === "success") {
      const handName = config.handType === "left" ? "左手" : "右手";
      logMessage(
        "success",
        `${config.interface} (${handName}): ${data.message}`,
      );
    } else {
      logMessage("error", `${config.interface}: ${data.error}`);
    }
  } catch (error) {
    logMessage("error", `${config.interface}: 停止动画失败 - ${error.message}`);
  }
}

// 重置所有启用手部
async function resetAllHands() {
  const enabledHands = getEnabledHands();
  if (enabledHands.length === 0) {
    logMessage("error", "没有启用的手部");
    return;
  }

  // 重置滑块值
  LinkerHandController.applyFingerPreset(LinkerHandController.PRESETS.OPEN);
  LinkerHandController.applyPalmPreset(
    LinkerHandController.PRESETS.PALM_NEUTRAL,
  );

  logMessage("info", `重置 ${enabledHands.length} 个启用的手部...`);

  // 停止所有动画
  await stopAllAnimations();

  // 发送重置姿态
  await sendAllFingerPoses();
  await sendAllPalmPoses();

  logMessage("info", "所有启用手部已重置完成");
}

// 自动触发按钮序列（数字手势）
async function triggerButtonsSequentially(interval = 2000) {
  const enabledHands = getEnabledHands();
  if (enabledHands.length === 0) {
    logMessage("error", "没有启用的手部");
    return;
  }

  logMessage("info", `开始自动数字手势序列 (${enabledHands.length} 个手部)`);

  const buttons = [
    document.getElementById("pose-1"),
    document.getElementById("pose-2"),
    document.getElementById("pose-3"),
    document.getElementById("pose-4"),
    document.getElementById("pose-5"),
    document.getElementById("pose-6"),
    document.getElementById("pose-7"),
    document.getElementById("pose-8"),
    document.getElementById("pose-9"),
  ];

  for (const button of buttons) {
    if (button) {
      button.click();
      await new Promise((resolve) => setTimeout(resolve, interval));
    }
  }

  // 然后执行所有预设手势
  const presetButtons = document.querySelectorAll(
    ".preset-grid button:not(.preset-num-pose)",
  );
  for (const button of presetButtons) {
    button.click();
    await new Promise((resolve) => setTimeout(resolve, interval));
  }

  logMessage("success", "数字手势序列完成");
}

// 日志消息
function logMessage(type, message) {
  const statusLog = document.getElementById("status-log");
  const timestamp = new Date().toLocaleTimeString();

  const logEntry = document.createElement("div");
  logEntry.className = "log-entry";

  let statusClass = "status-info";
  if (type === "success") statusClass = "status-success";
  else if (type === "error") statusClass = "status-error";

  logEntry.innerHTML = `
        <span class="status-indicator ${statusClass}"></span>
        <span class="log-timestamp">${timestamp}</span>
        ${message}
    `;

  statusLog.appendChild(logEntry);
  statusLog.scrollTop = statusLog.scrollHeight;

  // 保持最多 50 条日志
  const entries = statusLog.querySelectorAll(".log-entry");
  if (entries.length > 50) {
    statusLog.removeChild(entries[0]);
  }
}

// 启动状态更新器
function startStatusUpdater() {
  // 每 5 秒检查一次接口状态
  setInterval(async () => {
    await checkAllInterfaceStatus();
  }, 5000);

  // 每 30 秒刷新一次接口列表
  setInterval(async () => {
    const oldInterfaces = [...availableInterfaces];
    await loadAvailableInterfaces();

    // 如果接口发生变化，重新生成配置
    if (JSON.stringify(oldInterfaces) !== JSON.stringify(availableInterfaces)) {
      generateHandConfigs();
    }
  }, 30000);
}

// 添加调试功能
async function debugSystemStatus() {
  logMessage("info", "🔍 开始系统调试...");

  // 检查 HTML 元素
  const elements = {
    "hands-grid": document.getElementById("hands-grid"),
    "status-log": document.getElementById("status-log"),
    "enabled-hands-status": document.getElementById("enabled-hands-status"),
    "sensor-data": document.getElementById("sensor-data"),
  };

  Object.entries(elements).forEach(([name, element]) => {
    if (element) {
      logMessage("success", `✅ 元素 ${name} 存在`);
    } else {
      logMessage("error", `❌ 元素 ${name} 不存在`);
    }
  });

  // 检查全局变量
  logMessage("info", `可用接口：[${availableInterfaces.join(", ")}]`);
  logMessage("info", `手部配置数量：${Object.keys(handConfigs).length}`);
  logMessage("info", `启用手部数量：${getEnabledHands().length}`);

  // 测试 API 连通性
  try {
    logMessage("info", "测试 /api/health 连接...");
    const response = await fetch("/api/legacy/health");
    if (response.ok) {
      const data = await response.json();
      logMessage("success", "✅ 健康检查通过");
      console.log("Health Check Data:", data);
    } else {
      logMessage("error", `❌ 健康检查失败：HTTP ${response.status}`);
    }
  } catch (error) {
    logMessage("error", `❌ 健康检查异常：${error.message}`);
  }

  // 测试接口 API
  try {
    logMessage("info", "测试 /api/interfaces 连接...");
    const response = await fetch("/api/legacy/interfaces");
    if (response.ok) {
      const data = await response.json();
      logMessage("success", "✅ 接口 API 通过");
      console.log("Interfaces API Data:", data);
    } else {
      logMessage("error", `❌ 接口 API 失败：HTTP ${response.status}`);
    }
  } catch (error) {
    logMessage("error", `❌ 接口 API 异常：${error.message}`);
  }
}

// 导出全局函数供 HTML 按钮使用
window.triggerButtonsSequentially = triggerButtonsSequentially;
window.debugSystemStatus = debugSystemStatus;

// 添加全局错误处理
window.addEventListener("error", (event) => {
  logMessage("error", `全局错误：${event.error?.message || event.message}`);
  console.error("Global Error:", event.error);
});

window.addEventListener("unhandledrejection", (event) => {
  logMessage(
    "error",
    `未处理的 Promise 拒绝：${event.reason?.message || event.reason}`,
  );
  console.error("Unhandled Promise Rejection:", event.reason);
});

// 页面可见性变化时的处理
document.addEventListener("visibilitychange", () => {
  if (!document.hidden) {
    // 页面变为可见时，刷新状态
    checkAllInterfaceStatus();
  }
});

// 处理网络错误时的重连逻辑
window.addEventListener("online", () => {
  logMessage("info", "网络连接已恢复，正在重新连接...");
  initializeSystem();
});

window.addEventListener("offline", () => {
  logMessage("error", "网络连接已断开");
  showConnectionWarning();
});

// 键盘快捷键支持
document.addEventListener("keydown", (e) => {
  // Ctrl+R 刷新接口
  if (e.ctrlKey && e.key === "r") {
    e.preventDefault();
    logMessage("info", "快捷键刷新接口列表...");
    initializeSystem();
  }

  // Ctrl+Space 停止所有动画
  if (e.ctrlKey && e.code === "Space") {
    e.preventDefault();
    stopAllAnimations();
  }

  // Ctrl+A 选择/取消选择所有手部
  if (e.ctrlKey && e.key === "a") {
    e.preventDefault();
    toggleAllHands();
  }

  // 数字键 1-9 快速设置预设姿势
  if (e.key >= "1" && e.key <= "9" && !e.ctrlKey && !e.altKey) {
    const activeElement = document.activeElement;
    // 确保不在输入框中
    if (
      activeElement.tagName !== "INPUT" &&
      activeElement.tagName !== "SELECT"
    ) {
      const button = document.getElementById(`pose-${e.key}`);
      if (button) button.click();
    }
  }
});

// 切换所有手部启用状态
function toggleAllHands() {
  const enabledCount = Object.values(handConfigs).filter(
    (config) => config.enabled,
  ).length;
  const shouldEnable = enabledCount === 0;

  Object.keys(handConfigs).forEach((handId) => {
    handConfigs[handId].enabled = shouldEnable;
    const checkbox = document.getElementById(`${handId}_checkbox`);
    if (checkbox) {
      checkbox.checked = shouldEnable;
    }
    updateHandElement(handId);
  });

  updateEnabledHandsStatus();
  logMessage("info", `${shouldEnable ? "启用" : "禁用"}所有手部`);
}

// 工具提示功能
function addTooltips() {
  const tooltips = {
    "refresh-all": "刷新所有可用接口列表",
    "send-all-finger-poses": "向所有启用的手部发送当前手指关节位置",
    "send-all-palm-poses": "向所有启用的手部发送当前掌部关节位置",
    "reset-all-hands": "重置所有启用手部到默认位置",
    "stop-all-animations": "停止所有启用手部的动画",
    "start-wave": "启动所有启用手部的手指波浪动画",
    "start-sway": "启动所有启用手部的掌部摆动动画",
    "stop-animation": "停止所有启用手部的动画",
    "refill-core": "执行 Refill Core 动作序列",
  };

  Object.entries(tooltips).forEach(([id, text]) => {
    const element = document.getElementById(id);
    if (element) {
      element.title = text;
    }
  });
}

// 页面加载完成后添加工具提示
document.addEventListener("DOMContentLoaded", () => {
  addTooltips();
});

// ---eof

// 六手依次动画函数
async function startSequentialHandAnimation(
  animationType = "wave",
  interval = 500,
  cycles = 3,
) {
  const enabledHands = getEnabledHands();

  // 检查是否有足够的手部
  if (enabledHands.length === 0) {
    logMessage("error", "没有启用的手部");
    return;
  }

  // 确保按接口名称排序（can0, can1, can2...）
  const sortedHands = enabledHands.sort((a, b) => {
    const getInterfaceNumber = (iface) => {
      const match = iface.match(/(\d+)$/);
      return match ? Number.parseInt(match[1]) : 0;
    };
    return getInterfaceNumber(a.interface) - getInterfaceNumber(b.interface);
  });

  logMessage(
    "info",
    `开始六手依次动画 - 类型：${animationType}, 间隔：${interval}ms, 循环：${cycles}次`,
  );
  logMessage(
    "info",
    `动画顺序：${sortedHands.map((h) => h.interface).join(" → ")}`,
  );

  // 定义动画预设
  const animationPresets = {
    wave: {
      name: "手指波浪",
      fingerPoses: [
        [255, 255, 255, 255, 255, 255], // 完全张开
        [128, 128, 128, 128, 128, 128], // 中间位置
        [64, 64, 64, 64, 64, 64], // 握拳
        [128, 128, 128, 128, 128, 128], // 回到中间
      ],
      palmPose: [128, 128, 128, 128], // 掌部保持中立
    },

    thumbsUp: {
      name: "竖拇指传递",
      fingerPoses: [
        [255, 255, 0, 0, 0, 0], // 竖拇指
        [128, 128, 128, 128, 128, 128], // 恢复中立
      ],
      palmPose: [128, 128, 128, 128],
    },

    point: {
      name: "食指指点传递",
      fingerPoses: [
        [0, 0, 255, 0, 0, 0], // 食指指点
        [128, 128, 128, 128, 128, 128], // 恢复中立
      ],
      palmPose: [128, 128, 128, 128],
    },

    fistOpen: {
      name: "握拳张开",
      fingerPoses: [
        [64, 64, 64, 64, 64, 64], // 握拳
        [255, 255, 255, 255, 255, 255], // 张开
        [128, 128, 128, 128, 128, 128], // 中立
      ],
      palmPose: [128, 128, 128, 128],
    },

    numbers: {
      name: "数字倒计时",
      fingerPoses: [
        [255, 255, 255, 255, 255, 255], // 5
        [0, 57, 255, 255, 255, 255], // 4
        [0, 57, 255, 255, 255, 0], // 3
        [0, 57, 255, 255, 0, 0], // 2
        [0, 57, 255, 0, 0, 0], // 1
        [64, 64, 64, 64, 64, 64], // 握拳 (0)
      ],
      palmPoses: [
        [255, 109, 255, 118], // 5 对应的掌部
        [255, 109, 255, 118], // 4 对应的掌部
        [255, 109, 255, 118], // 3 对应的掌部
        [255, 109, 255, 118], // 2 对应的掌部
        [255, 109, 255, 118], // 1 对应的掌部
        [128, 128, 128, 128], // 0 对应的掌部
      ],
    },

    mexican: {
      name: "墨西哥波浪",
      fingerPoses: [
        [64, 64, 64, 64, 64, 64], // 起始握拳
        [128, 64, 64, 64, 64, 64], // 拇指起
        [255, 128, 64, 64, 64, 64], // 拇指 + 食指起
        [255, 255, 128, 64, 64, 64], // 前三指起
        [255, 255, 255, 128, 64, 64], // 前四指起
        [255, 255, 255, 255, 128, 64], // 前五指起
        [255, 255, 255, 255, 255, 255], // 全部张开
        [128, 255, 255, 255, 255, 128], // 波浪形
        [64, 128, 255, 255, 128, 64], // 继续波浪
        [64, 64, 128, 255, 128, 64], // 波浪收尾
        [64, 64, 64, 128, 64, 64], // 几乎回到握拳
        [64, 64, 64, 64, 64, 64], // 完全握拳
      ],
      palmPose: [128, 128, 128, 128],
    },
  };

  const preset = animationPresets[animationType] || animationPresets.wave;
  const fingerPoses = preset.fingerPoses;
  const palmPoses =
    preset.palmPoses || Array(fingerPoses.length).fill(preset.palmPose);

  // 执行动画循环
  for (let cycle = 0; cycle < cycles; cycle++) {
    logMessage("info", `${preset.name} - 第 ${cycle + 1}/${cycles} 轮`);

    // 每个动作姿势
    for (let poseIndex = 0; poseIndex < fingerPoses.length; poseIndex++) {
      const currentFingerPose = fingerPoses[poseIndex];
      const currentPalmPose = palmPoses[poseIndex];

      // 依次激活每只手
      for (let handIndex = 0; handIndex < sortedHands.length; handIndex++) {
        const hand = sortedHands[handIndex];
        const handName = hand.handType === "left" ? "左手" : "右手";

        // 先发送掌部姿态
        await sendPalmPoseToHand(hand, currentPalmPose);

        // 短暂延迟后发送手指姿态
        setTimeout(async () => {
          await sendFingerPoseToHand(hand, currentFingerPose);
        }, 50);

        logMessage(
          "info",
          `${hand.interface}(${handName}) 执行姿势 ${poseIndex + 1}/${
            fingerPoses.length
          }`,
        );

        // 等待间隔时间再激活下一只手
        await new Promise((resolve) => setTimeout(resolve, interval));
      }
    }

    // 循环间隔（如果有多轮）
    if (cycle < cycles - 1) {
      logMessage("info", `等待下一轮动画...`);
      await new Promise((resolve) => setTimeout(resolve, interval * 2));
    }
  }

  // 动画结束后，让所有手回到中立位置
  logMessage("info", "动画完成，恢复中立位置...");
  const neutralFingerPose = [128, 128, 128, 128, 128, 128];
  const neutralPalmPose = [128, 128, 128, 128];

  for (const hand of sortedHands) {
    await sendPalmPoseToHand(hand, neutralPalmPose);
    setTimeout(async () => {
      await sendFingerPoseToHand(hand, neutralFingerPose);
    }, 50);
    await new Promise((resolve) => setTimeout(resolve, 100));
  }

  logMessage("success", `六手依次动画 "${preset.name}" 完成！`);
}

// 扩展的动画控制函数
async function startCustomSequentialAnimation(config) {
  const {
    animationType = "wave",
    interval = 500,
    cycles = 3,
    direction = "forward", // 'forward', 'backward', 'bounce'
    simultaneousHands = 1, // 同时激活的手数
    staggerDelay = 100, // 同时激活手之间的错开延迟
  } = config;

  const enabledHands = getEnabledHands();

  if (enabledHands.length === 0) {
    logMessage("error", "没有启用的手部");
    return;
  }

  // 根据方向排序手部
  let sortedHands = enabledHands.sort((a, b) => {
    const getInterfaceNumber = (iface) => {
      const match = iface.match(/(\d+)$/);
      return match ? Number.parseInt(match[1]) : 0;
    };
    return getInterfaceNumber(a.interface) - getInterfaceNumber(b.interface);
  });

  if (direction === "backward") {
    sortedHands = sortedHands.reverse();
  }

  logMessage(
    "info",
    `开始自定义六手动画 - 方向：${direction}, 同时手数：${simultaneousHands}`,
  );

  // 执行动画逻辑...
  // 这里可以根据 simultaneousHands 参数同时控制多只手
  // 实现类似的动画逻辑，但支持更多自定义选项
}

// 预定义的快捷动画函数
async function startWaveAnimation() {
  await startSequentialHandAnimation("wave", 300, 2);
}

async function startThumbsUpRelay() {
  await startSequentialHandAnimation("thumbsUp", 400, 3);
}

async function startPointingRelay() {
  await startSequentialHandAnimation("point", 350, 2);
}

async function startNumberCountdown() {
  await startSequentialHandAnimation("numbers", 800, 1);
}

async function startMexicanWave() {
  await startSequentialHandAnimation("mexican", 200, 3);
}

async function startFistOpenWave() {
  await startSequentialHandAnimation("fistOpen", 400, 2);
}

// 高级组合动画：先正向再反向
async function startBidirectionalWave() {
  logMessage("info", "开始双向波浪动画...");

  // 正向波浪
  await startSequentialHandAnimation("wave", 300, 1);
  await new Promise((resolve) => setTimeout(resolve, 500));

  // 反向波浪（通过反转手部顺序实现）
  const originalGetEnabledHands = window.getEnabledHands;
  window.getEnabledHands = () => originalGetEnabledHands().reverse();

  await startSequentialHandAnimation("wave", 300, 1);

  // 恢复原始函数
  window.getEnabledHands = originalGetEnabledHands;

  logMessage("success", "双向波浪动画完成！");
}

// 导出函数到全局作用域
window.startSequentialHandAnimation = startSequentialHandAnimation;
window.startCustomSequentialAnimation = startCustomSequentialAnimation;
window.startWaveAnimation = startWaveAnimation;
window.startThumbsUpRelay = startThumbsUpRelay;
window.startPointingRelay = startPointingRelay;
window.startNumberCountdown = startNumberCountdown;
window.startMexicanWave = startMexicanWave;
window.startFistOpenWave = startFistOpenWave;
window.startBidirectionalWave = startBidirectionalWave;
