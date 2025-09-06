<template>
<aside class="rail">
<div class="rail-top">
<button class="rail-btn" :class="{ active: currentView === 'chat' }" @click="$emit('view-change', 'chat')" title="Chats">
<svg viewBox="0 0 24 24" class="ico"><path d="M4 4h16v12H7l-3 3V4z"/></svg>
<span class="rail-badge" v-if="totalUnread > 0">{{ totalUnread }}</span>
</button>
<button class="rail-btn" :class="{ active: currentView === 'contacts' }" @click="$emit('view-change', 'contacts')" title="Contacts">
<svg viewBox="0 0 24 24" class="ico"><path d="M16 4c0-1.11.89-2 2-2s2 .89 2 2-.89 2-2 2-2-.89-2-2M4 18v-1c0-1.1.9-2 2-2h2c1.1 0 2 .9 2 2v1h2v-1c0-1.1.9-2 2-2h2c1.1 0 2 .9 2 2v1h2c1.1 0 2-.9 2-2v-3H2v3c0 1.1.9 2 2 2h2M18 9c-1.1 0-2-.9-2-2s.9-2 2-2 2 .9 2 2-.9 2-2 2m-8 0c-1.1 0-2-.9-2-2s.9-2 2-2 2 .9 2 2-.9 2-2 2M6 9c-1.1 0-2-.9-2-2s.9-2 2-2 2 .9 2 2-.9 2-2 2"/></svg>
</button>
<button class="rail-btn" title="Calls"><svg viewBox="0 0 24 24" class="ico"><path d="M6.6 10.8c1.3 2.6 3.4 4.7 6 6l2-2c.3-.3.7-.4 1.1-.3 1 .3 2 .5 3 .5.6 0 1 .4 1 1V20c0 .6-.4 1-1 1C10.6 21 3 13.4 3 4c0-.6.4-1 1-1h3c.6 0 1 .4 1 1 0 1 .2 2 .5 3 .1.4 0 .8-.3 1.1l-1.6 1.7z"/></svg></button>
<button class="rail-btn" title="Status"><svg viewBox="0 0 24 24" class="ico"><path d="M12 2a10 10 0 100 20 10 10 0 000-20zm0 3a7 7 0 110 14 7 7 0 010-14z"/></svg></button>
</div>
<div class="rail-bottom">
<button class="rail-btn" :class="{ active: currentView === 'settings' }" @click="$emit('view-change', 'settings')" title="Settings">
<svg viewBox="0 0 24 24" class="ico"><path d="M19.4 13a7.8 7.8 0 000-2l2-1.5-2-3.4-2.3.6a7.8 7.8 0 00-1.7-1l-.3-2.4H9l-.3 2.4a7.8 7.8 0 00-1.7 1L4.7 6.1l-2 3.4L4.8 11a7.8 7.8 0 000 2l-2 1.5 2 3.4 2.3-.6c.5.4 1.1.7 1.7 1l.3 2.4h4.6l.3-2.4c.6-.3 1.2-.6 1.7-1l2.3.6 2-3.4L19.4 13zM12 15a3 3 0 110-6 3 3 0 010 6z"/></svg>
</button>
</div>
</aside>
</template>
<script setup lang="ts">
defineProps<{ totalUnread:number, currentView:string }>()
defineEmits<{ 'view-change': [view: string] }>()
</script>
<style scoped>
.rail{background:var(--panel-2);display:flex;flex-direction:column;justify-content:space-between;align-items:center;padding:10px 0;border-right:1px solid var(--panel-3)}
.rail-top,.rail-bottom{display:flex;flex-direction:column;gap:8px}
.rail-btn{position:relative;width:44px;height:44px;border:none;background:transparent;color:var(--muted);border-radius:12px;display:grid;place-items:center;cursor:pointer;transition:all 0.2s}
.rail-btn:hover{background:var(--hover);color:var(--text)}
.rail-btn.active{background:var(--brand);color:white}
.rail-badge{position:absolute;bottom:-4px;right:-4px;background:var(--brand);color:#002d24;font-weight:700;border-radius:10px;padding:1px 6px;font-size:11px}
.ico{width:22px;height:22px;fill:currentColor}
</style>