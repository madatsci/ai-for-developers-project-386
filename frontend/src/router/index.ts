import { createRouter, createWebHistory } from 'vue-router'
import BookingView from '@/views/BookingView.vue'
import HomeView from '@/views/HomeView.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/', name: 'home', component: HomeView },
    { path: '/booking', name: 'booking', component: BookingView },
  ],
})

export default router
