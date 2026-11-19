<script setup lang="ts">
defineProps<{ title: string; sub?: string }>()
</script>

<template>
  <div class="plate">
    <i class="screw tl" /><i class="screw tr" /><i class="screw bl" /><i class="screw br" />
    <p class="title"><span>{{ title }}</span></p>
    <!-- absolutely positioned: showing it never changes the plate height -->
    <p v-if="sub" class="sub">{{ sub }}</p>
  </div>
</template>

<style scoped>
.plate {
  position: relative; width: min(580px, 100%); margin: 0 auto; padding: 16px 48px;
  border-radius: 10px; text-align: center;
  background:
    repeating-linear-gradient(0deg, rgba(255,255,255,.07) 0 1px, rgba(0,0,0,.035) 1px 2px, rgba(255,255,255,0) 2px 3px),
    linear-gradient(100deg, rgba(255,255,255,0) 30%, rgba(255,255,255,.35) 48%, rgba(255,255,255,0) 66%),
    linear-gradient(180deg, #e9e9e7, #cfcfcc 55%, #bdbdba);
  box-shadow:
    inset 0 1px 0 rgba(255,255,255,.95),
    inset 0 -1px 0 rgba(0,0,0,.2),
    inset 1px 0 0 rgba(255,255,255,.5),
    inset -1px 0 0 rgba(0,0,0,.12),
    0 1px 2px rgba(0,0,0,.3),
    0 8px 16px rgba(0,0,0,.3),
    0 18px 40px rgba(0,0,0,.3);
}
.title {
  margin: 0; min-height: 3.3em; display: flex; align-items: center; justify-content: center;
  font-size: clamp(13px, 1.6vw, 17px); letter-spacing: .14em; text-transform: uppercase; line-height: 1.65; font-weight: 700;
  color: #45454a; text-shadow: 0 1px 0 rgba(255,255,255,.75), 0 -1px 0 rgba(0,0,0,.08);
}
.title span { display: -webkit-box; -webkit-line-clamp: 3; -webkit-box-orient: vertical; overflow: hidden; }
.sub {
  /* status line overlays the bottom edge so it never pushes the title off-centre */
  position: absolute; left: 48px; right: 48px; bottom: 5px; margin: 0; line-height: 1.4;
  font-size: 11px; letter-spacing: .08em; white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  color: #8a3b1e; text-shadow: 0 1px 0 rgba(255,255,255,.6);
}
.screw {
  position: absolute; width: 9px; height: 9px; border-radius: 50%;
  background: radial-gradient(circle at 35% 30%, #fff, #c4c4c1 45%, #7e7e7b);
  box-shadow: inset 0 0 0 1px rgba(0,0,0,.25), 0 1px 0 rgba(255,255,255,.8);
}
.screw::after { content: ''; position: absolute; left: 1px; right: 1px; top: 50%; height: 1px; margin-top: -.5px; background: rgba(0,0,0,.45); transform: rotate(35deg); }
.tl { left: 10px; top: 10px; } .tr { right: 10px; top: 10px; } .bl { left: 10px; bottom: 10px; } .br { right: 10px; bottom: 10px; }
</style>
