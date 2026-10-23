import { createRouter, createWebHistory } from 'vue-router'
import BookingCatalogView from '@/views/BookingCatalogView.vue'
import BookingView from '@/views/BookingView.vue'
import HomeView from '@/views/HomeView.vue'
import AdminPrototypeView from '@/views/admin-prototype/AdminPrototypeView.vue'
import CallsPrototypeView from '@/views/calls-prototype/CallsPrototypeView.vue'
import AdminWorkingHoursView from '@/views/AdminWorkingHoursView.vue'
import AdminSchedulesView from '@/views/AdminSchedulesView.vue'
import AdminCallTypesView from '@/views/AdminCallTypesView.vue'
import AdminCallsView from '@/views/AdminCallsView.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/', name: 'home', component: HomeView },
    { path: '/booking', name: 'booking', component: BookingCatalogView },
    { path: '/booking/:typeSlug', name: 'booking-type', component: BookingView },
    { path: '/admin/prototype', name: 'admin-prototype', component: AdminPrototypeView },
    {
      path: '/admin/calls',
      name: 'admin-calls',
      component: AdminCallsView,
    },
    {
      path: '/admin/call-types',
      name: 'admin-call-types',
      component: AdminCallTypesView,
    },
    {
      path: '/admin/schedules',
      name: 'admin-schedules',
      component: AdminSchedulesView,
    },
    {
      path: '/admin/working-hours',
      name: 'admin-working-hours',
      component: AdminWorkingHoursView,
    },
    {
      path: '/admin/calls-prototype',
      name: 'admin-calls-prototype',
      component: CallsPrototypeView,
    },
  ],
})

export default router
