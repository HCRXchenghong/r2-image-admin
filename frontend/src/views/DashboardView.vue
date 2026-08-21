<template>
  <div class="dashboard">
    <div class="page-head">
      <div>
        <h2>仪表盘</h2>
        <p>图片存储概览与服务器资源实时状态（每 15 秒自动刷新）</p>
      </div>
      <el-button :icon="Refresh" circle @click="loadAll" />
    </div>

    <el-row :gutter="16" class="stat-row">
      <el-col :xs="12" :sm="12" :md="6">
        <el-card shadow="never" class="stat-card">
          <div class="stat-icon orange"><el-icon><Picture /></el-icon></div>
          <div class="stat-label">图片总数</div>
          <div class="stat-value">{{ stats ? stats.images : '—' }}</div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="12" :md="6">
        <el-card shadow="never" class="stat-card">
          <div class="stat-icon green"><el-icon><Coin /></el-icon></div>
          <div class="stat-label">存储占用</div>
          <div class="stat-value">{{ stats ? formatBytes(stats.bytes) : '—' }}</div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="12" :md="6">
        <el-card shadow="never" class="stat-card">
          <div class="stat-icon blue"><el-icon><Cpu /></el-icon></div>
          <div class="stat-label">内存占用</div>
          <div class="stat-value">{{ res ? formatBytes(res.mem_alloc) : '—' }}</div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="12" :md="6">
        <el-card shadow="never" class="stat-card">
          <div class="stat-icon purple"><el-icon><FolderOpened /></el-icon></div>
          <div class="stat-label">分类数量</div>
          <div class="stat-value">{{ stats ? (stats.categories || []).length : '—' }}</div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="12" :md="6">
        <el-card shadow="never" class="stat-card">
          <div class="stat-icon slate"><el-icon><Monitor /></el-icon></div>
          <div class="stat-label">运行时长</div>
          <div class="stat-value small">{{ res ? formatUptime(res.uptime_seconds) : '—' }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="section-row">
      <el-col :xs="24" :md="12">
        <el-card shadow="never" class="panel">
          <template #header>
            <div class="panel-title">服务器资源</div>
          </template>

          <div v-if="res" class="resource-grid">
            <div class="resource-item">
              <span class="r-label">运行时长</span>
              <span class="r-value">{{ formatUptime(res.uptime_seconds) }}</span>
            </div>
            <div class="resource-item">
              <span class="r-label">Go 版本</span>
              <span class="r-value">{{ res.go_version }}</span>
            </div>
            <div class="resource-item">
              <span class="r-label">CPU 核心</span>
              <span class="r-value">{{ res.num_cpu }}</span>
            </div>
            <div class="resource-item">
              <span class="r-label">Goroutine</span>
              <span class="r-value">{{ res.num_goroutine }}</span>
            </div>
            <div class="resource-item">
              <span class="r-label">内存占用</span>
              <span class="r-value">{{ formatBytes(res.mem_alloc) }}</span>
            </div>
            <div class="resource-item">
              <span class="r-label">系统内存（保留）</span>
              <span class="r-value">{{ formatBytes(res.mem_sys) }}</span>
            </div>
            <div class="resource-item">
              <span class="r-label">运行协程</span>
              <span class="r-value">{{ res.num_goroutine }}</span>
            </div>
            <div class="resource-item">
              <span class="r-label">GC 次数</span>
              <span class="r-value">{{ res.num_gc }}</span>
            </div>
          </div>

          <div class="usage-block">
            <div class="usage-line">
              <span>应用内存使用</span>
              <span>{{ formatBytes(res ? res.mem_alloc : 0) }} / {{ formatBytes(res ? res.mem_sys : 0) }}</span>
            </div>
            <el-progress :percentage="memPercent" :stroke-width="10" :show-text="false" />
          </div>

          <div class="disk-block">
            <div class="disk-line">
              <span>磁盘占用</span>
              <span>{{ formatBytes(res ? res.disk_used : 0) }} / {{ formatBytes(res ? res.disk_total : 0) }}</span>
            </div>
            <el-progress :percentage="diskPercent" :stroke-width="10" :show-text="false" />
          </div>
        </el-card>
      </el-col>

      <el-col :xs="24" :md="12">
        <div class="panel-stack">
          <el-card shadow="never" class="panel">
            <template #header>
              <div class="panel-title">环境信息</div>
            </template>
            <div v-if="cfg" class="env-grid">
              <div class="env-item">
                <span class="e-label">存储后端</span>
                <span class="e-value">{{ cfg.storage_driver === 'r2' ? 'Cloudflare R2' : '本地磁盘' }}</span>
              </div>
              <div class="env-item">
                <span class="e-label">数据库</span>
                <span class="e-value">{{ cfg.db_driver }}</span>
              </div>
              <div class="env-item">
                <span class="e-label">Bucket</span>
                <span class="e-value">{{ cfg.bucket || '—' }}</span>
              </div>
              <div class="env-item">
                <span class="e-label">公开域名</span>
                <span class="e-value ellipsis">{{ cfg.public_base_url || '未设置' }}</span>
              </div>
              <div class="env-item">
                <span class="e-label">图片处理</span>
                <el-tag size="small" :type="cfg.processing ? 'success' : 'warning'">
                  {{ cfg.processing ? 'libvips 已启用' : '未启用' }}
                </el-tag>
              </div>
              <div class="env-item">
                <span class="e-label">AI 生图</span>
                <el-tag size="small" :type="cfg.ai_image_configured ? 'success' : 'danger'">
                  {{ cfg.ai_image_configured ? '已配置' : '未配置' }}
                </el-tag>
              </div>
              <div class="env-item">
                <span class="e-label">输出格式</span>
                <span class="e-value">{{ (cfg.formats || []).join(' / ') }}</span>
              </div>
              <div class="env-item">
                <span class="e-label">上传上限</span>
                <span class="e-value">{{ cfg.max_upload_mb }} MB</span>
              </div>
            </div>
          </el-card>

          <el-card shadow="never" class="panel">
            <template #header>
              <div class="panel-title">分类分布</div>
            </template>

            <el-table v-if="stats && (stats.categories || []).length" :data="stats.categories" size="default">
              <el-table-column prop="category" label="分类" min-width="140" />
              <el-table-column prop="count" label="图片数" width="100" />
              <el-table-column label="存储占用" width="120">
                <template #default="{ row }">{{ formatBytes(row.bytes) }}</template>
              </el-table-column>
            </el-table>

            <el-empty v-else description="还没有图片，去「图片管理」上传吧" />
          </el-card>
        </div>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { Refresh, Picture, Coin, FolderOpened, Monitor, Cpu } from '@element-plus/icons-vue'
import client, { apiError } from '../api/client'
import { formatBytes } from '../utils'
import { ElMessage } from 'element-plus'

const stats = ref(null)
const cfg = ref(null)
const res = ref(null)

const diskPercent = computed(() => {
  if (!res.value || !res.value.disk_total) return 0
  return Math.min(100, Math.round((res.value.disk_used / res.value.disk_total) * 100))
})

const memPercent = computed(() => {
  if (!res.value || !res.value.mem_sys) return 0
  return Math.min(100, Math.round((res.value.mem_alloc / res.value.mem_sys) * 100))
})

async function loadAll() {
  try {
    const [s, c, r] = await Promise.all([
      client.get('/stats'),
      client.get('/config'),
      client.get('/resources'),
    ])
    stats.value = s.data
    cfg.value = c.data
    res.value = r.data
  } catch (err) {
    ElMessage.error(apiError(err))
  }
}

let timer
onMounted(() => {
  loadAll()
  timer = window.setInterval(loadAll, 15000)
})
onBeforeUnmount(() => window.clearInterval(timer))

function formatUptime(seconds) {
  const s = Number(seconds || 0)
  if (s < 60) return s + ' 秒'
  if (s < 3600) return Math.floor(s / 60) + ' 分钟'
  if (s < 86400) return Math.floor(s / 3600) + ' 小时 ' + Math.floor((s % 3600) / 60) + ' 分钟'
  return Math.floor(s / 86400) + ' 天 ' + Math.floor((s % 86400) / 3600) + ' 小时'
}

</script>

<style scoped>
.dashboard {
  max-width: 1200px;
  margin: 0 auto;
}

.page-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 18px;
}

.page-head h2 {
  margin: 0;
  font-size: 20px;
  color: #1f2329;
}

.page-head p {
  margin: 4px 0 0;
  color: #8a9099;
  font-size: 13px;
}

.stat-row,
.section-row {
  margin-bottom: 16px;
}

.stat-card :deep(.el-card__body) {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.stat-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  border-radius: 10px;
  font-size: 18px;
  margin-bottom: 10px;
}

.stat-icon.orange {
  background: #fff3e8;
  color: #f97316;
}

.stat-icon.green {
  background: #e8f7ee;
  color: #22a06b;
}

.stat-icon.blue {
  background: #e8f1fd;
  color: #2f6fed;
}

.stat-icon.purple {
  background: #f1ebfd;
  color: #7c5cd6;
}

.stat-icon.slate {
  background: #eef0f3;
  color: #4b5563;
}

.stat-label {
  color: #8a9099;
  font-size: 13px;
}

.stat-value {
  font-size: 24px;
  font-weight: 600;
  color: #1f2329;
}

.stat-value.small {
  font-size: 18px;
}

.panel {
  border: 1px solid #ebeef5;
  border-radius: 10px;
}

.panel-title {
  font-weight: 600;
  color: #1f2329;
}

.resource-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 14px;
}

.resource-item {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.r-label {
  color: #8a9099;
  font-size: 12px;
}

.r-value {
  color: #1f2329;
  font-size: 16px;
  font-weight: 600;
}

.disk-block {
  margin-top: 18px;
}

.usage-block {
  margin-top: 18px;
}

.usage-line,
.disk-line {
  display: flex;
  justify-content: space-between;
  font-size: 13px;
  color: #4b5563;
  margin-bottom: 8px;
}

.usage-line {
  margin-top: 16px;
}

.panel-stack {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.env-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
}

.env-item {
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
}

.e-label {
  color: #8a9099;
  font-size: 12px;
}

.e-value {
  color: #1f2329;
  font-size: 13px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ellipsis {
  display: block;
}
</style>
