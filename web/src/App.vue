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
/* 全局样式覆盖 */
html, body, #app {
  margin: 0;
  padding: 0;
  height: 100%;
  font-family: 'Helvetica Neue', Helvetica, 'PingFang SC', 'Hiragino Sans GB',
  'Microsoft YaHei', '微软雅黑', Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  background-color: var(--el-bg-color-page);
  color: var(--el-text-color-primary);
}

* {
  box-sizing: border-box;
}

a {
  text-decoration: none;
}
</style>
