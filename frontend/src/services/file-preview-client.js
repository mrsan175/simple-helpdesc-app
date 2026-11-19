import { PREVIEW_TIMEOUT_MS, PreviewError, validatePreviewFile } from './file-preview.js'

function createWorker() {
  return new Worker(new URL('./file-preview.worker.js', import.meta.url), { type: 'module' })
}

// One short-lived worker per preview isolates malformed documents and releases
// the WASM heap on success, failure, source change, close, or timeout.
export function parseInPreviewWorker(blob, name, { signal, timeoutMs = PREVIEW_TIMEOUT_MS, workerFactory = createWorker } = {}) {
  return new Promise((resolve, reject) => {
    let worker
    let timer
    let finished = false
    const finish = (error, result) => {
      if (finished) return
      finished = true
      clearTimeout(timer)
      signal?.removeEventListener('abort', abort)
      worker?.terminate()
      if (error) reject(error)
      else resolve(result)
    }
    const abort = () => finish(new DOMException('Preview cancelled', 'AbortError'))
    if (signal?.aborted) return abort()
    try {
      validatePreviewFile(blob, name)
      worker = workerFactory()
      worker.onmessage = ({ data }) => {
        if (data?.error) finish(new PreviewError(data.error))
        else if (data?.result) finish(null, data.result)
        else finish(new PreviewError('worker'))
      }
      worker.onerror = (event) => {
        event.preventDefault?.()
        finish(new PreviewError('worker'))
      }
      worker.onmessageerror = () => finish(new PreviewError('worker'))
      signal?.addEventListener('abort', abort, { once: true })
      timer = setTimeout(() => finish(new PreviewError('timeout')), timeoutMs)
      blob.arrayBuffer().then((buffer) => {
        if (!finished) worker.postMessage({ buffer, name }, [buffer])
      }).catch(() => finish(new PreviewError('invalid')))
    } catch (error) {
      finish(error instanceof PreviewError ? error : new PreviewError('worker'))
    }
  })
}
