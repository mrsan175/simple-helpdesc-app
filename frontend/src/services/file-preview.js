export const MAX_PREVIEW_BYTES = 5 * 1024 * 1024
export const MAX_OFFICE_EXPANDED_BYTES = 20 * 1024 * 1024
export const MAX_ZIP_ENTRIES = 1000
export const MAX_PREVIEW_TEXT = 200_000
export const PREVIEW_TIMEOUT_MS = 15_000

const imageTypes = { jpg: 'image/jpeg', jpeg: 'image/jpeg', png: 'image/png', gif: 'image/gif', webp: 'image/webp' }
const officeExtensions = new Set(['doc', 'docx', 'xls', 'xlsx'])

export class PreviewError extends Error {
  constructor(code) {
    super(code)
    this.name = 'PreviewError'
    this.code = code
  }
}

export function getFileExtension(name = '') {
  const match = String(name).match(/\.([a-z0-9]+)$/i)
  return match ? match[1].toLowerCase() : ''
}

export function getPreviewType(name) {
  const ext = getFileExtension(typeof name === 'string' ? name : name?.name || name?.filename)
  if (imageTypes[ext]) return 'image'
  if (officeExtensions.has(ext)) return 'office'
  if (ext === 'txt' || ext === 'csv') return 'text'
  if (ext === 'pdf' || ext === 'zip') return ext
  return 'unsupported'
}

export function getPreviewMime(name) {
  return imageTypes[getFileExtension(name)] || (getFileExtension(name) === 'pdf' ? 'application/pdf' : 'text/plain')
}

export function validatePreviewFile(file, name = file?.name) {
  if (!file || !Number.isFinite(file.size) || file.size < 0) throw new PreviewError('invalid')
  if (file.size > MAX_PREVIEW_BYTES) throw new PreviewError('too_large')
  if (getPreviewType(name) === 'unsupported') throw new PreviewError('unsupported')
}

export function formatPreviewSize(bytes = 0) {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B'
  const unit = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), 3)
  return `${Number((bytes / 1024 ** unit).toFixed(1))} ${['B', 'KiB', 'MiB', 'GiB'][unit]}`
}

export function limitPreviewText(text, limit = MAX_PREVIEW_TEXT) {
  return { text: text.slice(0, limit), truncated: text.length > limit }
}

// Plain text only: neither CSV formulas nor markup are interpreted.
export function decodePreviewText(input, limit = MAX_PREVIEW_TEXT) {
  const bytes = asBytes(input)
  let encoding = 'utf-8'
  if (bytes[0] === 0xff && bytes[1] === 0xfe) encoding = 'utf-16le'
  if (bytes[0] === 0xfe && bytes[1] === 0xff) encoding = 'utf-16be'
  let text
  try {
    text = new TextDecoder(encoding, { fatal: encoding === 'utf-8' }).decode(bytes)
  } catch {
    text = new TextDecoder('windows-1252').decode(bytes)
  }
  return limitPreviewText(text.replace(/\u0000/g, '\ufffd'), limit)
}

function asBytes(input) {
  const bytes = input instanceof Uint8Array ? input : new Uint8Array(input)
  if (bytes.byteLength > MAX_PREVIEW_BYTES) throw new PreviewError('too_large')
  return bytes
}

function cleanName(name) {
  return name.replace(/[\u0000-\u001f\u007f-\u009f\u202a-\u202e\u2066-\u2069]/g, '\ufffd')
}

// Read bounded ZIP metadata only. Never follow archive paths or extract files.
// ZIP64/multipart archives are intentionally rejected rather than partially listed.
export function parseZipDirectory(input) {
  const bytes = asBytes(input)
  const view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength)
  const invalid = () => { throw new PreviewError('archive_invalid') }
  if (bytes.length < 22) invalid()
  let end = -1
  for (let i = bytes.length - 22; i >= Math.max(0, bytes.length - 65557); i--) {
    if (view.getUint32(i, true) === 0x06054b50 && i + 22 + view.getUint16(i + 20, true) === bytes.length) {
      end = i
      break
    }
  }
  if (end < 0) invalid()
  const count = view.getUint16(end + 10, true)
  const directorySize = view.getUint32(end + 12, true)
  const directoryStart = view.getUint32(end + 16, true)
  if (view.getUint16(end + 4, true) || view.getUint16(end + 6, true) || view.getUint16(end + 8, true) !== count ||
      count === 0xffff || directorySize === 0xffffffff || directoryStart === 0xffffffff) {
    throw new PreviewError('archive_unsupported')
  }
  if (count > MAX_ZIP_ENTRIES) throw new PreviewError('archive_limit')
  if (directoryStart + directorySize !== end || directoryStart > end) invalid()
  const entries = []
  let offset = directoryStart
  let expandedBytes = 0
  for (let i = 0; i < count; i++) {
    if (offset + 46 > end || view.getUint32(offset, true) !== 0x02014b50) invalid()
    const flags = view.getUint16(offset + 8, true)
    const method = view.getUint16(offset + 10, true)
    const compressedSize = view.getUint32(offset + 20, true)
    const size = view.getUint32(offset + 24, true)
    const nameLength = view.getUint16(offset + 28, true)
    const extraLength = view.getUint16(offset + 30, true)
    const commentLength = view.getUint16(offset + 32, true)
    const localOffset = view.getUint32(offset + 42, true)
    const next = offset + 46 + nameLength + extraLength + commentLength
    if (size === 0xffffffff || compressedSize === 0xffffffff || localOffset === 0xffffffff || view.getUint16(offset + 34, true)) {
      throw new PreviewError('archive_unsupported')
    }
    if (!nameLength || nameLength > 4096 || next > end || localOffset + 30 > directoryStart) invalid()
    if (view.getUint32(localOffset, true) !== 0x04034b50 || view.getUint16(localOffset + 6, true) !== flags ||
        view.getUint16(localOffset + 8, true) !== method) invalid()
    const localNameLength = view.getUint16(localOffset + 26, true)
    const localExtraLength = view.getUint16(localOffset + 28, true)
    const dataOffset = localOffset + 30 + localNameLength + localExtraLength
    if (localNameLength !== nameLength || dataOffset + compressedSize > directoryStart) invalid()
    for (let n = 0; n < nameLength; n++) {
      if (bytes[localOffset + 30 + n] !== bytes[offset + 46 + n]) invalid()
    }
    if (!(flags & 8) && (view.getUint32(localOffset + 18, true) !== compressedSize || view.getUint32(localOffset + 22, true) !== size)) invalid()
    const rawName = new TextDecoder(flags & 0x800 ? 'utf-8' : 'windows-1252').decode(bytes.subarray(offset + 46, offset + 46 + nameLength))
    entries.push({ name: cleanName(rawName), size, compressedSize, encrypted: Boolean(flags & 0x41), directory: rawName.endsWith('/'), method, dataOffset })
    expandedBytes += size
    offset = next
  }
  if (offset !== end) invalid()
  return { entries, expandedBytes }
}

// Verify actual streamed expansion as well as declared sizes before giving OOXML
// to WASM. A dishonest DEFLATE size must not bypass the decompression budget.
export async function validateOfficeArchive(input, extension) {
  const bytes = asBytes(input)
  const archive = parseZipDirectory(bytes)
  if (archive.entries.some((entry) => entry.encrypted)) throw new PreviewError('encrypted')
  if (archive.expandedBytes > MAX_OFFICE_EXPANDED_BYTES) throw new PreviewError('archive_limit')
  const names = new Set(archive.entries.map((entry) => entry.name))
  if (names.size !== archive.entries.length || !names.has('[Content_Types].xml') ||
      !names.has(extension === 'docx' ? 'word/document.xml' : 'xl/workbook.xml')) throw new PreviewError('office_invalid')
  let total = 0
  for (const entry of archive.entries) {
    const data = bytes.subarray(entry.dataOffset, entry.dataOffset + entry.compressedSize)
    if (entry.method === 0) {
      if (entry.size !== data.length) throw new PreviewError('archive_invalid')
      total += data.length
      continue
    }
    if (entry.method !== 8) throw new PreviewError('archive_unsupported')
    let reader
    try {
      reader = new Blob([data]).stream().pipeThrough(new DecompressionStream('deflate-raw')).getReader()
      let size = 0
      while (true) {
        const { value, done } = await reader.read()
        if (done) break
        size += value.length
        total += value.length
        if (total > MAX_OFFICE_EXPANDED_BYTES || size > entry.size) throw new PreviewError('archive_limit')
      }
      if (size !== entry.size) throw new PreviewError('archive_invalid')
    } catch (error) {
      if (error instanceof PreviewError) throw error
      throw new PreviewError('archive_invalid')
    } finally {
      if (reader) await reader.cancel().catch(() => {})
    }
  }
  return archive
}

export async function extractOfficeText(input, extension, WasmDocument) {
  const bytes = asBytes(input)
  if (!officeExtensions.has(extension)) throw new PreviewError('unsupported')
  if (extension === 'docx' || extension === 'xlsx') {
    // Password-protected OOXML is an OLE container, not a ZIP.
    if (isOle(bytes)) throw new PreviewError('encrypted')
    await validateOfficeArchive(bytes, extension)
  } else if (!isOle(bytes)) {
    throw new PreviewError('office_invalid')
  }
  let document
  try {
    document = new WasmDocument(bytes, extension)
    return limitPreviewText(document.plainText())
  } catch {
    throw new PreviewError('office_invalid')
  } finally {
    document?.free()
  }
}

function isOle(bytes) {
  return [0xd0, 0xcf, 0x11, 0xe0, 0xa1, 0xb1, 0x1a, 0xe1].every((value, i) => bytes[i] === value)
}

export function validatePdfSignature(input) {
  const bytes = asBytes(input)
  if (!new TextDecoder('ascii').decode(bytes.subarray(0, 1024)).includes('%PDF-')) throw new PreviewError('invalid')
}

export function previewErrorKey(error) {
  const codes = new Set(['too_large', 'unsupported', 'invalid', 'archive_invalid', 'archive_unsupported', 'archive_limit', 'encrypted', 'office_invalid', 'timeout', 'worker', 'image', 'remote'])
  return `preview_error_${codes.has(error?.code) ? error.code : 'invalid'}`
}
