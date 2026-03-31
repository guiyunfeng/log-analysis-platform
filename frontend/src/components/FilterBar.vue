<template>
  <div class="filter-bar">
    <el-form :inline="true" :model="filters" size="small">
      <el-form-item label="项目">
        <el-input
          v-model="filters.project"
          placeholder="输入项目名"
          clearable
          style="width: 140px"
          @clear="onFilter"
          @keyup.enter="onFilter"
        />
      </el-form-item>
      <el-form-item label="服务">
        <el-input
          v-model="filters.service"
          placeholder="输入服务名"
          clearable
          style="width: 140px"
          @clear="onFilter"
          @keyup.enter="onFilter"
        />
      </el-form-item>
      <el-form-item label="机器" v-if="showJob">
        <el-input
          v-model="filters.job"
          placeholder="输入机器标识"
          clearable
          style="width: 180px"
          @clear="onFilter"
          @keyup.enter="onFilter"
        />
      </el-form-item>
      <el-form-item>
        <el-button type="primary" @click="onFilter" :icon="Search">查询</el-button>
        <el-button @click="onReset">重置</el-button>
      </el-form-item>
    </el-form>
  </div>
</template>

<script setup>
import { reactive } from 'vue'
import { Search } from '@element-plus/icons-vue'

const props = defineProps({
  modelValue: {
    type: Object,
    default: () => ({ project: '', service: '', job: '' })
  },
  showJob: {
    type: Boolean,
    default: true
  }
})

const emit = defineEmits(['update:modelValue', 'filter'])

const filters = reactive({
  project: props.modelValue?.project || '',
  service: props.modelValue?.service || '',
  job: props.modelValue?.job || ''
})

const onFilter = () => {
  emit('update:modelValue', { ...filters })
  emit('filter', { ...filters })
}

const onReset = () => {
  filters.project = ''
  filters.service = ''
  filters.job = ''
  onFilter()
}
</script>

<style scoped>
.filter-bar {
  background: #fff;
  padding: 16px 20px 0;
  border-radius: 8px;
  margin-bottom: 16px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
}
</style>
