import { createRouter, createWebHistory } from 'vue-router'
import { useAuth } from './auth'

const routes = [
  { path: '/login', name: 'login', component: () => import('./views/LoginView.vue') },
  {
    path: '/',
    component: () => import('./layout/MainLayout.vue'),
    children: [
      { path: '', name: 'dashboard', component: () => import('./views/DashboardView.vue') },
      { path: 'gallery', name: 'gallery', component: () => import('./views/GalleryView.vue') },
      { path: 'ai', name: 'ai', component: () => import('./views/AIWorkstationView.vue') },
      { path: 'settings', name: 'settings', component: () => import('./views/SettingsView.vue') },
      { path: 'r2-guide', name: 'r2-guide', component: () => import('./views/R2GuideView.vue') },
    ],
  },
  { path: '/:pathMatch(.*)*', redirect: '/' },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to) => {
  const { token } = useAuth()
  if (to.path !== '/login' && !token.value) return '/login'
  if (to.path === '/login' && token.value) return '/'
  return true
})

export default router
