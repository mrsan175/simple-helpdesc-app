import { defineStore } from 'pinia'
import { authService } from '../services/auth.service'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: localStorage.getItem('token') || '',
    user: JSON.parse(localStorage.getItem('user') || 'null'),
    loading: false
  }),
  getters: {
    isAuthenticated: (state) => Boolean(state.token && state.user),
    isAdmin: (state) => state.user?.role === 'admin',
    isAgent: (state) => state.user?.role === 'agent',
    isRequester: (state) => state.user?.role === 'requester'
  },
  actions: {
    async login(credentials) {
      this.loading = true
      try {
        const res = await authService.login(credentials)
        this.setAuth(res.data)
        return res.data
      } finally {
        this.loading = false
      }
    },
    async register(data) {
      this.loading = true
      try {
        const res = await authService.register(data)
        this.setAuth(res.data)
        return res.data
      } finally {
        this.loading = false
      }
    },
    async fetchMe() {
      if (!this.token) return null
      try {
        const res = await authService.me()
        this.user = res.data
        localStorage.setItem('user', JSON.stringify(res.data))
        return res.data
      } catch (err) {
        this.logout()
        return null
      }
    },
    setAuth(authData) {
      this.token = authData.token
      this.user = authData.user
      localStorage.setItem('token', authData.token)
      localStorage.setItem('user', JSON.stringify(authData.user))
    },
    logout() {
      this.token = ''
      this.user = null
      localStorage.removeItem('token')
      localStorage.removeItem('user')
    }
  }
})
