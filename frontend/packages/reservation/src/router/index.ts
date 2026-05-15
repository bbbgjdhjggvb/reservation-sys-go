import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  {
    path: '/',
    name: 'reserve',
    component: () => import('@/views/ReserveView.vue'),
    meta: { title: '场地预约系统' },
  },
  {
    path: '/myorders',
    name: 'myorders',
    component: () => import('@/views/MyOrdersView.vue'),
    meta: { title: '我的预约' },
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: (to) => ({ path: '/', query: to.query, hash: to.hash }),
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to) => {
  document.title = (to.meta.title as string) || '场地预约系统'
})

export default router
