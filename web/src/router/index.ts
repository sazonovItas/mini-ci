import { createRouter, createWebHistory } from 'vue-router';
import WorkflowListView from '../views/WorkflowListView.vue';
import WorkflowDetailView from '../views/WorkflowDetailView.vue';
import YamlEditor from '../components/editor/YamlEditor.vue';

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: WorkflowListView },
    { path: '/workflows/:id', component: WorkflowDetailView },
    { path: '/create', component: YamlEditor },
    { path: '/edit/:existingId', component: YamlEditor, props: true },
  ]
});

export default router;
