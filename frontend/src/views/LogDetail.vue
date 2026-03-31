<template>
  <div class="log-detail-page">
    <!-- Search filters -->
    <div class="card toolbar">
      <el-form :inline="true" :model="filters" size="small">
        <el-form-item label="项目">
          <el-input v-model="filters.project" placeholder="项目名" clearable style="width: 120px" />
        </el-form-item>
        <el-form-item label="服务">
          <el-input v-model="filters.service" placeholder="服务名" clearable style="width: 120px" />
        </el-form-item>
        <el-form-item label="调用点">
          <el-input v-model="filters.caller_file" placeholder="调用点文件" clearable style="width: 160px" />
        </el-form-item>
        <el-form-item label="机器">
          <el-input v-model="filters.job" placeholder="机器标识" clearable style="width: 160px" />
        </el-form-item>
        <el-form-item label="关键字">
          <el-input
            v-model="filters.keyword"
            placeholder="搜索关键字"
            clearable
            style="width: 200px"
            @keyup.enter="loadLogs"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :icon="Search" @click="loadLogs" :loading="loading">查询</el-button>
          <el-button @click="resetFilters">重置</el-button>
        </el-form-item>
      </el-form>
      <div class="time-row">
        <TimeRangeSelector v-model="timeRange" @change="loadLogs" />
        <span class="result-count">共 {{ total }} 条记录</span>
      </div>
    </div>

    <!-- Log table -->
    <div class="card table-card">
      <el-table
        :data="logs"
        v-loading="loading"
        stripe
        size="small"
        style="width: 100%"
        row-class-name="log-row"
        @row-click="showDetail"
      >
        <el-table-column prop="timestamp" label="时间" width="180" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="timestamp">{{ formatTime(row.timestamp) }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="project" label="项目" width="100" show-overflow-tooltip />
        <el-table-column prop="service" label="服务" width="100" show-overflow-tooltip />
        <el-table-column prop="caller" label="调用点" width="180" show-overflow-tooltip />
        <el-table-column prop="content" label="错误内容" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="content-text">{{ row.content }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="job" label="机器" width="200" show-overflow-tooltip />
        <el-table-column label="操作" width="80">
          <template #default="{ row }">
            <el-button size="small" text type="primary" @click.stop="showDetail(row)">详情</el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- Pagination -->
      <div class="pagination-row">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[20, 50, 100, 200]"
          :total="total"
          layout="total, sizes, prev, pager, next"
          @size-change="loadLogs"
          @current-change="loadLogs"
        />
      </div>
    </div>

    <!-- Detail drawer -->
    <el-drawer v-model="detailVisible" title="日志详情" size="50%" direction="rtl">
      <div v-if="selectedLog" class="log-detail">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="时间戳">{{ selectedLog.timestamp }}</el-descriptions-item>
          <el-descriptions-item label="项目">{{ selectedLog.project }}</el-descriptions-item>
          <el-descriptions-item label="服务">{{ selectedLog.service }}</el-descriptions-item>
          <el-descriptions-item label="调用点">{{ selectedLog.caller }}</el-descriptions-item>
          <el-descriptions-item label="机器">{{ selectedLog.job }}</el-descriptions-item>
          <el-descriptions-item label="级别">
            <el-tag type="danger" size="small">{{ selectedLog.level || 'error' }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item v-if="selectedLog.trace" label="Trace ID">{{ selectedLog.trace }}</el-descriptions-item>
          <el-descriptions-item v-if="selectedLog.span" label="Span ID">{{ selectedLog.span }}</el-descriptions-item>
        </el-descriptions>
        <div class="content-block">
          <div class="content-label">错误内容:</div>
          <pre class="content-pre">{{ selectedLog.content }}</pre>
        </div>
      </div>
    </el-drawer>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { Search } from '@element-plus/icons-vue'
import TimeRangeSelector from '../components/TimeRangeSelector.vue'
import { getLogs } from '../api/index.js'

const route = useRoute()

const filters = reactive({
  project: route.query.project || '',
  service: route.query.service || '',
  caller_file: route.query.caller_file || '',
  job: route.query.job || '',
  keyword: ''
})

const timeRange = reactive({
  start: route.query.start ? parseInt(route.query.start) : null,
  end: route.query.end ? parseInt(route.query.end) : null,
  step: '5m'
})

const loading = ref(false)
const logs = ref([])
const total = ref(0)
const currentPage = ref(1)
const pageSize = ref(50)
const detailVisible = ref(false)
const selectedLog = ref(null)

const loadLogs = async () => {
  if (!timeRange.start) return

  loading.value = true
  try {
    const res = await getLogs({
      project: filters.project,
      service: filters.service,
      caller_file: filters.caller_file,
      job: filters.job,
      keyword: filters.keyword,
      start: timeRange.start,
      end: timeRange.end,
      limit: pageSize.value
    })
    logs.value = res?.data || []
    total.value = res?.total || 0
  } finally {
    loading.value = false
  }
}

const resetFilters = () => {
  filters.project = ''
  filters.service = ''
  filters.caller_file = ''
  filters.job = ''
  filters.keyword = ''
  loadLogs()
}

const showDetail = (row) => {
  selectedLog.value = row
  detailVisible.value = true
}

const formatTime = (ts) => {
  if (!ts) return '-'
  try {
    return new Date(ts).toLocaleString('zh-CN')
  } catch {
    return ts
  }
}

onMounted(() => {
  if (timeRange.start) {
    loadLogs()
  }
})
</script>

<style scoped>
.log-detail-page {
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

.time-row {
  padding-top: 12px;
  border-top: 1px solid #f0f0f0;
  margin-top: 4px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.result-count {
  font-size: 13px;
  color: #666;
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
  color: #ff4d4f;
  font-size: 12px;
}

:deep(.log-row) {
  cursor: pointer;
}

:deep(.log-row:hover td) {
  background: #fff7f7 !important;
}

.log-detail {
  padding: 0 4px;
}

.content-block {
  margin-top: 16px;
}

.content-label {
  font-weight: 600;
  margin-bottom: 8px;
  color: #333;
}

.content-pre {
  background: #f5f5f5;
  border-radius: 4px;
  padding: 12px;
  font-size: 13px;
  white-space: pre-wrap;
  word-break: break-all;
  color: #ff4d4f;
  border: 1px solid #e8e8e8;
}
</style>
