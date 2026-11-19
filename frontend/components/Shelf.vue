<script setup lang="ts">
import type { VinylRecord } from '~/composables/useRecords'

const props = defineProps<{ records: VinylRecord[]; activeId?: string | null; loading?: boolean }>()
const emit = defineEmits<{ select: [VinylRecord]; remove: [VinylRecord]; add: []; reorder: [string[]] }>()

// Local copy so the list can be rearranged live while dragging; resynced from props otherwise.
const items = ref<VinylRecord[]>([])
const dragId = ref<string | null>(null)
watch(() => props.records, (r) => { if (!dragId.value) items.value = [...r] }, { immediate: true })

const listEl = ref<HTMLElement | null>(null) // scroll container
const itemsGroup = ref<{ $el: HTMLElement } | null>(null)
const itemsEl = computed(() => itemsGroup.value?.$el ?? null) // positioned parent of the items
const drag = reactive({ dx: 0, dy: 0 })

const DRAG_THRESHOLD = 6 // px of mouse movement before a press becomes a drag
const LONG_PRESS = 300 // ms hold before a touch becomes a drag (so normal swipes still scroll)

let pending: { id: string; x: number; y: number; grabX: number; grabY: number; touch: boolean } | null = null
let pressTimer: ReturnType<typeof setTimeout> | undefined
let suppressClick = false
let lastPointer = { x: 0, y: 0 }

const horizontal = () => window.matchMedia('(max-width: 760px)').matches
const elFor = (id: string) => itemsEl.value?.querySelector<HTMLElement>(`[data-id="${id}"]`) ?? null

function onPointerDown(e: PointerEvent, r: VinylRecord) {
  if (e.button !== 0 || (e.target as HTMLElement).closest('.del')) return
  const rect = (e.currentTarget as HTMLElement).getBoundingClientRect()
  pending = { id: r.id, x: e.clientX, y: e.clientY, grabX: e.clientX - rect.left, grabY: e.clientY - rect.top, touch: e.pointerType !== 'mouse' }
  lastPointer = { x: e.clientX, y: e.clientY }
  window.addEventListener('pointermove', onPointerMove)
  window.addEventListener('pointerup', onPointerUp)
  window.addEventListener('pointercancel', onPointerUp)
  if (pending.touch) pressTimer = setTimeout(() => startDrag(), LONG_PRESS)
}

function startDrag() {
  if (!pending) return
  dragId.value = pending.id
  follow(lastPointer.x, lastPointer.y)
}

function onPointerMove(e: PointerEvent) {
  if (!pending) return
  lastPointer = { x: e.clientX, y: e.clientY }
  if (!dragId.value) {
    const moved = Math.hypot(e.clientX - pending.x, e.clientY - pending.y) > DRAG_THRESHOLD
    if (!moved) return
    if (pending.touch) return cleanup() // finger moved before the long-press: it's a scroll
    startDrag()
  }
  e.preventDefault()
  autoScroll(e.clientX, e.clientY)
  moveTo(e.clientX, e.clientY)
  follow(e.clientX, e.clientY)
}

// Where the dragged item should sit: count the other items whose centre is before the pointer.
// Uses layout offsets (not getBoundingClientRect) so items mid-animation don't cause jitter.
function moveTo(x: number, y: number) {
  const base = itemsEl.value!.getBoundingClientRect()
  const h = horizontal()
  const p = h ? x : y
  const others = items.value.filter(r => r.id !== dragId.value)
  let index = 0
  for (const r of others) {
    const el = elFor(r.id)
    if (!el) continue
    const mid = h ? base.left + el.offsetLeft + el.offsetWidth / 2 : base.top + el.offsetTop + el.offsetHeight / 2
    if (mid < p) index++
  }
  const from = items.value.findIndex(r => r.id === dragId.value)
  if (from === index) return
  const next = [...items.value]
  const [moved] = next.splice(from, 1)
  next.splice(index, 0, moved!)
  items.value = next
}

// keep the dragged sleeve under the pointer, locked to the list's axis
function follow(x: number, y: number) {
  const el = dragId.value && elFor(dragId.value)
  if (!el || !pending) return
  const base = itemsEl.value!.getBoundingClientRect()
  const h = horizontal()
  drag.dx = h ? x - pending.grabX - (base.left + el.offsetLeft) : 0
  drag.dy = h ? 0 : y - pending.grabY - (base.top + el.offsetTop)
}

function autoScroll(x: number, y: number) {
  const list = listEl.value
  if (!list) return
  const r = list.getBoundingClientRect()
  const EDGE = 48, STEP = 12
  if (horizontal()) {
    if (x < r.left + EDGE) list.scrollLeft -= STEP
    else if (x > r.right - EDGE) list.scrollLeft += STEP
  } else {
    if (y < r.top + EDGE) list.scrollTop -= STEP
    else if (y > r.bottom - EDGE) list.scrollTop += STEP
  }
}

function onPointerUp() {
  const wasDragging = !!dragId.value
  cleanup()
  if (!wasDragging) return
  // swallow the click that follows a drag
  suppressClick = true
  setTimeout(() => { suppressClick = false })
  const ids = items.value.map(r => r.id)
  if (ids.join() !== props.records.map(r => r.id).join()) emit('reorder', ids)
}

function cleanup() {
  clearTimeout(pressTimer)
  pending = null
  dragId.value = null
  drag.dx = drag.dy = 0
  window.removeEventListener('pointermove', onPointerMove)
  window.removeEventListener('pointerup', onPointerUp)
  window.removeEventListener('pointercancel', onPointerUp)
}

// while dragging with a finger, stop the page/shelf from scrolling
function blockTouchScroll(e: TouchEvent) { if (dragId.value) e.preventDefault() }
onMounted(() => window.addEventListener('touchmove', blockTouchScroll, { passive: false }))
onBeforeUnmount(() => { cleanup(); window.removeEventListener('touchmove', blockTouchScroll) })

function onClick(r: VinylRecord) {
  if (suppressClick) return
  emit('select', r)
}

// horizontal shelf (narrow screens): let a normal mouse wheel scroll it sideways
function onWheel(e: WheelEvent) {
  const list = listEl.value
  if (!list || !horizontal() || Math.abs(e.deltaY) <= Math.abs(e.deltaX)) return
  const max = list.scrollWidth - list.clientWidth
  if (max <= 0) return
  e.preventDefault()
  list.scrollLeft = Math.min(max, Math.max(0, list.scrollLeft + e.deltaY))
}

// keyboard: Alt + arrow keys move the focused record
function onKey(e: KeyboardEvent, r: VinylRecord) {
  if (e.key === 'Enter') return emit('select', r)
  if (!e.altKey) return
  const dir = ['ArrowUp', 'ArrowLeft'].includes(e.key) ? -1 : ['ArrowDown', 'ArrowRight'].includes(e.key) ? 1 : 0
  if (!dir) return
  e.preventDefault()
  const from = items.value.findIndex(x => x.id === r.id)
  const to = from + dir
  if (to < 0 || to >= items.value.length) return
  const next = [...items.value]
  ;[next[from], next[to]] = [next[to]!, next[from]!]
  items.value = next
  emit('reorder', next.map(x => x.id))
  nextTick(() => elFor(r.id)?.focus())
}
</script>

<template>
  <aside class="shelf">
    <div ref="listEl" class="list" :class="{ sorting: !!dragId }" @wheel="onWheel">
      <p v-if="loading && !records.length" class="empty">Loading the shelf…</p>
      <p v-else-if="!records.length" class="empty">The shelf is empty. Add a YouTube mix to get started.</p>
      <TransitionGroup ref="itemsGroup" tag="div" name="shelf" class="items" :class="{ sorting: !!dragId }">
        <div
          v-for="r in items" :key="r.id" :data-id="r.id"
          class="item" :class="{ active: r.id === activeId, dragging: r.id === dragId }"
          :style="r.id === dragId ? { '--dx': `${drag.dx}px`, '--dy': `${drag.dy}px` } : undefined"
          role="button" tabindex="0" title="Click to play · drag to reorder (Alt + arrows)"
          @pointerdown="onPointerDown($event, r)" @click="onClick(r)" @keydown="onKey($event, r)"
          @contextmenu="dragId && $event.preventDefault()"
        >
          <div class="sleeve-wrap">
            <div class="disc"><div class="vinyl" :style="{ '--cover': `url(${r.thumbnail})` }" /></div>
            <div class="sleeve" role="img" :aria-label="r.title" :style="{ backgroundImage: `url(${r.thumbnail})` }" />
            <button class="del" title="Remove from shelf" aria-label="Remove" @click.stop="emit('remove', r)">×</button>
          </div>
          <p class="name">{{ r.title }}</p>
        </div>
      </TransitionGroup>
    </div>
    <button class="add" @click="emit('add')">＋ Add a record</button>
  </aside>
</template>

<style scoped>
.shelf {
  position: relative; width: 236px; flex-shrink: 0; height: 100%; display: flex; flex-direction: column; z-index: 2;
  background:
    url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='180' height='180'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='.75' numOctaves='3' stitchTiles='stitch'/%3E%3CfeColorMatrix values='0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 .35 0'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)'/%3E%3C/svg%3E"),
    radial-gradient(ellipse at 30% 0%, rgba(255,190,130,.16), transparent 55%),
    linear-gradient(90deg, rgba(0,0,0,.28), rgba(0,0,0,0) 10%, rgba(0,0,0,0) 85%, rgba(0,0,0,.45)),
    linear-gradient(180deg, #5a3219, #3d2010 55%, #2a150a);
  box-shadow: inset -1px 0 0 rgba(255,255,255,.06), 6px 0 24px rgba(0,0,0,.55), 2px 0 3px rgba(0,0,0,.5);
}
/* stitching along the edge */
.shelf::after {
  content: ''; position: absolute; inset: 7px; border: 1px dashed rgba(230,190,150,.28); border-radius: 4px; pointer-events: none;
}
.list { flex: 1; overflow-y: auto; overflow-x: hidden; padding: 24px 16px 10px 22px; scrollbar-width: thin; scrollbar-color: #8a6448 transparent; }
.empty { color: var(--muted); font-size: 13px; line-height: 1.5; }
.items { position: relative; }
.item { position: relative; margin-bottom: 26px; cursor: pointer; outline: none; user-select: none; -webkit-user-select: none; -webkit-touch-callout: none; }
/* other records slide out of the way */
.shelf-move { transition: transform .28s cubic-bezier(.3,.7,.3,1); }
/* the record being dragged follows the pointer (!important beats TransitionGroup's inline FLIP transform) */
.item.dragging { z-index: 5; cursor: grabbing; transform: translate(var(--dx, 0), var(--dy, 0)) !important; transition: none !important; }
.item.dragging .sleeve { transform: scale(1.05) rotate(-1.5deg); box-shadow: inset 0 0 0 1px rgba(255,255,255,.15), 0 18px 36px rgba(0,0,0,.6), 0 4px 8px rgba(0,0,0,.4); }
.item.dragging .disc { transform: translateX(10px); }
.item.dragging .name { opacity: .6; }
.items.sorting .item:not(.dragging) { opacity: .85; }
.items.sorting .del { display: none; }
.sleeve-wrap { position: relative; width: 150px; height: 150px; }
.sleeve {
  position: relative; z-index: 1; width: 150px; height: 150px; display: block; border-radius: 8px;
  background-size: auto 134%; background-position: center; background-color: #222;
  box-shadow:
    inset 0 0 0 1px rgba(255,255,255,.12),
    inset -6px 0 10px -6px rgba(0,0,0,.6),
    1px 2px 2px rgba(0,0,0,.45),
    6px 10px 18px rgba(0,0,0,.5);
  transition: transform .3s ease, box-shadow .3s ease;
}
.sleeve::after { content: ''; position: absolute; inset: 0; border-radius: inherit; background: linear-gradient(120deg, rgba(255,255,255,.18), rgba(255,255,255,0) 40%); pointer-events: none; }
.disc {
  position: absolute; left: 5px; top: 5px; width: 140px; height: 140px; border-radius: 50%;
  transform: translateX(18px); transition: transform .45s cubic-bezier(.3,.7,.3,1);
  box-shadow: 4px 6px 12px rgba(0,0,0,.55);
}
.vinyl {
  position: absolute; inset: 0; border-radius: 50%;
  background:
    conic-gradient(from 300deg, rgba(255,255,255,0), rgba(255,255,255,.12) 6%, rgba(255,255,255,0) 13%, rgba(255,255,255,0) 50%, rgba(255,255,255,.1) 56%, rgba(255,255,255,0) 63%),
    radial-gradient(circle closest-side, #fff 0 3%, rgba(0,0,0,0) 3.5%),
    radial-gradient(circle closest-side, rgba(0,0,0,0) 0 33%, #0b0b0b 34% 38%, rgba(0,0,0,0) 38.5%),
    var(--cover) center / auto 48% no-repeat,
    repeating-radial-gradient(circle closest-side, #101010 0 1.5px, #1c1c1c 1.5px 3px);
  -webkit-mask: radial-gradient(circle closest-side, transparent 0 2.6%, #000 3%); mask: radial-gradient(circle closest-side, transparent 0 2.6%, #000 3%);
}
.item:hover .disc, .item:focus-visible .disc { transform: translateX(34px); }
.item:hover .sleeve { transform: translateY(-2px); }
.item.active .disc { transform: translateX(48px); }
.item.active .vinyl { animation: spin 7s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
.name { margin: 10px 0 0; font-size: 13px; line-height: 1.4; color: #c9b39c; text-shadow: 0 1px 1px rgba(0,0,0,.6); display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
.item.active .name { color: #ffdcb8; }
.item.active .sleeve { box-shadow: inset 0 0 0 1px rgba(255,255,255,.12), 0 0 0 2px var(--accent), 1px 2px 2px rgba(0,0,0,.45), 6px 10px 18px rgba(0,0,0,.5); }
.item:focus-visible .sleeve { box-shadow: 0 0 0 2px #fff; }
.del {
  position: absolute; z-index: 2; right: 6px; top: 6px; width: 24px; height: 24px; border-radius: 50%; border: 0;
  background: rgba(0,0,0,.7); color: #fff; font-size: 16px; line-height: 1; opacity: 0; transition: opacity .2s;
}
.item:hover .del, .del:focus-visible { opacity: 1; }
.add {
  position: relative; z-index: 1; margin: 10px 18px 18px; padding: 12px; border: 1px dashed #c9a383; border-radius: 4px;
  background: rgba(0,0,0,.12); color: #f0dccb; font-size: 14px; transition: background .2s;
}
.add:hover { background: rgba(255,255,255,.07); }
@media (max-width: 760px) {
  .shelf { width: 100%; height: auto; flex-direction: row; align-items: center; }
  .shelf::after { inset: 5px; }
  /* horizontal shelf: no native scrollbar (it appeared/disappeared and shifted the layout),
     soft fade at the edges hints there is more, swipe / wheel / drag to scroll */
  .list {
    flex: 1; min-width: 0; overflow-x: auto; overflow-y: hidden; padding: 10px 0 10px 12px;
    scrollbar-width: none; overscroll-behavior-x: contain; scroll-snap-type: x proximity; scroll-padding-left: 12px;
    -webkit-mask-image: linear-gradient(90deg, transparent 0, #000 12px, #000 calc(100% - 28px), transparent 100%);
            mask-image: linear-gradient(90deg, transparent 0, #000 12px, #000 calc(100% - 28px), transparent 100%);
  }
  .list::-webkit-scrollbar { display: none; }
  .list.sorting { scroll-snap-type: none; }
  .items { display: flex; gap: 14px; width: max-content; padding-right: 28px; }
  .item { scroll-snap-align: start; }
  .item { width: 96px; flex-shrink: 0; margin: 0; }
  .item.dragging .disc { transform: translateX(6px); }
  .sleeve-wrap, .sleeve { width: 72px; height: 72px; }
  .disc { left: 3px; top: 3px; width: 66px; height: 66px; transform: translateX(10px); }
  .item:hover .disc, .item:focus-visible .disc { transform: translateX(16px); }
  .item.active .disc { transform: translateX(22px); }
  .del { width: 20px; height: 20px; font-size: 14px; right: 3px; top: 3px; }
  .name { margin-top: 6px; font-size: 11px; -webkit-line-clamp: 1; }
  .add { flex-shrink: 0; margin: 10px 12px; padding: 8px 12px; font-size: 13px; white-space: nowrap; }
}
</style>
