<template>
<div class="row" :class="{ mine: msg.mine }">
<div v-if="!msg.mine" class="avatar small" :style="{ background: pickColor(msg.author) }">{{ initials(msg.author) }}</div>
<div class="bubble" :class="msg.mine ? 'self' : 'other'">
<div class="text" v-html="msg.text"></div>
<div class="stamp">{{ msg.time }}</div>
</div>
</div>
</template>
<script setup lang="ts">
import type { Msg } from '@/composables/useChatData'
import { initials, pickColor } from '@/utils/text'


defineProps<{ msg: Msg }>()
</script>
<style scoped>
.row {
  display: flex;
  align-items: flex-end;
  gap: 6px;
  margin: 2px 0;
}

.row.mine {
  justify-content: flex-end;
}

.avatar.small {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: #4c7a6f;
  display: grid;
  place-items: center;
  color: #fff;
  font-weight: 700;
  font-size: 12px;
}

.bubble {
  max-width: 70%;
  padding: 6px 8px;
  border-radius: 8px;
  font-size: 14px;
  line-height: 1.4;
  box-shadow: 0 1px 0 rgba(0,0,0,.2);
  position: relative;
  white-space: pre-wrap; /* agar line break di pesan */
  word-wrap: break-word;
}

/* Bubble orang lain */
.bubble.other {
  background: var(--bubble-other);
  border-top-left-radius: 0; /* mirip WA */
}

/* Bubble kita */
.bubble.self {
  background: var(--bubble-self);
  border-top-right-radius: 0; /* mirip WA */
  color: #fff;
}

.text :deep(b) { font-weight: 700; }
.text :deep(a) { color: #8ab4f8; text-decoration: underline; }

/* Timestamp kecil di pojok kanan bawah bubble */
.stamp {
  text-align: right;
  color: var(--muted);
  font-size: 11px;
  margin-top: 2px;
}
</style>