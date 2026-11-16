import api from './api'

export const userService = {
  list(role) {
    return api.get('/users', { params: role ? { role } : {} }).then((r) => r.data)
  },
  updateRole(id, role) {
    return api.patch(`/users/${id}/role`, { role }).then((r) => r.data)
  }
}
