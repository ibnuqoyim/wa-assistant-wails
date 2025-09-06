<template>
<div class="composer">
<button class="icon-btn" title="Emoji"><svg viewBox="0 0 24 24" class="ico"><path d="M12 2a10 10 0 100 20 10 10 0 000-20zm-4 8a1.5 1.5 0 110-3 1.5 1.5 0 010 3zm8 0a1.5 1.5 0 110-3 1.5 1.5 0 010 3zM7.5 14a4.5 4.5 0 009 0H7.5z"/></svg></button>
<button class="icon-btn" title="Attach"><svg viewBox="0 0 24 24" class="ico"><path d="M16.5 6.5l-7.8 7.8a3 3 0 104.2 4.2l8-8a5 5 0 10-7.1-7.1l-8.6 8.6" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg></button>
<input :value="modelValue" @keydown.enter.exact.prevent="$emit('send')" @input="$emit('update:modelValue', ($event.target as HTMLInputElement).value)" class="input" type="text" placeholder="Type a message"/>
<button class="icon-btn" title="Mic"><svg viewBox="0 0 24 24" class="ico"><path d="M12 14a3 3 0 003-3V6a3 3 0 10-6 0v5a3 3 0 003 3zm5-3a5 5 0 01-10 0M12 16v4m-3 0h6" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg></button>
<button class="send-btn" :disabled="!modelValue.trim()" @click="$emit('send')" title="Send"><svg viewBox="0 0 24 24" class="ico"><path d="M2 21l20-9L2 3v7l15 2-15 2v7z"/></svg></button>
</div>
</template>
<script setup lang="ts">
defineProps<{ modelValue:string }>()
</script>
<style scoped>
.composer{display:flex;align-items:center;gap:8px;padding:10px 12px;background:var(--panel);border-top:1px solid var(--panel-3)}
.input{flex:1;background:var(--panel-2);border:none;outline:none;color:var(--text);padding:10px 12px;border-radius:10px;font-size:14px}
.icon-btn{width:36px;height:36px;border:none;background:transparent;color:var(--muted);border-radius:10px;display:grid;place-items:center;cursor:pointer}
.icon-btn:hover{background:var(--hover);color:var(--text)}
.send-btn{width:40px;height:40px;border-radius:10px;border:none;background:var(--brand);color:#003d32;display:grid;place-items:center;cursor:pointer;font-weight:700}
.send-btn:disabled{opacity:.5;cursor:default}
.ico{width:22px;height:22px;fill:currentColor}
</style>