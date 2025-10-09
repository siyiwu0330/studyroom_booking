// src/router.js
import { createRouter, createWebHistory } from 'vue-router'
import RegisterPage from '../views/RegisterView.vue'

// Lazy-loaded examples (optional)
const HomePage  = () => import('../views/RegisterView.vue')
const LoginPage = () => import('../views/RegisterView.vue')

const routes = [
  { path: '/', name: 'home', component: HomePage },
  { path: '/register', name: 'register', component: RegisterPage },
  { path: '/login', name: 'login', component: LoginPage },
  // 404 fallback
  { path: '/:pathMatch(.*)*', name: 'not-found', component: { template: '<p>Not found</p>' } }
]

export const router = createRouter({
  history: createWebHistory(), // use createWebHashHistory() if you prefer # URLs or lack server config
  routes,
  scrollBehavior() {
    return { top: 0 }
  }
})
