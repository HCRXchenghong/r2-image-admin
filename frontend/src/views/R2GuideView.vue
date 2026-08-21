<template>
  <div class="r2-guide">
    <div class="guide-head">
      <el-button text :icon="ArrowLeft" @click="$router.push('/settings')">返回存储配置</el-button>
      <el-tag type="success" effect="light"><el-icon><Lock /></el-icon> 登录后可见</el-tag>
    </div>

    <template v-if="loading">
      <el-skeleton :rows="12" animated />
    </template>

    <template v-else-if="guide">
      <section class="hero-card">
        <div class="hero-icon"><el-icon><Cloudy /></el-icon></div>
        <div>
          <h1>{{ guide.title }}</h1>
          <p>{{ guide.subtitle }}</p>
          <div class="hero-actions">
            <el-button type="primary" @click="$router.push('/settings')">去填写存储配置</el-button>
            <el-link :href="guide.console_url" target="_blank" type="primary" :icon="TopRight">打开 Cloudflare 控制台</el-link>
          </div>
        </div>
      </section>

      <section class="flow-card">
        <div class="section-title">接入路径</div>
        <div class="flow">
          <div class="flow-node cloudflare"><el-icon><Cloudy /></el-icon><span>Cloudflare<br />控制台</span></div>
          <el-icon class="flow-arrow"><Right /></el-icon>
          <div class="flow-node bucket"><el-icon><FolderOpened /></el-icon><span>R2 Bucket<br />与 API Token</span></div>
          <el-icon class="flow-arrow"><Right /></el-icon>
          <div class="flow-node app"><el-icon><Setting /></el-icon><span>本后台<br />存储配置</span></div>
          <el-icon class="flow-arrow"><Right /></el-icon>
          <div class="flow-node domain"><el-icon><Link /></el-icon><span>自定义域名<br />公开图片</span></div>
        </div>
      </section>

      <section class="guide-section">
        <div class="section-title">四步完成配置</div>
        <div class="steps">
          <article v-for="step in guide.steps" :key="step.number" class="step-card">
            <div class="step-number">{{ step.number }}</div>
            <div class="step-body">
              <h2>{{ step.title }}</h2>
              <div class="console-path"><el-icon><Guide /></el-icon>{{ step.path }}</div>
              <p>{{ step.description }}</p>
              <ul>
                <li v-for="tip in step.tips" :key="tip"><el-icon><CircleCheck /></el-icon>{{ tip }}</li>
              </ul>
            </div>
          </article>
        </div>
      </section>

      <section class="guide-section">
        <div class="section-title">后台字段怎么填写</div>
        <el-table :data="guide.fields" class="fields-table" border>
          <el-table-column prop="field" label="后台字段" width="175" />
          <el-table-column prop="where" label="去哪里找" min-width="250" />
          <el-table-column prop="example" label="填写示例" min-width="220" />
          <el-table-column prop="note" label="注意事项" min-width="220" />
        </el-table>
      </section>

      <section class="guide-section two-column">
        <el-card shadow="never" class="info-card">
          <template #header><div class="card-label"><el-icon><Connection /></el-icon>预签名直传的 R2 CORS</div></template>
          <p>{{ guide.cors.description }}</p>
          <pre>{{ guide.cors.example }}</pre>
          <el-alert type="warning" :closable="false" title="将 admin.example.com 替换成你的真实后台域名；不要使用 *。" />
        </el-card>

        <el-card shadow="never" class="info-card verify-card">
          <template #header><div class="card-label"><el-icon><CircleCheck /></el-icon>配置后验证清单</div></template>
          <ol>
            <li v-for="item in guide.verify" :key="item">{{ item }}</li>
          </ol>
          <el-alert type="info" :closable="false" title="Access Key 与 Secret Key 只会在服务端保存，页面不会回显明文。" />
        </el-card>
      </section>
    </template>

    <el-result v-else icon="error" title="接入指南加载失败" sub-title="请刷新页面或重新登录后重试。">
      <template #extra><el-button type="primary" @click="load">重新加载</el-button></template>
    </el-result>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import {
  ArrowLeft, CircleCheck, Cloudy, Connection, FolderOpened, Guide, Link,
  Lock, Right, Setting, TopRight,
} from '@element-plus/icons-vue'
import client, { apiError } from '../api/client'
import { ElMessage } from 'element-plus'

const guide = ref(null)
const loading = ref(true)

async function load() {
  loading.value = true
  try {
    const res = await client.get('/guides/r2')
    guide.value = res.data
  } catch (err) {
    guide.value = null
    ElMessage.error(apiError(err))
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.r2-guide { max-width: 1180px; margin: 0 auto; padding-bottom: 32px; }
.guide-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 14px; }
.guide-head :deep(.el-tag) { display: inline-flex; gap: 5px; align-items: center; }
.hero-card { display: flex; gap: 22px; align-items: center; padding: 32px; border-radius: 16px; background: linear-gradient(120deg, #f38020 0%, #f9a64c 55%, #ffd3a5 100%); color: #fff; box-shadow: 0 12px 30px rgba(243, 128, 32, .2); }
.hero-icon { display: grid; flex: 0 0 auto; width: 74px; height: 74px; place-items: center; border: 1px solid rgba(255,255,255,.4); border-radius: 20px; background: rgba(255,255,255,.16); font-size: 38px; }
.hero-card h1 { margin: 0; font-size: 26px; }
.hero-card p { max-width: 760px; margin: 8px 0 18px; line-height: 1.7; opacity: .95; }
.hero-actions { display: flex; gap: 18px; align-items: center; }
.hero-actions :deep(.el-link) { color: #fff; font-size: 14px; }
.flow-card, .guide-section { margin-top: 18px; }
.flow-card { padding: 22px 28px; border: 1px solid #ebeef5; border-radius: 12px; background: #fff; }
.section-title { margin-bottom: 14px; color: #1f2329; font-size: 17px; font-weight: 600; }
.flow { display: flex; align-items: center; justify-content: space-between; gap: 10px; }
.flow-node { display: flex; flex: 1; min-width: 120px; gap: 9px; align-items: center; padding: 14px; border-radius: 10px; font-size: 13px; font-weight: 600; line-height: 1.45; }
.flow-node .el-icon { font-size: 23px; }
.cloudflare { background: #fff3e8; color: #db6b13; }.bucket { background: #f1ecff; color: #7656c4; }.app { background: #e9f3ff; color: #2f6fed; }.domain { background: #e8f8ef; color: #20855a; }.flow-arrow { flex: 0 0 auto; color: #a8abb2; }
.steps { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 14px; }
.step-card { display: flex; gap: 14px; padding: 18px; border: 1px solid #ebeef5; border-radius: 12px; background: #fff; }
.step-number { display: grid; flex: 0 0 auto; width: 30px; height: 30px; place-items: center; border-radius: 50%; background: #f38020; color: #fff; font-weight: 700; }
.step-body h2 { margin: 3px 0 8px; color: #303133; font-size: 16px; }.step-body p { margin: 10px 0; color: #606266; font-size: 13px; line-height: 1.75; }.console-path { display: inline-flex; gap: 5px; align-items: center; padding: 5px 8px; border-radius: 5px; background: #f5f7fa; color: #606266; font-size: 12px; }.step-body ul { display: grid; gap: 6px; margin: 10px 0 0; padding: 0; list-style: none; color: #6b7280; font-size: 12px; }.step-body li { display: flex; gap: 5px; line-height: 1.45; }.step-body li .el-icon { flex: 0 0 auto; margin-top: 2px; color: #22a06b; }
.fields-table { border-radius: 10px; overflow: hidden; }.two-column { display: grid; grid-template-columns: 1.1fr .9fr; gap: 16px; }.info-card { border-radius: 12px; }.card-label { display: flex; gap: 8px; align-items: center; color: #303133; font-weight: 600; }.card-label .el-icon { color: #f38020; }.info-card p, .info-card ol { margin: 0 0 14px; color: #606266; font-size: 13px; line-height: 1.75; }.info-card ol { padding-left: 22px; }.info-card pre { overflow: auto; margin: 0 0 14px; padding: 14px; border-radius: 8px; background: #1f2937; color: #e5e7eb; font-size: 12px; line-height: 1.6; white-space: pre-wrap; }.info-card :deep(.el-alert) { align-items: flex-start; }
@media (max-width: 860px) { .flow { align-items: stretch; flex-direction: column; }.flow-arrow { display: none; }.flow-node { min-width: 0; }.steps, .two-column { grid-template-columns: 1fr; }.hero-card { align-items: flex-start; padding: 24px; }.hero-icon { width: 54px; height: 54px; font-size: 28px; }.hero-actions { align-items: flex-start; flex-direction: column; gap: 10px; } }
@media (max-width: 560px) { .hero-card { gap: 14px; }.hero-card h1 { font-size: 21px; }.hero-icon { display: none; }.fields-table { font-size: 12px; } }
</style>
