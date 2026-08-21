<template>
  <el-container class="app-layout">
    <el-aside width="220px" class="sidebar">
      <div class="brand">
        <div class="brand-logo">R2</div>
        <div>
          <div class="brand-title">图床管理</div>
          <div class="brand-sub">Cloudflare R2</div>
        </div>
      </div>

      <el-menu :default-active="$route.path" router class="sidebar-menu">
        <el-menu-item index="/">
          <el-icon><Odometer /></el-icon>
          <span>仪表盘</span>
        </el-menu-item>
        <el-menu-item index="/gallery">
          <el-icon><PictureFilled /></el-icon>
          <span>图片管理</span>
        </el-menu-item>
        <el-menu-item index="/ai">
          <el-icon><MagicStick /></el-icon>
          <span>AI 生图工作站</span>
        </el-menu-item>
        <el-menu-item index="/settings">
          <el-icon><Setting /></el-icon>
          <span>系统设置</span>
        </el-menu-item>
      </el-menu>
    </el-aside>

    <el-container>
      <el-header class="topbar">
        <div class="topbar-title">R2 图床管理后台</div>
        <div class="topbar-right">
          <el-avatar :size="28" class="user-avatar">{{ avatarText }}</el-avatar>
          <span class="username">{{ username || 'admin' }}</span>
          <el-button text @click="handleLogout">
            <el-icon><SwitchButton /></el-icon>
            <span>退出</span>
          </el-button>
        </div>
      </el-header>

      <el-main class="content">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Odometer, PictureFilled, MagicStick, Setting, SwitchButton } from '@element-plus/icons-vue'
import { useAuth } from '../auth'

const router = useRouter()
const { username, logout, loadMe } = useAuth()

const avatarText = computed(() => (username.value || 'A').slice(0, 1).toUpperCase())

onMounted(() => {
  loadMe()
})

function handleLogout() {
  logout()
  router.push('/login')
}
</script>

<style scoped>
.app-layout {
  height: 100vh;
}

.sidebar {
  display: flex;
  flex-direction: column;
  background: #fff;
  border-right: 1px solid #e4e7ed;
}

.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 18px 16px;
}

.brand-logo {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  border-radius: 10px;
  background: linear-gradient(135deg, #ff9a4d, #f97316);
  color: #fff;
  font-weight: 700;
  font-size: 15px;
  box-shadow: 0 6px 16px rgba(249, 115, 22, 0.24);
}

.brand-title {
  font-size: 15px;
  font-weight: 600;
  color: #1f2329;
}

.brand-sub {
  font-size: 12px;
  color: #8a9099;
}

.sidebar-menu {
  flex: 1;
  border-right: none;
}

.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: #fff;
  border-bottom: 1px solid #e4e7ed;
}

.topbar-title {
  font-size: 15px;
  font-weight: 600;
  color: #1f2329;
}

.topbar-right {
  display: flex;
  align-items: center;
  gap: 10px;
}

.user-avatar {
  background: #f97316;
}

.username {
  font-size: 14px;
  color: #4b5563;
}

.content {
  background: #f5f7fa;
  padding: 20px;
}
</style>
