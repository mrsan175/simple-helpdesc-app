import assert from 'node:assert/strict'
import { createInflateRaw, deflateRawSync } from 'node:zlib'
import { Duplex } from 'node:stream'
import test from 'node:test'
import { parseInPreviewWorker } from '../src/services/file-preview-client.js'
import { previewMessages } from '../src/i18n/file-preview.js'
import {
  MAX_OFFICE_EXPANDED_BYTES, MAX_PREVIEW_BYTES, MAX_PREVIEW_TEXT,
  decodePreviewText, extractOfficeText, formatPreviewSize, getFileExtension, getPreviewMime, getPreviewType,
  limitPreviewText, parseZipDirectory, previewErrorKey, validateOfficeArchive, validatePdfSignature, validatePreviewFile
} from '../src/services/file-preview.js'

// Node 20 lacks deflate-raw streams; browsers use their native implementation.
try { new DecompressionStream('deflate-raw') } catch {
  const NativeStream = globalThis.DecompressionStream
  globalThis.DecompressionStream = function (format) {
    return format === 'deflate-raw' ? Duplex.toWeb(createInflateRaw()) : new NativeStream(format)
  }
}

// Small synthetic ZIPs let us mutate precisely the fields used in safety checks.
function zip(files = []) {
  const local = []
  const central = []
  let offset = 0
  for (const file of files) {
    const name = Buffer.from(file.name)
    const data = Buffer.from(file.text || '')
    const payload = file.deflate ? deflateRawSync(data) : data
    const size = file.size ?? data.length
    const header = Buffer.alloc(30)
    header.writeUInt32LE(0x04034b50)
    header.writeUInt16LE(file.encrypted ? 1 : 0x800, 6)
    header.writeUInt16LE(file.deflate ? 8 : 0, 8)
    header.writeUInt32LE(payload.length, 18)
    header.writeUInt32LE(size, 22)
    header.writeUInt16LE(name.length, 26)
    const directory = Buffer.alloc(46)
    directory.writeUInt32LE(0x02014b50)
    directory.writeUInt16LE(file.encrypted ? 1 : 0x800, 8)
    directory.writeUInt16LE(file.deflate ? 8 : 0, 10)
    directory.writeUInt32LE(payload.length, 20)
    directory.writeUInt32LE(size, 24)
    directory.writeUInt16LE(name.length, 28)
    directory.writeUInt32LE(offset, 42)
    local.push(header, name, payload)
    central.push(directory, name)
    offset += header.length + name.length + payload.length
  }
  const directory = Buffer.concat(central)
  const end = Buffer.alloc(22)
  end.writeUInt32LE(0x06054b50)
  end.writeUInt16LE(files.length, 8)
  end.writeUInt16LE(files.length, 10)
  end.writeUInt32LE(directory.length, 12)
  end.writeUInt32LE(offset, 16)
  return Buffer.concat([...local, directory, end])
}
const ooxml = (extra = []) => zip([{ name: '[Content_Types].xml', text: '<Types/>' }, { name: 'word/document.xml', text: '<document/>' }, ...extra])
const hasCode = (code) => (error) => error.code === code

test('all allowlisted extensions map to a safe local renderer, ignoring MIME and case', () => {
  for (const ext of ['jpg', 'jpeg', 'png', 'gif', 'webp']) assert.equal(getPreviewType(`private.${ext.toUpperCase()}`), 'image')
  for (const ext of ['doc', 'docx', 'xls', 'xlsx']) assert.equal(getPreviewType(`private.${ext}`), 'office')
  for (const ext of ['txt', 'csv']) assert.equal(getPreviewType(`private.${ext}`), 'text')
  assert.equal(getPreviewType({ filename: 'private.pdf' }), 'pdf')
  assert.equal(getPreviewType({ name: 'private.zip', type: 'text/html' }), 'zip')
  for (const name of ['private.svg', 'private.html', 'private.docx.exe', 'no-extension']) assert.equal(getPreviewType(name), 'unsupported')
  assert.equal(getFileExtension('private.DOCX'), 'docx')
  assert.equal(getPreviewMime('image.png'), 'image/png')
  assert.equal(getPreviewMime('private.pdf'), 'application/pdf')
})

test('5 MiB boundary applies to every preview and untrusted byte parser', () => {
  validatePreviewFile({ name: 'ok.txt', size: MAX_PREVIEW_BYTES })
  assert.throws(() => validatePreviewFile({ name: 'big.txt', size: MAX_PREVIEW_BYTES + 1 }), hasCode('too_large'))
  assert.throws(() => validatePreviewFile({ name: 'bad.txt', size: NaN }), hasCode('invalid'))
  assert.throws(() => validatePreviewFile({ name: 'bad.svg', size: 10 }), hasCode('unsupported'))
  assert.throws(() => decodePreviewText(new Uint8Array(MAX_PREVIEW_BYTES + 1)), hasCode('too_large'))
  assert.equal(formatPreviewSize(MAX_PREVIEW_BYTES), '5 MiB')
})

test('text and CSV remain literal; UTF-8/UTF-16/legacy text and limits are handled', () => {
  const unsafe = '<img src=x onerror=alert(1)>\n=HYPERLINK("https://evil.invalid")\n日本語'
  assert.equal(decodePreviewText(Buffer.from(unsafe)).text, unsafe)
  assert.equal(decodePreviewText(Buffer.from('valid UTF-8 \ufffd')).text, 'valid UTF-8 \ufffd')
  assert.equal(decodePreviewText(Buffer.from([0xff, 0xfe, 0x48, 0, 0x69, 0])).text, 'Hi')
  assert.equal(decodePreviewText(Buffer.from([0xfe, 0xff, 0, 0x48, 0, 0x69])).text, 'Hi')
  assert.equal(decodePreviewText(Buffer.from([0x63, 0x61, 0x66, 0xe9])).text, 'café')
  assert.deepEqual(limitPreviewText('x'.repeat(MAX_PREVIEW_TEXT + 1)), { text: 'x'.repeat(MAX_PREVIEW_TEXT), truncated: true })
  assert.deepEqual(decodePreviewText(Buffer.from('a'.repeat(1000)), 600), { text: 'a'.repeat(600), truncated: true })
})

test('ZIP contents are listed literally without extracting, even huge/encrypted entries', () => {
  const bytes = zip([{ name: '../<script>secret</script>.txt', text: 'secret', size: 0xfffffffe, encrypted: true }, { name: 'folder/' }, { name: 'folder/日本語.txt', text: 'private' }])
  const result = parseZipDirectory(bytes)
  assert.equal(result.entries.length, 3)
  assert.equal(result.entries[0].name, '../<script>secret</script>.txt')
  assert.equal(result.entries[0].encrypted, true)
  assert.equal(result.entries[0].size, 0xfffffffe)
  assert.equal(result.entries[1].directory, true)
  assert.equal(result.entries[2].name, 'folder/日本語.txt')
  assert.deepEqual(parseZipDirectory(zip()).entries, [])
  assert.equal(parseZipDirectory(zip([{ name: 'x\u202etxt.exe' }])).entries[0].name, 'x\ufffdtxt.exe')
})

test('malformed ZIP, mismatched local metadata, ZIP64, multipart, and too many entries fail safely', () => {
  for (const bytes of [Buffer.from('bad'), zip([{ name: 'a', text: 'abc' }]).subarray(0, 35)]) {
    assert.throws(() => parseZipDirectory(bytes), hasCode('archive_invalid'))
  }
  const badSize = zip([{ name: 'a', text: 'abc' }])
  badSize.writeUInt32LE(999, 22)
  assert.throws(() => parseZipDirectory(badSize), hasCode('archive_invalid'))
  const badOffset = zip([{ name: 'a' }])
  badOffset.writeUInt32LE(1234, badOffset.length - 6)
  assert.throws(() => parseZipDirectory(badOffset), hasCode('archive_invalid'))
  const zip64 = zip()
  zip64.writeUInt32LE(0xffffffff, zip64.length - 10)
  assert.throws(() => parseZipDirectory(zip64), hasCode('archive_unsupported'))
  const multipart = zip()
  multipart.writeUInt16LE(1, multipart.length - 18)
  assert.throws(() => parseZipDirectory(multipart), hasCode('archive_unsupported'))
  assert.throws(() => parseZipDirectory(zip(Array.from({ length: 1001 }, (_, i) => ({ name: `entry${i}` })))), hasCode('archive_limit'))
})

test('Office ZIP preflight enforces declared and actual expansion, encryption, and document structure', async () => {
  await validateOfficeArchive(ooxml(), 'docx')
  await validateOfficeArchive(ooxml([{ name: 'deflated.txt', text: 'hello'.repeat(100), deflate: true }]), 'docx')
  await assert.rejects(validateOfficeArchive(ooxml([{ name: 'bomb', size: MAX_OFFICE_EXPANDED_BYTES + 1 }]), 'docx'), hasCode('archive_limit'))
  // Declares only one byte but actually expands to 1 MiB: reject before WASM.
  await assert.rejects(validateOfficeArchive(ooxml([{ name: 'lying-bomb', text: 'x'.repeat(1024 * 1024), size: 1, deflate: true }]), 'docx'), hasCode('archive_limit'))
  await assert.rejects(validateOfficeArchive(ooxml([{ name: 'password', encrypted: true }]), 'docx'), hasCode('encrypted'))
  await assert.rejects(validateOfficeArchive(zip([{ name: 'not-office' }]), 'docx'), hasCode('office_invalid'))
  await assert.rejects(validateOfficeArchive(ooxml([{ name: 'word/document.xml' }]), 'docx'), hasCode('office_invalid'))
})

test('Office extraction only uses plainText, caps output and frees the WASM document', async () => {
  let freed = false
  class Document {
    plainText() { return 'x'.repeat(MAX_PREVIEW_TEXT + 1) }
    free() { freed = true }
    toHtml() { assert.fail('HTML must never be requested') }
  }
  const result = await extractOfficeText(ooxml(), 'docx', Document)
  assert.equal(result.text.length, MAX_PREVIEW_TEXT)
  assert.equal(result.truncated, true)
  assert.equal(freed, true)
  freed = false
  class Broken extends Document { plainText() { throw new Error('private parser detail') } }
  await assert.rejects(extractOfficeText(ooxml(), 'docx', Broken), hasCode('office_invalid'))
  assert.equal(freed, true)
  await assert.rejects(extractOfficeText(Buffer.from('not-OLE'), 'doc', Document), hasCode('office_invalid'))
  await assert.rejects(extractOfficeText(Buffer.from([0xd0, 0xcf, 0x11, 0xe0, 0xa1, 0xb1, 0x1a, 0xe1]), 'docx', Document), hasCode('encrypted'))
})

test('PDF signature and error messages do not trust arbitrary content', () => {
  validatePdfSignature(Buffer.from('%PDF-1.7\n'))
  assert.throws(() => validatePdfSignature(Buffer.from('<script>alert(1)</script>')), hasCode('invalid'))
  assert.equal(previewErrorKey({ code: '<script>' }), 'preview_error_invalid')
  assert.deepEqual(Object.keys(previewMessages.id).sort(), Object.keys(previewMessages.en).sort())
  for (const [key, value] of Object.entries(previewMessages.en)) {
    assert.match(key, /^preview_/)
    assert.ok(value.length)
  }
})

function fakeWorker() {
  return { terminated: false, posted: null, postMessage(data, transfer) { this.posted = { data, transfer } }, terminate() { this.terminated = true } }
}
const tick = () => new Promise((resolve) => setImmediate(resolve))

test('worker client transfers bytes and terminates on success/error/timeout/abort', async () => {
  const blob = new Blob(['private bytes'])
  const worker = fakeWorker()
  const success = parseInPreviewWorker(blob, 'private.doc', { workerFactory: () => worker })
  await tick()
  assert.equal(worker.posted.data.name, 'private.doc')
  assert.equal(new TextDecoder().decode(worker.posted.data.buffer), 'private bytes')
  assert.equal(worker.posted.transfer[0], worker.posted.data.buffer)
  worker.onmessage({ data: { result: { text: 'local only' } } })
  assert.deepEqual(await success, { text: 'local only' })
  assert.equal(worker.terminated, true)

  const bad = fakeWorker()
  const failure = parseInPreviewWorker(blob, 'x.zip', { workerFactory: () => bad })
  bad.onmessage({ data: { error: 'archive_invalid' } })
  await assert.rejects(failure, hasCode('archive_invalid'))
  assert.equal(bad.terminated, true)

  const slow = fakeWorker()
  await assert.rejects(parseInPreviewWorker(blob, 'x.zip', { workerFactory: () => slow, timeoutMs: 5 }), hasCode('timeout'))
  assert.equal(slow.terminated, true)

  const controller = new AbortController()
  const cancelled = fakeWorker()
  const pending = parseInPreviewWorker(blob, 'x.zip', { workerFactory: () => cancelled, signal: controller.signal })
  controller.abort()
  await assert.rejects(pending, { name: 'AbortError' })
  await tick()
  assert.equal(cancelled.terminated, true)
  assert.equal(cancelled.posted, null)

  let created = false
  await assert.rejects(parseInPreviewWorker(blob, 'x.zip', { workerFactory: () => { created = true }, signal: controller.signal }), { name: 'AbortError' })
  assert.equal(created, false)
})

test('worker startup and malformed worker responses fail without leaking a worker', async () => {
  await assert.rejects(parseInPreviewWorker(new Blob(), 'x.zip', { workerFactory: () => { throw new Error('blocked') } }), hasCode('worker'))
  const worker = fakeWorker()
  const pending = parseInPreviewWorker(new Blob(), 'x.zip', { workerFactory: () => worker })
  worker.onmessage({ data: {} })
  await assert.rejects(pending, hasCode('worker'))
  assert.equal(worker.terminated, true)
})
