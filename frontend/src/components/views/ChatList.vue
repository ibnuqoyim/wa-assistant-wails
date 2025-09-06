<template>
<section class="chatlist">
<header class="chatlist-header">
<div class="profile">
<div class="avatar">MQ</div>
<div class="me"><div class="name">M ibnu Qoyim</div><div class="sub">Online</div></div>
</div>
<div class="tools">
<button class="icon-btn" title="New chat"><svg viewBox="0 0 24 24" class="ico"><path d="M12 5v14M5 12h14" stroke="currentColor" stroke-width="2" fill="none" stroke-linecap="round"/></svg></button>
<button class="icon-btn" title="Menu"><svg viewBox="0 0 24 24" class="ico"><circle cx="5" cy="12" r="1.5"/><circle cx="12" cy="12" r="1.5"/><circle cx="19" cy="12" r="1.5"/></svg></button>
</div>
</header>


<div class="search">
<div class="search-box">
<svg viewBox="0 0 24 24" class="ico"><path d="M10 18a8 8 0 100-16 8 8 0 000 16zm11 3l-6-6" stroke="currentColor" stroke-width="2" fill="none" stroke-linecap="round"/></svg>
<input :value="query" @input="$emit('update:query', ($event.target as HTMLInputElement).value)" placeholder="Search or start new chat"/>
</div>
</div>


<div class="chat-items">
<article v-for="c in chats" :key="c.id" class="chat-item" :class="{active:c.id===selectedId}" @click="$emit('select', c.id)">
<div class="avatar" :style="{ background: pickColor(c.name) }">{{ initials(c.name) }}</div>
<div class="ci-main">
<div class="top"><div class="title">{{ c.name }}</div><div class="time">{{ c.time }}</div></div>
<div class="bottom"><div class="msg">{{ c.last }}</div><div class="meta"><span v-if="c.pinned" class="pin">📌</span><span v-if="c.unread" class="badge">{{ c.unread }}</span></div></div>
</div>
</article>
</div>
</section>
</template>
<script setup lang="ts">
import type { ChatItem } from '@/composables/useChatData'
import { initials, pickColor } from '@/utils/text'


defineProps<{ chats:ChatItem[]; selectedId:number; query:string }>()
</script>
<style scoped>
.chatlist{
  background: var(--panel);
  border-right: 1px solid var(--panel-3);
  display: grid;
  grid-template-rows: auto auto 1fr;
  /* pastikan tidak ada bg lain yang nempel */
  background-image: none !important;
}

.chatlist-header{
  display:flex;align-items:center;justify-content:space-between;
  padding:10px 12px;border-bottom:1px solid var(--panel-3)
}

.profile{display:flex;align-items:center;gap:10px}
.avatar{width:40px;height:40px;border-radius:50%;background:#4c7a6f;
  display:grid;place-items:center;color:#fff;font-weight:700}

.me .name{font-weight:600}
.me .sub{color:var(--muted);font-size:12px;margin-top:2px}

.tools{display:flex;gap:6px}
.icon-btn{width:36px;height:36px;border:none;background:transparent;
  color:var(--muted);border-radius:10px;display:grid;place-items:center;cursor:pointer}
.icon-btn:hover{background:var(--hover);color:var(--text)}
.ico{width:22px;height:22px;fill:currentColor}

/* --- SEARCH --------------------------------------------------------------- */
.search{
  padding:8px 10px;border-bottom:1px solid var(--panel-3);
  /* paksa hilangkan bg dari style lain */
  background: var(--panel) !important;
  background-image: none !important;
}
.search-box{
  display:flex;align-items:center;gap:8px;
  background: var(--panel-2);
  border-radius:8px;padding:8px 10px;color:var(--muted);
  /* jaga-jaga */
  background-image:none !important;
}
.search-box input{
  flex:1;background:transparent;border:none;outline:none;
  color:var(--text);font-size:14px;
}

/* --- CHAT LIST ------------------------------------------------------------ */
.chat-items{overflow:auto}

.chat-item{
  display:flex;gap:12px;padding:10px 12px;
  border-bottom:1px solid var(--panel-3);cursor:pointer
}
.chat-item:hover,.chat-item.active{background:var(--hover)}
.ci-main{flex:1;min-width:0}
.top{display:flex;justify-content:space-between;align-items:center}
.title{font-weight:600;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.time{color:var(--muted);font-size:12px;margin-left:8px}
.bottom{display:flex;justify-content:space-between;gap:10px;align-items:center}
.msg{color:var(--muted);font-size:13px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;max-width:220px}
.meta{display:flex;align-items:center;gap:6px}
.badge{background:var(--brand);color:#002d24;font-weight:700;padding:2px 6px;border-radius:10px;font-size:12px}
.pin{opacity:.8}
</style>