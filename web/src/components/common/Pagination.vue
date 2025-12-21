<template>
  <div class="pagination-bar">
    <div class="limit-selector">
      <label>Show:</label>
      <select :value="limit" @change="onLimitChange(Number(($event.target as HTMLSelectElement).value))">
        <option v-for="opt in [5, 10, 15, 25, 50]" :key="opt" :value="opt">{{ opt }}</option>
      </select>
    </div>

    <!-- Range Display (e.g. 0 - 10) -->
    <div class="range-info">
      Showing {{ offset }} - {{ offset + limit }}
    </div>

    <div class="controls">
      <button @click="$emit('update:offset', Math.max(0, offset - limit))" :disabled="offset === 0">
        Previous
      </button>

      <span class="page-info">Page {{ pageNum }}</span>

      <button @click="$emit('update:offset', offset + limit)">
        Next
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';

const props = defineProps<{
  limit: number;
  offset: number;
}>();

const emit = defineEmits(['update:limit', 'update:offset']);

const pageNum = computed(() => Math.floor(props.offset / props.limit) + 1);

const onLimitChange = (newLimit: number) => {
  emit('update:limit', newLimit);
  // Reset to first page when changing page size to avoid index issues
  emit('update:offset', 0);
};
</script>

<style scoped>
.pagination-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 0;
  border-top: 1px solid #333;
  margin-top: 1rem;
  color: #ccc;
  font-size: 0.9rem;
}

.limit-selector {
  display: flex;
  align-items: center;
  gap: 10px;
}

.limit-selector select {
  background: #333;
  color: white;
  border: 1px solid #444;
  padding: 4px 8px;
  border-radius: 4px;
  cursor: pointer;
}

.range-info {
  color: #666;
  font-family: monospace;
}

.controls {
  display: flex;
  align-items: center;
  gap: 5px;
}

.controls button {
  background: #333;
  color: #fff;
  border: none;
  padding: 6px 12px;
  border-radius: 4px;
  cursor: pointer;
  transition: background 0.2s;
}

.controls button:hover:not(:disabled) {
  background: #444;
}

.controls button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.page-info {
  margin: 0 15px;
  color: #fff;
  font-family: monospace;
}
</style>
