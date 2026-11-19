import { extractVideoId, lookupVideo } from '~/utils/youtube'

export interface VinylRecord {
  id: string
  videoId: string
  title: string
  author: string
  thumbnail: string
  position: number
}

const SHELF_KEY = 'vd-shelf'

/**
 * Two storage modes:
 * - API mode (NUXT_PUBLIC_API_BASE set): the Go backend keeps one shared shelf.
 * - Local mode (no API, e.g. GitHub Pages): each visitor's shelf lives in their browser,
 *   starting from the default shelf in public/records.json.
 */
export function useRecords() {
  const config = useRuntimeConfig()
  const apiBase = config.public.apiBase as string
  const useApi = !!apiBase
  const records = useState<VinylRecord[]>('records', () => [])
  const loading = useState('records-loading', () => false)

  function saveLocal() {
    try { localStorage.setItem(SHELF_KEY, JSON.stringify(records.value)) } catch {}
  }

  async function load() {
    loading.value = true
    try {
      if (useApi) {
        records.value = await $fetch<VinylRecord[]>('/api/records', { baseURL: apiBase })
        return
      }
      let saved: VinylRecord[] | null = null
      try { saved = JSON.parse(localStorage.getItem(SHELF_KEY) || 'null') } catch {}
      records.value = saved ?? await $fetch<VinylRecord[]>(`${config.app.baseURL}records.json`).catch(() => [])
    } finally {
      loading.value = false
    }
  }

  async function add(url: string) {
    if (useApi) {
      try {
        const rec = await $fetch<VinylRecord>('/api/records', { baseURL: apiBase, method: 'POST', body: { url } })
        records.value = [...records.value, rec]
        return rec
      } catch (e: any) {
        throw new Error(e?.data?.error || 'Could not add this record')
      }
    }
    const videoId = extractVideoId(url)
    if (!videoId) throw new Error('not a valid YouTube link')
    if (records.value.some(r => r.videoId === videoId)) throw new Error('record already on the shelf')
    const meta = await lookupVideo(videoId)
    const rec: VinylRecord = {
      id: crypto.randomUUID().slice(0, 16),
      videoId,
      ...meta,
      position: records.value.reduce((m, r) => Math.max(m, r.position), -1) + 1,
    }
    records.value = [...records.value, rec]
    saveLocal()
    return rec
  }

  async function remove(id: string) {
    if (useApi) await $fetch(`/api/records/${id}`, { baseURL: apiBase, method: 'DELETE' })
    records.value = records.value.filter(r => r.id !== id)
    if (!useApi) saveLocal()
  }

  return { records, loading, load, add, remove }
}
