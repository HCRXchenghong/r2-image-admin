<template>
  <div class="ai-workstation">
    <div class="page-head">
      <div>
        <h2>AI 生图工作站</h2>
        <p>输入提示词，调用 GPT 图片模型生成，可放弃或直接上传到图片管理</p>
      </div>
      <div class="head-status">
        <el-tag :type="cfg.ai_image_configured ? 'success' : 'danger'" size="large">
          {{ cfg.ai_image_configured ? 'AI 已配置' : 'AI 未配置' }}
        </el-tag>
        <el-button :icon="Refresh" circle @click="loadConfig" />
      </div>
    </div>

    <div v-if="!cfg.ai_image_configured" class="not-configured">
      <el-empty description="尚未配置 AI 生图 API Key">
        <el-button type="primary" @click="$router.push('/settings')">前往设置</el-button>
      </el-empty>
    </div>

    <div v-else class="workbench">
      <div class="left-panel">
        <div class="panel-title">生图参数</div>
        <el-input
          v-model="prompt"
          type="textarea"
          :rows="8"
          maxlength="1000"
          show-word-limit
          placeholder="描述你想生成的图片，例如：一只戴墨镜的橘猫，赛博朋克风格，高清摄影"
        />

        <div class="field-row">
          <span class="field-label">尺寸</span>
          <el-select v-model="size" style="width: 160px">
            <el-option label="1:1 正方形" value="1024x1024" />
            <el-option label="3:2 横版" value="1536x1024" />
            <el-option label="2:3 竖版" value="1024x1536" />
          </el-select>
        </div>

        <div class="field-row">
          <span class="field-label">模型</span>
          <el-input :model-value="cfg.ai_image_model || 'gpt-image-1'" disabled />
        </div>

        <div class="field-row">
          <span class="field-label">上传分类</span>
          <el-input v-model="uploadCategory" placeholder="如 ai" />
        </div>

        <el-button type="primary" size="large" :icon="MagicStick" :loading="generating" class="generate-btn" @click="generate">
          {{ generating ? '正在生成…' : '生成图片' }}
        </el-button>
      </div>

      <div class="right-panel">
        <div class="panel-title">生成结果</div>

        <div v-if="items.length === 0" class="empty-results">
          <el-empty description="还没有生成结果" />
        </div>

        <div v-else class="result-grid">
          <div v-for="(item, idx) in items" :key="item.id" class="result-card">
            <el-image :src="item.url" fit="contain" class="result-image" :preview-src-list="[item.url]" />
            <div class="result-info">
              <div class="result-meta">
                <span>{{ item.size }}</span>
                <el-tag v-if="item.status === 'uploaded'" type="success" size="small">已上传</el-tag>
                <el-tag v-else-if="item.status === 'uploading'" type="warning" size="small">上传中</el-tag>
                <el-tag v-else type="info" size="small">待处理</el-tag>
              </div>
              <div class="result-actions">
                <el-button
                  size="small"
                  type="primary"
                  :icon="UploadFilled"
                  :loading="item.status === 'uploading'"
                  :disabled="item.status === 'uploaded'"
                  @click="uploadItem(item)"
                >
                  {{ item.status === 'uploaded' ? '已上传' : '直接上传使用' }}
                </el-button>
                <el-button size="small" :icon="Delete" :disabled="item.status === 'uploading'" @click="discard(idx)">放弃</el-button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { MagicStick, UploadFilled, Delete, Refresh } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import client, { apiError } from '../api/client'

const cfg = ref({})
const prompt = ref('')
const size = ref('1024x1024')
const uploadCategory = ref('ai')
const generating = ref(false)
const items = ref([])
let seq = 0

async function loadConfig() {
  try {
    const res = await client.get('/config')
    cfg.value = res.data
  } catch (err) {
    ElMessage.error(apiError(err))
  }
}

async function generate() {
  const text = prompt.value.trim()
  if (!text) {
    ElMessage.warning('请先输入提示词')
    return
  }
  generating.value = true
  try {
    const res = await client.post('/ai/generate', { prompt: text, size: size.value })
    const data = res.data?.data || []
    const parsed = data
      .map((d) => d.b64_json || d.url || '')
      .filter(Boolean)
      .map((src) => ({ id: ++seq, url: toObjectUrl(src), src, size: size.value, status: 'pending' }))
    if (!parsed.length) {
      ElMessage.error('AI 接口未返回图片数据')
      return
    }
    items.value.unshift(...parsed)
    ElMessage.success('生成成功，共 ' + parsed.length + ' 张')
  } catch (err) {
    ElMessage.error(apiError(err))
  } finally {
    generating.value = false
  }
}

function toObjectUrl(src) {
  if (/^https?:\/\//.test(src)) return src
  const mimeMatch = src.match(/^data:([^;]+);/)
  if (mimeMatch) {
    const data = src.split(',')[1]
    const bytes = base64ToBytes(data)
    return URL.createObjectURL(new Blob([bytes], { type: mimeMatch[1] }))
  }
  return 'data:image/png;base64,' + src
}

function base64ToBytes(b64) {
  const bin = atob(b64)
  const bytes = new Uint8Array(bin.length)
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i)
  return bytes
}

async function uploadItem(item) {
  if (item.status !== 'pending') return
  item.status = 'uploading'
  try {
    const blob = await fetchBlob(item.src)
    const file = new File([blob], 'ai-' + Date.now() + '.' + extOf(item.src), { type: blob.type || 'image/png' })
    const fd = new FormData()
    fd.append('file', file)
    fd.append('category', uploadCategory.value || 'ai')
    await client.post('/images/direct', fd)
    item.status = 'uploaded'
    ElMessage.success('已上传到图片管理（分类：' + (uploadCategory.value || 'ai') + '）')
  } catch (err) {
    item.status = 'pending'
    ElMessage.error('上传失败：' + apiError(err))
  }
}

async function fetchBlob(src) {
  if (/^https?:\/\//.test(src)) {
    const res = await fetch(src)
    if (!res.ok) throw new Error('HTTP ' + res.status)
    return res.blob()
  }
  const data = src.replace(/^data:[^,]+;base64,/, '')
  const bytes = base64ToBytes(data)
  const mimeMatch = src.match(/^data:([^;]+);/)
  return new Blob([bytes], { type: mimeMatch ? mimeMatch[1] : 'image/png' })
}

function extOf(src) {
  const mimeMatch = src.match(/^data:([^;]+);/)
  if (mimeMatch) return mimeMatch[1].split('/')[1] || 'png'
  const path = src.split('?')[0]
  const dot = path.lastIndexOf('.')
  return dot >= 0 ? path.slice(dot + 1) : 'png'
}

function discard(idx) {
  const item = items.value[idx]
  if (item && /^blob:/.test(item.url)) URL.revokeObjectURL(item.url)
  items.value.splice(idx, 1)
}

onMounted(loadConfig)
</script>

<style scoped>
.ai-workstation {
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

.head-status {
  display: flex;
  align-items: center;
  gap: 10px;
}

.not-configured {
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 10px;
  padding: 40px 0;
}

.workbench {
  display: grid;
  grid-template-columns: 340px 1fr;
  gap: 16px;
}

.left-panel,
.right-panel {
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 10px;
  padding: 18px;
}

.panel-title {
  font-size: 15px;
  font-weight: 600;
  color: #1f2329;
  margin-bottom: 16px;
}

.field-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 14px;
}

.field-label {
  width: 70px;
  flex-shrink: 0;
  color: #4b5563;
  font-size: 13px;
}

.generate-btn {
  width: 100%;
  margin-top: 18px;
}

.empty-results {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 360px;
}

.result-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 14px;
}

.result-card {
  border: 1px solid #ebeef5;
  border-radius: 8px;
  overflow: hidden;
}

.result-image {
  width: 100%;
  aspect-ratio: 1;
  background: #f5f7fa;
}

.result-info {
  padding: 10px 12px;
}

.result-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
  font-size: 12px;
  color: #8a9099;
}

.result-actions {
  display: flex;
  gap: 8px;
}

@media (max-width: 900px) {
  .workbench {
    grid-template-columns: 1fr;
  }
}
</style>
