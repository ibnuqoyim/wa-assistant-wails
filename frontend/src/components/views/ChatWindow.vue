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


<div class="messages" ref="wrap" @scroll="handleScroll">
  <div v-if="isLoadingMore" class="loading-more">
    Loading more messages...
  </div>
  <MessageBubble v-for="m in sortedMessages" :key="m.id" :msg="m" />
</div>

<Composer v-model="draftLocal" @send="emit('send')" />



</section>
</template>
<script setup lang="ts">
import type { ChatItem, Msg } from '@/composables/useChatData'
import { ref, watch, computed } from 'vue'
import MessageBubble from './MessageBubble.vue'
import Composer from './Composer.vue'
import { initials, pickColor } from '@/utils/text'


const props = defineProps<{ 
  chat: ChatItem|undefined; 
  messages: Msg[]; 
  draft: string; 
  showInfo: boolean 
}>()

const emit = defineEmits<{
  (e:'send'):void;
  (e:'hide-info'):void;
  (e:'update:draft', v:string):void;
  (e:'load-more'):void;
}>()

const wrap = ref<HTMLDivElement|null>(null)
const draftLocal = ref(props.draft)
const isLoadingMore = ref(false)

// Compute sorted messages (newest first)
const sortedMessages = computed(() => {
  return [...props.messages].reverse()
})

// Watch for draft changes
watch(() => props.draft, v => draftLocal.value = v)
watch(draftLocal, v => emit('update:draft', v))

// Handle scroll to load more messages
let loadMoreThrottle = false
const handleScroll = async (event: Event) => {
  const element = event.target as HTMLElement
  if (loadMoreThrottle) return
  
  // Check if we're near the bottom (since messages are reversed, bottom is actually top)
  if (element.scrollTop < 100) {
    loadMoreThrottle = true
    isLoadingMore.value = true
    
    // Emit load-more event
    emit('load-more')
    
    // Reset throttle after 1 second
    setTimeout(() => {
      loadMoreThrottle = false
      isLoadingMore.value = false
    }, 1000)
  }
}

// Watch for message changes
watch(() => props.messages, () => {
  // Messages are automatically positioned correctly due to flex-direction: column-reverse
}, { deep: true })
</script>
<style scoped>
.chat {
  display: grid;
  grid-template-rows: 60px 1fr auto;
  position: relative;
  height: 100vh;
  max-height: 100vh;
  background-color: #0b141a;
}

.chat-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 16px;
  border-bottom: 1px solid var(--panel-3);
  background: var(--panel);
  height: 60px;
  position: sticky;
  top: 0;
  z-index: 10;
}

.chat-header .left {
  display: flex;
  align-items: center;
  gap: 10px;
}

.chat-header .title {
  font-weight: 600;
}

.chat-header .sub {
  color: var(--muted);
  font-size: 12px;
}

.avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: #4c7a6f;
  display: grid;
  place-items: center;
  color: #fff;
  font-weight: 700;
  flex-shrink: 0;
}

.chat-header .right {
  display: flex;
  align-items: center;
  gap: 6px;
}

.icon-btn {
  width: 36px;
  height: 36px;
  border: none;
  background: transparent;
  color: var(--muted);
  border-radius: 10px;
  display: grid;
  place-items: center;
  cursor: pointer;
}

.icon-btn:hover {
  background: var(--hover);
  color: var(--text);
}

.ico {
  width: 22px;
  height: 22px;
  fill: currentColor;
}

.messages {
  height: calc(100vh - 120px); /* 60px header + 60px composer */
  overflow-y: auto;
  padding: 20px 16px;
  display: flex;
  flex-direction: column-reverse; /* reverse the direction */
  gap: 4px;
  background-color: #0b141a;
}

.messages::-webkit-scrollbar {
  width: 6px;
}

.messages::-webkit-scrollbar-track {
  background: transparent;
}

.messages::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.1);
  border-radius: 3px;
}

.messages::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.2);
}

.loading-more {
  text-align: center;
  padding: 10px;
  color: var(--muted);
  font-size: 14px;
  background: rgba(255, 255, 255, 0.05);
  border-radius: 8px;
  margin: 10px 0;
}
</style>