<template>
  <el-config-provider :locale="locale">
    <router-view />
  </el-config-provider>
</template>

<script setup>
import { computed, onMounted, watchEffect } from 'vue'
import { useRoute } from 'vue-router'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import { applyDocumentTitle, syncPublicSiteTitle } from '@/utils/site'

const locale = computed(() => zhCn)
const route = useRoute()

watchEffect(() => {
  applyDocumentTitle(route.meta?.title || '')
})

onMounted(async () => {
  await syncPublicSiteTitle()
})
</script>

<style>
/* 全局基础样式 */
html, body, #app {
  margin: 0;
  padding: 0;
  height: 100%;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Helvetica Neue', Helvetica, 'PingFang SC',
    'Hiragino Sans GB', 'Microsoft YaHei', '微软雅黑', Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  background-color: var(--app-bg-page, var(--el-bg-color-page));
  color: var(--el-text-color-primary);
  font-feature-settings: 'tnum' on, 'lnum' on;
}

* {
  box-sizing: border-box;
}

a {
  text-decoration: none;
  transition: color 0.15s ease;
}

/* 平滑滚动 */
html {
  scroll-behavior: smooth;
}

/* Focus 样式统一 */
:focus-visible {
  outline: 2px solid var(--el-color-primary-light-3);
  outline-offset: 2px;
}

:focus:not(:focus-visible) {
  outline: none;
}
</style>
