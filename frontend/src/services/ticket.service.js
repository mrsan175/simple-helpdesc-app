import api from './api'

export const ticketService = {
  list(params = {}) {
    return api.get('/tickets', { params }).then((r) => r.data)
  },
  get(id) {
    return api.get(`/tickets/${id}`).then((r) => r.data)
  },
  create(payload) {
    return api.post('/tickets', payload).then((r) => r.data)
  },
  update(id, payload) {
    return api.patch(`/tickets/${id}`, payload).then((r) => r.data)
  },
  addComment(id, payload) {
    return api.post(`/tickets/${id}/comments`, payload).then((r) => r.data)
  },
  stats() {
    return api.get('/stats').then((r) => r.data)
  }
}
