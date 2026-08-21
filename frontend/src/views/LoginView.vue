<template>
  <div class="login-page">
    <div class="login-brand">
      <div class="login-brand-inner">
        <div class="logo">R2</div>
        <h1>R2 图床管理后台</h1>
        <p>自托管图片存储、处理与分发，支持 Cloudflare R2 和 AI 生图。</p>
        <div class="feature-list">
          <span>多尺寸 WebP / AVIF</span>
          <span>URL 保持不变</span>
          <span>AI 生图工作站</span>
        </div>
      </div>
    </div>

    <div class="login-form">
      <el-card class="login-card" shadow="never">
        <div class="card-title">登录</div>
        <div class="card-sub">使用管理员账号登录</div>

        <el-form :model="form" @submit.prevent="submit">
          <el-form-item>
            <el-input v-model="form.username" placeholder="用户名" size="large" :prefix-icon="User" />
          </el-form-item>
          <el-form-item>
            <el-input
              v-model="form.password"
              type="password"
              placeholder="密码"
              size="large"
              show-password
              :prefix-icon="Lock"
              @keyup.enter="submit"
            />
          </el-form-item>

          <el-alert v-if="error" :title="error" type="error" :closable="false" class="error-alert" />

          <el-button
            type="primary"
            size="large"
            class="login-btn"
            :loading="loading"
            @click="submit"
          >
            登录
          </el-button>
        </el-form>


      </el-card>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { User, Lock } from '@element-plus/icons-vue'
import { useAuth } from '../auth'
import { apiError } from '../api/client'

const router = useRouter()
const { login } = useAuth()
const form = reactive({ username: '', password: '' })
const error = ref('')
const loading = ref(false)

async function submit() {
  error.value = ''
  loading.value = true
  try {
    await login(form.username, form.password)
    router.push('/')
  } catch (err) {
    error.value = apiError(err)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  display: grid;
  grid-template-columns: 1.1fr 1fr;
  min-height: 100vh;
  background: #fff;
}

.login-brand {
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(150deg, #1f2329 0%, #2b2f36 55%, #3a2514 100%);
  color: #fff;
  padding: 48px;
}

.login-brand-inner {
  max-width: 430px;
}

.logo {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 52px;
  height: 52px;
  border-radius: 14px;
  background: linear-gradient(135deg, #ff9a4d, #f97316);
  font-weight: 700;
  font-size: 20px;
  margin-bottom: 28px;
  box-shadow: 0 12px 30px rgba(249, 115, 22, 0.3);
}

.login-brand h1 {
  font-size: 28px;
  font-weight: 600;
  margin: 0 0 12px;
}

.login-brand p {
  color: #b6bbc2;
  line-height: 1.8;
  margin: 0;
}

.feature-list {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 28px;
}

.feature-list span {
  padding: 6px 12px;
  border-radius: 999px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  font-size: 13px;
  color: #d8dbe0;
}

.login-form {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px;
  background: #f5f7fa;
}

.login-card {
  width: 100%;
  max-width: 400px;
  border: 1px solid #ebeef5;
  border-radius: 12px;
  padding: 12px 8px;
}

.card-title {
  font-size: 22px;
  font-weight: 600;
  color: #1f2329;
}

.card-sub {
  margin: 6px 0 24px;
  color: #8a9099;
  font-size: 14px;
}

.login-btn {
  width: 100%;
}

.error-alert {
  margin-bottom: 16px;
}

@media (max-width: 768px) {
  .login-page {
    grid-template-columns: 1fr;
  }
  .login-brand {
    display: none;
  }
}
</style>
