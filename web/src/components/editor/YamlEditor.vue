<template>
  <div class="editor-page">
    <!-- Header Toolbar -->
    <header class="editor-header">
      <div class="left-controls">
        <router-link to="/" class="back-link">Back</router-link>
        <input v-model="name" placeholder="Workflow Name" class="name-input" />
      </div>
      <div class="right-controls">
        <span class="validation-status" :class="{ error: !isValid }">
          {{ error || 'Valid YAML' }}
        </span>
        <button @click="save" :disabled="!isValid || isSaving" class="save-btn">
          {{ isSaving ? 'Saving...' : 'Save Workflow' }}
        </button>
      </div>
    </header>

    <!-- Code Editor Area -->
    <main class="editor-body">
      <codemirror v-model="yamlContent" placeholder="Enter workflow configuration..."
        :style="{ height: '100%', width: '100%' }" :autofocus="true" :indent-with-tab="true" :tab-size="2"
        :extensions="extensions" @change="validate" />
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, shallowRef } from 'vue';
import { useRouter } from 'vue-router';
import yaml from 'js-yaml';
import { Codemirror } from 'vue-codemirror';
import { yaml as yamlLang } from '@codemirror/lang-yaml';
import { oneDark } from '@codemirror/theme-one-dark';
import { apiClient } from '../../api/client';

const props = defineProps<{ existingId?: string }>();
const router = useRouter();

// Editor Configuration
const extensions = shallowRef([yamlLang(), oneDark]);

// State
const name = ref('');
const yamlContent = ref(
  `jobs:
  - name: example-job
    run:
      image: alpine:latest
      scripts:
        - name: hello
          command: [echo, "Hello World"]`
);
const error = ref('');
const isValid = ref(true);
const isSaving = ref(false);

// Validation Logic
const validate = () => {
  try {
    if (!yamlContent.value.trim()) {
      throw new Error("Config cannot be empty");
    }
    yaml.load(yamlContent.value);
    error.value = '';
    isValid.value = true;
  } catch (e: any) {
    // Truncate long yaml error messages for the UI
    error.value = e.message.split('\n')[0];
    isValid.value = false;
  }
};

// Data Loading
const loadExisting = async () => {
  if (!props.existingId) return;

  try {
    const res = await apiClient.get(`/workflows/${props.existingId}`);
    name.value = res.data.name;
    // Convert JSON config back to YAML
    yamlContent.value = yaml.dump(res.data.config);
    validate();
  } catch (e) {
    console.error("Failed to load workflow", e);
  }
};

// Save Action
const save = async () => {
  if (!name.value) {
    alert("Please enter a workflow name");
    return;
  }

  try {
    isSaving.value = true;
    const config = yaml.load(yamlContent.value);
    const payload = { name: name.value, config };

    if (props.existingId) {
      await apiClient.put(`/workflows/${props.existingId}`, payload);
    } else {
      await apiClient.post('/workflows', payload);
    }
    router.push('/');
  } catch (e) {
    alert("Failed to save workflow");
    console.error(e);
  } finally {
    isSaving.value = false;
  }
};

onMounted(loadExisting);
</script>

<style scoped>
.editor-page {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background-color: #1e1e1e;
  /* Matches OneDark bg */
}

/* Header Styles */
.editor-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 20px;
  background-color: #2c2c2c;
  border-bottom: 1px solid #111;
  height: 60px;
  flex-shrink: 0;
}

.left-controls {
  display: flex;
  align-items: center;
  gap: 15px;
  flex: 1;
}

.back-link {
  color: #aaa;
  text-decoration: none;
  font-weight: bold;
  font-size: 0.9rem;
}

.back-link:hover {
  color: #fff;
}

.name-input {
  background: #1a1a1a;
  border: 1px solid #444;
  color: #fff;
  padding: 8px 12px;
  border-radius: 4px;
  font-size: 1rem;
  width: 300px;
}

.name-input:focus {
  border-color: #3d7cf9;
  outline: none;
}

.right-controls {
  display: flex;
  align-items: center;
  gap: 20px;
}

.validation-status {
  font-family: monospace;
  font-size: 0.85rem;
  color: #3cb371;
  /* Green by default */
}

.validation-status.error {
  color: #e74c3c;
}

.save-btn {
  background-color: #3d7cf9;
  color: white;
  border: none;
  padding: 8px 24px;
  border-radius: 4px;
  cursor: pointer;
  font-weight: 600;
  transition: background 0.2s;
}

.save-btn:disabled {
  background-color: #555;
  cursor: not-allowed;
  opacity: 0.7;
}

.save-btn:hover:not(:disabled) {
  background-color: #2b63d6;
}

/* Main Editor Body */
.editor-body {
  flex: 1;
  overflow: hidden;
  /* Let CodeMirror handle scrolling */
  position: relative;
}

/* Deep selector to customize CodeMirror font/size if needed */
:deep(.cm-editor) {
  font-size: 14px;
  font-family: 'Fira Code', 'Consolas', monospace;
}

:deep(.cm-scroller) {
  overflow: auto;
}
</style>
