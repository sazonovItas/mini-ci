<template>
  <div class="layout">
    <!-- Left Sidebar: Builds List -->
    <aside class="sidebar">
      <div class="sidebar-head">
        <h3>Builds</h3>
        <button @click="runBuild" :disabled="isTriggering" class="run-btn">
          {{ isTriggering ? '...' : '▶ Run' }}
        </button>
      </div>
      <div class="build-list">
        <div v-for="b in builds" :key="b.id" class="build-item" :class="{ active: selectedBuildId === b.id }"
          @click="selectBuild(b.id)">
          <StatusIcon :status="b.status" :key="b.status" />
          <div class="b-info">
            <span class="b-id">#{{ b.id.slice(0, 8) }}</span>
          </div>
        </div>
      </div>
    </aside>

    <!-- Center & Right: Content -->
    <main class="content" v-if="currentBuild">
      <!-- Build Header -->
      <div class="top-bar">
        <h2>Build #{{ currentBuild.id.slice(0, 8) }}</h2>
        <div class="actions">
          <span class="status-badge" :class="currentBuild.status">{{ currentBuild.status }}</span>
          <button v-if="isRunnable(currentBuild.status)" @click="abortBuild" class="abort-btn">Abort</button>
        </div>
      </div>

      <div class="workspace">
        <!-- Center: Pipeline Graph (Jobs) -->
        <div class="graph-container">
          <div class="pipeline">
            <div v-for="(job, idx) in jobs" :key="job.id" class="step">
              <div class="job-node" :class="{ active: selectedJobId === job.id, [job.status]: true }"
                @click="selectJob(job.id)">
                <span class="job-name">{{ job.name }}</span>
                <StatusIcon :status="job.status" />
              </div>
              <div v-if="idx < jobs.length - 1" class="connector"></div>
            </div>
          </div>
        </div>

        <!-- Right/Bottom: Job Details (Tasks + Logs) -->
        <div class="inspector" v-if="selectedJob">
          <div class="task-list">
            <h4>Tasks in {{ selectedJob.name }}</h4>
            <div v-for="t in tasks" :key="t.id" class="task-row" :class="{ active: selectedTaskId === t.id }"
              @click="selectTask(t.id)">
              <StatusIcon :status="t.status" />
              <span>{{ t.name }}</span>
            </div>
          </div>

          <div class="log-pane" v-if="selectedTaskId">
            <LogViewer :task-id="selectedTaskId" />
          </div>
          <div class="empty-state" v-else>Select a task to view logs</div>
        </div>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { useRoute } from 'vue-router';
import { apiClient } from '../api/client';
import { useSocket } from '../composables/useSocket';
import { type Build, type Job, type Task, Status } from '../types';
import StatusIcon from '../components/common/StatusIcon.vue';
import LogViewer from '../components/logs/LogViewer.vue';

const route = useRoute();
const workflowId = route.params.id as string;
const { onEvent } = useSocket();

// Data State
const builds = ref<Build[]>([]);
const selectedBuildId = ref<string | null>(null);
const jobs = ref<Job[]>([]);
const selectedJobId = ref<string | null>(null);
const tasks = ref<Task[]>([]);
const selectedTaskId = ref<string | null>(null);
const isTriggering = ref(false);

const currentBuild = computed(() => builds.value.find(b => b.id === selectedBuildId.value));
const selectedJob = computed(() => jobs.value.find(j => j.id === selectedJobId.value));

const isRunnable = (s: Status) => s === Status.Pending || s === Status.Started;

// Subscription Handles (to unsubscribe when switching)
let unsubJobStatus: (() => void) | null = null;
let unsubTaskStatus: (() => void) | null = null;

// --- 1. Load Initial Builds ---
const loadBuilds = async () => {
  const res = await apiClient.get(`/workflows/${workflowId}/builds`);
  builds.value = res.data.reverse();

  if (!selectedBuildId.value && builds.value.length) {
    selectBuild(builds.value[0].id);
  }
};

// --- 2. Select Build (Switch Job Status Listeners) ---
const selectBuild = async (id: string) => {
  selectedBuildId.value = id;
  selectedJobId.value = null;
  selectedTaskId.value = null;
  tasks.value = [];

  // A. Fetch Jobs
  const res = await apiClient.get(`/builds/${id}/jobs`);
  jobs.value = res.data;

  // B. Switch Socket Listener for Jobs
  if (unsubJobStatus) {
    unsubJobStatus();
    unsubJobStatus = null;
  }

  // Subscribe to: build:{id}:job:status
  // Payload: { id: "job-id", status: "...", ... }
  unsubJobStatus = onEvent(`build:${id}:job:status`, (payload: any) => {
    const job = jobs.value.find(j => j.id === payload.id);
    if (job) job.status = payload.status;
  });

  // Auto-select first job
  if (jobs.value.length) {
    selectJob(jobs.value[0].id);
  }
};

// --- 3. Select Job (Switch Task Status Listeners) ---
const selectJob = async (id: string) => {
  selectedJobId.value = id;
  selectedTaskId.value = null;

  // A. Fetch Tasks
  const res = await apiClient.get(`/jobs/${id}/tasks`);
  tasks.value = res.data;

  // B. Switch Socket Listener for Tasks
  if (unsubTaskStatus) {
    unsubTaskStatus();
    unsubTaskStatus = null;
  }

  // Subscribe to: job:{id}:task:status
  // Payload: { id: "task-id", status: "...", ... }
  unsubTaskStatus = onEvent(`job:${id}:task:status`, (payload: any) => {
    const task = tasks.value.find(t => t.id === payload.id);
    if (task) task.status = payload.status;
  });
};

const selectTask = (id: string) => {
  selectedTaskId.value = id;
};

// --- Actions ---

const runBuild = async () => {
  isTriggering.value = true;
  try {
    const res = await apiClient.post(`/workflows/${workflowId}/builds`);
    builds.value.unshift(res.data);
    selectBuild(res.data.id);
  } catch (e) {
    alert("Cannot start build (maybe one is already running?)");
  } finally {
    isTriggering.value = false;
  }
};

const abortBuild = async () => {
  if (currentBuild.value) {
    await apiClient.post(`/builds/${currentBuild.value.id}/abort`);
  }
};

// --- Lifecycle ---

onMounted(() => {
  loadBuilds();

  // Global Listener for Build Statuses on this Workflow
  // Payload: { id: "build-id", status: "...", ... }
  onEvent(`workflow:${workflowId}:build:status`, (payload: any) => {
    const b = builds.value.find(x => x.id === payload.id);
    if (b) {
      b.status = payload.status;
    } else {
      // New build created by someone else
      loadBuilds();
    }
  });
});
</script>

<style scoped>
.layout {
  display: flex;
  height: 100vh;
  background: #1a1a1a;
  color: #eee;
  overflow: hidden;
}

/* Sidebar */
.sidebar {
  width: 220px;
  background: #222;
  border-right: 1px solid #333;
  display: flex;
  flex-direction: column;
}

.sidebar-head {
  padding: 15px;
  border-bottom: 1px solid #333;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.run-btn {
  background: #3d7cf9;
  color: white;
  border: none;
  padding: 5px 10px;
  cursor: pointer;
  border-radius: 3px;
}

.build-list {
  overflow-y: auto;
  flex: 1;
}

.build-item {
  padding: 12px 15px;
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  border-bottom: 1px solid #2a2a2a;
}

.build-item:hover {
  background: #2a2a2a;
}

.build-item.active {
  background: #333;
  border-left: 3px solid #3d7cf9;
}

/* Content */
.content {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.top-bar {
  padding: 15px 20px;
  background: #222;
  border-bottom: 1px solid #333;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.status-badge {
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 0.8rem;
  text-transform: uppercase;
  font-weight: bold;
  background: #444;
}

.status-badge.succeeded {
  background: #3cb371;
  color: #000;
}

.status-badge.failed {
  background: #e74c3c;
  color: white;
}

.abort-btn {
  background: #8e44ad;
  color: white;
  border: none;
  padding: 5px 15px;
  margin-left: 10px;
  cursor: pointer;
}

/* Workspace */
.workspace {
  flex: 1;
  display: flex;
  overflow: hidden;
}

/* Graph */
.graph-container {
  flex: 2;
  background: #161616;
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 20px;
  overflow: auto;
}

.pipeline {
  display: flex;
  align-items: center;
}

.job-node {
  width: 140px;
  height: 80px;
  background: #222;
  border: 2px solid #444;
  border-radius: 6px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  transition: 0.2s;
}

.job-node:hover {
  background: #2a2a2a;
}

.job-node.active {
  border-color: #3d7cf9;
  background: #2a2a2a;
}

.job-node.succeeded {
  border-color: #3cb371;
}

.job-node.failed {
  border-color: #e74c3c;
}

.connector {
  width: 40px;
  height: 2px;
  background: #444;
}

/* Inspector */
.inspector {
  flex: 1;
  min-width: 400px;
  background: #1e1e1e;
  border-left: 1px solid #333;
  display: flex;
  flex-direction: column;
}

.task-list {
  padding: 15px;
  border-bottom: 1px solid #333;
  max-height: 40%;
  overflow-y: auto;
}

.task-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px;
  cursor: pointer;
  border-radius: 4px;
}

.task-row:hover {
  background: #2a2a2a;
}

.task-row.active {
  background: #333;
}

.log-pane {
  flex: 1;
  overflow: hidden;
}

.empty-state {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #555;
}
</style>
