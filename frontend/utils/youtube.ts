// Browser-side twin of backend/internal/youtube: parse links and read metadata via oEmbed
// (YouTube's oEmbed endpoint sends CORS headers, so no server is needed).
const ID_RE = /^[A-Za-z0-9_-]{11}$/

export function extractVideoId(raw: string): string | null {
  raw = raw.trim()
  if (ID_RE.test(raw)) return raw
  let u: URL
  try { u = new URL(raw.includes('://') ? raw : `https://${raw}`) } catch { return null }
  const host = u.hostname.toLowerCase().replace(/^(www\.|m\.)/, '')
  let id = ''
  if (host === 'youtu.be') id = u.pathname.split('/')[1] ?? ''
  else if (['youtube.com', 'music.youtube.com', 'youtube-nocookie.com'].includes(host)) {
    id = u.searchParams.get('v') ?? ''
    if (!id) {
      const [kind, value] = u.pathname.split('/').filter(Boolean)
      if (['embed', 'shorts', 'live', 'v'].includes(kind ?? '')) id = value ?? ''
    }
  }
  return ID_RE.test(id) ? id : null
}

export const thumbnailUrl = (id: string) => `https://i.ytimg.com/vi/${id}/hqdefault.jpg`

export async function lookupVideo(id: string) {
  const q = new URLSearchParams({ url: `https://www.youtube.com/watch?v=${id}`, format: 'json' })
  const res = await fetch(`https://www.youtube.com/oembed?${q}`)
  if (!res.ok) throw new Error('This video is private, removed or cannot be embedded.')
  const body = await res.json() as { title?: string; author_name?: string }
  if (!body.title?.trim()) throw new Error('This video is unavailable.')
  return { title: body.title, author: body.author_name ?? '', thumbnail: thumbnailUrl(id) }
}
