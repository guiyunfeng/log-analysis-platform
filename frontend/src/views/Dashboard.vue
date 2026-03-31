<template>
  <div class="dashboard-page">
    <!-- Toolbar -->
    <div class="toolbar card">
      <FilterBar v-model="filters" @filter="loadData" />
      <div class="toolbar-bottom">
        <TimeRangeSelector v-model="timeRange" @change="loadData" />
      </div>
    </div>

    <!-- Summary cards -->
    <el-row :gutter="16" class="summary-row">
      <el-col :span="6" v-for="card in summaryCards" :key="card.label">
        <div class="summary-card card">
          <div class="card-icon" :style="{ background: card.color }">
            <el-icon size="24" color="#fff"><component :is="card.icon" /></el-icon>
          </div>
          <div class="card-info">
            <div class="card-label">{{ card.label }}</div>
            <div class="card-value">{{ card.value }}</div>
          </div>
        </div>
      </el-col>
    </el-row>

    <!-- Error trend chart -->
    <div class="card chart-card">
      <div class="chart-header">
        <span class="chart-title">错误趋势</span>
        <span class="chart-subtitle">（按服务分组）</span>
      </div>
      <div v-if="trendLoading" class="chart-loading">
        <el-icon class="is-loading" size="32"><Loading /></el-icon>
      </div>
      <div v-else-if="trendSeries.length === 0" class="chart-empty">
        <el-empty description="暂无数据" :image-size="80" />
      </div>
      <v-chart v-else :option="trendOption" style="height: 380px" autoresize />
    </div>

    <!-- Summary pie charts -->
    <el-row :gutter="16">
      <el-col :span="12">
        <div class="card chart-card">
          <div class="chart-header">
            <span class="chart-title">按服务分布</span>
          </div>
          <div v-if="summaryLoading" class="chart-loading">
            <el-icon class="is-loading" size="32"><Loading /></el-icon>
          </div>
          <v-chart v-else :option="servicesPieOption" style="height: 300px" autoresize />
        </div>
      </el-col>
      <el-col :span="12">
        <div class="card chart-card">
          <div class="chart-header">
            <span class="chart-title">按项目分布</span>
          </div>
          <div v-if="summaryLoading" class="chart-loading">
            <el-icon class="is-loading" size="32"><Loading /></el-icon>
          </div>
          <v-chart v-else :option="projectsPieOption" style="height: 300px" autoresize />
        </div>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart, PieChart } from 'echarts/charts'
import {
  TitleComponent, TooltipComponent, LegendComponent,
  GridComponent, DataZoomComponent
} from 'echarts/components'
import FilterBar from '../components/FilterBar.vue'
import TimeRangeSelector from '../components/TimeRangeSelector.vue'
import { getDashboardTrend, getDashboardSummary } from '../api/index.js'

use([CanvasRenderer, LineChart, PieChart, TitleComponent, TooltipComponent,
  LegendComponent, GridComponent, DataZoomComponent])

const filters = reactive({ project: '', service: '', job: '' })
const timeRange = reactive({ start: null, end: null, step: '5m' })

const trendLoading = ref(false)
const summaryLoading = ref(false)
const trendSeries = ref([])
const summaryData = ref({})

const summaryCards = computed(() => {
  const byService = summaryData.value?.by_service || []
  const byProject = summaryData.value?.by_project || []
  const totalErrors = byService.reduce((s, i) => s + i.count, 0)
  const activeServices = byService.length
  const activeProjects = byProject.length

  return [
    { label: '总错误数', value: totalErrors.toLocaleString(), icon: 'Warning', color: '#ff4d4f' },
    { label: '活跃服务', value: activeServices, icon: 'Connection', color: '#1890ff' },
    { label: '活跃项目', value: activeProjects, icon: 'FolderOpened', color: '#52c41a' },
    { label: '机器数', value: (summaryData.value?.by_job || []).length, icon: 'Monitor', color: '#faad14' }
  ]
})

const trendOption = computed(() => {
  if (!trendSeries.value.length) return {}

  const series = trendSeries.value.map(s => ({
    name: s.name,
    type: 'line',
    smooth: true,
    data: (s.data || []).map(p => [p.time, Number(p.value.toFixed(4))]),
    areaStyle: { opacity: 0.1 }
  }))

  return {
    tooltip: {
      trigger: 'axis',
      formatter: (params) => {
        const time = new Date(params[0].axisValue).toLocaleString('zh-CN')
        let html = `<b>${time}</b><br/>`
        params.forEach(p => {
          html += `${p.marker}${p.seriesName}: ${(p.value[1] * 60).toFixed(1)} 次/分<br/>`
        })
        return html
      }
    },
    legend: { type: 'scroll', bottom: 0 },
    grid: { top: 20, left: 60, right: 20, bottom: 60 },
    xAxis: {
      type: 'time',
      axisLabel: { formatter: (v) => new Date(v).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }) }
    },
    yAxis: { type: 'value', name: '错误率(次/秒)', nameTextStyle: { fontSize: 11 } },
    dataZoom: [{ type: 'inside' }, { type: 'slider', height: 20 }],
    series
  }
})

const buildPieOption = (items, title) => {
  if (!items || items.length === 0) return { title: { text: '暂无数据', left: 'center', top: 'middle' } }
  const data = items.slice(0, 10).map(i => ({ name: i.name, value: i.count }))
  return {
    tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
    legend: { type: 'scroll', bottom: 0, textStyle: { fontSize: 11 } },
    series: [{
      type: 'pie',
      radius: ['35%', '65%'],
      center: ['50%', '45%'],
      data,
      label: { formatter: '{b}\n{d}%', fontSize: 11 }
    }]
  }
}

const servicesPieOption = computed(() => buildPieOption(summaryData.value?.by_service, '服务分布'))
const projectsPieOption = computed(() => buildPieOption(summaryData.value?.by_project, '项目分布'))

const loadData = async () => {
  if (!timeRange.start) return

  const params = {
    project: filters.project,
    service: filters.service,
    job: filters.job,
    start: timeRange.start,
    end: timeRange.end,
    step: timeRange.step
  }

  trendLoading.value = true
  summaryLoading.value = true

  try {
    const [trendRes, summaryRes] = await Promise.allSettled([
      getDashboardTrend(params),
      getDashboardSummary(params)
    ])

    if (trendRes.status === 'fulfilled') {
      trendSeries.value = trendRes.value?.data || []
    }
    if (summaryRes.status === 'fulfilled') {
      summaryData.value = summaryRes.value?.data || {}
    }
  } finally {
    trendLoading.value = false
    summaryLoading.value = false
  }
}

onMounted(() => {
  // TimeRangeSelector initializes with 1h range automatically
})
</script>

<style scoped>
.dashboard-page {
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
}

.summary-row {
  margin: 0 !important;
}

.summary-card {
  display: flex;
  align-items: center;
  padding: 20px;
  gap: 16px;
}

.card-icon {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.card-label {
  font-size: 13px;
  color: #999;
  margin-bottom: 4px;
}

.card-value {
  font-size: 24px;
  font-weight: 700;
  color: #333;
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

.chart-subtitle {
  font-size: 13px;
  color: #999;
  margin-left: 8px;
}

.chart-loading, .chart-empty {
  height: 200px;
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>
