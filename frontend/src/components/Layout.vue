<template>
  <el-container class="layout-container">
    <!-- Sidebar -->
    <el-aside :width="isCollapsed ? '64px' : '220px'" class="sidebar">
      <div class="logo-area">
        <el-icon size="24" color="#409EFF"><DataAnalysis /></el-icon>
        <span v-if="!isCollapsed" class="logo-text">日志分析平台</span>
      </div>
      <el-menu
        :default-active="activeMenu"
        class="sidebar-menu"
        :collapse="isCollapsed"
        router
        background-color="#001529"
        text-color="#ffffffa6"
        active-text-color="#ffffff"
      >
        <el-menu-item index="/dashboard">
          <el-icon><TrendCharts /></el-icon>
          <template #title>错误大盘</template>
        </el-menu-item>
        <el-menu-item index="/topn">
          <el-icon><List /></el-icon>
          <template #title>TopN 排行</template>
        </el-menu-item>
        <el-menu-item index="/logs">
          <el-icon><Document /></el-icon>
          <template #title>日志明细</template>
        </el-menu-item>
        <el-menu-item index="/alert-rules">
          <el-icon><Bell /></el-icon>
          <template #title>告警规则</template>
        </el-menu-item>
        <el-menu-item index="/alert-history">
          <el-icon><Clock /></el-icon>
          <template #title>告警历史</template>
        </el-menu-item>
        <el-menu-item index="/settings">
          <el-icon><Setting /></el-icon>
          <template #title>系统设置</template>
        </el-menu-item>
      </el-menu>
      <div class="collapse-btn" @click="toggleCollapse">
        <el-icon>
          <component :is="isCollapsed ? 'Expand' : 'Fold'" />
        </el-icon>
      </div>
    </el-aside>

    <!-- Main content -->
    <el-container>
      <el-header class="header">
        <div class="header-title">{{ currentTitle }}</div>
        <div class="header-right">
          <el-tag type="success" size="small">系统运行中</el-tag>
        </div>
      </el-header>
      <el-main class="main-content">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const isCollapsed = ref(false)

const activeMenu = computed(() => route.path)
const currentTitle = computed(() => route.meta?.title || '日志分析告警平台')

const toggleCollapse = () => {
  isCollapsed.value = !isCollapsed.value
}
</script>

<style scoped>
.layout-container {
  height: 100vh;
  overflow: hidden;
}

.sidebar {
  background-color: #001529;
  transition: width 0.3s;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.logo-area {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  border-bottom: 1px solid #ffffff1a;
  padding: 0 16px;
  overflow: hidden;
}

.logo-text {
  color: #fff;
  font-size: 16px;
  font-weight: 600;
  white-space: nowrap;
}

.sidebar-menu {
  flex: 1;
  border-right: none;
  overflow-y: auto;
}

.sidebar-menu:not(.el-menu--collapse) {
  width: 220px;
}

.collapse-btn {
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: #ffffffa6;
  border-top: 1px solid #ffffff1a;
  transition: color 0.2s;
}

.collapse-btn:hover {
  color: #fff;
}

.header {
  background: #fff;
  border-bottom: 1px solid #e8e8e8;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  box-shadow: 0 1px 4px rgba(0, 21, 41, 0.08);
}

.header-title {
  font-size: 18px;
  font-weight: 600;
  color: #333;
}

.main-content {
  background: #f0f2f5;
  overflow-y: auto;
  padding: 20px;
}
</style>
