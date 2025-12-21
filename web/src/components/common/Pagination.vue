<template>
  <div class="pagination-bar">
    <div class="limit-selector">
      <label>Show:</label>
      <select :value="limit" @change="$emit('update:limit', Number(($event.target as HTMLSelectElement).value))">
        <option v-for="opt in [5, 10, 15, 25, 50]" :key="opt" :value="opt">{{ opt }}</option>
      </select>
    </div>

    <div class="controls">
      <button @click="$emit('prev')" :disabled="offset === 0">Previous</button>
      <span class="page-info">Page {{ pageNum }}</span>
      <button @click="$emit('next')">Next</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';

const props = defineProps<{ limit: number; offset: number }>();
defineEmits(['update:limit', 'next', 'prev']);

const pageNum = computed(() => Math.floor(props.offset / props.limit) + 1);
</script>

<style scoped>
.pagination-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 0;
  border-top: 1px solid #333;
  margin-top: 1rem;
}

.limit-selector {
  color: #aaa;
}

.controls button {
  background: #333;
  color: #fff;
  border: none;
  padding: 5px 10px;
  cursor: pointer;
}

.controls button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.page-info {
  margin: 0 10px;
  color: #fff;
}
</style>
