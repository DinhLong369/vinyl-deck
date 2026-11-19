<script setup lang="ts">
import type { VinylRecord } from '~/composables/useRecords'

const { records, loading, load, add, remove } = useRecords()
const yt = useYouTubePlayer('yt-player', { onEnded: () => onTrackEnded() })

const activeId = ref<string | null>(null)
const volume = ref(70)
const deckStyle = ref<1 | 2>(1)
const armDown = ref(false)
const showAdd = ref(false)
const showInfo = ref(false)
const addUrl = ref('')
const addError = ref('')
const adding = ref(false)
const loadError = ref('')

const active = computed(() => records.value.find(r => r.id === activeId.value) || null)
const plateTitle = computed(() => active.value?.title || 'Pick a record from the shelf')
const plateSub = computed(() => yt.error.value || (yt.buffering.value ? 'Dropping the needle…' : ''))

// small, per-viewer preferences
function pref<T>(key: string, r: Ref<T>) {
  try { const v = localStorage.getItem(key); if (v !== null) r.value = JSON.parse(v) } catch {}
  watch(r, v => { try { localStorage.setItem(key, JSON.stringify(v)) } catch {} })
}

onMounted(async () => {
  pref('vd-volume', volume); pref('vd-style', deckStyle); pref('vd-active', activeId)
  try { await load() } catch { loadError.value = 'Cannot reach the API server.' }
  if (!active.value) activeId.value = records.value[0]?.id ?? null
  await yt.init(volume.value)
  if (active.value) yt.load(active.value.videoId, false)
})

// dev-only handle for debugging in the browser console
if (import.meta.dev && import.meta.client) (window as any).__deck = { yt, records, activeId }

watch(volume, v => yt.setVolume(v))
let liftTimer: ReturnType<typeof setTimeout> | undefined
watch(yt.playing, (p) => {
  clearTimeout(liftTimer)
  if (p) { armDown.value = true; return }
  // only lift when playback has really stopped, not for a blip between player states
  liftTimer = setTimeout(() => { if (!yt.playing.value && !swapTimer) armDown.value = false }, 350)
})
watch(yt.error, e => { if (e) armDown.value = false })

// if the browser refuses to start playback, don't leave the needle sitting on a silent record
let startGuard: ReturnType<typeof setTimeout> | undefined
function startPlayback() {
  armDown.value = true
  yt.play()
  clearTimeout(startGuard)
  startGuard = setTimeout(() => { if (!yt.playing.value) armDown.value = false }, 4000)
}

function toggle() {
  if (!active.value) { showAdd.value = true; return }
  if (yt.playing.value) { clearTimeout(liftTimer); armDown.value = false; yt.pause() }
  else startPlayback()
}

// live streams have no timeline: the needle stays where it was dropped
const liveSpot = ref(0)
const progress = computed(() => {
  if (yt.isLive.value || yt.duration.value <= 0) return liveSpot.value
  return yt.currentTime.value / yt.duration.value
})
// remember the last real position so a restart lands on it instead of jumping to the rim first
watch(progress, (p) => { if (!yt.isLive.value && yt.duration.value > 0) liveSpot.value = p })

// tonearm dragged by hand: picking it up silences the record immediately,
// putting it back on the record resumes exactly where it stopped (no jump).
let swapTimer: ReturnType<typeof setTimeout> | undefined
function onGrab() {
  if (swapTimer) {
    // a record swap was in progress: put the new record on, but let the user drop the needle
    clearTimeout(swapTimer); swapTimer = undefined
    if (active.value) yt.load(active.value.videoId, false)
  }
  yt.mute() // instant silence; pause() alone lets a little audio through
  yt.pause()
}
function onDrop(f: number) {
  yt.unMute()
  if (!active.value) { armDown.value = false; showAdd.value = true; return }
  liveSpot.value = f
  startPlayback()
}
function onLift() {
  yt.unMute()
  armDown.value = false
  yt.pause()
}

// Put a record on the platter. With autoplay the arm lifts, the record is swapped, then the needle drops again.
function switchTo(r: VinylRecord, autoplay: boolean) {
  clearTimeout(swapTimer)
  activeId.value = r.id
  liveSpot.value = 0 // new record: needle starts at the first groove
  if (autoplay) {
    armDown.value = false
    yt.pause()
    // wait until the arm has fully swung back to its rest (1.4s) before dropping it again
    swapTimer = setTimeout(() => { swapTimer = undefined; yt.load(r.videoId, true); armDown.value = true }, 1700)
  } else {
    swapTimer = undefined
    yt.load(r.videoId, false)
  }
}

function select(r: VinylRecord) {
  if (r.id === activeId.value) return toggle()
  switchTo(r, yt.playing.value || !!swapTimer)
}

function restart() {
  yt.seekTo(0)
  startPlayback()
}

// next / previous record on the shelf (wraps around)
function step(dir: 1 | -1, autoplay = yt.playing.value || !!swapTimer) {
  const list = records.value
  if (!list.length) return
  // "previous" a few seconds into a track restarts it, like most players
  if (dir === -1 && autoplay && yt.currentTime.value > 3 && !yt.isLive.value) return restart()
  const i = list.findIndex(r => r.id === activeId.value)
  const next = list[(i + dir + list.length) % list.length]
  if (next.id === activeId.value) return autoplay ? restart() : undefined
  switchTo(next, autoplay)
}

function onTrackEnded() {
  step(1, true)
}

async function onRemove(r: VinylRecord) {
  if (r.id === activeId.value) { yt.pause(); armDown.value = false; activeId.value = null }
  await remove(r.id)
}

async function submitAdd() {
  addError.value = ''
  if (!addUrl.value.trim()) return
  adding.value = true
  try {
    const rec = await add(addUrl.value.trim())
    addUrl.value = ''; showAdd.value = false
    select(rec)
  } catch (e: any) {
    addError.value = e.message
  } finally { adding.value = false }
}

function fmt(s: number) {
  s = Math.floor(s); const h = Math.floor(s / 3600), m = Math.floor(s % 3600 / 60), sec = s % 60
  return (h ? `${h}:${String(m).padStart(2, '0')}` : `${m}`) + `:${String(sec).padStart(2, '0')}`
}
</script>

<template>
  <div class="app">
    <Shelf :records="records" :active-id="activeId" :loading="loading" @select="select" @remove="onRemove" @add="showAdd = true" />

    <main class="stage">
      <button class="info" aria-label="About" @click="showInfo = true">i</button>
      <p v-if="loadError" class="banner">{{ loadError }}</p>
      <Turntable
        v-model:volume="volume" v-model:deck-style="deckStyle"
        :playing="yt.playing.value && !yt.buffering.value" :arm-down="armDown" :progress="progress"
        :label="active?.thumbnail" @toggle="toggle" @next="step(1)" @prev="step(-1)" @grab="onGrab" @drop="onDrop" @lift="onLift"
      />
      <Nameplate :title="plateTitle" :sub="plateSub" />
    </main>

    <div id="yt-player" class="yt-hidden" />

    <Modal :open="showAdd" title="Add a record" @close="showAdd = false">
      <form class="add-form" @submit.prevent="submitAdd">
        <label for="yt-url">YouTube link</label>
        <input id="yt-url" v-model="addUrl" type="url" placeholder="https://www.youtube.com/watch?v=…" autofocus required>
        <p v-if="addError" class="err">{{ addError }}</p>
        <div class="actions">
          <button type="button" class="ghost" @click="showAdd = false">Cancel</button>
          <button type="submit" class="primary" :disabled="adding">{{ adding ? 'Adding…' : 'Add to shelf' }}</button>
        </div>
      </form>
    </Modal>

    <Modal :open="showInfo" title="Now playing" @close="showInfo = false">
      <div class="about">
        <template v-if="active">
          <img :src="active.thumbnail" alt="">
          <p class="np-title">{{ active.title }}</p>
          <p class="np-meta">{{ active.author }}<span v-if="yt.duration.value"> · {{ fmt(yt.currentTime.value) }} / {{ fmt(yt.duration.value) }}</span></p>
          <a :href="`https://www.youtube.com/watch?v=${active.videoId}`" target="_blank" rel="noopener">Open on YouTube ↗</a>
        </template>
        <p v-else class="np-meta">Nothing on the platter yet.</p>
        <hr>
        <h3>About</h3>
        <p class="np-meta">A calm turntable for long focus mixes. Add any embeddable YouTube mix to the shelf, drop the needle and get to work. When a record ends the next one on the shelf starts by itself.</p>
      </div>
    </Modal>
  </div>
</template>

<style scoped>
.app { display: flex; height: 100vh; }
.stage { position: relative; flex: 1; background: radial-gradient(ellipse 70% 60% at 55% 42%, rgba(255,255,255,.06), rgba(0,0,0,0) 70%), radial-gradient(ellipse at 50% 50%, rgba(0,0,0,0) 55%, rgba(0,0,0,.45) 100%); min-width: 0; overflow-y: auto; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 28px; padding: 64px 40px 32px; }
.info { position: absolute; right: 20px; top: 20px; width: 36px; height: 36px; border-radius: 50%; border: 1px solid #5d5d5d; background: radial-gradient(circle at 40% 30%, #3a3a3a, #161616); color: #ddd; box-shadow: 0 2px 6px rgba(0,0,0,.5), inset 0 1px 0 rgba(255,255,255,.1); font-family: Georgia, serif; font-size: 16px; }
.banner { margin: 0; padding: 8px 14px; border-radius: 6px; background: #5a2a1c; color: #ffd9c9; font-size: 13px; }
.add-form { display: flex; flex-direction: column; gap: 10px; }
.add-form label { font-size: 13px; color: var(--muted); }
.add-form input { padding: 11px 12px; border-radius: 6px; border: 1px solid #5c4a3d; background: #1b1714; color: var(--ink); font-size: 14px; }
.add-form input:focus { outline: 2px solid var(--accent); outline-offset: 1px; }
.err { margin: 0; color: #ff9f7f; font-size: 13px; }
.actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 6px; }
.ghost, .primary { padding: 9px 16px; border-radius: 6px; font-size: 14px; }
.ghost { background: none; border: 1px solid #5c4a3d; color: var(--ink); }
.primary { background: var(--accent); border: 0; color: #1d120b; font-weight: 600; }
.primary:disabled { opacity: .6; }
.about img { width: 100%; aspect-ratio: 16/9; object-fit: cover; border-radius: 6px; }
.np-title { margin: 12px 0 4px; font-weight: 600; line-height: 1.4; }
.np-meta { margin: 0 0 10px; color: var(--muted); font-size: 13px; line-height: 1.55; }
.about a { color: #f2b48d; font-size: 13px; }
.about hr { border: 0; border-top: 1px solid #4b3d33; margin: 16px 0; }
.about h3 { margin: 0 0 6px; font-size: 14px; }
@media (max-width: 760px) {
  .app { flex-direction: column; }
  .stage { padding: 56px 16px 24px; justify-content: flex-start; }
}
</style>
