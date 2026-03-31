import axios from 'axios'
import { ElMessage } from 'element-plus'

const api = axios.create({
  baseURL: '/api',
  timeout: 30000
})

api.interceptors.response.use(
  response => response.data,
  error => {
    const msg = error.response?.data?.error || error.message || '请求失败'
    ElMessage.error(msg)
    return Promise.reject(error)
  }
)

// Dashboard APIs
export const getDashboardTrend = (params) => api.get('/dashboard/error-trend', { params })
export const getDashboardSummary = (params) => api.get('/dashboard/error-summary', { params })

// TopN APIs
export const getTopNServices = (params) => api.get('/topn/services', { params })
export const getTopNCallers = (params) => api.get('/topn/callers', { params })

// Log detail APIs
export const getLogs = (params) => api.get('/logs', { params })

// Alert rule APIs
export const getAlertRules = () => api.get('/alert-rules')
export const createAlertRule = (data) => api.post('/alert-rules', data)
export const updateAlertRule = (id, data) => api.put(`/alert-rules/${id}`, data)
export const deleteAlertRule = (id) => api.delete(`/alert-rules/${id}`)
export const toggleAlertRule = (id) => api.put(`/alert-rules/${id}/toggle`)

// Alert history APIs
export const getAlertHistory = (params) => api.get('/alert-history', { params })
export const resolveAlertHistory = (id) => api.put(`/alert-history/${id}/resolve`)

// Settings APIs
export const getSettings = () => api.get('/settings')
export const updateSettings = (data) => api.put('/settings', data)

export default api
