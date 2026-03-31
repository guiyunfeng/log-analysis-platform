<template>
  <div class="alert-history-page">
    <!-- Filters -->
    <div class="card toolbar">
      <el-form :inline="true" :model="filters" size="small">
        <el-form-item label="级别">
          <el-select v-model="filters.severity" clearable placeholder="所有" style="width: 120px" @change="loadHistory">
            <el-option label="🔴 Critical" value="critical" />
            <el-option label="⚠️ Warning" value="warning" />
            <el-option label="🔇 Noise" value="noise" />
          </el-select>
        </el-form-item>
        <el-form-item label="服务">
          <el-input v-model="filters.service" placeholder="服务名" clearable style="width: 140px" @clear="loadHistory" />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="filters.resolved" clearable placeholder="所有" style="width: 100px" @change="loadHistory">
            <el-option label="未处理" value="false" />
            <el-option label="已处理" value="true" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="loadHistory" :loading="loading">查询</el-button>
          <el-button @click="resetFilters">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <!-- Stats summary -->
    <el-row :gutter="16">
      <el-col :span="6" v-for="stat in stats" :key="stat.label">
        <div class="stat-card card">
          <div class="stat-value" :style="{ color: stat.color }">{{ stat.value }}</div>
          <div class="stat-label">{{ stat.label }}</div>
        </div>
      </el-col>
    </el-row>

    <!-- History table -->
    <div class="card table-card">
      <el-table :data="history" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="created_at" label="时间" width="165">
          <template #default="{ row }">
            <span class="timestamp">{{ formatTime(row.created_at) }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="severity" label="级别" width="100">
          <template #default="{ row }">
            <el-tag :type="severityType(row.severity)" size="small">{{ row.severity.toUpperCase() }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="project" label="项目" width="100" show-overflow-tooltip />
        <el-table-column prop="service" label="服务" width="100" show-overflow-tooltip />
        <el-table-column prop="caller_file" label="调用点" width="160" show-overflow-tooltip />
        <el-table-column prop="error_count" label="错误数" width="90">
          <template #default="{ row }">
            <el-tag type="danger" size="small">{{ row.error_count }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="comparison" label="环比" width="100">
          <template #default="{ row }">
            <span :class="row.comparison?.startsWith('↑') ? 'up' : 'down'">{{ row.comparison }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="sample_content" label="示例报错" min-width="200" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="content-text">{{ row.sample_content }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="resolved" label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="row.resolved ? 'success' : 'warning'" size="small">
              {{ row.resolved ? '已处理' : '未处理' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="notified" label="推送" width="80">
          <template #default="{ row }">
            <el-tag :type="row.notified ? 'success' : 'info'" size="small">
              {{ row.notified ? '已推' : '未推' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button
              v-if="!row.resolved"
              size="small"
              text
              type="success"
              @click="resolve(row)"
            >
              标记处理
            </el-button>
            <span v-else class="resolved-time">{{ formatTime(row.resolved_at) }}</span>
          </template>
        </el-table-column>
      </el-table>

      <!-- Pagination -->
      <div class="pagination-row">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[20, 50, 100]"
          :total="total"
          layout="total, sizes, prev, pager, next"
          @size-change="loadHistory"
          @current-change="loadHistory"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getAlertHistory, resolveAlertHistory } from '../api/index.js'

const loading = ref(false)
const history = ref([])
const total = ref(0)
const currentPage = ref(1)
const pageSize = ref(20)

const filters = reactive({
  severity: '',
  service: '',
  resolved: ''
})

const stats = computed(() => {
  const allData = history.value
  return [
    { label: '本页告警总数', value: allData.length, color: '#333' },
    { label: 'Critical', value: allData.filter(h => h.severity === 'critical').length, color: '#ff4d4f' },
    { label: 'Warning', value: allData.filter(h => h.severity === 'warning').length, color: '#faad14' },
    { label: '未处理', value: allData.filter(h => !h.resolved).length, color: '#ff7875' }
  ]
})

const severityType = (severity) => {
  return { critical: 'danger', warning: 'warning', noise: 'info' }[severity] || 'info'
}

const formatTime = (ts) => {
  if (!ts) return '-'
  try {
    return new Date(ts).toLocaleString('zh-CN')
  } catch {
    return ts
  }
}

const loadHistory = async () => {
  loading.value = true
  try {
    const res = await getAlertHistory({
      severity: filters.severity,
      service: filters.service,
      resolved: filters.resolved,
      page: currentPage.value,
      page_size: pageSize.value
    })
    history.value = res?.data || []
    total.value = res?.total || 0
  } finally {
    loading.value = false
  }
}

const resetFilters = () => {
  filters.severity = ''
  filters.service = ''
  filters.resolved = ''
  loadHistory()
}

const resolve = async (row) => {
  await resolveAlertHistory(row.id)
  ElMessage.success('已标记为已处理')
  row.resolved = true
  await loadHistory()
}

onMounted(loadHistory)
</script>

<style scoped>
.alert-history-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.card {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
}

.toolbar {
  padding: 16px 20px;
}

.stat-card {
  padding: 20px;
  text-align: center;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  margin-bottom: 4px;
}

.stat-label {
  font-size: 13px;
  color: #999;
}

.table-card {
  padding: 0;
  overflow: hidden;
}

.pagination-row {
  padding: 16px 20px;
  display: flex;
  justify-content: flex-end;
  border-top: 1px solid #f0f0f0;
}

.timestamp {
  font-family: monospace;
  font-size: 12px;
  color: #666;
}

.content-text {
  font-size: 12px;
  color: #ff4d4f;
}

.up { color: #ff4d4f; font-weight: 600; }
.down { color: #52c41a; font-weight: 600; }

.resolved-time {
  font-size: 11px;
  color: #999;
}
</style>
