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
  <div class="editView">
    <div>
      <div>
        <b-alert show variant="dark">
          <h5 class="alert-heading">Editing ...</h5>
        </b-alert>
      </div>
      <div>
        <b-button pill variant="outline-primary" @click="refreshAlbums"
          >Refresh</b-button
        >
        <b-form-select
          v-model="selectedItem"
          @change="fetchAlbumData(selectedItem)"
        >
          <option v-for="item in items" :key="item.Title" :value="item.ISN">{{
            item.Title
          }}</option>
        </b-form-select>
      </div>
      <div>
        <CreateAlbum />
        <!--b-button
          pill
          v-b-toggle.collapse-1
          variant="outline-primary"
          @show="loadPictureBase()"
          >New Album</b-button
        >
        <b-collapse id="collapse-1" class="mt-2">
          <b-card>
            <p class="card-text">Define basic parameters for new album:</p>
            <div>
              <b-form @submit="onSubmit" inline>
                <b-form-group
                  id="inline-form-input-name"
                  v-model="Album.Title"
                  label="Title"
                  label-for="input-1"
                  placeholder="New Album name"
                >
                  <b-form-input
                    id="input-1"
                    v-model="form.email"
                    required
                    placeholder="Enter Album name"
                  ></b-form-input>
                </b-form-group>
                <b-form-select
                  v-model="selectedPicBaseItem"
                  @change="fetchPictureBase()"
                  label="Picture base"
                >
                  <option v-for="p in pictures" :key="p" :value="p">{{
                    p
                  }}</option>
                </b-form-select>

                <b-button variant="primary" type="submit">Save</b-button>
              </b-form>
            </div>
          </b-card>
        </b-collapse-->
      </div>
    </div>
    <div>
      selectedItem:
      <strong>{{ selectedItem }}</strong>
    </div>
    <div>
      <b-form @submit="onUpdate">
        <div>
          <b-form-input type="text" v-model="Album.Title"></b-form-input>
          <div>{{ new Date(Album.Generated * 1000) }}</div>
        </div>
        <b-button pill variant="outline-primary" type="submit">Update</b-button>
        <b-button pill variant="outline-primary" @click="deleteRecord"
          >Delete</b-button
        >
        <b-table
          ref="picTable"
          striped
          flip-list-move
          hover
          :items="Album.Pictures"
          :fields="fields"
        >
          <template v-slot:cell(index)="data">{{ data.index + 1 }}</template>
          <template v-slot:cell(order)="data">
            <b-form-input
              type="number"
              @change="changeOrder(data.index + 1, $event)"
              :value="data.index + 1"
            ></b-form-input>
            <b-form-input
              type="text"
              v-model="Album.Pictures[data.index].Md5"
            ></b-form-input>
            <b-form-input
              type="text"
              v-model="Album.Pictures[data.index].MIMEType"
            ></b-form-input>
            <b-form-input
              type="text"
              v-model="Album.Pictures[data.index].Fill"
            ></b-form-input>
          </template>
          <template v-slot:cell(Description)="data">
            <b-form-input
              type="text"
              v-model="Album.Pictures[data.index].Description"
            ></b-form-input>
          </template>
          <template v-slot:cell(Md5)="data">
            <img
              :src="Thumbnail(data.item.Md5)"
              class="rounded"
              :alt="'Error loading'"
            />
          </template>
        </b-table>
        <b-button pill variant="outline-primary" @click="addPicture"
          >Add Picture</b-button
        >
      </b-form>
    </div>
  </div>
</template>

<script lang="ts">
import { Component, Prop, Watch, Vue } from 'vue-property-decorator';
import {
  AlertPlugin,
  InputGroupPlugin,
  FormSelectPlugin,
  CardPlugin,
  FormCheckboxPlugin,
  FormInputPlugin,
  FormPlugin,
  FormDatepickerPlugin,
  FormGroupPlugin,
} from 'bootstrap-vue';
import store from '../store';
import { image } from '../images';
import { albums } from '../albums';
import 'bootstrap/dist/css/bootstrap.css';
import 'bootstrap-vue/dist/bootstrap-vue.css';
import CreateAlbum from './CreateAlbum.vue';

Vue.use(FormSelectPlugin);
Vue.use(FormCheckboxPlugin);
Vue.use(FormInputPlugin);
Vue.use(InputGroupPlugin);
Vue.use(FormPlugin);
Vue.use(FormDatepickerPlugin);
Vue.use(FormGroupPlugin);
Vue.use(AlertPlugin);
Vue.use(CardPlugin);

@Component({
  components: {
    CreateAlbum,
  },
})
export default class EditView extends Vue {
  @Prop(String) private readonly id: string | undefined;
 public data() {
    return {
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
      pictures: [],
      a: store.state.albumsData,
      items: store.state.albums,
      selectedItem: '',
      selectedPicBaseItem: '',
      fields: [
        'index',
        { key: 'order', sortable: true },
        { key: 'Description', sortable: true },
        { key: 'Md5' },
      ],
    };
  }
  @Watch('a')
public  changeAlbum(newVal: any, oldVal: any) {
    // console.log('Change Album');
    this.$data.albums = newVal.find(
      (album: any) => album.id === this.$data.selectedItem,
    );
    if (this.$data.albums && this.$data.albums !== null) {
      this.adaptAlbum(this.$data.albums);
    }
  }
public  created() {
    console.log('Create editor');
    this.$data.Album.Pictures = [];
    const items = this.getItems();
    if (items.length < 2) {
      store.dispatch('INIT_ALBUMS', '');
    }
    const promise = image.loadPictureBases();
    promise.then(
      (response: any) => {
        this.$data.pictures = response;
        // console.log('Loaded all pictures ' + JSON.stringify(this.pictures));
        return response.data;
      },
      (error: any) => console.log('Load error: ' + error),
    );
  }
public  changeOrder(from: any, to: any) {
    console.log('Change ' + from + ' to ' + to);
    if (to === from) {
      return;
    }
    console.log('Pictures before ' + JSON.stringify(this.$data.Album.Pictures));
    const pics = [];
    for (let i = 0; i < this.$data.Album.Pictures.length; i++) {
      if (i === to - 1) {
        pics.push(this.$data.Album.Pictures[from - 1]);
      }
      if (i !== from - 1) {
        pics.push(this.$data.Album.Pictures[i]);
      }
    }
    this.$data.Album.Pictures = pics;
    console.log('Pictures after ' + JSON.stringify(this.$data.Album.Pictures));
    (this.$refs.picTable as any).refresh();
  }
public  getItems() {
    return store.state.albums;
  }
public  deleteRecord() {
    albums.deleteAlbum(this.$data.Isn);
  }
 public addPicture() {
    const x = {
      Description: 'Extra',
      Fill: 'Fill',
      Interval: 8000,
      MIMEType: 'image/jpeg',
      Md5: this.$data.Album.Pictures[0].Md5,
      Size: {
        Height: this.$data.Album.Pictures[0].Md5.h,
        Width: this.$data.Album.Pictures[0].Md5.w,
      },
    };

    this.$data.Album.Pictures.push(x);
  }
public  refreshAlbums() {
    store.commit('CLEAR', '');
    store.dispatch('INIT_ALBUMS', '');
  }
 public adaptAlbum(albumCard: any) {
    // console.log('Receive ' + JSON.stringify(albumCard));
    this.$data.Isn = albumCard.id;
    this.$data.Album.Title = albumCard.Title;
    this.$data.Album.Date = albumCard.date;
    this.$data.Album.Generated = Math.floor(new Date().getTime() / 1000);
    this.$data.Album.Pictures = [];
    albumCard.pictures.forEach((element: any) => {
      const x = {
        Description: element.title,
        Fill: element.fill,
        Interval: 8000,
        MIMEType: element.MIMEType,
        Md5: element.msrc,
        Size: { Height: element.h, Width: element.w },
      };
      this.$data.Album.Pictures.push(x);
    });
    if (this.$data.Album.Pictures.length > 0) {
      this.$data.Album.Metadata.Thumbnail = this.$data.Album.Pictures[0].Md5;
    }
    this.$data.Album.Metadata.AlbumDescription = albumCard.Title;
    // console.log('Found ' + JSON.stringify(this.Album));
  }
 public fetchAlbumData(idx: string) {
    // console.log('Select and fetch <' + this.selectedItem + '> ' + idx);
    if (!this.$data.selectedItem) {
      return;
    }
    // console.log('Get album fetch');
    const a = store.getters.getAlbumById(this.$data.selectedItem);
    if (a) {
      // console.log('GOT: ' + JSON.stringify(a));
      this.adaptAlbum(a);
      return;
    } else {
      console.log('FAIL: ' + this.$data.selectedItem);
    }
    store.dispatch('LOAD_THUMBS', {
      nr: this.$data.selectedItem,
      loadImage: false,
    });
  }
 private loadPictureBase() {
    console.log('Load picture base');
  }
  private fetchPictureBase() {
    console.log('Fetch picture base');
    this.$data.Album.Pictures = [];
    image
      .loadPictureDirectory(this.$data.selectedPicBaseItem)
      .then((element: any) => {
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
  private Thumbnail(data: any) {
    // console.log('Thumbnail: ' + JSON.stringify(data));
    const i = store.getters.getThumbnailByMd5(data);
    if (i) {
      return i.src;
    }
    return '';
  }
  private save(evt: any) {
    console.log('Save clicked');
  }
  private onSubmit(evt: any) {
    evt.preventDefault();
    albums.storeAlbums(this.$data.Album);
  }
  private onUpdate(evt: any) {
    evt.preventDefault();
    this.$data.Album.Metadata.Thumbnail = this.$data.Album.Pictures[0].Md5;
    albums.updateAlbums(this.$data.Isn, this.$data.Album);
    store.commit('CLEAR', '');
    store.dispatch('LOAD_THUMBS', { nr: this.$data.Isn, loadImage: false });
  }
  private onReset(evt: any) {
    evt.preventDefault();
    // Reset our form values
    this.$data.form.email = '';
    this.$data.form.name = '';
    this.$data.form.food = null;
    this.$data.form.checked = [];
    // Trick to reset/clear native browser form validation state
    this.$data.show = false;
    this.$nextTick(() => {
      this.$data.show = true;
    });
  }
}
</script>

<style></style>
