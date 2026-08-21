<template>
  <div class="gallery">
    <div class="page-head">
      <div>
        <h2>图片管理</h2>
        <p>共 {{ total }} 张图片</p>
      </div>
      <div class="head-actions">
        <el-input
          v-model="q"
          placeholder="搜索文件名 / 路径"
          :prefix-icon="Search"
          clearable
          style="width: 240px"
          @input="onSearch"
          @clear="onSearch('')"
        />
        <el-button :icon="Refresh" circle @click="load" />
        <el-button type="primary" :icon="UploadFilled" @click="openUpload">上传图片</el-button>
      </div>
    </div>

    <div class="category-bar">
      <el-tag :effect="category === '' ? 'dark' : 'plain'" class="cat-tag" @click="selectCategory('')">全部</el-tag>
      <el-tag
        v-for="c in categories"
        :key="c.category"
        :effect="category === c.category ? 'dark' : 'plain'"
        class="cat-tag"
        @click="selectCategory(c.category)"
      >
        {{ c.category }} · {{ c.count }}
      </el-tag>
    </div>

    <div v-if="loading && items.length === 0" class="loading-box">
      <el-icon class="is-loading"><Loading /></el-icon>
    </div>

    <el-empty v-else-if="items.length === 0" description="没有符合条件的图片" />

    <div v-else class="grid">
      <div v-for="item in items" :key="item.id" class="grid-item" @click="openDetail(item)">
        <el-image :src="item.thumb_url" fit="cover" class="thumb" lazy />
        <div class="grid-info">
          <div class="name">{{ item.name }}</div>
          <div class="meta">
            <span>{{ item.width > 0 ? item.width + '×' + item.height : '—' }}</span>
            <span>{{ formatBytes(item.size_bytes) }}</span>
          </div>
          <div class="cat">{{ item.category }}</div>
        </div>
        <el-tag v-if="item.direct" size="small" class="direct-tag">原图</el-tag>
      </div>
    </div>

    <div v-if="totalPages > 1" class="pager">
      <el-pagination
        v-model:current-page="page"
        :page-size="pageSize"
        :total="total"
        layout="prev, pager, next, total"
        background
        @current-change="load"
      />
    </div>

    <el-dialog v-model="detailVisible" title="图片详情" width="680px" destroy-on-close>
      <div v-if="detail" class="detail">
        <div class="detail-preview">
          <el-image :src="detail.url || detail.thumb_url" fit="contain" :preview-src-list="[detail.url || detail.thumb_url]" />
        </div>

        <el-descriptions :column="3" border size="small" class="detail-desc">
          <el-descriptions-item label="文件名">{{ detail.name }}</el-descriptions-item>
          <el-descriptions-item label="分类">{{ detail.category }}</el-descriptions-item>
          <el-descriptions-item label="尺寸">{{ detail.width > 0 ? detail.width + '×' + detail.height : '—' }}</el-descriptions-item>
          <el-descriptions-item label="存储占用">{{ formatBytes(detail.size_bytes) }}</el-descriptions-item>
          <el-descriptions-item label="类型">{{ detail.content_type }}</el-descriptions-item>
          <el-descriptions-item label="上传时间">{{ formatDate(detail.created_at) }}</el-descriptions-item>
        </el-descriptions>

        <div v-if="detail.variants && detail.variants.length" class="variants">
          <el-table :data="detail.variants" size="small">
            <el-table-column prop="label" label="变体" width="90" />
            <el-table-column label="尺寸 / 格式" min-width="130">
              <template #default="{ row }">{{ row.width > 0 ? row.width + '×' + row.height : '—' }} · {{ row.format }}</template>
            </el-table-column>
            <el-table-column label="大小" width="100">
              <template #default="{ row }">{{ formatBytes(row.size_bytes) }}</template>
            </el-table-column>
            <el-table-column label="" width="70">
              <template #default="{ row }">
                <el-button size="small" text @click="copy(row.url)">复制</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>

        <div class="detail-actions">
          <el-button size="small" @click="copy(detail.url || detail.thumb_url)">复制链接</el-button>
          <el-button size="small" @click="copy(markdown(detail))">复制 Markdown</el-button>
          <el-button size="small" @click="copy(buildSrcset(detail))">复制 srcset</el-button>
          <el-button v-if="detail.original_url" size="small" @click="copy(detail.original_url)">复制原图链接</el-button>
        </div>

        <div class="detail-foot">
          <el-button type="danger" :icon="Delete" @click="doDelete(detail)">删除</el-button>
          <div class="foot-right">
            <input ref="replaceInput" type="file" accept="image/jpeg,image/png,image/webp,image/avif,image/gif,image/tiff,image/bmp,image/heic,image/heif" class="hidden-input" @change="onReplaceFile" />
            <el-button type="primary" :icon="Switch" :loading="replacing" @click="replaceInput?.click()">替换图片（URL 不变）</el-button>
          </div>
        </div>
      </div>
    </el-dialog>

    <el-dialog v-model="uploadVisible" title="上传图片" width="620px" destroy-on-close>
      <div class="upload-dialog">
        <div class="upload-row">
          <span class="label">上传方式</span>
          <el-radio-group v-model="mode">
            <el-radio-button value="smart">智能处理</el-radio-button>
            <el-radio-button value="direct">原图上传</el-radio-button>
          </el-radio-group>
        </div>

        <div class="upload-row">
          <span class="label">分类</span>
          <el-input v-model="uploadCategory" placeholder="如 products/sku-001" style="flex:1" />
        </div>

        <div v-if="mode === 'smart' && !processing" class="upload-tip">
          服务器未启用 libvips 图片处理，智能上传会失败，请使用「原图上传」。
        </div>

        <input ref="uploadInput" type="file" accept="image/jpeg,image/png,image/webp,image/avif,image/gif,image/tiff,image/bmp,image/heic,image/heif" multiple class="hidden-input" @change="addFiles" />

        <el-button :icon="Plus" @click="uploadInput?.click()">选择文件</el-button>

        <div v-if="queue.length" class="queue">
          <div v-for="(it, idx) in queue" :key="it.id" class="queue-item">
            <div class="q-name">{{ it.file.name }}</div>
            <div class="q-size">{{ formatBytes(it.file.size) }}</div>
            <el-progress v-if="it.status === 'uploading'" :percentage="it.progress" :stroke-width="8" style="width: 140px" />
            <el-tag v-else-if="it.status === 'done'" type="success" size="small">完成</el-tag>
            <el-tag v-else-if="it.status === 'error'" type="danger" size="small">失败</el-tag>
            <el-button v-if="it.status !== 'uploading'" text :icon="Close" @click="removeQueueItem(idx)" />
          </div>
        </div>

        <div class="dialog-foot">
          <el-button @click="uploadVisible = false">取消</el-button>
          <el-button type="primary" :loading="uploading" @click="startUpload">开始上传</el-button>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import {
  Search, Refresh, UploadFilled, Delete, Switch, Plus, Close, Loading,
} from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import client, { apiError } from '../api/client'
import { copyText, buildSrcset, formatBytes, formatDate, markdown } from '../utils'

const pageSize = 24
const items = ref([])
const total = ref(0)
const page = ref(1)
const q = ref('')
const category = ref('')
const categories = ref([])
const loading = ref(false)
const detail = ref(null)
const detailVisible = ref(false)
const replacing = ref(false)
const replaceInput = ref(null)

const uploadVisible = ref(false)
const mode = ref('smart')
const uploadCategory = ref('products')
const queue = ref([])
const uploading = ref(false)
const uploadInput = ref(null)
const processing = ref(true)

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

let searchTimer

async function load() {
  loading.value = true
  try {
    const res = await client.get('/images', {
      params: { page: page.value, pageSize, q: q.value || undefined, category: category.value || undefined },
    })
    items.value = res.data.items || []
    total.value = res.data.total || 0
  } catch (err) {
    ElMessage.error(apiError(err))
  } finally {
    loading.value = false
  }
}

async function loadCategories() {
  try {
    const res = await client.get('/images/categories')
    categories.value = res.data.categories || []
  } catch {
    /* ignore */
  }
}

function onSearch() {
  window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(() => {
    page.value = 1
    load()
  }, 350)
}

function selectCategory(c) {
  category.value = c
  page.value = 1
  load()
}

async function openDetail(item) {
  try {
    const res = await client.get('/images/' + item.id)
    detail.value = res.data
    detailVisible.value = true
  } catch (err) {
    ElMessage.error(apiError(err))
  }
}

async function copy(text) {
  await copyText(text)
  ElMessage.success('已复制')
}

async function doDelete(item) {
  try {
    await client.delete('/images/' + item.id)
    ElMessage.success('已删除 ' + item.name)
    detailVisible.value = false
    load()
    loadCategories()
  } catch (err) {
    ElMessage.error(apiError(err))
  }
}

function onReplaceFile(e) {
  const file = e.target.files?.[0]
  if (file) doReplace(file)
  e.target.value = ''
}

async function doReplace(file) {
  if (!detail.value) return
  replacing.value = true
  try {
    const fd = new FormData()
    fd.append('file', file)
    const res = await client.put('/images/' + detail.value.id, fd)
    detail.value = res.data
    ElMessage.success('替换成功，原 URL 保持不变')
    load()
  } catch (err) {
    ElMessage.error(apiError(err))
  } finally {
    replacing.value = false
  }
}

function openUpload() {
  uploadVisible.value = true
  client.get('/config').then((res) => {
    processing.value = !!res.data.processing
    if (!res.data.processing) mode.value = 'direct'
  }).catch(() => undefined)
}

function addFiles(e) {
  const files = Array.from(e.target.files || [])
  files.forEach((file) => {
    queue.value.push({ id: Date.now() + Math.random(), file, status: 'waiting', progress: 0 })
  })
  e.target.value = ''
}

function removeQueueItem(idx) {
  queue.value.splice(idx, 1)
}

async function startUpload() {
  const pending = queue.value.filter((it) => it.status === 'waiting' || it.status === 'error')
  if (!pending.length) return
  uploading.value = true
  for (const it of pending) {
    it.status = 'uploading'
    it.progress = 0
    try {
      const fd = new FormData()
      fd.append('file', it.file)
      fd.append('category', uploadCategory.value)
      const res = await client.post(mode.value === 'smart' ? '/images' : '/images/direct', fd, {
        onUploadProgress: (e) => {
          if (e.total) it.progress = Math.round((e.loaded / e.total) * 100)
        },
      })
      it.status = 'done'
      it.progress = 100
      ElMessage.success(it.file.name + ' 上传成功')
      void res
    } catch (err) {
      it.status = 'error'
      ElMessage.error(it.file.name + ' 上传失败：' + apiError(err))
    }
  }
  uploading.value = false
  load()
  loadCategories()
}

onMounted(() => {
  load()
  loadCategories()
  client.get('/config').then((res) => {
    processing.value = !!res.data.processing
  }).catch(() => undefined)
})
</script>

<style scoped>
.gallery {
  max-width: 1200px;
  margin: 0 auto;
}

.page-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
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

.head-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.category-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 16px;
}

.cat-tag {
  cursor: pointer;
}

.loading-box {
  display: flex;
  justify-content: center;
  padding: 60px 0;
  font-size: 28px;
  color: #c0c4cc;
}

.grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 14px;
}

@media (max-width: 900px) {
  .grid {
    grid-template-columns: repeat(3, 1fr);
  }
}

@media (max-width: 640px) {
  .grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

.grid-item {
  position: relative;
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 10px;
  overflow: hidden;
  cursor: pointer;
  transition: box-shadow 0.2s, transform 0.2s;
}

.grid-item:hover {
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08);
  transform: translateY(-2px);
}

.thumb {
  width: 100%;
  aspect-ratio: 1;
  background: #f5f7fa;
}

.grid-info {
  padding: 10px 12px;
}

.name {
  font-size: 14px;
  color: #1f2329;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.meta {
  display: flex;
  justify-content: space-between;
  margin-top: 4px;
  font-size: 12px;
  color: #8a9099;
}

.cat {
  margin-top: 2px;
  font-size: 12px;
  color: #a8abb2;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.direct-tag {
  position: absolute;
  top: 8px;
  right: 8px;
}

.pager {
  display: flex;
  justify-content: center;
  margin-top: 18px;
}

.detail-preview {
  display: flex;
  justify-content: center;
  padding: 16px;
  background: #f5f7fa;
  border-radius: 8px;
  margin-bottom: 16px;
}

.detail-desc {
  margin-bottom: 16px;
}

.variants {
  margin-bottom: 16px;
}

.detail-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 16px;
}

.detail-foot {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 16px;
  border-top: 1px solid #ebeef5;
}

.upload-row {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-bottom: 14px;
}

.label {
  width: 70px;
  color: #4b5563;
  font-size: 14px;
}

.upload-tip {
  margin-bottom: 14px;
  padding: 10px 12px;
  border-radius: 8px;
  background: #fdf6ec;
  color: #b88230;
  font-size: 13px;
}

.queue {
  margin-top: 16px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.queue-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  border: 1px solid #ebeef5;
  border-radius: 8px;
}

.q-name {
  flex: 1;
  font-size: 13px;
  color: #1f2329;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.q-size {
  font-size: 12px;
  color: #a8abb2;
}

.dialog-foot {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 18px;
}

.hidden-input {
  display: none;
}
</style>
