<script setup lang="ts">
import type { VinylRecord } from '~/composables/useRecords'
defineProps<{ records: VinylRecord[]; activeId?: string | null; loading?: boolean }>()
const emit = defineEmits<{ select: [VinylRecord]; remove: [VinylRecord]; add: [] }>()
</script>

<template>
  <aside class="shelf">
    <div class="list">
      <p v-if="loading && !records.length" class="empty">Loading the shelf…</p>
      <p v-else-if="!records.length" class="empty">The shelf is empty. Add a YouTube mix to get started.</p>
      <div
        v-for="r in records" :key="r.id" class="item" :class="{ active: r.id === activeId }"
        role="button" tabindex="0" @click="emit('select', r)" @keydown.enter="emit('select', r)"
      >
        <div class="sleeve-wrap">
          <div class="disc"><div class="vinyl" :style="{ '--cover': `url(${r.thumbnail})` }" /></div>
          <div class="sleeve" role="img" :aria-label="r.title" :style="{ backgroundImage: `url(${r.thumbnail})` }" />
          <button class="del" title="Remove from shelf" aria-label="Remove" @click.stop="emit('remove', r)">×</button>
        </div>
        <p class="name">{{ r.title }}</p>
      </div>
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
.item { margin-bottom: 26px; cursor: pointer; outline: none; }
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
  .list { display: flex; gap: 14px; overflow-x: auto; overflow-y: hidden; padding: 10px 12px; }
  .item { width: 96px; flex-shrink: 0; margin: 0; }
  .sleeve-wrap, .sleeve { width: 72px; height: 72px; }
  .disc { left: 3px; top: 3px; width: 66px; height: 66px; transform: translateX(10px); }
  .item:hover .disc, .item:focus-visible .disc { transform: translateX(16px); }
  .item.active .disc { transform: translateX(22px); }
  .del { width: 20px; height: 20px; font-size: 14px; right: 3px; top: 3px; }
  .name { margin-top: 6px; font-size: 11px; -webkit-line-clamp: 1; }
  .add { flex-shrink: 0; margin: 10px 12px; padding: 8px 12px; font-size: 13px; white-space: nowrap; }
}
</style>
