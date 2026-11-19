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
  updateComment(id, payload) {
    return api.patch(`/comments/${id}`, payload).then((r) => r.data)
  },
  deleteComment(id) {
    return api.delete(`/comments/${id}`)
  },
  uploadCommentAttachment(commentId, file) {
    const formData = new FormData()
    formData.append('file', file)
    return api.post(`/comments/${commentId}/attachments`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    }).then((r) => r.data)
  },
  replaceAttachment(id, file) {
    const formData = new FormData()
    formData.append('file', file)
    return api.patch(`/attachments/${id}`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    }).then((r) => r.data)
  },
  deleteAttachment(id) {
    return api.delete(`/attachments/${id}`)
  },
  uploadAttachment(id, file) {
    const formData = new FormData()
    formData.append('file', file)
    return api.post(`/tickets/${id}/attachments`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    }).then((r) => r.data)
  },
  downloadAttachment(attachmentId) {
    const token = localStorage.getItem('token')
    const baseURL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api'
    // Direct link with token or window open
    window.open(`${baseURL}/attachments/${attachmentId}/download?token=${token}`, '_blank')
  },
  stats() {
    return api.get('/stats').then((r) => r.data)
  },
  adminStats() {
    return api.get('/admin/stats').then((r) => r.data)
  }
}
