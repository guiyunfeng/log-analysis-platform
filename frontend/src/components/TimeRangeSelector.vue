<template>
  <div class="time-range-selector">
    <el-radio-group v-model="selectedRange" @change="onQuickChange" size="small">
      <el-radio-button v-for="opt in quickOptions" :key="opt.value" :label="opt.value">
        {{ opt.label }}
      </el-radio-button>
    </el-radio-group>
    <el-date-picker
      v-if="selectedRange === 'custom'"
      v-model="customRange"
      type="datetimerange"
      range-separator="至"
      start-placeholder="开始时间"
      end-placeholder="结束时间"
      size="small"
      style="margin-left: 12px; width: 380px"
      @change="onCustomChange"
    />
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'

const props = defineProps({
  modelValue: {
    type: Object,
    default: () => ({ start: null, end: null })
  }
})

const emit = defineEmits(['update:modelValue', 'change'])

const quickOptions = [
  { label: '5分钟', value: '5m', minutes: 5 },
  { label: '15分钟', value: '15m', minutes: 15 },
  { label: '1小时', value: '1h', minutes: 60 },
  { label: '6小时', value: '6h', minutes: 360 },
  { label: '12小时', value: '12h', minutes: 720 },
  { label: '24小时', value: '24h', minutes: 1440 },
  { label: '7天', value: '7d', minutes: 10080 },
  { label: '自定义', value: 'custom', minutes: null }
]

const selectedRange = ref('1h')
const customRange = ref(null)

const getTimeRange = (minutes) => {
  const end = Math.floor(Date.now() / 1000)
  const start = end - minutes * 60
  return { start, end }
}

const getStep = (minutes) => {
  if (minutes <= 15) return '1m'
  if (minutes <= 60) return '5m'
  if (minutes <= 360) return '15m'
  if (minutes <= 1440) return '1h'
  return '6h'
}

const onQuickChange = (val) => {
  const opt = quickOptions.find(o => o.value === val)
  if (!opt || !opt.minutes) return
  const range = getTimeRange(opt.minutes)
  emit('update:modelValue', { ...range, step: getStep(opt.minutes) })
  emit('change', { ...range, step: getStep(opt.minutes) })
}

const onCustomChange = (val) => {
  if (!val || val.length < 2) return
  const start = Math.floor(val[0].getTime() / 1000)
  const end = Math.floor(val[1].getTime() / 1000)
  const minutes = (end - start) / 60
  emit('update:modelValue', { start, end, step: getStep(minutes) })
  emit('change', { start, end, step: getStep(minutes) })
}

// Initialize with default range
onQuickChange('1h')
</script>

<style scoped>
.time-range-selector {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
</style>
