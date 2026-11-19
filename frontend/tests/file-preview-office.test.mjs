import assert from 'node:assert/strict'
import { spawnSync } from 'node:child_process'
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'
import { Worker } from 'node:worker_threads'
import { createInflateRaw } from 'node:zlib'
import { Duplex } from 'node:stream'
import test from 'node:test'
import { build } from 'vite'
import vue from '@vitejs/plugin-vue'
import init, { WasmDocument } from 'office-oxide-wasm/web'
import { extractOfficeText, parseZipDirectory } from '../src/services/file-preview.js'

// Node 20 compatibility for the browser-only deflate-raw stream API.
try { new DecompressionStream('deflate-raw') } catch {
  const NativeStream = globalThis.DecompressionStream
  globalThis.DecompressionStream = function (format) {
    return format === 'deflate-raw' ? Duplex.toWeb(createInflateRaw()) : new NativeStream(format)
  }
}

const root = fileURLToPath(new URL('..', import.meta.url))
const packageRoot = dirname(fileURLToPath(import.meta.resolve('office-oxide-wasm/package.json')))
await init({ module_or_path: await readFile(join(packageRoot, 'web/office_oxide_bg.wasm')) })

const documentXml = `<?xml version="1.0" encoding="UTF-8"?>
<office:document xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0" xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0" office:version="1.2" office:mimetype="application/vnd.oasis.opendocument.text">
<office:body><office:text><text:p>Private preview fixture &lt;script&gt;not executable&lt;/script&gt;</text:p><text:p>Indonesia 12345</text:p></office:text></office:body></office:document>`
const spreadsheetXml = `<?xml version="1.0" encoding="UTF-8"?>
<office:document xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0" xmlns:table="urn:oasis:names:tc:opendocument:xmlns:table:1.0" xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0" office:version="1.2" office:mimetype="application/vnd.oasis.opendocument.spreadsheet">
<office:body><office:spreadsheet><table:table table:name="Private sheet"><table:table-row><table:table-cell office:value-type="string"><text:p>Private preview fixture</text:p></table:table-cell><table:table-cell office:value-type="float" office:value="12345"><text:p>12345</text:p></table:table-cell></table:table-row></table:table></office:spreadsheet></office:body></office:document>`

async function officeFixtures(directory) {
  await writeFile(join(directory, 'document.fodt'), documentXml)
  await writeFile(join(directory, 'spreadsheet.fods'), spreadsheetXml)
  const fixtures = new Map()
  for (const [extension, filter, source] of [
    ['doc', 'MS Word 97', 'document.fodt'],
    ['docx', 'Office Open XML Text', 'document.fodt'],
    ['xls', 'MS Excel 97', 'spreadsheet.fods'],
    ['xlsx', 'Calc MS Excel 2007 XML', 'spreadsheet.fods']
  ]) {
    const result = spawnSync('libreoffice', [
      `-env:UserInstallation=${pathToFileURL(join(directory, 'profile')).href}`,
      '--headless', '--convert-to', `${extension}:${filter}`, '--outdir', directory, join(directory, source)
    ], { timeout: 30_000, encoding: 'utf8' })
    assert.equal(result.status, 0, `LibreOffice ${extension}: ${result.stderr || result.error || result.stdout}`)
    fixtures.set(extension, await readFile(join(directory, `${source.split('.')[0]}.${extension}`)))
  }
  return fixtures
}

async function bundlePreview() {
  const output = await build({
    root,
    configFile: false,
    publicDir: false,
    plugins: [vue()],
    logLevel: 'silent',
    base: '/private-app/',
    build: {
      write: false,
      minify: false,
      rollupOptions: { preserveEntrySignatures: 'strict', input: [resolve(root, 'src/components/shared/FilePreview.vue'), resolve(root, 'src/components/shared/DraftFilePreview.vue')] }
    }
  })
  return output.output
}

// Execute exactly the worker emitted by Vite (including its web WASM loader),
// adapting only the messaging/fetch environment. No server or network is used.
function runBundledWorker(script, wasm, name, bytes) {
  return new Promise((resolve, reject) => {
    const worker = new Worker(`
      const { parentPort, workerData } = require('node:worker_threads')
      try { new DecompressionStream('deflate-raw') } catch {
        const NativeStream = globalThis.DecompressionStream
        globalThis.DecompressionStream = function (format) {
          return format === 'deflate-raw'
            ? require('node:stream').Duplex.toWeb(require('node:zlib').createInflateRaw())
            : new NativeStream(format)
        }
      }
      global.self = global
      self.location = { href: 'https://local.invalid/private-app/assets/worker.js' }
      self.postMessage = (data) => parentPort.postMessage(data)
      global.fetch = async (url) => {
        if (!String(url).endsWith('.wasm')) throw new Error('Unexpected network request: ' + url)
        return new Response(workerData.wasm, { headers: { 'Content-Type': 'application/wasm' } })
      }
      eval(workerData.script)
      parentPort.on('message', (data) => self.onmessage({ data }))
    `, { eval: true, workerData: { script, wasm } })
    const timer = setTimeout(() => { worker.terminate(); reject(new Error('Bundled worker timed out')) }, 15_000)
    const finish = (error, data) => {
      clearTimeout(timer)
      worker.terminate()
      if (error) reject(error)
      else resolve(data)
    }
    worker.once('error', (error) => finish(error))
    worker.once('message', (data) => finish(null, data))
    const copy = Uint8Array.from(bytes)
    worker.postMessage({ name, buffer: copy.buffer }, [copy.buffer])
  })
}

test('Vite packages both components, worker and local WASM asset with a non-root deployment base', { timeout: 60_000 }, async () => {
  const output = await bundlePreview()
  const worker = output.find((asset) => /file-preview\.worker.*\.js$/.test(asset.fileName))
  const wasm = output.find((asset) => /office_oxide_bg.*\.wasm$/.test(asset.fileName))
  assert.ok(worker, 'Vite must emit the worker even before parent integrations exist')
  assert.ok(wasm, 'Vite must emit the package WASM, not leave a missing relative URL')
  const script = String(worker.source || worker.code)
  assert.ok(script.includes(`/private-app/${wasm.fileName}`), 'worker must resolve the WASM asset under the deployment base')
  assert.ok(wasm.source.byteLength > 100_000)
  const emptyZip = Buffer.alloc(22)
  emptyZip.writeUInt32LE(0x06054b50)
  assert.deepEqual(await runBundledWorker(script, wasm.source, 'empty.zip', emptyZip), { result: { entries: [] } })
  const invalidOffice = await runBundledWorker(script, wasm.source, 'bad.doc', Buffer.from('not a document'))
  assert.equal(invalidOffice.error, 'office_invalid')
  const unavailableWasm = await runBundledWorker(script, new Uint8Array(), 'bad.doc', Buffer.from('not a document'))
  assert.equal(unavailableWasm.error, 'worker', 'WASM load errors should not blame the document')
})

const hasLibreOffice = spawnSync('libreoffice', ['--version'], { timeout: 10_000 }).status === 0

test('real DOC/DOCX/XLS/XLSX fixtures extract private text through web WASM and the Vite-built worker', { skip: !hasLibreOffice && 'LibreOffice is not installed; install it to generate Office fixtures', timeout: 180_000 }, async (t) => {
  const directory = await mkdtemp(join(tmpdir(), 'file-preview-office-'))
  try {
    const fixtures = await officeFixtures(directory)
    const output = await bundlePreview()
    const worker = output.find((asset) => /file-preview\.worker.*\.js$/.test(asset.fileName))
    const wasm = output.find((asset) => /office_oxide_bg.*\.wasm$/.test(asset.fileName))
    for (const [extension, bytes] of fixtures) {
      await t.test(extension, async () => {
        const result = await extractOfficeText(bytes, extension, WasmDocument)
        assert.match(result.text, /Private preview fixture/)
        assert.match(result.text, /12345/)
        assert.equal(result.truncated, false)
        if (extension === 'docx') assert.match(result.text, /not executable/)
        if (extension.endsWith('x')) assert.ok(parseZipDirectory(bytes).expandedBytes < 20 * 1024 * 1024)
        const bundled = await runBundledWorker(String(worker.source || worker.code), wasm.source, `fixture.${extension}`, bytes)
        assert.equal(bundled.error, undefined, `bundled worker ${extension}: ${bundled.error}`)
        assert.equal(bundled.result.text, result.text)
      })
    }
  } finally {
    await rm(directory, { recursive: true, force: true })
  }
})
