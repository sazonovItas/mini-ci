<template>
  <div class="log-window">
    <div class="log-scroll" ref="scrollContainer">
      <div v-for="(l, i) in logs" :key="i" class="log-entry">
        <span class="ts">{{ formatTime(l.time) }}</span>
        <span class="msg">{{ l.msg }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, onMounted } from 'vue';
import { format } from 'date-fns';
import { apiClient } from '../../api/client';
import { useSocket } from '../../composables/useSocket';
import type { LogMessage } from '../../types';

const props = defineProps<{ taskId: string }>();
const logs = ref<LogMessage[]>([]);
const scrollContainer = ref<HTMLDivElement | null>(null);
const { onEvent } = useSocket();

// Track the current listener to unsubscribe when task changes
let unsubscribe: (() => void) | null = null;

const formatTime = (t: string) => {
  try { return format(new Date(t), 'HH:mm:ss'); }
  catch { return t; }
};

const scrollToBottom = () => {
  nextTick(() => {
    if (scrollContainer.value) {
      scrollContainer.value.scrollTop = scrollContainer.value.scrollHeight;
    }
  });
};

const loadLogs = async (id: string) => {
  // 1. Unsubscribe from previous task logs
  if (unsubscribe) {
    unsubscribe();
    unsubscribe = null;
  }

  logs.value = [];

  // 2. Fetch History
  try {
    const res = await apiClient.get(`/tasks/${id}/logs?limit=5000`);
    logs.value = res.data || [];
    scrollToBottom();
  } catch (e) {
    console.error("Log fetch failed", e);
  }

  // 3. Subscribe to new task logs
  // Backend emits: task:{id}:logs
  unsubscribe = onEvent(`task:${id}:logs`, (data: any) => {
    // Backend payload: { id: "...", messages: [...] }
    if (data.messages) {
      logs.value.push(...data.messages);
      scrollToBottom();
    }
  });
};

onMounted(() => {
  if (props.taskId) loadLogs(props.taskId);
});

watch(() => props.taskId, (newId) => {
  if (newId) loadLogs(newId);
});
</script>

<style scoped>
.log-window {
  background: #101010;
  height: 100%;
  border-radius: 4px;
  font-family: 'Consolas', monospace;
  font-size: 13px;
}

.log-scroll {
  height: 100%;
  overflow-y: auto;
  padding: 10px;
}

.log-entry {
  display: flex;
  gap: 10px;
  line-height: 1.4;
  color: #ddd;
}

.ts {
  color: #666;
  user-select: none;
  flex-shrink: 0;
}

.msg {
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
