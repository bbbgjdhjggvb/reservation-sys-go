import { createRouter, createWebHistory, type RouteLocationNormalized } from 'vue-router'

const routes = [
  {
    path: '/',
    name: 'reserve',
    component: () => import('@/views/ReserveView.vue'),
    meta: { title: '深圳大学校友之家场地预约' },
  },
  {
    path: '/myorders',
    name: 'myorders',
    component: () => import('@/views/MyOrdersView.vue'),
    meta: { title: '我的预约' },
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: (to: RouteLocationNormalized) => ({ path: '/', query: to.query, hash: to.hash }),
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to) => {
  document.title = (to.meta.title as string) || '深圳大学校友之家场地预约'
})

export default router
