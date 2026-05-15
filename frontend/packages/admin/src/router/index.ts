import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  {
    path: '/admin/login',
    name: 'login',
    component: () => import('@/views/LoginView.vue'),
    meta: { title: '管理员登录' },
  },
  {
    path: '/admin/dashboard',
    name: 'dashboard',
    component: () => import('@/views/DashboardView.vue'),
    meta: { title: '审核中心', requiresAuth: true },
  },
  {
    path: '/admin',
    redirect: '/admin/login',
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/admin/login',
  },
]

const router = createRouter({
  history: createWebHistory('/admin/'),
  routes,
})

router.beforeEach((to) => {
  document.title = (to.meta.title as string) || '场地预约管理系统'

  if (to.meta.requiresAuth) {
    const token = localStorage.getItem('admin_token')
    if (!token) {
      return { name: 'login', query: { redirect: to.fullPath } }
    }
  }

  if (to.name === 'login' && localStorage.getItem('admin_token')) {
    return { name: 'dashboard' }
  }
})

export default router
