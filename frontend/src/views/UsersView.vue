<template>
  <div class="space-y-6">
    <div>
      <h1 class="text-2xl font-bold tracking-tight text-slate-900">Manajemen User</h1>
      <p class="text-sm text-slate-500 mt-1">Kelola role seluruh pengguna helpdesk.</p>
    </div>

    <LoadingSkeleton v-if="loading" :rows="4" />

    <ErrorAlert v-else-if="error" :message="error" :show-retry="true" @retry="fetchUsers" />

    <EmptyState v-else-if="users.length === 0" title="Belum ada user" message="Belum ada pengguna terdaftar." />

    <div v-else class="card overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead class="border-b bg-slate-50">
            <tr>
              <th class="text-left font-medium text-slate-600 px-4 py-3">Nama</th>
              <th class="text-left font-medium text-slate-600 px-4 py-3">Email</th>
              <th class="text-left font-medium text-slate-600 px-4 py-3">Role</th>
              <th class="text-left font-medium text-slate-600 px-4 py-3">Tanggal Bergabung</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr v-for="u in users" :key="u.id" class="hover:bg-slate-50/60">
              <td class="px-4 py-3">
                <div class="flex items-center gap-3">
                  <div class="w-8 h-8 rounded-full bg-slate-100 text-slate-600 font-semibold text-xs flex items-center justify-center">
                    {{ u.name?.charAt(0).toUpperCase() }}
                  </div>
                  <span class="font-medium text-slate-900">{{ u.name }}</span>
                  <span v-if="u.id === auth.user?.id" class="text-[10px] uppercase font-semibold text-slate-400">Anda</span>
                </div>
              </td>
              <td class="px-4 py-3 text-slate-600">{{ u.email }}</td>
              <td class="px-4 py-3">
                <select
                  :value="u.role"
                  class="input h-8 py-1 text-xs w-32"
                  :disabled="savingId === u.id"
                  @change="changeRole(u, $event.target.value)"
                >
                  <option value="requester">Requester</option>
                  <option value="agent">Agent</option>
                  <option value="admin">Admin</option>
                </select>
              </td>
              <td class="px-4 py-3 text-slate-500 whitespace-nowrap">
                <p>{{ formatDate(u.created_at) }}</p>
                <p class="text-xs text-slate-400">{{ formatTime(u.created_at) }}</p>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <Toast :message="toast.message" :type="toast.type" :visible="toast.visible" />
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { useAuthStore } from '../stores/auth'
import { userService } from '../services/user.service'
import LoadingSkeleton from '../components/shared/LoadingSkeleton.vue'
import ErrorAlert from '../components/shared/ErrorAlert.vue'
import EmptyState from '../components/shared/EmptyState.vue'
import Toast from '../components/shared/Toast.vue'

const auth = useAuthStore()
const users = ref([])
const loading = ref(false)
const error = ref('')
const savingId = ref(null)
const toast = reactive({ message: '', type: 'success', visible: false })

let toastTimer = null

const fetchUsers = async () => {
  loading.value = true
  error.value = ''
  try {
    const res = await userService.list()
    users.value = res.data || []
  } catch (err) {
    error.value = err.message || 'Gagal memuat daftar user'
  } finally {
    loading.value = false
  }
}

const changeRole = async (user, newRole) => {
  if (user.role === newRole) return
  savingId.value = user.id
  try {
    await userService.updateRole(user.id, newRole)
    user.role = newRole
    showToast('Role berhasil diperbarui')
  } catch (err) {
    showToast(err.message || 'Gagal memperbarui role', 'error')
    await fetchUsers()
  } finally {
    savingId.value = null
  }
}

const showToast = (message, type = 'success') => {
  toast.message = message
  toast.type = type
  toast.visible = true
  if (toastTimer) clearTimeout(toastTimer)
  toastTimer = setTimeout(() => {
    toast.visible = false
  }, 3000)
}

const formatDate = (value) => {
  if (!value) return '-'
  return new Date(value).toLocaleDateString('id-ID', { day: '2-digit', month: 'short', year: 'numeric' })
}

const formatTime = (value) => {
  if (!value) return '-'
  return new Date(value).toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' })
}

onMounted(fetchUsers)
</script>