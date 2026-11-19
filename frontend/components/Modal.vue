<script setup lang="ts">
const props = defineProps<{ open: boolean; title: string }>()
const dialog = ref<HTMLElement | null>(null)
watch(() => props.open, async (o) => {
  if (!o) return
  await nextTick()
  dialog.value?.querySelector<HTMLElement>('input, textarea, select, button.primary')?.focus()
})
const emit = defineEmits<{ close: [] }>()
function onKey(e: KeyboardEvent) { if (e.key === 'Escape') emit('close') }
onMounted(() => window.addEventListener('keydown', onKey))
onBeforeUnmount(() => window.removeEventListener('keydown', onKey))
</script>

<template>
  <Transition name="fade">
    <div v-if="open" class="overlay" @click.self="emit('close')">
      <div ref="dialog" class="dialog" role="dialog" :aria-label="title">
        <header><h2>{{ title }}</h2><button class="x" aria-label="Close" @click="emit('close')">×</button></header>
        <slot />
      </div>
    </div>
  </Transition>
</template>

<style scoped>
.overlay { position: fixed; inset: 0; z-index: 50; background: rgba(0,0,0,.6); display: grid; place-items: center; padding: 16px; }
.dialog { width: min(460px, 100%); background: #2a2420; border: 1px solid #4b3d33; border-radius: 16px; padding: 22px; box-shadow: 0 20px 50px rgba(0,0,0,.6); }
header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 14px; }
h2 { margin: 0; font-size: 18px; font-weight: 600; }
.x { border: 0; background: none; color: var(--muted); font-size: 24px; line-height: 1; }
.fade-enter-active, .fade-leave-active { transition: opacity .2s; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
</style>
