import { createRouter, createWebHistory } from 'vue-router'
import Layout from '../components/Layout.vue'

const routes = [
  {
    path: '/',
    component: Layout,
    redirect: '/dashboard',
    children: [
      {
        path: '/dashboard',
        name: 'Dashboard',
        component: () => import('../views/Dashboard.vue'),
        meta: { title: '错误大盘' }
      },
      {
        path: '/topn',
        name: 'TopN',
        component: () => import('../views/TopN.vue'),
        meta: { title: 'TopN 排行' }
      },
      {
        path: '/logs',
        name: 'LogDetail',
        component: () => import('../views/LogDetail.vue'),
        meta: { title: '日志明细' }
      },
      {
        path: '/alert-rules',
        name: 'AlertRules',
        component: () => import('../views/AlertRules.vue'),
        meta: { title: '告警规则' }
      },
      {
        path: '/alert-history',
        name: 'AlertHistory',
        component: () => import('../views/AlertHistory.vue'),
        meta: { title: '告警历史' }
      },
      {
        path: '/settings',
        name: 'Settings',
        component: () => import('../views/Settings.vue'),
        meta: { title: '系统设置' }
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
