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
          <router-link :to="`/edit/${wf.id}`" class="action-link">Edit</router-link>

          <!-- Delete Button -->
          <button @click="deleteWorkflow(wf.id)" class="delete-btn" title="Delete Workflow">
            <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none"
              stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <polyline points="3 6 5 6 21 6"></polyline>
              <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
            </svg>
          </button>
        </div>
      </div>
    </div>

    <Pagination v-model:limit="limit" v-model:offset="offset" />
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue';
import { apiClient } from '../api/client';
import type { Workflow } from '../types';
import StatusIcon from '../components/common/StatusIcon.vue';
import Pagination from '../components/common/Pagination.vue';
import { useSocket } from '../composables/useSocket';

const workflows = ref<Workflow[]>([]);
const limit = ref(10);
const offset = ref(0);
const { onEvent } = useSocket();

const fetchWorkflows = async () => {
  const res = await apiClient.get<Workflow[]>(`/workflows?limit=${limit.value}&offset=${offset.value}`);
  workflows.value = res.data || []; // Handle null response for empty DB
};

const deleteWorkflow = async (id: string) => {
  if (!confirm("Are you sure you want to delete this workflow?")) return;

  try {
    await apiClient.delete(`/workflows/${id}`);
    // If deleting the last item on a page, go back one page
    if (workflows.value.length === 1 && offset.value > 0) {
      offset.value = Math.max(0, offset.value - limit.value);
    } else {
      fetchWorkflows();
    }
  } catch (e) {
    alert("Failed to delete workflow");
    console.error(e);
  }
};

// Listen for any workflow status change
onEvent('workflow:status', (e: any) => {
  const wf = workflows.value.find(w => w.id === e.workflowId);
  if (wf) {
    if (wf.currBuild) wf.currBuild.status = e.status;
    else fetchWorkflows();
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
  font-weight: 500;
}

.create-btn:hover {
  background: #2b63d6;
}

.wf-card {
  display: flex;
  align-items: center;
  background: #222;
  padding: 15px;
  margin-bottom: 10px;
  border-radius: 4px;
  border-left: 4px solid #444;
  transition: transform 0.1s;
}

.wf-card:hover {
  background: #2a2a2a;
}

.wf-status {
  margin-right: 15px;
}

.wf-info {
  flex: 1;
}

.wf-name {
  font-size: 1.1rem;
  color: #fff;
  text-decoration: none;
  font-weight: bold;
}

.wf-name:hover {
  text-decoration: underline;
}

.wf-meta {
  color: #888;
  font-size: 0.8rem;
  margin-top: 4px;
}

.wf-actions {
  display: flex;
  align-items: center;
  gap: 15px;
}

.action-link {
  color: #aaa;
  font-size: 0.9rem;
  text-decoration: none;
}

.action-link:hover {
  color: #fff;
}

.delete-btn {
  background: transparent;
  border: none;
  color: #666;
  cursor: pointer;
  padding: 4px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  transition: all 0.2s;
}

.delete-btn:hover {
  background: #3a1c1c;
  color: #e74c3c;
}
</style>
