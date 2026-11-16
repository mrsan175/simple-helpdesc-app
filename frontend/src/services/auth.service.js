import api from './api'

export const authService = {
  login(payload) {
    return api.post('/auth/login', payload).then((r) => r.data)
  },
  register(payload) {
    return api.post('/auth/register', payload).then((r) => r.data)
  },
  me() {
    return api.get('/me').then((r) => r.data)
  }
}
