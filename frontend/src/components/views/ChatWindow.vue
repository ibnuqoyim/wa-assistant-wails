<template>
<section class="chat">
<header class="chat-header" v-if="chat">
<div class="left">
<div class="avatar" :style="{ background: pickColor(chat.name) }">{{ initials(chat.name) }}</div>
<div><div class="title">{{ chat.name }}</div><div class="sub">Group • last seen recently</div></div>
</div>
<div class="right">
<button class="icon-btn" title="Search"><svg viewBox="0 0 24 24" class="ico"><path d="M10 18a8 8 0 100-16 8 8 0 000 16zm11 3l-6-6" stroke="currentColor" stroke-width="2" fill="none" stroke-linecap="round"/></svg></button>
<button class="icon-btn" title="Voice"><svg viewBox="0 0 24 24" class="ico"><path d="M6.6 10.8c1.3 2.6 3.4 4.7 6 6l2-2c.3-.3.7-.4 1.1-.3 1 .3 2 .5 3 .5.6 0 1 .4 1 1V20c0 .6-.4 1-1 1C10.6 21 3 13.4 3 4c0-.6.4-1 1-1h3c.6 0 1 .4 1 1 0 1 .2 2 .5 3 .1.4 0 .8-.3 1.1l-1.6 1.7z"/></svg></button>
<button class="icon-btn" title="Video"><svg viewBox="0 0 24 24" class="ico"><path d="M3 6h11a2 2 0 012 2v8a2 2 0 01-2 2H3V6zm15 3l4-2v10l-4-2V9z"/></svg></button>
<button class="icon-btn" title="More"><svg viewBox="0 0 24 24" class="ico"><circle cx="12" cy="5" r="1.5"/><circle cx="12" cy="12" r="1.5"/><circle cx="12" cy="19" r="1.5"/></svg></button>
</div>
</header>


<div class="messages" ref="wrap">
<MessageBubble v-for="m in messages" :key="m.id" :msg="m" />
</div>


<Composer v-model="draftLocal" @send="emit('send')" />


<div class="infobar" v-if="showInfo">
<div>You were added by someone who's not in your contacts</div>
<div class="actions">
<button class="pill">Exit group</button>
<button class="pill">Report</button>
<button class="pill primary" @click="emit('hide-info')">OK</button>
</div>
</div>
</section>
</template>
<script setup lang="ts">
import type { ChatItem, Msg } from '@/composables/useChatData'
import { ref, watch, nextTick } from 'vue'
import MessageBubble from './MessageBubble.vue'
import Composer from './Composer.vue'
import { initials, pickColor } from '@/utils/text'


const props = defineProps<{ chat: ChatItem|undefined; messages: Msg[]; draft: string; showInfo: boolean }>()
const emit = defineEmits<{(e:'send'):void;(e:'hide-info'):void;(e:'update:draft', v:string):void}>()


const wrap = ref<HTMLDivElement|null>(null)
const draftLocal = ref(props.draft)


watch(()=>props.draft, v=>draftLocal.value=v)
watch(draftLocal, v=>emit('update:draft', v))


watch(()=>props.messages, () => {
nextTick(()=>{ if(wrap.value){ wrap.value.scrollTop = wrap.value.scrollHeight } })
}, { deep:true })
</script>
<style scoped>
.chat{display:grid;grid-template-rows:auto 1fr auto auto;position:relative}
.chat-header{display:flex;align-items:center;justify-content:space-between;padding:10px 16px;border-bottom:1px solid var(--panel-3)}
.chat-header .left{display:flex;align-items:center;gap:10px}
.chat-header .title{font-weight:600}
.chat-header .sub{color:var(--muted);font-size:12px}
.avatar{width:40px;height:40px;border-radius:50%;background:#4c7a6f;display:grid;place-items:center;color:#fff;font-weight:700}
.chat-header .right {
  display: flex;
  align-items: center;
  gap: 6px;         /* jarak antar ikon */
}

/* optional jika ada global CSS yang bikin button block */
.icon-btn {
  display: inline-grid;  /* atau inline-flex */
}
.infobar{position:absolute;left:16px;right:16px;bottom:64px;background:var(--panel);border:1px solid var(--panel-3);padding:10px 12px;border-radius:10px;display:flex;align-items:center;justify-content:space-between;gap:10px;color:var(--muted)}
.pill{background:transparent;border:1px solid var(--panel-3);color:var(--text);padding:6px 10px;border-radius:999px;cursor:pointer}
.pill.primary{background:var(--brand);color:#003d32;border-color:var(--brand)}
.ico{width:22px;height:22px;fill:currentColor}
.icon-btn{width:36px;height:36px;border:none;background:transparent;color:var(--muted);border-radius:10px;display:grid;place-items:center;cursor:pointer}
.icon-btn:hover{background:var(--hover);color:var(--text)}
.messages {
  background-color: #0b141a;
  overflow-y: auto;
  padding: 20px 16px;
  display: flex;
  flex-direction: column;
  gap: 4px; /* jarak antar pesan lebih kecil */
}
</style>