export const TICKET_STATUSES = ['open', 'in_progress', 'resolved', 'closed']

export const TICKET_PRIORITIES = ['low', 'medium', 'high', 'urgent']

export const TICKET_CATEGORIES = [
  'IT Support',
  'Hardware',
  'Software',
  'Network',
  'HR/Facility'
]

export const STATUS_LABELS = {
  open: 'Open',
  in_progress: 'In Progress',
  resolved: 'Resolved',
  closed: 'Closed'
}

export const PRIORITY_LABELS = {
  low: 'Low',
  medium: 'Medium',
  high: 'High',
  urgent: 'Urgent'
}

// Client-side transition map: target status -> roles allowed
export const TRANSITIONS = {
  open: {
    in_progress: ['agent', 'admin'],
    closed: ['requester', 'admin']
  },
  in_progress: {
    resolved: ['agent', 'admin']
  },
  resolved: {
    closed: ['requester', 'agent', 'admin'],
    open: ['requester']
  },
  closed: {}
}

export function canTransition(currentStatus, targetStatus, role) {
  const allowed = TRANSITIONS[currentStatus]?.[targetStatus]
  return Array.isArray(allowed) && allowed.includes(role)
}
