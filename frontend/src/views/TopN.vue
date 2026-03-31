<template>
  <div class="topn-page">
    <!-- Toolbar -->
    <div class="card toolbar">
      <FilterBar v-model="filters" @filter="loadData" />
      <div class="toolbar-bottom">
        <TimeRangeSelector v-model="timeRange" @change="loadData" />
        <div class="topn-controls">
          <span class="label">展示数量:</span>
          <el-radio-group v-model="limit" @change="loadData" size="small">
            <el-radio-button :label="5">Top 5</el-radio-button>
            <el-radio-button :label="10">Top 10</el-radio-button>
            <el-radio-button :label="20">Top 20</el-radio-button>
          </el-radio-group>
          <span class="label" style="margin-left: 16px">展示方式:</span>
          <el-radio-group v-model="displayMode" size="small">
            <el-radio-button label="chart">图表</el-radio-button>
            <el-radio-button label="table">表格</el-radio-button>
            <el-radio-button label="both">两者</el-radio-button>
          </el-radio-group>
        </div>
      </div>
    </div>

    <!-- Services TopN -->
    <div class="card chart-card">
      <div class="chart-header">
        <span class="chart-title">🔥 错误最多的服务 Top{{ limit }}</span>
      </div>
      <el-skeleton :loading="servicesLoading" animated>
        <template #default>
          <div v-if="!services.length" class="empty-state">
            <el-empty description="暂无数据" :image-size="80" />
          </div>
          <div v-else>
            <v-chart
              v-if="displayMode !== 'table'"
              :option="servicesBarOption"
              style="height: 300px"
              autoresize
            />
            <el-table
              v-if="displayMode !== 'chart'"
              :data="services"
              stripe
              size="small"
              style="margin-top: 16px"
            >
              <el-table-column type="index" label="排名" width="60" />
              <el-table-column prop="name" label="服务名" />
              <el-table-column prop="extra.project" label="项目" />
              <el-table-column prop="count" label="错误数" sortable>
                <template #default="{ row }">
                  <el-tag type="danger">{{ row.count.toLocaleString() }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="100">
                <template #default="{ row }">
                  <el-button size="small" text type="primary" @click="drillDown(row.name, '')">
                    下钻查看
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </template>
      </el-skeleton>
    </div>

    <!-- Callers TopN -->
    <div class="card chart-card">
      <div class="chart-header">
        <span class="chart-title">📁 错误最多的调用点 Top{{ limit }}</span>
      </div>
      <el-skeleton :loading="callersLoading" animated>
        <template #default>
          <div v-if="!callers.length" class="empty-state">
            <el-empty description="暂无数据" :image-size="80" />
          </div>
          <div v-else>
            <v-chart
              v-if="displayMode !== 'table'"
              :option="callersBarOption"
              style="height: 300px"
              autoresize
            />
            <el-table
              v-if="displayMode !== 'chart'"
              :data="callers"
              stripe
              size="small"
              style="margin-top: 16px"
            >
              <el-table-column type="index" label="排名" width="60" />
              <el-table-column prop="name" label="调用点" show-overflow-tooltip />
              <el-table-column prop="extra.service" label="服务" width="120" />
              <el-table-column prop="count" label="错误数" sortable>
                <template #default="{ row }">
                  <el-tag type="danger">{{ row.count.toLocaleString() }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="100">
                <template #default="{ row }">
                  <el-button size="small" text type="primary" @click="drillDown(row.extra?.service, row.name)">
                    下钻查看
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </template>
      </el-skeleton>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed } from 'vue'
import { useRouter } from 'vue-router'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { BarChart } from 'echarts/charts'
import { TitleComponent, TooltipComponent, GridComponent, LegendComponent } from 'echarts/components'
import FilterBar from '../components/FilterBar.vue'
import TimeRangeSelector from '../components/TimeRangeSelector.vue'
import { getTopNServices, getTopNCallers } from '../api/index.js'

use([CanvasRenderer, BarChart, TitleComponent, TooltipComponent, GridComponent, LegendComponent])

const router = useRouter()
const filters = reactive({ project: '', service: '', job: '' })
const timeRange = reactive({ start: null, end: null, step: '5m' })
const limit = ref(10)
const displayMode = ref('both')
const servicesLoading = ref(false)
const callersLoading = ref(false)
const services = ref([])
const callers = ref([])

const buildBarOption = (data, color) => {
  const items = [...data].reverse()
  return {
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    grid: { top: 10, left: 120, right: 60, bottom: 20 },
    xAxis: { type: 'value', minInterval: 1 },
    yAxis: {
      type: 'category',
      data: items.map(i => i.name),
      axisLabel: {
        fontSize: 11,
        width: 110,
        overflow: 'truncate',
        formatter: v => v.length > 20 ? v.substring(0, 20) + '...' : v
      }
    },
    series: [{
      type: 'bar',
      data: items.map(i => i.count),
      itemStyle: { color, borderRadius: [0, 4, 4, 0] },
      label: { show: true, position: 'right', formatter: '{c}' }
    }]
  }
}

const servicesBarOption = computed(() => buildBarOption(services.value, '#ff4d4f'))
const callersBarOption = computed(() => buildBarOption(callers.value, '#1890ff'))

const loadData = async () => {
  if (!timeRange.start) return

  const params = {
    project: filters.project,
    service: filters.service,
    job: filters.job,
    start: timeRange.start,
    end: timeRange.end,
    limit: limit.value
  }

  servicesLoading.value = true
  callersLoading.value = true

  try {
    const [svcRes, callerRes] = await Promise.allSettled([
      getTopNServices(params),
      getTopNCallers(params)
    ])
    if (svcRes.status === 'fulfilled') services.value = svcRes.value?.data || []
    if (callerRes.status === 'fulfilled') callers.value = callerRes.value?.data || []
  } finally {
    servicesLoading.value = false
    callersLoading.value = false
  }
}

const drillDown = (service, callerFile) => {
  router.push({
    path: '/logs',
    query: {
      service: service || '',
      caller_file: callerFile || '',
      start: timeRange.start,
      end: timeRange.end
    }
  })
}
</script>

<style scoped>
.topn-page {
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

.toolbar-bottom {
  padding-top: 12px;
  border-top: 1px solid #f0f0f0;
  margin-top: 4px;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 16px;
}

.topn-controls {
  display: flex;
  align-items: center;
  gap: 8px;
}

.label {
  font-size: 13px;
  color: #666;
}

.chart-card {
  padding: 20px;
}

.chart-header {
  margin-bottom: 16px;
}

.chart-title {
  font-size: 16px;
  font-weight: 600;
  color: #333;
}

.empty-state {
  padding: 40px 0;
  display: flex;
  justify-content: center;
}
</style>
