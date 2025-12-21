<template>
  <div :class="['status-icon', status]" :title="status">
    <div v-if="isRunning" class="pulse-ring"></div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { Status } from '../../types';

const props = defineProps<{ status: Status }>();

const isRunning = computed(() =>
  props.status === Status.Pending || props.status === Status.Started
);
</script>

<style scoped>
.status-icon {
  width: 18px;
  height: 18px;
  border-radius: 50%;
  position: relative;
  background-color: #555;
  flex-shrink: 0;
}

/* Concourse-ish Colors */
.succeeded {
  background-color: #3cb371;
}

.failed,
.errored {
  background-color: #e74c3c;
}

.aborted {
  background-color: #8e44ad;
}

.started,
.pending {
  background-color: #f1c40f;
}

.created {
  background-color: #95a5a6;
}

.pulse-ring {
  position: absolute;
  top: -4px;
  left: -4px;
  right: -4px;
  bottom: -4px;
  border: 2px solid #f1c40f;
  border-radius: 50%;
  animation: pulse 1.5s infinite;
  opacity: 0;
}

@keyframes pulse {
  0% {
    transform: scale(0.8);
    opacity: 0.8;
  }

  100% {
    transform: scale(1.5);
    opacity: 0;
  }
}
</style>
