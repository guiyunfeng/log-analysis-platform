<template>
  <div class="alert-rules-page">
    <!-- Header -->
    <div class="card page-header">
      <div class="header-left">
        <span class="page-title">告警规则管理</span>
        <span class="page-desc">管理告警触发条件、级别和静默时间</span>
      </div>
      <el-button type="primary" :icon="Plus" @click="openDialog()">新增规则</el-button>
    </div>

    <!-- Rules table -->
    <div class="card table-card">
      <el-table :data="rules" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="name" label="规则名称" width="160" show-overflow-tooltip />
        <el-table-column prop="severity" label="级别" width="100">
          <template #default="{ row }">
            <el-tag :type="severityType(row.severity)" size="small">{{ severityLabel(row.severity) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="匹配条件" min-width="180">
          <template #default="{ row }">
            <div class="condition-cell">
              <span v-if="row.project" class="cond-item">项目: {{ row.project }}</span>
              <span v-if="row.service" class="cond-item">服务: {{ row.service }}</span>
              <span v-if="row.caller_file" class="cond-item">调用点: {{ row.caller_file }}</span>
              <span v-if="row.content_pattern" class="cond-item pattern">内容: {{ row.content_pattern }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="阈值" width="150">
          <template #default="{ row }">
            <span>{{ row.time_window / 60 }}分钟内 &gt; {{ row.threshold }} 次</span>
          </template>
        </el-table-column>
        <el-table-column label="标签" width="120" show-overflow-tooltip>
          <template #default="{ row }">
            <span v-if="row.labels" class="labels-cell">{{ row.labels }}</span>
            <span v-else class="text-muted">-</span>
          </template>
        </el-table-column>
        <el-table-column label="生效时间" width="140">
          <template #default="{ row }">
            <div v-if="row.effective_start || row.effective_days" class="time-cell">
              <span v-if="row.effective_start && row.effective_end">{{ row.effective_start }}–{{ row.effective_end }}</span>
              <span v-if="row.effective_days" class="days-tag">{{ formatDays(row.effective_days) }}</span>
            </div>
            <span v-else class="text-muted">全天</span>
          </template>
        </el-table-column>
        <el-table-column prop="silence_minutes" label="静默(分)" width="90" />
        <el-table-column prop="enabled" label="状态" width="90">
          <template #default="{ row }">
            <el-switch
              v-model="row.enabled"
              @change="toggleRule(row)"
              active-color="#52c41a"
            />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="130">
          <template #default="{ row }">
            <el-button size="small" text type="primary" @click="openDialog(row)">编辑</el-button>
            <el-popconfirm title="确认删除该规则？" @confirm="deleteRule(row.id)">
              <template #reference>
                <el-button size="small" text type="danger">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- Edit/Create dialog -->
    <el-dialog
      v-model="dialogVisible"
      :title="editingRule ? '编辑规则' : '新增规则'"
      width="680px"
      destroy-on-close
    >
      <el-form :model="form" :rules="formRules" ref="formRef" label-width="110px">
        <!-- 基础信息 -->
        <el-divider content-position="left">基础信息</el-divider>
        <el-form-item label="规则名称" prop="name">
          <el-input v-model="form.name" placeholder="规则名称" />
        </el-form-item>
        <el-form-item label="告警级别" prop="severity">
          <el-select v-model="form.severity" style="width: 100%">
            <el-option label="🔴 Critical（立即推送）" value="critical" />
            <el-option label="⚠️ Warning（聚合推送）" value="warning" />
            <el-option label="🔇 Noise（静默，不推送）" value="noise" />
          </el-select>
        </el-form-item>
        <el-form-item label="标签">
          <el-input v-model="form.labels" placeholder="env:prod,team:backend" />
        </el-form-item>
        <el-form-item label="规则描述">
          <el-input v-model="form.description" type="textarea" :rows="2" placeholder="规则说明（可选）" />
        </el-form-item>

        <!-- 匹配条件 -->
        <el-divider content-position="left">匹配条件（空=匹配所有）</el-divider>
        <el-form-item label="项目">
          <el-input v-model="form.project" placeholder="项目名（空=所有）" />
        </el-form-item>
        <el-form-item label="服务">
          <el-input v-model="form.service" placeholder="服务名（空=所有）" />
        </el-form-item>
        <el-form-item label="调用点">
          <el-input v-model="form.caller_file" placeholder="调用点文件（空=所有）" />
        </el-form-item>
        <el-form-item label="内容关键字">
          <el-input v-model="form.content_pattern" placeholder="支持正则，如: not found|ErrCode" />
        </el-form-item>

        <!-- 触发条件 -->
        <el-divider content-position="left">触发条件</el-divider>
        <el-form-item label="时间窗口">
          <el-input-number v-model="form.time_window" :min="60" :max="86400" :step="60" />
          <span style="margin-left: 8px; color: #666">秒（{{ form.time_window / 60 }}分钟）</span>
        </el-form-item>
        <el-form-item label="次数阈值">
          <el-input-number v-model="form.threshold" :min="0" :max="100000" />
          <span style="margin-left: 8px; color: #666">次（超过此值触发）</span>
        </el-form-item>
        <el-form-item label="静默时间">
          <el-input-number v-model="form.silence_minutes" :min="0" :max="1440" />
          <span style="margin-left: 8px; color: #666">分钟（同规则重复告警间隔）</span>
        </el-form-item>
        <el-form-item label="最大告警次数">
          <el-input-number v-model="form.max_alert_count" :min="0" :max="10000" />
          <span style="margin-left: 8px; color: #666">次/天（0=不限制）</span>
        </el-form-item>

        <!-- 生效时间 -->
        <el-divider content-position="left">生效时间</el-divider>
        <el-form-item label="生效星期">
          <el-checkbox-group v-model="effectiveDaysArr">
            <el-checkbox :value="1" label="周一" />
            <el-checkbox :value="2" label="周二" />
            <el-checkbox :value="3" label="周三" />
            <el-checkbox :value="4" label="周四" />
            <el-checkbox :value="5" label="周五" />
            <el-checkbox :value="6" label="周六" />
            <el-checkbox :value="7" label="周日" />
          </el-checkbox-group>
        </el-form-item>
        <el-form-item label="生效时段">
          <el-time-select
            v-model="form.effective_start"
            start="00:00"
            step="00:30"
            end="23:30"
            placeholder="开始时间"
            style="width: 140px"
          />
          <span style="margin: 0 8px; color: #666">至</span>
          <el-time-select
            v-model="form.effective_end"
            start="00:00"
            step="00:30"
            end="23:30"
            placeholder="结束时间"
            style="width: 140px"
          />
          <span style="margin-left: 8px; color: #999; font-size: 12px">（空=全天）</span>
        </el-form-item>

        <!-- 通知配置 -->
        <el-divider content-position="left">通知配置</el-divider>
        <el-form-item label="通知渠道">
          <el-select
            v-model="notifyChannelsArr"
            multiple
            placeholder="默认发往全部启用渠道"
            style="width: 100%"
          >
            <el-option
              v-for="ch in availableChannels"
              :key="ch.id"
              :label="`[${ch.type}] ${ch.name}`"
              :value="ch.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="恢复通知">
          <el-switch v-model="form.notify_recovery" />
          <span style="margin-left: 8px; color: #666; font-size: 12px">告警恢复后发送通知</span>
        </el-form-item>
        <el-form-item label="恢复判定窗口" v-if="form.notify_recovery">
          <el-input-number v-model="form.recovery_window" :min="60" :max="86400" :step="60" />
          <span style="margin-left: 8px; color: #666">秒（{{ form.recovery_window / 60 }}分钟）</span>
        </el-form-item>

        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveRule" :loading="saving">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { getAlertRules, createAlertRule, updateAlertRule, deleteAlertRule, toggleAlertRule, getNotifyChannels } from '../api/index.js'

const loading = ref(false)
const saving = ref(false)
const rules = ref([])
const dialogVisible = ref(false)
const editingRule = ref(null)
const formRef = ref(null)
const availableChannels = ref([])

const defaultForm = {
  name: '',
  severity: 'warning',
  project: '',
  service: '',
  caller_file: '',
  content_pattern: '',
  time_window: 300,
  threshold: 1,
  silence_minutes: 30,
  enabled: true,
  labels: '',
  description: '',
  effective_start: '',
  effective_end: '',
  effective_days: '1,2,3,4,5',
  notify_channels: '',
  notify_recovery: false,
  recovery_window: 600,
  max_alert_count: 0,
}

const form = reactive({ ...defaultForm })

// Derived arrays for checkbox-group / multi-select bindings
const effectiveDaysArr = computed({
  get() {
    if (!form.effective_days) return []
    return form.effective_days.split(',').map(Number).filter(Boolean)
  },
  set(val) {
    form.effective_days = val.slice().sort((a, b) => a - b).join(',')
  }
})

const notifyChannelsArr = computed({
  get() {
    if (!form.notify_channels) return []
    return form.notify_channels.split(',').map(Number).filter(Boolean)
  },
  set(val) {
    form.notify_channels = val.join(',')
  }
})

const formRules = {
  name: [{ required: true, message: '请输入规则名称', trigger: 'blur' }],
  severity: [{ required: true, message: '请选择告警级别', trigger: 'change' }]
}

const severityType = (severity) => {
  return { critical: 'danger', warning: 'warning', noise: 'info' }[severity] || 'info'
}

const severityLabel = (severity) => {
  return { critical: 'Critical', warning: 'Warning', noise: 'Noise' }[severity] || severity
}

const dayNames = { 1: '一', 2: '二', 3: '三', 4: '四', 5: '五', 6: '六', 7: '日' }
const formatDays = (days) => {
  if (!days) return ''
  return days.split(',').map(d => `周${dayNames[d] || d}`).join(' ')
}

const loadRules = async () => {
  loading.value = true
  try {
    const res = await getAlertRules()
    rules.value = res?.data || []
  } finally {
    loading.value = false
  }
}

const loadChannels = async () => {
  try {
    const res = await getNotifyChannels()
    availableChannels.value = res?.data || res || []
  } catch (e) {
    // ignore
  }
}

const openDialog = (rule = null) => {
  editingRule.value = rule
  if (rule) {
    Object.assign(form, { ...defaultForm, ...rule })
  } else {
    Object.assign(form, { ...defaultForm })
  }
  dialogVisible.value = true
}

const saveRule = async () => {
  await formRef.value.validate()
  saving.value = true
  try {
    if (editingRule.value) {
      await updateAlertRule(editingRule.value.id, form)
      ElMessage.success('规则已更新')
    } else {
      await createAlertRule(form)
      ElMessage.success('规则已创建')
    }
    dialogVisible.value = false
    await loadRules()
  } finally {
    saving.value = false
  }
}

const deleteRule = async (id) => {
  await deleteAlertRule(id)
  ElMessage.success('规则已删除')
  await loadRules()
}

const toggleRule = async (rule) => {
  await toggleAlertRule(rule.id)
  ElMessage.success(rule.enabled ? '已启用' : '已禁用')
}

onMounted(() => {
  loadRules()
  loadChannels()
})
</script>

<style scoped>
.alert-rules-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.card {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
}

.page-header {
  padding: 16px 24px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.header-left {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.page-title {
  font-size: 18px;
  font-weight: 600;
  color: #333;
}

.page-desc {
  font-size: 13px;
  color: #999;
}

.table-card {
  padding: 0;
  overflow: hidden;
}

:deep(.el-table) {
  border-radius: 8px;
}

.condition-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.cond-item {
  font-size: 12px;
  color: #333;
  background: #f5f5f5;
  border-radius: 4px;
  padding: 1px 6px;
  display: inline-block;
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.cond-item.pattern {
  background: #fff7e6;
  color: #d46b08;
}

.labels-cell {
  font-size: 12px;
  color: #666;
}

.time-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
  font-size: 12px;
  color: #555;
}

.days-tag {
  font-size: 11px;
  color: #999;
}

.text-muted {
  color: #ccc;
  font-size: 12px;
}
</style>

    <!-- Rules table -->
    <div class="card table-card">
      <el-table :data="rules" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="name" label="规则名称" width="180" show-overflow-tooltip />
        <el-table-column prop="severity" label="级别" width="100">
          <template #default="{ row }">
            <el-tag :type="severityType(row.severity)" size="small">{{ severityLabel(row.severity) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="匹配条件" min-width="200">
          <template #default="{ row }">
            <div class="condition-cell">
              <span v-if="row.project" class="cond-item">项目: {{ row.project }}</span>
              <span v-if="row.service" class="cond-item">服务: {{ row.service }}</span>
              <span v-if="row.caller_file" class="cond-item">调用点: {{ row.caller_file }}</span>
              <span v-if="row.content_pattern" class="cond-item pattern">内容: {{ row.content_pattern }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="阈值" width="150">
          <template #default="{ row }">
            <span>{{ row.time_window / 60 }}分钟内 &gt; {{ row.threshold }} 次</span>
          </template>
        </el-table-column>
        <el-table-column prop="silence_minutes" label="静默(分钟)" width="100" />
        <el-table-column prop="enabled" label="状态" width="90">
          <template #default="{ row }">
            <el-switch
              v-model="row.enabled"
              @change="toggleRule(row)"
              active-color="#52c41a"
            />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="130">
          <template #default="{ row }">
            <el-button size="small" text type="primary" @click="openDialog(row)">编辑</el-button>
            <el-popconfirm title="确认删除该规则？" @confirm="deleteRule(row.id)">
              <template #reference>
                <el-button size="small" text type="danger">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- Edit/Create dialog -->
    <el-dialog
      v-model="dialogVisible"
      :title="editingRule ? '编辑规则' : '新增规则'"
      width="600px"
      destroy-on-close
    >
      <el-form :model="form" :rules="formRules" ref="formRef" label-width="100px">
        <el-form-item label="规则名称" prop="name">
          <el-input v-model="form.name" placeholder="规则名称" />
        </el-form-item>
        <el-form-item label="告警级别" prop="severity">
          <el-select v-model="form.severity" style="width: 100%">
            <el-option label="🔴 Critical（立即推送）" value="critical" />
            <el-option label="⚠️ Warning（聚合推送）" value="warning" />
            <el-option label="🔇 Noise（静默，不推送）" value="noise" />
          </el-select>
        </el-form-item>
        <el-divider>匹配条件（空=匹配所有）</el-divider>
        <el-form-item label="项目">
          <el-input v-model="form.project" placeholder="项目名（空=所有）" />
        </el-form-item>
        <el-form-item label="服务">
          <el-input v-model="form.service" placeholder="服务名（空=所有）" />
        </el-form-item>
        <el-form-item label="调用点">
          <el-input v-model="form.caller_file" placeholder="调用点文件（空=所有）" />
        </el-form-item>
        <el-form-item label="内容关键字">
          <el-input v-model="form.content_pattern" placeholder="支持正则，如: not found|ErrCode" />
        </el-form-item>
        <el-divider>触发条件</el-divider>
        <el-form-item label="时间窗口">
          <el-input-number v-model="form.time_window" :min="60" :max="86400" :step="60" />
          <span style="margin-left: 8px; color: #666">秒（{{ form.time_window / 60 }}分钟）</span>
        </el-form-item>
        <el-form-item label="次数阈值">
          <el-input-number v-model="form.threshold" :min="0" :max="100000" />
          <span style="margin-left: 8px; color: #666">次（超过此值触发）</span>
        </el-form-item>
        <el-form-item label="静默时间">
          <el-input-number v-model="form.silence_minutes" :min="0" :max="1440" />
          <span style="margin-left: 8px; color: #666">分钟（同规则重复告警间隔）</span>
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveRule" :loading="saving">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>
