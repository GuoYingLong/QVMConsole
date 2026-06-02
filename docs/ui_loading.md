# UI 加载效果说明文档

## 概述
为了提供更优质、更具现代感的视觉体验，本项目全局移除了老式的加载图标与传统转圈效果，升级为：
1. **全局顶部进度条**：用于路由跳转及大区域 API 请求。
2. **“抖音风格”缺口圆环**：用于所有局部加载场景（Element Plus 的 `v-loading` 和按钮加载状态）。

## 详细功能描述

### 1. 顶部加载进度条 (NProgress)
- **技术栈**：使用了第三方库 `nprogress`。
- **配置位置**：
  - `web/src/router/index.js`：在路由切换的前置与后置守卫中被触发，页面跳转时会在顶部出现进度指示。
  - `web/src/utils/request.js`：全局请求拦截器和响应拦截器集成。发起的每一个非静默（`silent: false`） API 请求都会触发进度条，提升用户对系统后台处理状态的感知。
  - `web/src/style.css`：进度条颜色统一跟随主题色（`var(--el-color-primary)`）。

### 2. 缺口圆环加载动画 (TikTok 风格)
- **实现原理**：由于系统中大量使用了 Element Plus 的库封装，传统的做法需要逐一修改组件，为了确保全局一致性并降低维护成本，项目在 `web/src/style.css` 中采取 CSS 覆盖的方式实现全局拦截和样式替换。
- **覆盖点**：
  - `v-loading` 容器（如表格加载、大区块加载等）：强制隐藏默认的 `.el-loading-spinner .circular` SVG 图标，并通过伪元素 `::before` 结合 `border-right-color: transparent` 和无限旋转 `@keyframes custom-spin` 动画实现。
  - `el-button` 上的 `:loading="true"` 属性：强制隐藏默认的 `el-icon.is-loading` 内置 SVG，结合前者的自定义动画进行相同的边框渲染。
- **特点**：跟随当前生效的 Element Plus `var(--el-color-primary)`，确保能在深色模式和浅色模式下自然融入。

## 使用指引
- 若需要在某些特定的后台轮询请求中**关闭**顶部进度条响应（防止不断出现蓝条），请在 API 的 Request Config 中传入 `silent: true` 参数，比如：
  ```javascript
  axios.get('/api/status', { silent: true })
  ```
- 一般的按钮加载或表格加载，继续像往常一样使用 `v-loading="loading"` 或 `<el-button :loading="true">` 即可，样式将自动呈现为新的“缺口圆环”。
