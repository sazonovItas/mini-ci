<template>
  <div class="layout">
    <!-- Left Sidebar: Builds List -->
    <aside class="sidebar">
      <!-- NEW: Back Navigation -->
      <div class="sidebar-nav">
        <router-link to="/" class="back-link">
          <span class="icon"></span>Workflows
        </router-link>
      </div>

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
            <div v-for="(job, idx) in sortedJobs" :key="job.id" class="step-wrapper">

              <!-- Job Node -->
              <div class="job-node" :class="{ active: selectedJobId === job.id, [job.status]: true }"
                @click="selectJob(job.id)">
                <div class="node-header">
                  <span class="job-name">{{ job.name }}</span>
                  <StatusIcon :status="job.status" />
                </div>
              </div>

              <div v-if="idx < sortedJobs.length - 1" class="connector-line"></div>
            </div>
          </div>
        </div>

        <!-- Right/Bottom: Job Details (Tasks + Logs) -->
        <div class="inspector" v-if="selectedJob">
          <div class="task-header">
            <h4>{{ selectedJob.name }}</h4>
            <span class="sub-text">Job Tasks</span>
          </div>

          <div class="task-list">
            <div v-for="t in sortedTasks" :key="t.id" class="task-row" :class="{ active: selectedTaskId === t.id }"
              @click="selectTask(t.id)">
              <StatusIcon :status="t.status" />
              <span class="task-name">{{ t.name }}</span>
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
import { sortByPlan } from '../utils/plan';
import Status, { type Build, type Job, type Task } from '../types';
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

// Cleanups for manual subscription management
let unsubJobStatus: (() => void) | null = null;
let unsubTaskStatus: (() => void) | null = null;

// --- Computed ---

const currentBuild = computed(() => builds.value.find(b => b.id === selectedBuildId.value));
const selectedJob = computed(() => jobs.value.find(j => j.id === selectedJobId.value));

const isRunnable = (s: Status) => s === Status.Pending || s === Status.Started;

const sortedJobs = computed(() => {
  if (!currentBuild.value) return [];
  return sortByPlan(jobs.value, currentBuild.value.plan);
});

const sortedTasks = computed(() => {
  if (!selectedJob.value) return [];
  return sortByPlan(tasks.value, selectedJob.value.plan);
});

// --- Actions ---

const loadBuilds = async () => {
  try {
    const res = await apiClient.get(`/workflows/${workflowId}/builds`);
    builds.value = res.data.reverse();

    // Select latest build if none selected
    if (!selectedBuildId.value && builds.value.length) {
      selectBuild(builds.value[0]!.id);
    }
  } catch (e) { console.error("Load builds error", e); }
};

const selectBuild = async (id: string) => {
  // 1. Reset state
  selectedBuildId.value = id;
  selectedJobId.value = null;
  selectedTaskId.value = null;
  tasks.value = [];

  // 2. Clear old job listener
  if (unsubJobStatus) { unsubJobStatus(); unsubJobStatus = null; }

  // 3. Subscribe NEW listener (for jobs in this build)
  console.log(`🔌 Subscribing to jobs for build: ${id}`);
  unsubJobStatus = onEvent(`build:${id}:job:status`, (payload: any) => {
    const job = jobs.value.find(j => j.id === payload.id);
    if (job) {
      job.status = payload.status;
    }
  });

  // 4. Fetch Data
  try {
    const res = await apiClient.get(`/builds/${id}/jobs`);
    jobs.value = res.data;

    // 5. Auto-select active job
    const ordered = sortByPlan(jobs.value, currentBuild.value?.plan);
    if (ordered.length > 0) {
      const activeJob = ordered.find(j => j.status === Status.Started || j.status === Status.Pending);
      selectJob(activeJob ? activeJob.id : ordered[0]!.id);
    }
  } catch (e) { console.error("Load jobs error", e); }
};

const selectJob = async (id: string) => {
  // 1. Reset state
  selectedJobId.value = id;
  selectedTaskId.value = null;

  // 2. Clear old task listener
  if (unsubTaskStatus) { unsubTaskStatus(); unsubTaskStatus = null; }

  // 3. Subscribe NEW listener (for tasks in this job)
  console.log(`🔌 Subscribing to tasks for job: ${id}`);
  unsubTaskStatus = onEvent(`job:${id}:task:status`, (payload: any) => {
    const t = tasks.value.find(x => x.id === payload.id);
    if (t) {
      t.status = payload.status;
    }
  });

  // 4. Fetch Data
  try {
    const res = await apiClient.get(`/jobs/${id}/tasks`);
    tasks.value = res.data;

    // 5. Auto-select first task
    const job = jobs.value.find(j => j.id === id);
    if (job) {
      const orderedTasks = sortByPlan(tasks.value, job.plan);
      if (orderedTasks.length > 0) {
        selectTask(orderedTasks[0]!.id);
      }
    }
  } catch (e) { console.error("Load tasks error", e); }
};

const selectTask = (id: string) => {
  selectedTaskId.value = id;
};

const runBuild = async () => {
  isTriggering.value = true;
  try {
    const res = await apiClient.post(`/workflows/${workflowId}/builds`);
    builds.value.unshift(res.data);
    selectBuild(res.data.id);
  } catch (e) {
    alert("Failed to start build");
  } finally {
    isTriggering.value = false;
  }
};

const abortBuild = async () => {
  if (currentBuild.value) {
    await apiClient.post(`/builds/${currentBuild.value.id}/abort`);
  }
};

// Lifecycle
onMounted(() => {
  loadBuilds();

  // Listen for Build Status changes (Global for this workflow)
  onEvent(`workflow:${workflowId}:build:status`, (payload: any) => {
    const b = builds.value.find(x => x.id === payload.id);
    if (b) {
      b.status = payload.status;
    } else {
      loadBuilds(); // New build appeared
    }
  });
});
</script>

<style scoped>
.layout {
  display: flex;
  height: 100vh;
  background: #151515;
  color: #eee;
  overflow: hidden;
}

/* Sidebar */
.sidebar {
  width: 240px;
  background: #1e1e1e;
  border-right: 1px solid #333;
  display: flex;
  flex-direction: column;
}

/* Navigation Back Link */
.sidebar-nav {
  padding: 12px 15px;
  border-bottom: 1px solid #333;
  background-color: #252525;
}

.back-link {
  color: #aaa;
  text-decoration: none;
  font-size: 0.9rem;
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 5px;
  transition: color 0.2s;
}

.back-link:hover {
  color: #fff;
}

.back-link .icon {
  font-size: 1.1rem;
  line-height: 1;
}

.sidebar-head {
  padding: 15px;
  border-bottom: 1px solid #333;
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: #252525;
}

.run-btn {
  background: #3d7cf9;
  color: white;
  border: none;
  padding: 6px 12px;
  cursor: pointer;
  border-radius: 4px;
  font-weight: 600;
  font-size: 0.9rem;
}

.run-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.build-list {
  overflow-y: auto;
  flex: 1;
}

.build-item {
  padding: 12px 15px;
  display: flex;
  align-items: center;
  gap: 12px;
  cursor: pointer;
  border-bottom: 1px solid #2a2a2a;
  transition: background 0.2s;
}

.build-item:hover {
  background: #2a2a2a;
}

.build-item.active {
  background: #2a2a2a;
  border-left: 3px solid #3d7cf9;
  padding-left: 12px;
}

.b-info {
  font-family: monospace;
  font-size: 0.95rem;
  color: #ccc;
}

/* Content */
.content {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.top-bar {
  padding: 15px 25px;
  background: #1e1e1e;
  border-bottom: 1px solid #333;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.status-badge {
  padding: 5px 10px;
  border-radius: 4px;
  font-size: 0.8rem;
  text-transform: uppercase;
  font-weight: bold;
  background: #333;
  letter-spacing: 0.5px;
}

.status-badge.succeeded {
  background: #2e7d32;
  color: #fff;
}

.status-badge.failed,
.status-badge.errored {
  background: #c62828;
  color: white;
}

.status-badge.started,
.status-badge.pending {
  background: #f9a825;
  color: #000;
}

.status-badge.aborted {
  background: #6a1b9a;
  color: white;
}

.abort-btn {
  background: #c62828;
  color: white;
  border: none;
  padding: 6px 16px;
  margin-left: 15px;
  cursor: pointer;
  border-radius: 4px;
  font-weight: 600;
}

/* Workspace */
.workspace {
  flex: 1;
  display: flex;
  overflow: hidden;
  background: #121212;
}

/* Graph Area */
.graph-container {
  flex: 2;
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 40px;
  overflow: auto;
  background-image: radial-gradient(#222 1px, transparent 1px);
  background-size: 20px 20px;
}

.pipeline {
  display: flex;
  align-items: center;
}

.step-wrapper {
  display: flex;
  align-items: center;
}

.job-node {
  width: 160px;
  height: 70px;
  background: #1e1e1e;
  border: 2px solid #444;
  border-radius: 6px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s;
  position: relative;
  z-index: 2;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.3);
}

.job-node:hover {
  transform: translateY(-2px);
  border-color: #666;
}

.job-node.active {
  border-color: #3d7cf9;
  box-shadow: 0 0 0 2px rgba(61, 124, 249, 0.3);
  background: #252525;
}

.job-node.succeeded {
  border-color: #2e7d32;
}

.job-node.failed {
  border-color: #c62828;
}

.job-node.started {
  border-color: #f9a825;
}

.node-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 15px;
  width: 100%;
}

.job-name {
  font-weight: 500;
  font-size: 0.95rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* Connector Line */
.connector-line {
  width: 50px;
  height: 2px;
  background: #444;
  position: relative;
  z-index: 1;
}

/* Inspector / Logs */
.inspector {
  flex: 1;
  min-width: 450px;
  max-width: 600px;
  background: #1e1e1e;
  border-left: 1px solid #333;
  display: flex;
  flex-direction: column;
  box-shadow: -5px 0 15px rgba(0, 0, 0, 0.2);
}

.task-header {
  padding: 15px 20px;
  border-bottom: 1px solid #333;
  background: #252525;
}

.task-header h4 {
  margin: 0;
  font-size: 1.1rem;
}

.sub-text {
  font-size: 0.8rem;
  color: #888;
}

.task-list {
  padding: 0;
  overflow-y: auto;
  max-height: 35vh;
  background: #1a1a1a;
}

.task-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 20px;
  cursor: pointer;
  border-bottom: 1px solid #222;
  transition: background 0.1s;
}

.task-row:hover {
  background: #252525;
}

.task-row.active {
  background: #2c2c2c;
  border-left: 3px solid #3d7cf9;
  padding-left: 17px;
}

.task-name {
  font-size: 0.9rem;
}

.log-pane {
  flex: 1;
  overflow: hidden;
  background: #000;
  border-top: 1px solid #333;
}

.empty-state {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #555;
  background: #151515;
  border-top: 1px solid #333;
}
</style>
