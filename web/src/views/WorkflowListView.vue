<template>
  <div class="page">
    <div class="header">
      <h1>Workflows</h1>
      <router-link to="/create" class="create-btn">+ Create</router-link>
    </div>

    <div class="wf-list">
      <div v-for="wf in workflows" :key="wf.id" class="wf-card">
        <div class="wf-status">
          <StatusIcon :status="wf.currBuild?.status || 'created'" />
        </div>
        <div class="wf-info">
          <router-link :to="`/workflows/${wf.id}`" class="wf-name">{{ wf.name }}</router-link>
          <div class="wf-meta">
            ID: {{ wf.id }} | Last Build: {{ wf.currBuild ? '#' + wf.currBuild.id.slice(0, 8) : 'None' }}
          </div>
        </div>
        <div class="wf-actions">
          <router-link :to="`/edit/${wf.id}`">Edit</router-link>
        </div>
      </div>
    </div>

    <Pagination :limit="limit" :offset="offset" @update:limit="limit = $event" @next="offset += limit"
      @prev="offset = Math.max(0, offset - limit)" />
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue';
import { apiClient } from '../api/client';
import { Workflow } from '../types';
import StatusIcon from '../components/common/StatusIcon.vue';
import Pagination from '../components/common/Pagination.vue';
import { useSocket } from '../composables/useSocket';

const workflows = ref<Workflow[]>([]);
const limit = ref(10);
const offset = ref(0);
const { onEvent } = useSocket();

const fetchWorkflows = async () => {
  const res = await apiClient.get<Workflow[]>(`/workflows?limit=${limit.value}&offset=${offset.value}`);
  workflows.value = res.data;
};

// Listen for any workflow status change
onEvent('workflow:status', (e: any) => {
  const wf = workflows.value.find(w => w.id === e.workflowId); // Ensure backend sends workflowId in event
  if (wf) {
    if (wf.currBuild) wf.currBuild.status = e.status;
    else fetchWorkflows(); // New build created
  }
});

watch([limit, offset], fetchWorkflows);
onMounted(fetchWorkflows);
</script>

<style scoped>
.page {
  max-width: 900px;
  margin: 0 auto;
  padding: 20px;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.create-btn {
  background: #3d7cf9;
  color: white;
  padding: 8px 16px;
  text-decoration: none;
  border-radius: 4px;
}

.wf-card {
  display: flex;
  align-items: center;
  background: #222;
  padding: 15px;
  margin-bottom: 10px;
  border-radius: 4px;
  border-left: 4px solid #444;
}

.wf-status {
  margin-right: 15px;
}

.wf-info {
  flex: 1;
}

.wf-name {
  font-size: 1.2rem;
  color: #fff;
  text-decoration: none;
  font-weight: bold;
}

.wf-meta {
  color: #888;
  font-size: 0.8rem;
  margin-top: 4px;
}

.wf-actions a {
  color: #aaa;
  font-size: 0.9rem;
}
</style>
