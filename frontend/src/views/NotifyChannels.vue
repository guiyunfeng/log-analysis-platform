<template>
  <div class="notify-channels-page">
    <!-- Header actions -->
    <div class="toolbar">
      <el-button type="primary" :icon="Plus" @click="openDialog()">新增渠道</el-button>
    </div>

    <!-- Channel list -->
    <div class="card">
      <el-table :data="channels" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="name" label="渠道名称" />
        <el-table-column prop="type" label="类型" width="120">
          <template #default="{ row }">
            <el-tag :type="typeTagMap[row.type]?.tag || 'info'" size="small">
              {{ typeTagMap[row.type]?.label || row.type }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-switch
              :model-value="row.enabled"
              @change="toggleChannel(row)"
            />
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatTime(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="220">
          <template #default="{ row }">
            <el-button size="small" @click="openDialog(row)">编辑</el-button>
            <el-button size="small" type="success" @click="testChannel(row)" :loading="testingId === row.id">测试</el-button>
            <el-button size="small" type="danger" @click="deleteChannel(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- Create/Edit dialog -->
    <el-dialog
      v-model="dialogVisible"
      :title="form.id ? '编辑通知渠道' : '新增通知渠道'"
      width="600px"
      @close="resetForm"
    >
      <el-form :model="form" label-width="130px" size="default">
        <el-form-item label="渠道名称" required>
          <el-input v-model="form.name" placeholder="如：运维钉钉群" />
        </el-form-item>
        <el-form-item label="渠道类型" required>
          <el-select v-model="form.type" @change="onTypeChange" style="width: 100%">
            <el-option label="钉钉 (DingTalk)" value="dingtalk" />
            <el-option label="企业微信 (WeCom)" value="wecom" />
            <el-option label="邮箱 (Email)" value="email" />
            <el-option label="Telegram" value="telegram" />
            <el-option label="飞书 (Feishu)" value="feishu" />
            <el-option label="📞 电话告警 (Phone)" value="phone" />
          </el-select>
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>

        <!-- DingTalk config -->
        <template v-if="form.type === 'dingtalk'">
          <el-form-item label="Webhook URL" required>
            <el-input v-model="form.config.webhook" placeholder="https://oapi.dingtalk.com/robot/send?access_token=xxx" />
          </el-form-item>
          <el-form-item label="安全类型">
            <el-select v-model="form.config.security_type" style="width: 100%">
              <el-option label="关键词" value="keyword" />
              <el-option label="加签 (HMAC-SHA256)" value="sign" />
              <el-option label="IP 白名单" value="ip_whitelist" />
            </el-select>
          </el-form-item>
          <el-form-item label="加签密钥" v-if="form.config.security_type === 'sign'">
            <el-input v-model="form.config.sign_secret" type="password" show-password placeholder="SEC开头的密钥" />
          </el-form-item>
          <el-form-item label="关键词" v-if="form.config.security_type === 'keyword'">
            <el-input v-model="dingKeywordsStr" placeholder="多个关键词逗号分隔，如: 告警,故障,异常" />
          </el-form-item>
          <el-form-item label="@手机号">
            <el-input v-model="dingAtMobilesStr" placeholder="多个手机号逗号分隔（可选）" />
          </el-form-item>
          <el-form-item label="@所有人">
            <el-switch v-model="form.config.at_all" />
          </el-form-item>
        </template>

        <!-- WeCom config -->
        <template v-if="form.type === 'wecom'">
          <el-form-item label="Webhook URL" required>
            <el-input v-model="form.config.webhook" placeholder="https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx" />
          </el-form-item>
        </template>

        <!-- Email config -->
        <template v-if="form.type === 'email'">
          <el-form-item label="SMTP 服务器" required>
            <el-input v-model="form.config.smtp_host" placeholder="smtp.qq.com" />
          </el-form-item>
          <el-form-item label="SMTP 端口" required>
            <el-input-number v-model="form.config.smtp_port" :min="1" :max="65535" />
          </el-form-item>
          <el-form-item label="用户名" required>
            <el-input v-model="form.config.username" placeholder="xxx@qq.com" />
          </el-form-item>
          <el-form-item label="密码" required>
            <el-input v-model="form.config.password" type="password" show-password />
          </el-form-item>
          <el-form-item label="发件人" required>
            <el-input v-model="form.config.from" placeholder="xxx@qq.com" />
          </el-form-item>
          <el-form-item label="收件人" required>
            <el-input
              v-model="emailToStr"
              placeholder="多个地址用英文逗号分隔"
            />
          </el-form-item>
          <el-form-item label="启用 TLS">
            <el-switch v-model="form.config.use_tls" />
          </el-form-item>
        </template>

        <!-- Telegram config -->
        <template v-if="form.type === 'telegram'">
          <el-form-item label="Bot Token" required>
            <el-input v-model="form.config.bot_token" placeholder="xxx:yyy" />
          </el-form-item>
          <el-form-item label="Chat ID" required>
            <el-input v-model="form.config.chat_id" placeholder="-100123456" />
          </el-form-item>
        </template>

        <!-- Feishu config -->
        <template v-if="form.type === 'feishu'">
          <el-form-item label="Webhook URL" required>
            <el-input v-model="form.config.webhook" placeholder="https://open.feishu.cn/open-apis/bot/v2/hook/xxx" />
          </el-form-item>
        </template>

        <!-- Phone config -->
        <template v-if="form.type === 'phone'">
          <el-form-item label="云服务商" required>
            <el-select v-model="form.config.provider" style="width: 100%">
              <el-option label="阿里云" value="aliyun" />
              <el-option label="腾讯云" value="tencent" />
              <el-option label="自定义 Webhook" value="webhook" />
            </el-select>
          </el-form-item>
          <el-form-item label="AccessKeyID" v-if="form.config.provider !== 'webhook'">
            <el-input v-model="form.config.access_key_id" placeholder="AccessKeyID" />
          </el-form-item>
          <el-form-item label="SecretKey" v-if="form.config.provider !== 'webhook'">
            <el-input v-model="form.config.secret_key" type="password" show-password placeholder="SecretKey" />
          </el-form-item>
          <el-form-item label="TTS 模板ID" v-if="form.config.provider !== 'webhook'">
            <el-input v-model="form.config.template_id" placeholder="语音通知模板ID" />
          </el-form-item>
          <el-form-item label="显示号码" v-if="form.config.provider !== 'webhook'">
            <el-input v-model="form.config.called_show_number" placeholder="主叫显示号码" />
          </el-form-item>
          <el-form-item label="Webhook URL" v-if="form.config.provider === 'webhook'">
            <el-input v-model="form.config.webhook_url" placeholder="https://your-gateway/voice-alert" />
          </el-form-item>
          <el-form-item label="被叫号码" required>
            <el-input
              v-model="phoneNumbersStr"
              type="textarea"
              :rows="3"
              placeholder="每行一个手机号"
            />
          </el-form-item>
        </template>
      </el-form>

      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveChannel" :loading="saving">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import {
  getNotifyChannels,
  createNotifyChannel,
  updateNotifyChannel,
  deleteNotifyChannel,
  toggleNotifyChannel,
  testNotifyChannel
} from '../api/index.js'

const loading = ref(false)
const saving = ref(false)
const testingId = ref(null)
const dialogVisible = ref(false)
const channels = ref([])
const emailToStr = ref('')
const phoneNumbersStr = ref('')
const dingKeywordsStr = ref('')
const dingAtMobilesStr = ref('')

const typeTagMap = {
  dingtalk: { label: '钉钉', tag: 'primary' },
  wecom:    { label: '企业微信', tag: 'success' },
  email:    { label: '邮箱', tag: 'warning' },
  telegram: { label: 'Telegram', tag: 'info' },
  feishu:   { label: '飞书', tag: 'danger' },
  phone:    { label: '电话告警', tag: '' }
}

const defaultConfig = {
  dingtalk: { webhook: '', security_type: 'keyword', sign_secret: '', keywords: [], keyword: '', at_mobiles: [], at_all: false },
  wecom:    { webhook: '' },
  email:    { smtp_host: '', smtp_port: 465, username: '', password: '', from: '', to: [], use_tls: true },
  telegram: { bot_token: '', chat_id: '' },
  feishu:   { webhook: '' },
  phone:    { provider: 'webhook', access_key_id: '', secret_key: '', template_id: '', called_show_number: '', phone_numbers: [], webhook_url: '' }
}

const form = reactive({
  id: null,
  name: '',
  type: 'dingtalk',
  config: { ...defaultConfig.dingtalk },
  enabled: true
})

const loadChannels = async () => {
  loading.value = true
  try {
    const res = await getNotifyChannels()
    channels.value = res?.data || res || []
  } catch (e) {
    console.error('Failed to load notify channels:', e)
  } finally {
    loading.value = false
  }
}

const onTypeChange = () => {
  form.config = { ...(defaultConfig[form.type] || {}) }
  emailToStr.value = ''
  phoneNumbersStr.value = ''
  dingKeywordsStr.value = ''
  dingAtMobilesStr.value = ''
}

const openDialog = (row) => {
  if (row) {
    form.id = row.id
    form.name = row.name
    form.type = row.type
    form.enabled = row.enabled
    try {
      const parsed = typeof row.config === 'string' ? JSON.parse(row.config) : row.config
      form.config = { ...(defaultConfig[row.type] || {}), ...parsed }
      if (row.type === 'email' && Array.isArray(form.config.to)) {
        emailToStr.value = form.config.to.join(', ')
      }
      if (row.type === 'phone' && Array.isArray(form.config.phone_numbers)) {
        phoneNumbersStr.value = form.config.phone_numbers.join('\n')
      }
      if (row.type === 'dingtalk') {
        dingKeywordsStr.value = Array.isArray(form.config.keywords) ? form.config.keywords.join(',') : (form.config.keyword || '')
        dingAtMobilesStr.value = Array.isArray(form.config.at_mobiles) ? form.config.at_mobiles.join(',') : ''
      }
    } catch {
      form.config = { ...(defaultConfig[row.type] || {}) }
    }
  } else {
    resetForm()
  }
  dialogVisible.value = true
}

const resetForm = () => {
  form.id = null
  form.name = ''
  form.type = 'dingtalk'
  form.config = { ...defaultConfig.dingtalk }
  form.enabled = true
  emailToStr.value = ''
  phoneNumbersStr.value = ''
  dingKeywordsStr.value = ''
  dingAtMobilesStr.value = ''
}

const saveChannel = async () => {
  if (!form.name.trim()) {
    ElMessage.warning('请填写渠道名称')
    return
  }
  if (form.type === 'email') {
    form.config.to = emailToStr.value.split(',').map(s => s.trim()).filter(Boolean)
  }
  if (form.type === 'phone') {
    form.config.phone_numbers = phoneNumbersStr.value.split('\n').map(s => s.trim()).filter(Boolean)
  }
  if (form.type === 'dingtalk') {
    form.config.keywords = dingKeywordsStr.value.split(',').map(s => s.trim()).filter(Boolean)
    // Also set legacy single keyword for backward compatibility
    form.config.keyword = form.config.keywords[0] || ''
    form.config.at_mobiles = dingAtMobilesStr.value.split(',').map(s => s.trim()).filter(Boolean)
  }
  const payload = {
    name: form.name,
    type: form.type,
    config: JSON.stringify(form.config),
    enabled: form.enabled
  }
  saving.value = true
  try {
    if (form.id) {
      await updateNotifyChannel(form.id, payload)
      ElMessage.success('渠道已更新')
    } else {
      await createNotifyChannel(payload)
      ElMessage.success('渠道已创建')
    }
    dialogVisible.value = false
    loadChannels()
  } finally {
    saving.value = false
  }
}

const toggleChannel = async (row) => {
  const previous = row.enabled
  try {
    await toggleNotifyChannel(row.id)
    row.enabled = !row.enabled
  } catch (e) {
    row.enabled = previous
  }
}

const testChannel = async (row) => {
  testingId.value = row.id
  try {
    await testNotifyChannel(row.id)
    ElMessage.success('测试消息已发送，请检查对应渠道')
  } catch (e) {
    // error is shown by interceptor
  } finally {
    testingId.value = null
  }
}

const deleteChannel = async (row) => {
  await ElMessageBox.confirm(`确定删除渠道「${row.name}」？`, '确认删除', { type: 'warning' })
  try {
    await deleteNotifyChannel(row.id)
    ElMessage.success('已删除')
    loadChannels()
  } catch (e) {
    // ignore
  }
}

const formatTime = (t) => {
  if (!t) return '-'
  return new Date(t).toLocaleString('zh-CN', { hour12: false })
}

onMounted(loadChannels)
</script>

<style scoped>
.notify-channels-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.toolbar {
  display: flex;
  justify-content: flex-end;
}

.card {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
  padding: 16px;
}
</style>
