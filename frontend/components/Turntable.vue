<script setup lang="ts">
const props = defineProps<{
  playing: boolean
  armDown: boolean
  /** playback progress 0..1, drives where the needle sits on the record */
  progress?: number
  label?: string
  volume: number
  deckStyle: 1 | 2
}>()
const emit = defineEmits<{
  toggle: []
  next: []
  prev: []
  /** user picked the arm up */
  grab: []
  /** needle dropped on the record at position 0 (outer edge) .. 1 (label) */
  drop: [number]
  /** arm released off the record */
  lift: []
  'update:volume': [number]
  'update:deckStyle': [1 | 2]
}>()
const vol = computed({ get: () => props.volume, set: v => emit('update:volume', Number(v)) })

// ---------- tonearm geometry (all in cqw, relative to the plinth) ----------
const PIVOT = { x: 82.5, y: 14.5 }
const STYLUS = { x: -2.77, y: 43 } // stylus tip in arm-local coords at 0deg
const CENTER = { x: 37, y: 36.2 } // record centre
const R_OUT = 28 // record edge: needle dropped inside this counts as "on the record"
const R_START = 23.5 // where the music starts: a little into the grooves, not on the rim
const R_IN = 11.8 // last groove, just outside the label
const REST = 6
const MAX = 50

function stylusRadius(deg: number) {
  const t = (deg * Math.PI) / 180
  const x = PIVOT.x + STYLUS.x * Math.cos(t) - STYLUS.y * Math.sin(t)
  const y = PIVOT.y + STYLUS.x * Math.sin(t) + STYLUS.y * Math.cos(t)
  return Math.hypot(x - CENTER.x, y - CENTER.y)
}
// lookup table angle -> position on record (0 outer .. 1 inner, null = off the record)
const table = Array.from({ length: MAX * 10 + 1 }, (_, i) => {
  const deg = i / 10
  const r = stylusRadius(deg)
  return { deg, f: r > R_OUT || r < R_IN ? null : Math.max(0, (R_START - r) / (R_START - R_IN)) }
})
function fractionAt(deg: number) {
  return table[Math.round(Math.min(MAX, Math.max(0, deg)) * 10)].f
}
function angleFor(f: number) {
  f = Math.min(1, Math.max(0, f))
  let best = table.find(e => e.f !== null)!
  for (const e of table) if (e.f !== null && Math.abs(e.f - f) < Math.abs(best.f! - f)) best = e
  return best.deg
}

const plinth = ref<HTMLElement | null>(null)
const dragAngle = ref<number | null>(null)
const angle = computed(() => {
  if (dragAngle.value !== null) return dragAngle.value
  if (!props.armDown) return REST
  return angleFor(Math.max(0.005, props.progress ?? 0))
})
const overRecord = computed(() => dragAngle.value !== null && fractionAt(dragAngle.value) !== null)

// What is actually rendered. Large changes (lift, drop, record swap) swing with an eased
// 1.4s transition; small changes from the 1s progress tick creep linearly. Ticks that arrive
// while a swing is still running are held back, otherwise each tick would restart the eased
// transition from wherever the arm is and the motion would visibly stutter.
const TRAVEL_MS = 1400
const shown = ref(angle.value)
const motion = ref<'travel' | 'track' | 'none'>('none')
let travelUntil = 0
let heldTick: ReturnType<typeof setTimeout> | undefined
watch(angle, (target) => {
  clearTimeout(heldTick)
  if (dragAngle.value !== null) { motion.value = 'none'; shown.value = target; return }
  const now = performance.now()
  if (Math.abs(target - shown.value) >= 1) {
    motion.value = 'travel'
    shown.value = target
    travelUntil = now + TRAVEL_MS
  } else if (now < travelUntil) {
    heldTick = setTimeout(() => { motion.value = 'track'; shown.value = angle.value }, travelUntil - now)
  } else {
    motion.value = 'track'
    shown.value = target
  }
})
onBeforeUnmount(() => clearTimeout(heldTick))

function pointerAngle(e: PointerEvent) {
  const rect = plinth.value!.getBoundingClientRect()
  const cq = rect.width / 100
  const dx = e.clientX - (rect.left + PIVOT.x * cq)
  const dy = e.clientY - (rect.top + PIVOT.y * cq)
  // 0deg = arm pointing straight down, positive = swinging left over the record
  const deg = (Math.atan2(-dx, dy) * 180) / Math.PI
  return Math.min(MAX, Math.max(0, deg))
}
let grabOffset = 0
function onArmDown(e: PointerEvent) {
  if (e.button !== 0) return
  e.preventDefault()
  ;(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId)
  grabOffset = angle.value - pointerAngle(e)
  dragAngle.value = angle.value
  emit('grab')
}
function onArmMove(e: PointerEvent) {
  if (dragAngle.value === null) return
  dragAngle.value = Math.min(MAX, Math.max(0, pointerAngle(e) + grabOffset))
}
function onArmUp() {
  if (dragAngle.value === null) return
  const f = fractionAt(dragAngle.value)
  if (f === null) {
    dragAngle.value = null
    emit('lift')
  } else {
    // keep the arm where it was dropped until the parent reports the new progress
    emit('drop', f)
    dragAngle.value = null // arm glides to the playback position
  }
}
// keyboard: arrows nudge the arm, Enter drops / Escape lifts
function onArmKey(e: KeyboardEvent) {
  if (e.key === 'ArrowLeft' || e.key === 'ArrowRight') {
    e.preventDefault()
    const base = props.armDown ? angle.value : 28
    const next = Math.min(MAX, Math.max(REST, base + (e.key === 'ArrowLeft' ? 1 : -1)))
    const f = fractionAt(next)
    f === null ? emit('lift') : emit('drop', f)
  } else if (e.key === 'Escape') emit('lift')
}

// straight arm tube, in arm-local cqw (pivot at 0,0, headshell at 0,37.6)
const ROD = 'M0 1 L0 37.6'

// Strobe dots on the platter rim (viewBox 200x200, centre 100,100), two staggered rows.
const DOTS = 132
const strobe = [96.2, 92.4].flatMap((r, row) =>
  Array.from({ length: DOTS }, (_, i) => {
    const a = ((i + row * 0.5) / DOTS) * Math.PI * 2
    return { x: +(100 + r * Math.cos(a)).toFixed(2), y: +(100 + r * Math.sin(a)).toFixed(2) }
  }),
)
// Screw holes on the tonearm base platform.
const holes = Array.from({ length: 8 }, (_, i) => {
  const a = (i / 8) * Math.PI * 2 + Math.PI / 8
  return { x: +(50 + 36 * Math.cos(a)).toFixed(2), y: +(50 + 36 * Math.sin(a)).toFixed(2) }
})
</script>

<template>
  <div class="deck" :class="`style-${deckStyle}`">
    <span class="hinge left" /><span class="hinge right" />
    <div ref="plinth" class="plinth">
      <div class="grain" />

      <!-- platter -->
      <div class="platter">
        <svg class="rim" viewBox="0 0 200 200" aria-hidden="true">
          <circle v-for="(d, i) in strobe" :key="i" :cx="d.x" :cy="d.y" r="1.05" class="dot" />
        </svg>
        <div class="record" :class="{ spinning: playing }">
          <div class="grooves" />
          <div class="label" :style="label ? { backgroundImage: `url(${label})` } : {}" />
          <div class="spindle" />
        </div>
        <div class="sheen" />
      </div>

      <!-- tonearm -->
      <!-- arm rest: sits exactly under the rod at the REST angle -->
      <div class="cue"><span /></div>
      <div class="arm-base">
        <svg class="arm-plate" viewBox="0 0 100 100" aria-hidden="true">
          <defs>
            <radialGradient id="plate" cx="40%" cy="35%" r="70%">
              <stop offset="0" stop-color="#ffffff" />
              <stop offset=".55" stop-color="#d4d4d4" />
              <stop offset="1" stop-color="#9c9c9c" />
            </radialGradient>
          </defs>
          <circle cx="50" cy="50" r="48" fill="url(#plate)" stroke="#8c8c8c" stroke-width="1" />
          <circle cx="50" cy="50" r="44" fill="none" stroke="rgba(255,255,255,.7)" stroke-width=".8" />
          <circle v-for="(h, i) in holes" :key="i" :cx="h.x" :cy="h.y" r="2.6" fill="#2a2a2a" stroke="#f2f2f2" stroke-width=".7" />
        </svg>
      </div>
      <div
        class="arm" :class="[`m-${motion}`, { down: armDown || overRecord, dragging: dragAngle !== null }]"
        :style="{ transform: `rotate(${shown}deg)` }"
        role="slider" tabindex="0" aria-label="Tonearm: drag onto the record to play, off to stop"
        :aria-valuenow="Math.round((fractionAt(shown) ?? 0) * 100)" aria-valuemin="0" aria-valuemax="100"
        @pointerdown="onArmDown" @pointermove="onArmMove" @pointerup="onArmUp" @pointercancel="onArmUp" @keydown="onArmKey"
      >
        <div class="counterweight" />
        <div class="collar" />
        <svg class="rod" viewBox="-3 0 6 38.5" aria-hidden="true">
          <defs>
            <!-- round tube shading across the rod -->
            <linearGradient id="tube" gradientUnits="userSpaceOnUse" x1="-.5" y1="0" x2=".5" y2="0">
              <stop offset="0" stop-color="#7a7a7a" />
              <stop offset=".25" stop-color="#cfcfcf" />
              <stop offset=".42" stop-color="#ffffff" />
              <stop offset=".65" stop-color="#d6d6d6" />
              <stop offset="1" stop-color="#8c8c8c" />
            </linearGradient>
          </defs>
          <path class="rod-hit" :d="ROD" />
          <path class="rod-edge" :d="ROD" />
          <path class="rod-body" :d="ROD" />
        </svg>
        <div class="headshell">
          <span class="lift" />
          <span class="body" />
          <span class="cart" />
        </div>
        <div class="hub" />
      </div>

      <div class="knob"><span /></div>

      <button class="start" :class="{ on: playing }" aria-label="Start or stop" @click="emit('toggle')">
        <span class="bar" />
        <svg class="ico" viewBox="0 0 16 16" aria-hidden="true">
          <path v-if="playing" d="M4 3h3v10H4zM9 3h3v10H9z" />
          <path v-else d="M5 2.5v11l8.5-5.5z" />
        </svg>
      </button>

      <div class="volume">
        <span class="cap">VOLUME</span>
        <div class="fader">
          <div class="ticks"><i v-for="n in 9" :key="n" /></div>
          <div class="slot" />
          <input v-model="vol" type="range" min="0" max="100" step="1" aria-label="Volume">
        </div>
      </div>

      <div class="style-sel">
        <span class="cap">STYLE</span>
        <div class="btns">
          <button v-for="s in [1, 2] as const" :key="s" :class="{ on: deckStyle === s }" :aria-pressed="deckStyle === s" @click="emit('update:deckStyle', s)">
            <span class="led" />{{ s }}
          </button>
        </div>
      </div>

      <div class="track">
        <span class="cap">TRACK</span>
        <div class="btns">
          <button aria-label="Previous record" title="Previous record" @click="emit('prev')">
            <svg viewBox="0 0 16 16" aria-hidden="true"><path d="M3 3h2v10H3zM13 3v10L6 8z" /></svg>
          </button>
          <button aria-label="Next record" title="Next record" @click="emit('next')">
            <svg viewBox="0 0 16 16" aria-hidden="true"><path d="M11 3h2v10h-2zM3 3v10l7-5z" /></svg>
          </button>
        </div>
      </div>

      <div class="brand">VINYL DECK</div>
    </div>
  </div>
</template>

<style scoped>
/* All geometry is in container-query units (1cqw = 1% of deck width) so the deck scales as one piece. */
.deck {
  position: relative;
  container-type: inline-size;
  width: min(900px, 100%);
  padding-top: 4cqw;
  user-select: none;
  -webkit-user-select: none;
  --p-hi: #ececea;
  --p-mid: #d6d6d3;
  --p-lo: #c4c4c1;
  --p-edge1: #a9a9a6;
  --p-edge2: #8b8b88;
  --p-ink: #7a7a77;
  --panel: linear-gradient(90deg, #ececea, #d9d9d6);
  --top-hl: rgba(255, 255, 255, .95);
}
.deck.style-2 {
  --p-hi: #3d3d40;
  --p-mid: #2a2a2c;
  --p-lo: #1c1c1e;
  --p-edge1: #111;
  --p-edge2: #050505;
  --p-ink: #9b9b9b;
  --panel: linear-gradient(90deg, #2c2c2f, #222224);
  --top-hl: rgba(255, 255, 255, .14);
}

/* ---------- plinth ---------- */
.plinth {
  position: relative;
  aspect-ratio: 880 / 640;
  border-radius: 2.2cqw;
  background:
    radial-gradient(120% 90% at 30% 0%, rgba(255,255,255,.35), rgba(255,255,255,0) 55%),
    linear-gradient(170deg, var(--p-hi) 0%, var(--p-mid) 60%, var(--p-lo) 100%);
  box-shadow:
    inset 0 .15cqw .1cqw var(--top-hl),
    inset 0 -.35cqw .6cqw rgba(0,0,0,.1),
    0 .3cqw .2cqw rgba(0,0,0,.18),
    0 .5cqw 0 var(--p-edge1),
    0 1.6cqw 2.4cqw rgba(0,0,0,.35),
    0 4cqw 7cqw rgba(0,0,0,.4);
  transition: background .5s, box-shadow .5s;
}
.grain {
  position: absolute; inset: 0; border-radius: inherit; pointer-events: none; opacity: .35; mix-blend-mode: overlay;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='200' height='200'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='1.2' numOctaves='2' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)'/%3E%3C/svg%3E");
}
.hinge {
  position: absolute; top: 0; width: 10.5cqw; height: 4.6cqw; border-radius: 1cqw 1cqw .3cqw .3cqw;
  background: linear-gradient(180deg, #2f2f2f, #121212 75%);
  box-shadow: inset 0 .1cqw 0 rgba(255,255,255,.12), 0 .4cqw .8cqw rgba(0,0,0,.5);
}
.hinge.left { left: 15.5cqw; } .hinge.right { left: 73cqw; }

/* ---------- platter ---------- */
.platter {
  position: absolute; left: 5cqw; top: 4.2cqw; width: 64cqw; height: 64cqw; border-radius: 50%;
  background:
    conic-gradient(from 15deg, #fbfbfb, #bdbdbd 10%, #efefef 20%, #a9a9a9 32%, #f4f4f4 45%, #b8b8b8 57%, #fdfdfd 70%, #a6a6a6 83%, #ececec 92%, #fbfbfb);
  box-shadow:
    inset 0 0 0 .18cqw #8e8e8e,
    inset 0 0 0 .38cqw #f6f6f6,
    0 .25cqw .3cqw rgba(0,0,0,.35),
    0 1.2cqw 2cqw rgba(0,0,0,.35),
    0 2.6cqw 4cqw rgba(0,0,0,.25);
}
.rim { position: absolute; inset: 0; width: 100%; height: 100%; }
.dot { fill: #4b4b4b; stroke: rgba(255,255,255,.55); stroke-width: .25; }
.record {
  will-change: transform;
  position: absolute; inset: 3.7cqw; border-radius: 50%;
  background: #0c0c0c;
  box-shadow:
    0 0 0 .15cqw #030303,
    0 0 0 .3cqw rgba(255,255,255,.08),
    0 .3cqw .9cqw rgba(0,0,0,.7);
  animation: spin 4s linear infinite; animation-play-state: paused;
}
.record.spinning { animation-play-state: running; }
.grooves {
  position: absolute; inset: 0; border-radius: 50%;
  background:
    /* edge & lead-in */
    radial-gradient(circle closest-side, transparent 0 96.5%, #191919 97% 98%, #070707 98.5%),
    /* track gaps */
    radial-gradient(circle closest-side,
      transparent 0 50%, #232323 50.3% 50.9%, transparent 51.2% 62%, #232323 62.3% 62.9%,
      transparent 63.2% 73%, #232323 73.3% 73.9%, transparent 74.2% 84%, #232323 84.3% 84.9%, transparent 85.2%),
    /* run-out area around the label */
    radial-gradient(circle closest-side, #0b0b0b 0 41%, transparent 41.5%),
    /* fine grooves */
    repeating-radial-gradient(circle closest-side, #0d0d0d 0 .11cqw, #171717 .11cqw .2cqw, #0f0f0f .2cqw .28cqw);
}
.label {
  position: absolute; left: 50%; top: 50%; width: 37%; height: 37%; transform: translate(-50%, -50%);
  border-radius: 50%;
  background: radial-gradient(circle closest-side, #f2e1c4, #c78d4f);
  background-size: auto 136%; background-position: center;
  box-shadow: 0 0 0 .2cqw #050505, inset 0 0 0 .25cqw rgba(0,0,0,.25), inset 0 0 1.2cqw rgba(0,0,0,.25);
}
.spindle {
  position: absolute; left: 50%; top: 50%; width: 1.5cqw; height: 1.5cqw; transform: translate(-50%, -50%);
  border-radius: 50%; background: radial-gradient(circle at 35% 30%, #fff, #c8c8c8 40%, #7a7a7a);
  box-shadow: 0 .15cqw .3cqw rgba(0,0,0,.6);
}
/* light reflection stays still while the record turns */
.sheen {
  position: absolute; inset: 3.7cqw; border-radius: 50%; pointer-events: none;
  background:
    conic-gradient(from 315deg, rgba(255,255,255,0) 0deg, rgba(255,255,255,.13) 18deg, rgba(255,255,255,0) 40deg,
      rgba(255,255,255,0) 180deg, rgba(255,255,255,.11) 198deg, rgba(255,255,255,0) 220deg, rgba(255,255,255,0) 360deg),
    radial-gradient(circle at 30% 25%, rgba(255,255,255,.06), transparent 50%);
  -webkit-mask: radial-gradient(circle closest-side, transparent 0 39%, #000 40% 97%, transparent 98%);
          mask: radial-gradient(circle closest-side, transparent 0 39%, #000 40% 97%, transparent 98%);
}
@keyframes spin { to { transform: rotate(360deg); } }

/* ---------- tonearm ---------- */
.arm-base {
  position: absolute; left: 73.5cqw; top: 5.5cqw; width: 18cqw; height: 18cqw; border-radius: 50%;
  background: radial-gradient(circle at 45% 40%, #2e2e2e, #141414 60%, #0a0a0a);
  box-shadow:
    inset 0 .15cqw .1cqw rgba(255,255,255,.12),
    inset 0 -.3cqw .6cqw rgba(0,0,0,.6),
    0 .3cqw .4cqw rgba(0,0,0,.35),
    0 1.2cqw 2cqw rgba(0,0,0,.35);
}
.arm-plate { position: absolute; left: 22%; top: 22%; width: 56%; height: 56%; filter: drop-shadow(0 .3cqw .4cqw rgba(0,0,0,.5)); }
.arm {
  position: absolute; left: 82.5cqw; top: 14.5cqw; width: 0; height: 0;
  filter: drop-shadow(1.6cqw 2.2cqw 1.1cqw rgba(0,0,0,.32));
  will-change: transform;
  outline: none; touch-action: none;
}
.arm.m-travel { transition: transform 1.4s cubic-bezier(.45,.05,.25,1); }
.arm.m-track { transition: transform 1s linear; }
.arm.down { filter: drop-shadow(.7cqw 1cqw .5cqw rgba(0,0,0,.4)); }
/* while held the arm is lifted: bigger, softer shadow and no easing lag */
.arm.dragging { transition: none; filter: drop-shadow(1.8cqw 2.6cqw 1.3cqw rgba(0,0,0,.3)); }
.arm .rod, .arm .headshell, .arm .counterweight { cursor: grab; }
.arm.dragging .rod, .arm.dragging .headshell, .arm.dragging .counterweight { cursor: grabbing; }
.arm:focus-visible .headshell .body { outline: .25cqw solid var(--accent); outline-offset: .3cqw; }
.rod { position: absolute; left: -3cqw; top: 0; width: 6cqw; height: 38.5cqw; overflow: visible; }
.rod path { fill: none; stroke-linecap: round; }
.rod-hit { stroke: transparent; stroke-width: 3; pointer-events: stroke; }
.rod-edge { stroke: rgba(70,70,70,.55); stroke-width: 1.06; pointer-events: none; }
.rod-body { stroke: url(#tube); stroke-width: .9; pointer-events: none; }
.counterweight {
  position: absolute; left: -2.9cqw; top: -11.5cqw; width: 5.8cqw; height: 8cqw; border-radius: 1.2cqw 1.2cqw .7cqw .7cqw;
  background:
    repeating-linear-gradient(0deg, rgba(0,0,0,0) 0 .42cqw, rgba(0,0,0,.22) .42cqw .52cqw, rgba(255,255,255,.25) .52cqw .58cqw),
    linear-gradient(90deg, #7d7d7d, #cfcfcf 20%, #ffffff 40%, #c4c4c4 65%, #8f8f8f 85%, #666);
  box-shadow: inset 0 .2cqw .1cqw rgba(255,255,255,.6), inset 0 -.3cqw .2cqw rgba(0,0,0,.25);
}
.collar {
  position: absolute; left: -1.5cqw; top: -3.8cqw; width: 3cqw; height: 1.4cqw; border-radius: .3cqw;
  background: linear-gradient(90deg, #111, #3a3a3a 40%, #111);
}
.hub {
  position: absolute; left: -3.1cqw; top: -3.1cqw; width: 6.2cqw; height: 6.2cqw; border-radius: 50%;
  background:
    radial-gradient(circle closest-side, #0e0e0e 0 26%, #9a9a9a 28% 32%, transparent 34%),
    radial-gradient(circle at 38% 32%, #ffffff, #dcdcdc 40%, #a3a3a3 75%, #858585);
  box-shadow: inset 0 0 0 .15cqw rgba(0,0,0,.25), 0 .2cqw .5cqw rgba(0,0,0,.45);
}
.headshell { position: absolute; left: -2.6cqw; top: 37cqw; width: 5.2cqw; height: 8cqw; transform: rotate(24deg); transform-origin: 50% 0; }
.headshell .body {
  position: absolute; left: .6cqw; top: 0; width: 3.8cqw; height: 6.8cqw; border-radius: .5cqw .5cqw .8cqw .8cqw;
  background:
    repeating-linear-gradient(90deg, rgba(0,0,0,0) 0 .55cqw, rgba(0,0,0,.18) .55cqw .65cqw),
    linear-gradient(90deg, #8e8e8e, #f6f6f6 45%, #b9b9b9 75%, #808080);
  box-shadow: inset 0 0 0 .12cqw rgba(0,0,0,.25);
}
.headshell .lift { position: absolute; left: 4.1cqw; top: 1.6cqw; width: 2cqw; height: .55cqw; border-radius: .3cqw; background: linear-gradient(#e8e8e8, #8c8c8c); transform: rotate(-12deg); }
.headshell .cart { position: absolute; left: 1.3cqw; top: 5.8cqw; width: 2.4cqw; height: 1.7cqw; border-radius: .2cqw .2cqw .4cqw .4cqw; background: linear-gradient(#333, #0b0b0b); }

/* ---------- small controls ---------- */
.cue {
  position: absolute; left: 77.08cqw; top: 41cqw; width: 5cqw; height: 2.6cqw; border-radius: 1.3cqw;
  transform: rotate(6deg);
  background: radial-gradient(ellipse at 50% 25%, #6a6a6a, #1d1d1d 55%, #050505);
  box-shadow: 0 .25cqw .3cqw rgba(0,0,0,.45), 0 .8cqw 1.2cqw rgba(0,0,0,.3);
}
/* the groove the rod lies in */
.cue span { position: absolute; left: 50%; top: 18%; width: 1.3cqw; height: 64%; transform: translateX(-50%); border-radius: .3cqw; background: linear-gradient(90deg, #000, #2a2a2a 50%, #000); box-shadow: inset 0 .1cqw .2cqw rgba(0,0,0,.8); }
.knob {
  position: absolute; left: 3.6cqw; top: 50.5cqw; width: 5.4cqw; height: 5.4cqw; border-radius: 50%;
  background: radial-gradient(circle at 36% 30%, #6b6b6b, #262626 38%, #0a0a0a 70%, #000);
  box-shadow: inset 0 -.2cqw .4cqw rgba(255,255,255,.08), 0 .3cqw .3cqw rgba(0,0,0,.45), 0 1cqw 1.4cqw rgba(0,0,0,.35);
}
.knob span {
  position: absolute; left: 50%; top: 50%; width: .65cqw; height: .65cqw; transform: translate(-50%, -50%); border-radius: 50%;
  background: #ff5a36; box-shadow: 0 0 .5cqw #ff5a36, 0 0 1.2cqw rgba(255,80,40,.6);
}

.start {
  position: absolute; left: 3cqw; top: 61.3cqw; width: 10.8cqw; height: 9cqw; border: 0; border-radius: 1.3cqw;
  background: linear-gradient(180deg, #ffffff, #f0f0ee 55%, #dfdfdc); color: #3a3a3a;
  box-shadow:
    0 0 0 .1cqw rgba(0,0,0,.22),
    inset 0 .15cqw .05cqw #fff,
    inset 0 -.4cqw .5cqw rgba(0,0,0,.08),
    0 .35cqw .3cqw rgba(0,0,0,.22),
    0 1cqw 1.6cqw rgba(0,0,0,.22);
  display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 1.4cqw;
  transition: transform .1s, box-shadow .1s;
}
.start:hover { background: linear-gradient(180deg, #ffffff, #f6f6f4 55%, #e6e6e3); }
.start:active {
  transform: translateY(.25cqw);
  box-shadow: 0 0 0 .1cqw rgba(0,0,0,.25), inset 0 .25cqw .5cqw rgba(0,0,0,.14), 0 .1cqw .2cqw rgba(0,0,0,.25);
}
.start .ico { width: 2.2cqw; height: 2.2cqw; fill: #3a3a3a; }
.start .bar { width: 3.6cqw; height: .6cqw; border-radius: .3cqw; background: #4a2a22; transition: background .2s, box-shadow .2s; }
.start.on .bar { background: var(--accent); box-shadow: 0 0 .8cqw var(--accent); }

.cap { display: block; font-size: 1.2cqw; letter-spacing: .22cqw; color: var(--p-ink); font-weight: 600; text-align: center; }
.volume { position: absolute; left: 84.5cqw; top: 25.5cqw; width: 11cqw; }
.fader {
  position: relative; margin: 1.3cqw auto 0; width: 8cqw; height: 22cqw; border-radius: 1cqw;
  background: var(--panel);
  box-shadow: inset 0 .2cqw .6cqw rgba(0,0,0,.18), inset 0 -.1cqw .1cqw rgba(255,255,255,.6), 0 0 0 .08cqw rgba(0,0,0,.12), 0 .12cqw .05cqw rgba(255,255,255,.7);
}
.style-2 .fader { box-shadow: inset 0 .15cqw .4cqw rgba(0,0,0,.6), 0 0 0 .12cqw #000; }
.ticks { position: absolute; left: 1.1cqw; top: 2.4cqw; bottom: 2.4cqw; display: flex; flex-direction: column; justify-content: space-between; }
.ticks i { display: block; width: 1.5cqw; height: .18cqw; background: #4a4a4a; }
.style-2 .ticks i { background: #8a8a8a; }
.slot { position: absolute; left: 4.9cqw; top: 1.6cqw; bottom: 1.6cqw; width: .55cqw; border-radius: .3cqw; background: #0b0b0b; box-shadow: inset 0 .1cqw .2cqw #000, 0 .1cqw 0 rgba(255,255,255,.5); }
.fader input {
  position: absolute; left: 3.1cqw; top: 1cqw; height: 20cqw; width: 4.2cqw; margin: 0;
  writing-mode: vertical-lr; direction: rtl; -webkit-appearance: none; appearance: none; background: transparent; cursor: ns-resize;
}
.fader input::-webkit-slider-runnable-track { width: 100%; background: transparent; }
.fader input::-moz-range-track { background: transparent; }
.fader input::-webkit-slider-thumb {
  -webkit-appearance: none; width: 4.4cqw; height: 3.2cqw; border-radius: .3cqw;
  background:
    linear-gradient(180deg, rgba(0,0,0,0) 46%, #555 46% 54%, rgba(0,0,0,0) 54%),
    repeating-linear-gradient(180deg, #fafafa 0 .38cqw, #c9c9c9 .38cqw .5cqw),
    #e5e5e5;
  box-shadow: inset 0 0 0 .1cqw rgba(0,0,0,.25), 0 .25cqw .2cqw rgba(0,0,0,.35), 0 .8cqw 1cqw rgba(0,0,0,.35);
}
.fader input::-moz-range-thumb {
  width: 4.4cqw; height: 3.2cqw; border: 0; border-radius: .3cqw;
  background:
    linear-gradient(180deg, rgba(0,0,0,0) 46%, #555 46% 54%, rgba(0,0,0,0) 54%),
    repeating-linear-gradient(180deg, #fafafa 0 .38cqw, #c9c9c9 .38cqw .5cqw),
    #e5e5e5;
  box-shadow: inset 0 0 0 .1cqw rgba(0,0,0,.25), 0 .25cqw .2cqw rgba(0,0,0,.35), 0 .8cqw 1cqw rgba(0,0,0,.35);
}
.fader input:focus-visible { outline: .2cqw solid var(--accent); outline-offset: .3cqw; }

.style-sel { position: absolute; left: 84.5cqw; top: 51cqw; width: 11cqw; }
.track { position: absolute; left: 84.5cqw; top: 60.5cqw; width: 11cqw; }

/* key well: the buttons sit in a recessed tray */
.btns {
  display: flex; width: fit-content; gap: .5cqw; margin: .9cqw auto 0; padding: .45cqw; border-radius: 1cqw;
  background: linear-gradient(180deg, rgba(0,0,0,.14), rgba(0,0,0,.04));
  box-shadow: inset 0 .15cqw .35cqw rgba(0,0,0,.35), inset 0 -.05cqw 0 rgba(255,255,255,.25), 0 .1cqw 0 rgba(255,255,255,.75);
}
.style-2 .btns { background: rgba(0,0,0,.35); box-shadow: inset 0 .15cqw .35cqw rgba(0,0,0,.8), 0 .1cqw 0 rgba(255,255,255,.08); }
.btns button {
  position: relative; width: 4.3cqw; height: 4.3cqw; padding: 0; border: 0; border-radius: .8cqw;
  background: linear-gradient(180deg, #ffffff, #efefed 50%, #dcdcd9);
  color: #3a3a3a; font-size: 1.2cqw; font-weight: 700;
  box-shadow:
    0 0 0 .08cqw rgba(0,0,0,.25),
    inset 0 .14cqw .05cqw #fff,
    inset 0 -.3cqw .4cqw rgba(0,0,0,.08),
    0 .25cqw .3cqw rgba(0,0,0,.25),
    0 .6cqw 1cqw rgba(0,0,0,.15);
  display: flex; flex-direction: column; align-items: center; justify-content: center; gap: .35cqw;
  transition: transform .08s, box-shadow .08s, background .15s;
}
.btns button:hover { background: linear-gradient(180deg, #ffffff, #f6f6f4 50%, #e4e4e1); }
.btns button:active, .btns button.on {
  transform: translateY(.14cqw);
  background: linear-gradient(180deg, #e2e2df, #ececea);
  box-shadow: 0 0 0 .08cqw rgba(0,0,0,.28), inset 0 .2cqw .45cqw rgba(0,0,0,.18), 0 .08cqw .12cqw rgba(0,0,0,.25);
}
.btns button:focus-visible { outline: .2cqw solid var(--accent); outline-offset: .25cqw; }
.btns .led { width: 1.8cqw; height: .4cqw; border-radius: .2cqw; background: #3a3a3a; box-shadow: inset 0 .05cqw .1cqw rgba(0,0,0,.6); transition: background .2s, box-shadow .2s; }
.btns button.on .led { background: var(--accent); box-shadow: 0 0 .5cqw var(--accent), 0 0 1.1cqw rgba(232,116,59,.55); }
.btns svg { width: 1.6cqw; height: 1.6cqw; fill: #2e2e2e; filter: drop-shadow(0 .06cqw 0 rgba(255,255,255,.8)); }
.btns button:active svg { fill: var(--accent); }

.brand {
  position: absolute; left: 66cqw; top: 64cqw; font-size: 1.85cqw; letter-spacing: .8cqw; color: var(--p-ink); font-weight: 500;
  padding-bottom: .7cqw; border-bottom: .14cqw solid currentColor; opacity: .9;
  text-shadow: 0 .1cqw 0 rgba(255,255,255,.6);
}
.style-2 .brand { text-shadow: 0 -.1cqw 0 rgba(0,0,0,.8); }

@media (prefers-reduced-motion: reduce) { .record { animation-duration: 10s; } .arm { transition-duration: .2s; } }
</style>
