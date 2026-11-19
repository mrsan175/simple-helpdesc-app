import assert from 'node:assert/strict'
import { spawn, spawnSync } from 'node:child_process'
import { mkdtemp, readFile, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import test from 'node:test'
import { createServer } from 'vite'
import vue from '@vitejs/plugin-vue'

// Optional dependency-free browser smoke test. Set FILE_PREVIEW_BROWSER to a
// Chromium-compatible executable, or use the Brave installation on this host.
const executable = process.env.FILE_PREVIEW_BROWSER || '/opt/brave.com/brave/brave'
const hasBrowser = typeof WebSocket !== 'undefined' && spawnSync(executable, ['--version'], { timeout: 5000 }).status === 0
const delay = (ms) => new Promise((resolve) => setTimeout(resolve, ms))
async function until(check, label) {
  for (let i = 0; i < 100; i++) {
    const result = await check().catch(() => null)
    if (result) return result
    await delay(100)
  }
  throw new Error(`Timed out: ${label}`)
}

const harness = `
import { createApp, h, ref, nextTick } from 'vue'
import { createPinia } from 'pinia'
import FilePreview from '/src/components/shared/FilePreview.vue'
import DraftFilePreview from '/src/components/shared/DraftFilePreview.vue'
import { ticketService } from '/src/services/ticket.service.js'
import { parseInPreviewWorker } from '/src/services/file-preview-client.js'
import { translations } from '/src/i18n/index.js'
import { previewMessages } from '/src/i18n/file-preview.js'
Object.assign(translations.id, previewMessages.id)
Object.assign(translations.en, previewMessages.en)
const file = ref(null)
const attachment = ref(null)
const open = ref(false)
window.checkWorker = async () => { try { await parseInPreviewWorker(new Blob(['invalid']), 'bad.doc') } catch (error) { return error.code } }
window.calls = []
window.aborts = 0
window.liveUrls = new Set()
const createUrl = URL.createObjectURL.bind(URL)
const revokeUrl = URL.revokeObjectURL.bind(URL)
URL.createObjectURL = (blob) => { const url = createUrl(blob); window.liveUrls.add(url); return url }
URL.revokeObjectURL = (url) => { window.liveUrls.delete(url); revokeUrl(url) }
ticketService.previewAttachment = (id, signal) => {
  window.calls.push(id)
  return new Promise((resolve, reject) => {
    signal.addEventListener('abort', () => { window.aborts++; reject(new DOMException('aborted', 'AbortError')) })
    if (id !== 'slow') resolve(new Blob(['remote private text']))
  })
}
const app = createApp({ setup() { return () => h('main', [
  h('button', { id: 'outside' }, 'Outside'),
  h(DraftFilePreview, { file: file.value, onPreview: () => { open.value = true }, onRemove: () => { file.value = null } }),
  open.value ? h(FilePreview, { file: file.value, attachment: attachment.value, contextLabel: 'Private comment context', onClose: () => { open.value = false } }) : null
]) } })
app.use(createPinia())
app.mount('#app')
window.setLocal = async (bytes, name, modal = true) => {
  attachment.value = null
  file.value = new File([new Uint8Array(bytes)], name)
  open.value = modal
  await nextTick()
}
window.setRemote = async (id) => { file.value = null; attachment.value = { id, filename: 'remote.txt', file_size: 40 }; open.value = true; await nextTick() }
window.closePreview = async () => { open.value = false; await nextTick() }
window.ready = true
`

function pdfFixture() {
  const parts = ['%PDF-1.4\n']
  const offsets = [0]
  const stream = 'BT /F1 24 Tf 50 150 Td (Private PDF preview) Tj ET'
  const objects = [
    '<< /Type /Catalog /Pages 2 0 R >>',
    '<< /Type /Pages /Kids [3 0 R] /Count 1 >>',
    '<< /Type /Page /Parent 2 0 R /MediaBox [0 0 400 250] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>',
    '<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>',
    `<< /Length ${stream.length} >>\nstream\n${stream}\nendstream`
  ]
  for (let i = 0; i < objects.length; i++) {
    offsets.push(Buffer.byteLength(parts.join('')))
    parts.push(`${i + 1} 0 obj\n${objects[i]}\nendobj\n`)
  }
  const xref = Buffer.byteLength(parts.join(''))
  parts.push(`xref\n0 6\n0000000000 65535 f \n${offsets.slice(1).map((offset) => String(offset).padStart(10, '0') + ' 00000 n ').join('\n')}\ntrailer\n<< /Size 6 /Root 1 0 R >>\nstartxref\n${xref}\n%%EOF\n`)
  return [...Buffer.from(parts.join(''))]
}

test('browser: local draft/modal safety, focus, cleanup, cancellation and PDF embedding', { skip: !hasBrowser && 'Set FILE_PREVIEW_BROWSER to run the Chromium smoke test (Node 22+)', timeout: 90_000 }, async (t) => {
  const profile = await mkdtemp(join(tmpdir(), 'file-preview-browser-'))
  let server
  let browser
  let socket
  const pending = new Map()
  const consoleErrors = []
  const requests = []
  let sequence = 0
  try {
    server = await createServer({
      root: fileURLToPath(new URL('..', import.meta.url)), configFile: false,
      logLevel: 'silent', server: { host: '127.0.0.1', port: 0, hmr: false },
      plugins: [vue(), {
        name: 'file-preview-test-harness',
        resolveId(id) { if (id === '/__file-preview-test.js') return '\0file-preview-harness' },
        load(id) { if (id === '\0file-preview-harness') return harness },
        configureServer(vite) {
          vite.middlewares.use((req, res, next) => {
            if (req.url !== '/__file-preview-test') return next()
            res.setHeader('Content-Type', 'text/html')
            res.end('<!doctype html><div id="app"></div><script type="module" src="/__file-preview-test.js"></script>')
          })
        }
      }]
    })
    await server.listen()
    browser = spawn(executable, ['--headless=new', '--no-sandbox', '--disable-gpu', '--disable-dev-shm-usage', '--remote-debugging-port=0', `--user-data-dir=${profile}`, 'about:blank'], { stdio: 'ignore' })
    const port = await until(async () => (await readFile(join(profile, 'DevToolsActivePort'), 'utf8')).split('\n')[0], 'browser debugger')
    const pages = await (await fetch(`http://127.0.0.1:${port}/json/list`)).json()
    socket = new WebSocket(pages.find((page) => page.type === 'page').webSocketDebuggerUrl)
    await new Promise((resolve, reject) => { socket.addEventListener('open', resolve, { once: true }); socket.addEventListener('error', reject, { once: true }) })
    socket.addEventListener('message', ({ data }) => {
      const message = JSON.parse(data)
      if (message.id) {
        const callback = pending.get(message.id)
        pending.delete(message.id)
        if (message.error) callback.reject(new Error(JSON.stringify(message.error)))
        else callback.resolve(message.result)
      }
      if (message.method === 'Runtime.exceptionThrown') consoleErrors.push(message.params.exceptionDetails.exception?.description || message.params.exceptionDetails.text)
      if (message.method === 'Log.entryAdded' && message.params.entry.level === 'error') consoleErrors.push(`${message.params.entry.text} ${message.params.entry.url || ''}`)
      if (message.method === 'Network.requestWillBeSent') requests.push(message.params.request)
    })
    const cdp = (method, params = {}, sessionId) => new Promise((resolve, reject) => {
      const id = ++sequence
      pending.set(id, { resolve, reject })
      socket.send(JSON.stringify({ id, method, params, sessionId }))
    })
    const evaluate = async (expression) => {
      const result = await cdp('Runtime.evaluate', { expression, awaitPromise: true, returnByValue: true })
      if (result.exceptionDetails) throw new Error(JSON.stringify(result.exceptionDetails))
      return result.result.value
    }
    await cdp('Runtime.enable')
    await cdp('Log.enable')
    await cdp('Network.enable')
    await cdp('Page.enable')
    await cdp('Page.navigate', { url: `http://127.0.0.1:${server.httpServer.address().port}/__file-preview-test` })
    await until(() => evaluate('window.ready'), 'Vue harness')

    assert.equal(await until(() => evaluate('window.checkWorker?.()'), 'development worker/WASM'), 'office_invalid', 'development worker must load its WASM asset')
    const unsafe = '<img src=x onerror="window.unsafe=true">,=HYPERLINK("https://private.invalid")'
    await evaluate(`window.setLocal(${JSON.stringify([...Buffer.from(unsafe)])}, 'private.csv')`)
    await until(() => evaluate('document.querySelector("dialog pre")?.textContent'), 'literal CSV')
    assert.equal(await evaluate('document.querySelector("dialog pre").textContent'), unsafe)
    assert.equal(await evaluate('Boolean(window.unsafe)'), false)
    assert.equal(await evaluate('window.calls.length'), 0)
    assert.match(await evaluate('document.querySelector("dialog").textContent'), /Private comment context/)
    assert.equal(await evaluate('document.querySelector("dialog").open'), true)
    assert.equal(await evaluate('document.activeElement === document.querySelector("dialog button")'), true)
    await cdp('Input.dispatchKeyEvent', { type: 'keyDown', key: 'Tab', code: 'Tab', windowsVirtualKeyCode: 9 })
    assert.equal(await evaluate('document.activeElement === document.querySelector("dialog button")'), true)
    await cdp('Input.dispatchKeyEvent', { type: 'keyDown', key: 'Escape', code: 'Escape', windowsVirtualKeyCode: 27 })
    await until(async () => !(await evaluate('Boolean(document.querySelector("dialog"))')), 'Escape close')

    const png = [...Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=', 'base64')]
    await evaluate(`window.setLocal(${JSON.stringify(png)}, 'private.png')`)
    await until(() => evaluate('document.querySelector("dialog img")?.complete'), 'image preview')
    assert.equal(await evaluate('window.liveUrls.size'), 2, 'draft and modal each own a local URL')
    await evaluate('window.closePreview()')
    assert.equal(await evaluate('window.liveUrls.size'), 1)
    await evaluate(`window.setLocal([111,107], 'next.txt', false)`)
    assert.equal(await evaluate('window.liveUrls.size'), 0)

    await evaluate('window.setRemote("slow")')
    await evaluate(`window.setLocal([108,111,99,97,108], 'local.txt')`)
    await until(() => evaluate('document.querySelector("dialog pre")?.textContent === "local"'), 'stale remote cancellation')
    assert.equal(await evaluate('window.aborts'), 1)
    assert.equal(await evaluate('window.calls.length'), 1)

    await evaluate(`window.setLocal(${JSON.stringify(pdfFixture())}, 'private.pdf')`)
    await until(() => evaluate('Boolean(document.querySelector("dialog iframe"))'), 'sandboxed PDF frame')
    await delay(1500)
    assert.equal(await evaluate('document.querySelector("dialog iframe").getAttribute("sandbox")'), 'allow-scripts')
    assert.equal(await evaluate('window.calls.length'), 1, 'PDF stays local')
    const pdfTarget = (await cdp('Target.getTargets')).targetInfos.find((target) => target.type === 'iframe' && target.url.startsWith('blob:'))
    assert.ok(pdfTarget, 'sandboxed PDF should load in an isolated child target')
    // Chromium blocks PDF plugins in sandboxed frames, so verify the local
    // new-tab fallback really loads a PDF rather than merely checking an iframe.
    assert.equal(await evaluate('document.querySelector("dialog a").rel'), 'noopener noreferrer')
    await cdp('Runtime.evaluate', { expression: 'document.querySelector("dialog a").click()', userGesture: true })
    const pdfTab = await until(async () => (await cdp('Target.getTargets')).targetInfos.find((target) => target.type === 'page' && target.url.startsWith('blob:')), 'local PDF tab')
    const { sessionId } = await cdp('Target.attachToTarget', { targetId: pdfTab.targetId, flatten: true })
    const pdfTree = await until(async () => {
      const tree = await cdp('Page.getFrameTree', {}, sessionId)
      return tree.frameTree.frame.mimeType === 'application/pdf' && tree
    }, 'browser PDF document')
    assert.equal(pdfTree.frameTree.frame.mimeType, 'application/pdf')
    const opener = await cdp('Runtime.evaluate', { expression: 'window.opener === null', returnByValue: true }, sessionId)
    assert.equal(opener.result.value, true)
    await cdp('Target.closeTarget', { targetId: pdfTab.targetId })
    assert.ok(!requests.some((request) => request.method !== 'GET'), 'preview must never upload')
    assert.ok(!requests.some((request) => request.url.includes('private.invalid')), 'CSV links/formulas must not execute')
    assert.deepEqual(consoleErrors.filter((error) => !error.includes('favicon')), [])
    await evaluate('window.closePreview()')
    assert.equal(await evaluate('window.liveUrls.size'), 0)
  } finally {
    const errors = consoleErrors.filter((error) => !error.includes('favicon'))
    if (errors.length) t.diagnostic(`Browser errors: ${errors.join('; ')}`)
    socket?.close()
    browser?.kill('SIGKILL')
    await server?.close()
    await delay(200)
    await rm(profile, { recursive: true, force: true })
  }
})
