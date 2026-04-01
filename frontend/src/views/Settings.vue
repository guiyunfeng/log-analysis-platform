<template>
  <div class="settings-page">
    <div class="card settings-card">
      <div class="section-title">连接配置</div>
      <el-form :model="form" label-width="160px" size="default">
        <el-form-item label="Loki 服务地址">
          <el-input
            v-model="form.loki_url"
            placeholder="http://your-loki:3100"
            style="width: 400px"
          />
          <el-button style="margin-left: 8px" @click="testLoki" :loading="testingLoki">
            测试连接
          </el-button>
        </el-form-item>
      </el-form>

      <el-divider />

      <div class="section-title">钉钉安全配置</div>
      <el-form :model="form" label-width="160px" size="default">
        <el-form-item label="全局钉钉加签密钥">
          <el-input
            v-model="form.dingtalk_secret"
            type="password"
            show-password
            placeholder="SEC开头的加签密钥（可留空）"
            style="width: 400px"
          />
          <span class="hint">用于在全局 DingTalk 渠道中开启 HMAC-SHA256 加签安全模式</span>
        </el-form-item>
        <el-form-item label="全局钉钉关键词">
          <el-input
            v-model="form.dingtalk_keywords"
            placeholder="告警,故障,异常（多个逗号分隔）"
            style="width: 400px"
          />
          <span class="hint">多个关键词，消息中至少包含其一才能通过关键词安全验证</span>
        </el-form-item>
      </el-form>

      <el-divider />

      <div class="section-title">告警全局配置</div>
      <el-form :model="form" label-width="160px" size="default">
        <el-form-item label="突增倍数阈值">
          <el-input-number v-model="form.spike_multiplier" :min="1" :max="100" />
          <span class="hint">当前5分钟错误数是过去1小时均值的N倍时触发突增告警</span>
        </el-form-item>
        <el-form-item label="全局错误阈值">
          <el-input-number v-model="form.global_threshold" :min="1" :max="10000" />
          <span class="hint">未匹配规则的服务，5分钟内超过此数量触发 warning</span>
        </el-form-item>
        <el-form-item label="全局时间窗口">
          <el-input-number v-model="form.global_time_window" :min="60" :max="86400" :step="60" />
          <span class="hint">秒（{{ form.global_time_window ? form.global_time_window / 60 : 0 }}分钟）</span>
        </el-form-item>
        <el-form-item label="全局静默时间">
          <el-input-number v-model="form.global_silence_minutes" :min="0" :max="1440" />
          <span class="hint">分钟（同服务+级别告警的最小间隔）</span>
        </el-form-item>
        <el-form-item label="Warning聚合间隔">
          <el-input-number v-model="form.warning_batch_interval" :min="1" :max="60" />
          <span class="hint">分钟（warning告警汇总推送的间隔）</span>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="saveSettings" :loading="saving">
            保存配置
          </el-button>
        </el-form-item>
      </el-form>
    </div>

    <!-- About -->
    <div class="card about-card">
      <div class="section-title">关于</div>
      <el-descriptions :column="2" border>
        <el-descriptions-item label="平台名称">日志分析告警平台</el-descriptions-item>
        <el-descriptions-item label="版本">v1.0.0</el-descriptions-item>
        <el-descriptions-item label="后端">Go + Gin</el-descriptions-item>
        <el-descriptions-item label="前端">Vue3 + ECharts + Element Plus</el-descriptions-item>
        <el-descriptions-item label="数据库">MySQL 8.0</el-descriptions-item>
        <el-descriptions-item label="日志源">Loki</el-descriptions-item>
      </el-descriptions>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getSettings, updateSettings } from '../api/index.js'

const saving = ref(false)
const testingLoki = ref(false)

const form = reactive({
  loki_url: '',
  dingtalk_secret: '',
  dingtalk_keywords: '',
  spike_multiplier: 10,
  global_threshold: 100,
  global_time_window: 300,
  global_silence_minutes: 30,
  warning_batch_interval: 5
})

const loadSettings = async () => {
  try {
    const res = await getSettings()
    const data = res?.data || {}
    if (data.loki_url) form.loki_url = data.loki_url
    if (data.dingtalk_secret !== undefined) form.dingtalk_secret = data.dingtalk_secret
    if (data.dingtalk_keywords !== undefined) form.dingtalk_keywords = data.dingtalk_keywords
    if (data.spike_multiplier) form.spike_multiplier = Number(data.spike_multiplier)
    if (data.global_threshold) form.global_threshold = Number(data.global_threshold)
    if (data.global_time_window) form.global_time_window = Number(data.global_time_window)
    if (data.global_silence_minutes) form.global_silence_minutes = Number(data.global_silence_minutes)
    if (data.warning_batch_interval) form.warning_batch_interval = Number(data.warning_batch_interval)
  } catch (e) {
    // ignore
  }
}

const saveSettings = async () => {
  saving.value = true
  try {
    const payload = {
      loki_url: form.loki_url,
      dingtalk_secret: form.dingtalk_secret,
      dingtalk_keywords: form.dingtalk_keywords,
      spike_multiplier: String(form.spike_multiplier),
      global_threshold: String(form.global_threshold),
      global_time_window: String(form.global_time_window),
      global_silence_minutes: String(form.global_silence_minutes),
      warning_batch_interval: String(form.warning_batch_interval)
    }
    await updateSettings(payload)
    ElMessage.success('配置已保存')
  } finally {
    saving.value = false
  }
}

const testLoki = async () => {
  if (!form.loki_url) {
    ElMessage.warning('请先填写 Loki 地址')
    return
  }
  testingLoki.value = true
  try {
    const res = await fetch(`${form.loki_url}/ready`)
    if (res.ok) {
      ElMessage.success('Loki 连接成功')
    } else {
      ElMessage.error(`Loki 连接失败: ${res.status}`)
    }
  } catch (e) {
    ElMessage.error(`Loki 连接失败: ${e.message}`)
  } finally {
    testingLoki.value = false
  }
}

onMounted(loadSettings)
</script>

<style scoped>
.settings-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.card {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
}

.settings-card, .about-card {
  padding: 24px;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: #333;
  margin-bottom: 20px;
  padding-bottom: 12px;
  border-bottom: 1px solid #f0f0f0;
}

.hint {
  margin-left: 12px;
  font-size: 12px;
  color: #999;
}
</style>
