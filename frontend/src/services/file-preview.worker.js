import init, { WasmDocument } from 'office-oxide-wasm/web'
import { extractOfficeText, getFileExtension, getPreviewType, parseZipDirectory, PreviewError } from './file-preview.js'

self.onmessage = async ({ data }) => {
  try {
    const bytes = new Uint8Array(data.buffer)
    const type = getPreviewType(data.name)
    if (type === 'zip') {
      const archive = parseZipDirectory(bytes)
      // Only metadata leaves the worker; no archive contents are decompressed.
      self.postMessage({ result: { entries: archive.entries.map(({ name, size, compressedSize, encrypted, directory }) => ({ name, size, compressedSize, encrypted, directory })) } })
    } else if (type === 'office') {
      // The package does not export its WASM file. Resolve it from this source
      // module: Vite's dev optimizer relocates the package JS to .vite/deps,
      // which breaks the package's implicit relative URL. Build emits a hash.
      const wasmUrl = new URL('../../node_modules/office-oxide-wasm/web/office_oxide_bg.wasm', import.meta.url)
      try {
        await init({ module_or_path: wasmUrl })
      } catch {
        throw new PreviewError('worker')
      }
      self.postMessage({ result: await extractOfficeText(bytes, getFileExtension(data.name), WasmDocument) })
    } else {
      throw new PreviewError('unsupported')
    }
  } catch (error) {
    self.postMessage({ error: error instanceof PreviewError ? error.code : 'office_invalid' })
  }
}
