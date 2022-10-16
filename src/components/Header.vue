<!--
 * Copyright (c) 2020-2022 Thorsten A. Knieling
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.-->

<template>
  <div class="header">
    <b-navbar toggleable="lg" type="dark" variant="dark" sticky="true" fixed="top">
      <b-navbar-brand href="#">
        <img src="Icon-60.JPG" class="d-inline-block align-top" alt="Bitgarten">
        Bitgarten</b-navbar-brand>

      <b-navbar-toggle target="nav-collapse"></b-navbar-toggle>

      <b-collapse id="nav-collapse" is-nav>
        <b-navbar-nav>
          <b-nav-item to="/">&Uuml;bersicht</b-nav-item>
          <b-nav-item to="/thumbnail">Thumbnail</b-nav-item>
          <b-nav-item v-if="checked" to="/editor">Editor</b-nav-item>
        </b-navbar-nav>
        <b-navbar-nav class="ml-auto" right>
          <b-nav-item v-on:click="refresh">Refresh</b-nav-item>
          <b-nav-item v-on:click="logout">Logout</b-nav-item>
        </b-navbar-nav>
      </b-collapse>
    </b-navbar>
  </div>
</template>

<script lang="ts">
import { Component, Vue, Watch } from 'vue-property-decorator';
import {
  NavbarPlugin,
  ButtonPlugin,
  FormInputPlugin,
  TablePlugin,
  FormCheckboxPlugin,
} from 'bootstrap-vue';
import { albums } from '../albums';
import { config } from '../config';
import store from '../store';
import 'bootstrap/dist/css/bootstrap.css';
import 'bootstrap-vue/dist/bootstrap-vue.css';
import { userService } from '../user.service';

Vue.use(NavbarPlugin);
Vue.use(ButtonPlugin);
Vue.use(FormInputPlugin);
Vue.use(FormCheckboxPlugin);
Vue.use(TablePlugin);

// @Component({
//     NavbarPlugin,
//     ButtonPlugin,
//     FormInputPlugin,
//     TablePlugin,
//      store,
// })
export default class Header extends Vue {
  private checked = store.state.editorMode;
  private data() {
    return {
      checked: this.checked,
      results: store.state.albums,
    };
  }
  @Watch('results')
  private onResultsChanged(value: string, oldValue: string) {
    console.log('Results changed');
  }
  private changeChecked(evt: any, e: any) {
    console.log('Changex checked' + !evt);
    store.state.editorMode = !evt;
    this.checked = store.state.editorMode;
    console.log('Change checked ' + store.state.editorMode);
  }
  private setChecked(c: boolean) {
    this.checked = c;
  }
  private getChecked() {
    return this.checked;
  }
  private refresh() {
    console.log('Refresh image list');
    store.commit('CLEAR', '');
    location.reload();
  }
  private logout() {
    console.log('Call logout');
    userService.logout();
    store.commit('CLEAR', '');
    location.reload();
  }
}
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped>
a {
  text-decoration: none;
  color: black;
}

.router-link-exact-active {
  color: black;
}
</style>
