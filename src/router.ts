import Vue from 'vue';
import Router from 'vue-router';
import Home from './views/Album.vue';
import LoginPage from './login/LoginPage.vue';

Vue.use(Router);

export default new Router({
  routes: [
    {
      path: '/',
      name: 'home',
      component: Home,
    },
    { path: '/login', name: 'login', component: LoginPage },
    {
      path: '/pictures/:id',
      name: 'pictures',
      component: () => import('./views/Pictures.vue'),
    },
    {
      path: '/editor',
      name: 'editor',
      component: () => import('./views/Editor.vue'),
    },
    {
      path: '/about',
      name: 'about',
      // route level code-splitting
      // this generates a separate chunk (about.[hash].js) for this route
      // which is lazy-loaded when the route is visited.
      component: () => import(/* webpackChunkName: "about" */ './views/About.vue'),
    },
  ],
});
