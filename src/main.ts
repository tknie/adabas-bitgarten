import Vue from 'vue';
import App from './App.vue';
import router from './router';
import store from './store';
import './registerServiceWorker';

Vue.config.productionTip = false;

router.beforeEach((to: any, from: any, next: any) => {
  // redirect to login page if not logged in and trying to access a restricted page
  const publicPages = ['/login'];
  const authRequired = !publicPages.includes(to.path);
  const loggedIn = localStorage.getItem('user');

  if (authRequired && !loggedIn) {
    return next({
      path: '/login',
      query: { returnUrl: to.path },
    });
  }

  next();
});

new Vue({
  router,
  store,
  render: (h) => h(App),
  created() {
    // store.dispatch('INIT_ALBUMS', '');
  },
}).$mount('#app');
