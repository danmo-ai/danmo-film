import { createRouter, createWebHashHistory } from 'vue-router';
import type { RouteRecordRaw } from 'vue-router';

const routes: RouteRecordRaw[] = [
  { path: '/', redirect: '/workspace' },
  {
    path: '/workspace',
    component: () => import('@/views/LongVideoWorkspaceLayout.vue'),
    children: [
      {
        path: '',
        name: 'long_video_projects',
        component: () => import('@/views/LongVideoWorkspaceHome.vue'),
      },
      {
        path: 'project/:seriesId',
        name: 'long_video_series',
        component: () => import('@/views/LongVideoSeriesView.vue'),
      },
      {
        path: 'episode/:projectId',
        name: 'long_video_create',
        component: () => import('@/views/LongVideoCreateView.vue'),
      },
    ],
  },
  /** Legacy paths */
  { path: '/projects', redirect: '/workspace' },
  { path: '/series/:seriesId', redirect: (to) => `/workspace/project/${to.params.seriesId}` },
  { path: '/episode/:projectId', redirect: (to) => `/workspace/episode/${to.params.projectId}` },
  { path: '/long_video_create', redirect: '/workspace' },
  {
    path: '/settings',
    name: 'settings',
    component: () => import('@/views/SettingsView.vue'),
  },
];

const router = createRouter({
  history: createWebHashHistory(),
  routes,
});

export default router;
