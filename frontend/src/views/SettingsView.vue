<template>
  <div class="settings">
    <div class="page-head">
      <div>
        <h2>系统设置</h2>
        <p>配置存储、图片处理、AI 生图与安全参数，保存后自动重启生效</p>
      </div>
      <div class="head-actions">
        <div class="restart-switch">
          <span>保存后自动重启</span>
          <el-switch v-model="autoRestart" :loading="savingRestart" @change="saveRestart" />
        </div>
        <el-button :icon="Refresh" circle @click="load" />
      </div>
    </div>

    <el-row :gutter="16" class="group-row">
      <el-col :xs="24" :md="12">
        <el-card shadow="never" class="group-card">
          <div class="group-head">
            <div class="group-title">
              <span class="group-icon orange"><el-icon><Box /></el-icon></span>
              <div>
                <div class="name">存储配置</div>
                <div class="desc">本地磁盘或 Cloudflare R2 对象存储</div>
              </div>
            </div>
            <el-button type="primary" plain @click="openDialog('storage')">配置</el-button>
          </div>
          <div class="group-body">
            <el-descriptions :column="1" size="small">
              <el-descriptions-item label="存储驱动">
                <el-tag size="small" :type="cfg.storage_driver === 'r2' ? 'warning' : 'info'">
                  {{ cfg.storage_driver === 'r2' ? 'Cloudflare R2' : '本地磁盘' }}
                </el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="Bucket">{{ cfg.bucket || '—' }}</el-descriptions-item>
              <el-descriptions-item label="公开域名">{{ cfg.public_base_url || '未设置' }}</el-descriptions-item>
            </el-descriptions>
          </div>
        </el-card>
      </el-col>

      <el-col :xs="24" :md="12">
        <el-card shadow="never" class="group-card">
          <div class="group-head">
            <div class="group-title">
              <span class="group-icon green"><el-icon><MagicStick /></el-icon></span>
              <div>
                <div class="name">图片处理</div>
                <div class="desc">多尺寸变体、输出格式与压缩质量</div>
              </div>
            </div>
            <el-button type="primary" plain @click="openDialog('image')">配置</el-button>
          </div>
          <div class="group-body">
            <el-descriptions :column="1" size="small">
              <el-descriptions-item label="生成尺寸">{{ cfg.sizes || [] }} px</el-descriptions-item>
              <el-descriptions-item label="输出格式">{{ (cfg.formats || []).join(' / ') }}</el-descriptions-item>
              <el-descriptions-item label="压缩质量">{{ cfg.quality }}%</el-descriptions-item>
              <el-descriptions-item label="保留原图">
                <el-tag size="small" :type="cfg.keep_original ? 'success' : 'info'">
                  {{ cfg.keep_original ? '是' : '否' }}
                </el-tag>
              </el-descriptions-item>
            </el-descriptions>
          </div>
        </el-card>
      </el-col>

      <el-col :xs="24" :md="12">
        <el-card shadow="never" class="group-card">
          <div class="group-head">
            <div class="group-title">
              <span class="group-icon blue"><el-icon><UploadFilled /></el-icon></span>
              <div>
                <div class="name">上传限制</div>
                <div class="desc">单文件大小上限</div>
              </div>
            </div>
            <el-button type="primary" plain @click="openDialog('upload')">配置</el-button>
          </div>
          <div class="group-body">
            <el-descriptions :column="1" size="small">
              <el-descriptions-item label="单文件上限">{{ cfg.max_upload_mb }} MB</el-descriptions-item>
            </el-descriptions>
          </div>
        </el-card>
      </el-col>

      <el-col :xs="24" :md="12">
        <el-card shadow="never" class="group-card">
          <div class="group-head">
            <div class="group-title">
              <span class="group-icon purple"><el-icon><Cpu /></el-icon></span>
              <div>
                <div class="name">AI 生图</div>
                <div class="desc">OpenAI Images API 网关、密钥与模型</div>
              </div>
            </div>
            <el-button type="primary" plain @click="openDialog('ai')">配置</el-button>
          </div>
          <div class="group-body">
            <el-descriptions :column="1" size="small">
              <el-descriptions-item label="接口地址">{{ cfg.ai_image_api_url || '—' }}</el-descriptions-item>
              <el-descriptions-item label="模型">{{ cfg.ai_image_model || '—' }}</el-descriptions-item>
              <el-descriptions-item label="API Key">
                <el-tag size="small" :type="cfg.ai_image_configured ? 'success' : 'danger'">
                  {{ cfg.ai_image_configured ? '已配置' : '未配置' }}
                </el-tag>
              </el-descriptions-item>
            </el-descriptions>
          </div>
        </el-card>
      </el-col>

      <el-col :xs="24" :md="12">
        <el-card shadow="never" class="group-card">
          <div class="group-head">
            <div class="group-title">
              <span class="group-icon red"><el-icon><Lock /></el-icon></span>
              <div>
                <div class="name">安全</div>
                <div class="desc">管理员密码、会话与跨域来源控制</div>
              </div>
            </div>
            <el-button type="primary" plain @click="openDialog('security')">配置</el-button>
          </div>
          <div class="group-body">
            <el-descriptions :column="1" size="small">
              <el-descriptions-item label="管理员账号">{{ username || 'admin' }}</el-descriptions-item>
              <el-descriptions-item label="密码状态">
                <el-tag size="small" type="success">仅服务器保存</el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="会话有效期">{{ cfg.jwt_ttl_hours || '—' }} 小时</el-descriptions-item>
              <el-descriptions-item label="审计留存">{{ cfg.audit_retention_days || '—' }} 天</el-descriptions-item>
            </el-descriptions>
          </div>
        </el-card>
      </el-col>

      <el-col :xs="24" :md="12">
        <el-card shadow="never" class="group-card">
          <div class="group-head">
            <div class="group-title">
              <span class="group-icon slate"><el-icon><DataLine /></el-icon></span>
              <div>
                <div class="name">运行环境</div>
                <div class="desc">数据库、图片处理与健康状态</div>
              </div>
            </div>
            <el-button :icon="Refresh" circle @click="load" />
          </div>
          <div class="group-body">
            <el-descriptions :column="1" size="small">
              <el-descriptions-item label="数据库">{{ cfg.db_driver }}</el-descriptions-item>
              <el-descriptions-item label="图片处理">
                <el-tag size="small" :type="cfg.processing ? 'success' : 'warning'">
                  {{ cfg.processing ? 'libvips 已启用' : '未启用（原图上传可用）' }}
                </el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="服务健康">
                <el-tag size="small" :type="health ? 'success' : 'danger'">{{ health ? '正常' : '异常' }}</el-tag>
              </el-descriptions-item>
            </el-descriptions>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-dialog v-model="dialog.visible" :title="dialog.title" width="560px" destroy-on-close>
      <div v-if="dialog.type === 'storage'" class="form-body">
        <el-form label-width="110px">
          <el-form-item label="存储驱动">
            <el-radio-group v-model="form.storage_driver">
              <el-radio-button value="local">本地磁盘</el-radio-button>
              <el-radio-button value="r2">Cloudflare R2</el-radio-button>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="R2 Account ID">
            <el-input v-model="form.r2_account_id" placeholder="留空则不修改" />
          </el-form-item>
          <el-form-item label="Access Key ID">
            <el-input v-model="form.r2_access_key_id" placeholder="留空则不修改" />
          </el-form-item>
          <el-form-item label="Secret Key">
            <el-input v-model="form.r2_secret_access_key" type="password" show-password placeholder="留空则不修改" />
          </el-form-item>
          <el-form-item label="Bucket">
            <el-input v-model="form.r2_bucket" placeholder="如 site-images" />
          </el-form-item>
          <el-form-item label="公开域名">
            <el-input v-model="form.public_base_url" placeholder="如 https://img.example.com" />
          </el-form-item>
          <div class="form-tip">
            切换为 R2 后必须完整填写 Account ID、两个 Key 和公开域名，否则校验会失败。
            <el-link type="primary" :underline="false" @click="$router.push('/r2-guide')">不知道去哪里找？查看 R2 接入指南 <el-icon><TopRight /></el-icon></el-link>
          </div>
        </el-form>
      </div>

      <div v-else-if="dialog.type === 'image'" class="form-body">
        <el-form label-width="110px">
          <el-form-item label="生成尺寸">
            <el-input v-model="form.img_sizes" placeholder="如 400,800,1200,1600" />
          </el-form-item>
          <el-form-item label="输出格式">
            <el-select v-model="form.img_formats" multiple placeholder="选择格式">
              <el-option label="WebP" value="webp" />
              <el-option label="AVIF" value="avif" />
              <el-option label="JPEG" value="jpeg" />
              <el-option label="PNG" value="png" />
            </el-select>
          </el-form-item>
          <el-form-item label="压缩质量">
            <el-input-number v-model="form.img_quality" :min="1" :max="100" />
          </el-form-item>
          <el-form-item label="保留原图">
            <el-switch v-model="form.img_keep_original" />
          </el-form-item>
        </el-form>
      </div>

      <div v-else-if="dialog.type === 'upload'" class="form-body">
        <el-form label-width="110px">
          <el-form-item label="单文件上限">
            <el-input-number v-model="form.max_upload_mb" :min="1" :max="200" />
            <span class="unit">MB</span>
          </el-form-item>
        </el-form>
      </div>

      <div v-else-if="dialog.type === 'ai'" class="form-body">
        <el-form label-width="110px">
          <el-form-item label="接口地址">
            <el-input v-model="form.ai_image_api_url" placeholder="https://api.openai.com/v1/images/generations" />
          </el-form-item>
          <el-form-item label="API Key">
            <el-input v-model="form.ai_image_api_key" type="password" show-password placeholder="留空则不修改" />
          </el-form-item>
          <el-form-item label="模型">
            <div class="model-row">
              <el-select
                v-model="form.ai_image_model"
                filterable
                allow-create
                default-first-option
                placeholder="同步后可选择模型，也可手动输入"
                style="flex: 1"
              >
                <el-option v-for="m in aiModels" :key="m" :label="m" :value="m" />
              </el-select>
              <el-button :loading="syncingModels" @click="syncModels">同步上游模型</el-button>
            </div>
          </el-form-item>
          <div class="form-tip">Key 保存在服务器 .env 文件中，仅用于后端请求 OpenAI Images API，前端不会读取明文。</div>
        </el-form>
      </div>

      <div v-else-if="dialog.type === 'security'" class="form-body">
        <el-form label-width="110px">
          <el-form-item label="管理员账号">
            <el-input :model-value="username || 'admin'" disabled />
          </el-form-item>
          <el-form-item label="新密码">
            <el-input v-model="form.admin_password" type="password" show-password placeholder="留空则不修改，至少 12 位" />
          </el-form-item>
          <el-form-item label="JWT Secret">
            <el-input v-model="form.jwt_secret" type="password" show-password placeholder="留空则不修改；建议 32 位以上随机串" />
          </el-form-item>
          <el-form-item label="允许跨域来源">
            <el-input v-model="form.cors_allowed_origins" placeholder="留空仅同源；多个来源用英文逗号分隔" />
          </el-form-item>
          <div class="form-tip">修改密码会自动轮换 JWT 密钥并使所有会话失效。生产环境应保持跨域来源为空，或仅填写精确的 HTTPS 管理后台地址。</div>
        </el-form>
      </div>

      <template #footer>
        <el-button @click="dialog.visible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveDialog">保存并重启</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { Refresh, Box, MagicStick, UploadFilled, Cpu, Lock, DataLine, TopRight } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import client, { apiError } from '../api/client'
import { useAuth } from '../auth'

const { username } = useAuth()
const cfg = ref({})
const health = ref(true)
const autoRestart = ref(true)
const savingRestart = ref(false)
const saving = ref(false)

const dialog = reactive({ visible: false, type: '', title: '' })
const form = reactive({})
const aiModels = ref([])
const syncingModels = ref(false)

async function load() {
  try {
    const [c, h] = await Promise.all([client.get('/config'), client.get('/health')])
    cfg.value = c.data
    autoRestart.value = !!c.data.auto_restart
    health.value = !!h.data.ok
  } catch (err) {
    ElMessage.error(apiError(err))
  }
}

function openDialog(type) {
  const c = cfg.value
  Object.keys(form).forEach((k) => delete form[k])
  if (type === 'storage') {
    Object.assign(form, {
      storage_driver: c.storage_driver || 'local',
      r2_account_id: '',
      r2_access_key_id: '',
      r2_secret_access_key: '',
      r2_bucket: c.bucket || '',
      public_base_url: c.public_base_url || '',
    })
    dialog.title = '配置存储'
  } else if (type === 'image') {
    Object.assign(form, {
      img_sizes: (c.sizes || []).join(','),
      img_formats: c.formats || [],
      img_quality: c.quality ?? 82,
      img_keep_original: !!c.keep_original,
    })
    dialog.title = '配置图片处理'
  } else if (type === 'upload') {
    Object.assign(form, { max_upload_mb: c.max_upload_mb || 20 })
    dialog.title = '配置上传限制'
  } else if (type === 'ai') {
    Object.assign(form, {
      ai_image_api_url: c.ai_image_api_url || '',
      ai_image_api_key: '',
      ai_image_model: c.ai_image_model || '',
    })
    aiModels.value = []
    dialog.title = '配置 AI 生图'
  } else if (type === 'security') {
    Object.assign(form, { admin_password: '', jwt_secret: '', cors_allowed_origins: (c.cors_allowed_origins || []).join(',') })
    dialog.title = '配置安全'
  }
  dialog.type = type
  dialog.visible = true
}

function buildPayload() {
  const payload = {}
  const t = dialog.type
  if (t === 'storage') {
    if (form.storage_driver) payload.storage_driver = form.storage_driver
    if (form.r2_account_id) payload.r2_account_id = form.r2_account_id
    if (form.r2_access_key_id) payload.r2_access_key_id = form.r2_access_key_id
    if (form.r2_secret_access_key) payload.r2_secret_access_key = form.r2_secret_access_key
    if (form.r2_bucket) payload.r2_bucket = form.r2_bucket
    if (form.public_base_url) payload.public_base_url = form.public_base_url
  } else if (t === 'image') {
    if (form.img_sizes) payload.img_sizes = form.img_sizes
    if (form.img_formats && form.img_formats.length) payload.img_formats = form.img_formats.join(',')
    if (form.img_quality != null) payload.img_quality = form.img_quality
    payload.img_keep_original = !!form.img_keep_original
  } else if (t === 'upload') {
    if (form.max_upload_mb != null) payload.upload_max_mb = form.max_upload_mb
  } else if (t === 'ai') {
    if (form.ai_image_api_url) payload.ai_image_api_url = form.ai_image_api_url
    if (form.ai_image_api_key) payload.ai_image_api_key = form.ai_image_api_key
    if (form.ai_image_model) payload.ai_image_model = form.ai_image_model
  } else if (t === 'security') {
    if (form.admin_password) payload.admin_password = form.admin_password
    if (form.jwt_secret) payload.jwt_secret = form.jwt_secret
    payload.cors_allowed_origins = form.cors_allowed_origins || ''
  }
  payload.auto_restart = autoRestart.value
  return payload
}

async function syncModels() {
  if (!form.ai_image_api_url) {
    ElMessage.warning('请先填写接口地址')
    return
  }
  syncingModels.value = true
  try {
    const res = await client.post('/ai/models/sync', {
      api_url: form.ai_image_api_url,
      api_key: form.ai_image_api_key || '',
    })
    aiModels.value = res.data.models || []
    if (!aiModels.value.length) {
      ElMessage.warning('上游未返回可用模型，可手动输入')
    } else {
      ElMessage.success('已同步 ' + aiModels.value.length + ' 个模型')
    }
  } catch (err) {
    ElMessage.error(apiError(err))
  } finally {
    syncingModels.value = false
  }
}

async function saveDialog() {
  saving.value = true
  try {
    const res = await client.put('/config', buildPayload())
    ElMessage.success(res.data.message || '配置已保存')
    dialog.visible = false
    if (res.data.auto_restart !== false) {
      ElMessage.warning('服务即将自动重启，页面会短暂断开，请稍后刷新')
    }
    window.setTimeout(() => load().catch(() => undefined), 2200)
  } catch (err) {
    ElMessage.error(apiError(err))
  } finally {
    saving.value = false
  }
}

async function saveRestart(val) {
  savingRestart.value = true
  try {
    const res = await client.put('/config', { auto_restart: val })
    ElMessage.success(res.data.message || '已更新')
  } catch (err) {
    autoRestart.value = !val
    ElMessage.error(apiError(err))
  } finally {
    savingRestart.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.settings {
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

.head-actions {
  display: flex;
  align-items: center;
  gap: 14px;
}

.restart-switch {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: #4b5563;
}

.group-row {
  row-gap: 16px;
}

.group-card {
  border: 1px solid #ebeef5;
  border-radius: 10px;
}

.group-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 14px;
  border-bottom: 1px solid #f0f2f5;
}

.group-title {
  display: flex;
  align-items: center;
  gap: 12px;
}

.group-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  border-radius: 8px;
  font-size: 18px;
}

.group-icon.orange { background: #fff3e8; color: #f97316; }
.group-icon.green { background: #e8f7ee; color: #22a06b; }
.group-icon.blue { background: #e8f1fd; color: #2f6fed; }
.group-icon.purple { background: #f1ebfd; color: #7c5cd6; }
.group-icon.red { background: #fdecec; color: #d54941; }
.group-icon.slate { background: #eef0f3; color: #4b5563; }

.name {
  font-size: 15px;
  font-weight: 600;
  color: #1f2329;
}

.desc {
  font-size: 12px;
  color: #8a9099;
}

.group-body {
  padding-top: 14px;
}

.form-body {
  padding: 4px 8px;
}

.form-tip {
  margin-top: 4px;
  padding: 10px 12px;
  border-radius: 8px;
  background: #f5f7fa;
  color: #6b7280;
  font-size: 12px;
  line-height: 1.6;
}

.unit {
  margin-left: 8px;
  color: #8a9099;
  font-size: 13px;
}

.model-row {
  display: flex;
  gap: 8px;
  width: 100%;
}

:deep(.el-dialog__title) {
  font-weight: 600;
}
</style>
