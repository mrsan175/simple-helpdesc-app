/**
 * SSE realtime client.
 * Usage: const { onEvent, disconnect } = useRealtime()
 * Events: 'ticket_created', 'ticket_updated', 'comment_added', 'attachment_added'
 */

const BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api'

let _es = null
const _handlers = new Map()

export function useRealtime() {
  function connect() {
    if (_es) return
    const token = localStorage.getItem('token')
    if (!token) return

    _es = new EventSource(`${BASE_URL}/events?token=${encodeURIComponent(token)}`)

    _es.onmessage = (e) => dispatch(e.type || 'message', e.data)

    for (const eventType of ['ticket_created', 'ticket_updated', 'comment_added', 'attachment_added']) {
      _es.addEventListener(eventType, (e) => dispatch(eventType, e.data))
    }

    _es.onerror = () => {
      // Reconnect in 5s; EventSource auto-reconnects for transient failures
    }
  }

  function dispatch(type, raw) {
    try {
      const payload = JSON.parse(raw)
      const handlers = _handlers.get(type) || []
      handlers.forEach((fn) => fn(payload))
    } catch (_) {}
  }

  function onEvent(type, fn) {
    if (!_handlers.has(type)) _handlers.set(type, [])
    _handlers.get(type).push(fn)
    connect()
    return () => {
      const list = _handlers.get(type) || []
      const idx = list.indexOf(fn)
      if (idx > -1) list.splice(idx, 1)
    }
  }

  function disconnect() {
    if (_es) {
      _es.close()
      _es = null
    }
    _handlers.clear()
  }

  return { onEvent, disconnect, connect }
}
