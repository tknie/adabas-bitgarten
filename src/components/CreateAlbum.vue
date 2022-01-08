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
  <div class='createalbum p-2'>
    <b-button v-b-modal.modal-1 variant='primary'>Create Album</b-button>

    <b-modal
      @ok='handleOk'
      id='modal-1'
      size='xl'
      variant='outline-danger'
      title='Create Album'
    >
      <p class='my-4'>Please provide new Album parameters</p>
      <b-form>
        <b-form-group
          label-cols-lg='3'
          label='Album'
          label-size='lg'
          label-class='font-weight-bold pt-0'
          class='mb-0'
        >
          <b-form-group
            label-cols-sm='3'
            label='Album name:'
            label-align-sm='right'
            label-for='nested-dbid'
          >
            <b-form-input v-model='Album.Title' id='nested-dbid'></b-form-input>
          </b-form-group>
          <b-form-group
            label-cols-sm='3'
            label='Description:'
            label-align-sm='right'
            label-for='nested-name'
          >
            <b-form-input
              v-model='Album.Metadata.AlbumDescription'
              id='nested-name'
            ></b-form-input>
          </b-form-group>
          <b-form-group
            label-cols-sm='3'
            label='Base images:'
            label-align-sm='right'
            label-for='nested-checkpoint'
          >
            <b-form-select
              v-model='selected'
              v-on:change='selectedPictureBase'
              :options='pictures'
            ></b-form-select>
          </b-form-group>
        </b-form-group>
      </b-form>
      <b-table
        ref='picTable'
        striped
        flip-list-move
        hover
        :items='Album.Pictures'
        :fields='fields'
      >
        <template v-slot:cell(Name)='data'
          >{{ data.item.Name }} [{{ data.item.Md5 }}]
        </template>
        <template v-slot:cell(thumbnail)='data'>
          <img
            :src='Thumbnail(data.item.Md5)'
            class='rounded'
            :alt='"Error loading"'
          />
        </template>
      </b-table>
    </b-modal>
  </div>
</template>

<script lang='ts'>
import { Component, Prop, Vue } from 'vue-property-decorator';
import axios from 'axios';
import {
  ModalPlugin,
  InputGroupPlugin,
  FormSelectPlugin,
  FormCheckboxPlugin,
  FormInputPlugin,
  FormPlugin,
  FormDatepickerPlugin,
  FormGroupPlugin,
} from 'bootstrap-vue';
import { image } from '../images';
import store from '../store';
import { albums } from '../albums';

Vue.use(FormSelectPlugin);
Vue.use(FormCheckboxPlugin);
Vue.use(FormInputPlugin);
Vue.use(InputGroupPlugin);
Vue.use(FormPlugin);
Vue.use(FormDatepickerPlugin);
Vue.use(FormGroupPlugin);
Vue.use(ModalPlugin);

@Component
export default class CreateAlbum extends Vue {
  @Prop(String) private readonly url: string | undefined;
  public data() {
    return {
      selected: null,
      pictures: [],
      fields: [
        { key: 'Name', text: 'Name' },
        { key: 'Description', text: 'Description' },
        { key: 'MIMEType', text: 'MIMEType' },
        { key: 'thumbnail', text: 'thumbnail' },
      ],
      Album: {
        Date: 0,
        Directory: '',
        Generated: 0,
        Metadata: {
          AlbumDescription: '',
          Thumbnail: '',
        },
        Title: '',
        Pictures: [],
      },
      db: null,
    };
  }
  public created() {
    console.log('Created');
    this.$data.Isn = 0;
    this.$data.Album.Date = Math.floor(new Date().getTime() / 1000);
    this.$data.Album.Generated = Math.floor(new Date().getTime() / 1000);
    this.$data.Album.Pictures = [];
    this.$data.pictures = [];
    image.loadPictureBases().then((p) => {
      this.$data.pictures = p;
    });
  }
  public Thumbnail(data: any) {
    // console.log('Thumbnail: ' + JSON.stringify(data));
    const i = store.getters.getThumbnailByMd5(data);
    if (i) {
      return i.src;
    }
    return '';
  }
  public handleOk(bvModalEvt: any) {
    console.log('Handle OK');
    // const getConfig = {
    //   headers: authHeader('application/json'),
    // };
    // axios
    //   .post(
    //     config.Url() + '/adabas/database/' + this.$data.db.dbid() + '/file',
    //     this.$data.createFile,
    //     getConfig
    //   )
    //   .then(function(response) {
    //     console.log(response);
    //   })
    //   .catch(function(error) {
    //     console.log(
    //       error.response.statusText + ':' + JSON.stringify(error.response)
    //     );
    //   });
    albums.storeAlbums(this.$data.Album);
  }
  public onSubmit(evt: any) {
    evt.preventDefault();
    albums.storeAlbums(this.$data.Album);
  }
  public selectedPictureBase(myarg: any) {
    console.log('Select picture base ' + this.$data.selected);
    image.loadPictureDirectory(this.$data.selected).then((element: any) => {
      // console.log('Fetched picture base ' + JSON.stringify(element));
      this.$data.Isn = 0;
      this.$data.Album.Date = Math.floor(new Date().getTime() / 1000);
      this.$data.Album.Generated = Math.floor(new Date().getTime() / 1000);
      this.$data.Album.Pictures = [];
      element.forEach((p: any) => {
        const x = {
          Description: p.title,
          Fill: 'fill',
          Interval: 8000,
          MIMEType: 'image/jpeg',
          Md5: p.msrc,
          Name: p.title,
          Size: { Height: 1280, Width: 960 },
        };
        this.$data.Album.Pictures.push(x);
        // console.log('Load thumb: ' + p.msrc);
        store.dispatch('LOAD_THUMB', p.msrc);
      });
    });
  }
}
</script>

<!-- Add 'scoped' attribute to limit CSS to this component only -->
<style scoped>
h3 {
  margin: 40px 0 0;
}
ul {
  list-style-type: none;
  padding: 0;
}
li {
  display: inline-block;
  margin: 0 10px;
}
a {
  color: #42b983;
}
</style>
