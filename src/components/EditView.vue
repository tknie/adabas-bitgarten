<template>
  <div class="editView">
    <div>
      <div>
        <b-alert show variant="success">
          <h5 class="alert-heading">Editing ...</h5>
        </b-alert>
      </div>
      <div>
        <b-button pill variant="outline-primary" @click="refreshAlbums">Refresh</b-button>
        <b-form-select v-model="selectedItem" @change="fetchAlbumData(selectedItem)">
          <option v-for="item in items" :key="item.Title" :value="item.ISN">{{item.Title}}</option>
        </b-form-select>
      </div>
      <div>
        <b-button
          pill
          v-b-toggle.collapse-1
          variant="outline-primary"
          @show="loadPictureBase()"
        >New Album</b-button>
        <b-collapse id="collapse-1" class="mt-2">
          <b-card>
            <p class="card-text">Define basic parameters for new album:</p>
            <div>
              <b-form @submit="onSubmit" inline>
                <b-input-group>
                  <label for="inline-form-input-name">Title</label>
                  <b-input
                    id="inline-form-input-name"
                    v-model="Album.Title"
                    placeholder="New Album name"
                  ></b-input>
                  <label for="inline-form-input-name">Picture base</label>
                  <b-form-select v-model="selectedPicBaseItem" @change="fetchPictureBase()">
                    <option v-for="p in pictures" :key="p" :value="p">{{p}}</option>
                  </b-form-select>

                  <b-button variant="primary" type="submit">Save</b-button>
                </b-input-group>
              </b-form>
            </div>
          </b-card>
        </b-collapse>
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
          <div>{{new Date(Album.Generated*1000)}}</div>
        </div>
        <b-button pill variant="outline-primary" type="submit">Update</b-button>
        <b-button pill variant="outline-primary" @click="deleteRecord">Delete</b-button>
        <b-table
          ref="picTable"
          striped
          flip-list-move
          hover
          :items="Album.Pictures"
          :fields="fields"
        >
          <template v-slot:cell(index)="data">{{data.index+1}}</template>
          <template v-slot:cell(order)="data">
            <b-form-input
              type="number"
              @change="changeOrder(data.index+1,$event)"
              :value="data.index + 1"
            ></b-form-input>
          </template>
          <template v-slot:cell(Description)="data">
            <b-form-input type="text" v-model="Album.Pictures[data.index].Description"></b-form-input>
          </template>
          <template v-slot:cell(Md5)="data">
            <img :src="Thumbnail(data.item.Md5)" class="rounded" :alt="'Error loading'" />
          </template>
          <template v-slot:cell(MIMEtype)="data">
            <b-form-input type="text" v-model="Album.Pictures[data.index].Md5"></b-form-input>
            <b-form-input type="text" v-model="Album.Pictures[data.index].MIMEType"></b-form-input>
            <b-form-input type="text" v-model="Album.Pictures[data.index].Fill"></b-form-input>
          </template>
        </b-table>
        <b-button pill variant="outline-primary" @click="addPicture">Add Picture</b-button>
      </b-form>
    </div>
  </div>
</template>

<script lang='ts'>
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

Vue.use(FormSelectPlugin);
Vue.use(FormCheckboxPlugin);
Vue.use(FormInputPlugin);
Vue.use(InputGroupPlugin);
Vue.use(FormPlugin);
Vue.use(FormDatepickerPlugin);
Vue.use(FormGroupPlugin);
Vue.use(AlertPlugin);
Vue.use(CardPlugin);

export default {
  extends: Vue,
  props: {
    id: {
      type: String,
    },
  },
  data() {
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
        { key: 'MIMEtype' },
      ],
    };
  },
  watch: {
    a(newVal: any, oldVal: any) {
      // console.log('Change Album');
      this.albums = newVal.find((album: any) => album.id === this.selectedItem);
      if (this.albums && this.albums !== null) {
        this.adaptAlbum(this.albums);
      }
    },
  },
  created() {
    console.log('Create editor');
    this.Album.Pictures = [];
    const items = this.getItems();
    if (items.length < 2) {
      store.dispatch('INIT_ALBUMS', '');
    }
    const promise = image.loadPictureBases();
    promise.then(
      (response: any) => {
        this.pictures = response;
        // console.log('Loaded all pictures ' + JSON.stringify(this.pictures));
        return response.data;
      },
      (error: any) => console.log('Load error: ' + error),
    );
  },
  methods: {
    changeOrder(from: any, to: any) {
      console.log('Change ' + from + ' to ' + to);
      if (to === from) {
        return;
      }
      console.log('Pictures before ' + JSON.stringify(this.Album.Pictures));
      const pics = [];
      for (let i = 0; i < this.Album.Pictures.length; i++) {
        if (i === to - 1) {
          pics.push(this.Album.Pictures[from - 1]);
        }
        if (i !== from - 1) {
          pics.push(this.Album.Pictures[i]);
        }
      }
      this.Album.Pictures = pics;
      console.log('Pictures after ' + JSON.stringify(this.Album.Pictures));
      this.$refs.picTable.refresh();
    },
    getItems() {
      return store.state.albums;
    },
    deleteRecord() {
      albums.deleteAlbum(this.Isn);
    },
    addPicture() {
      const x = {
        Description: 'Extra',
        Fill: 'Fill',
        Interval: 8000,
        MIMEType: 'image/jpeg',
        Md5: this.Album.Pictures[0].Md5,
        Size: {
          Height: this.Album.Pictures[0].Md5.h,
          Width: this.Album.Pictures[0].Md5.w,
        },
      };

      this.Album.Pictures.push(x);
    },
    refreshAlbums() {
      store.commit('CLEAR', '');
      store.dispatch('INIT_ALBUMS', '');
    },
    adaptAlbum(albumCard: any) {
      // console.log('Receive ' + JSON.stringify(albumCard));
      this.Isn = albumCard.id;
      this.Album.Title = albumCard.Title;
      this.Album.Date = albumCard.date;
      this.Album.Generated = Math.floor(new Date().getTime() / 1000);
      this.Album.Pictures = [];
      albumCard.pictures.forEach((element: any) => {
        const x = {
          Description: element.title,
          Fill: element.fill,
          Interval: 8000,
          MIMEType: element.MIMEType,
          Md5: element.msrc,
          Size: { Height: element.h, Width: element.w },
        };
        this.Album.Pictures.push(x);
      });
      if (this.Album.Pictures.length > 0) {
        this.Album.Metadata.Thumbnail = this.Album.Pictures[0].Md5;
      }
      this.Album.Metadata.AlbumDescription = albumCard.Title;
      // console.log('Found ' + JSON.stringify(this.Album));
    },
    fetchAlbumData(idx: string) {
      // console.log('Select and fetch <' + this.selectedItem + '> ' + idx);
      if (!this.selectedItem) {
        return;
      }
      // console.log('Get album fetch');
      const a = store.getters.getAlbumById(this.selectedItem);
      if (a) {
        // console.log('GOT: ' + JSON.stringify(a));
        this.adaptAlbum(a);
        return;
      } else {
        console.log('FAIL: ' + this.selectedItem);
      }
      store.dispatch('INIT_ALBUM', { nr: this.selectedItem, loadImage: false });
    },
    loadPictureBase() {
      console.log('Load picture base');
    },
    fetchPictureBase() {
      console.log('Fetch picture base');
      this.Album.Pictures = [];
      image
        .loadPictureDirectory(this.selectedPicBaseItem)
        .then((element: any) => {
          // console.log('Fetched picture base ' + JSON.stringify(element));
          this.Isn = 0;
          this.Album.Date = Math.floor(new Date().getTime() / 1000);
          this.Album.Generated = Math.floor(new Date().getTime() / 1000);
          this.Album.Pictures = [];
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
            this.Album.Pictures.push(x);
            // console.log('Load thumb: ' + p.msrc);
            store.dispatch('LOAD_THUMB', p.msrc);
          });
        });
    },
    Thumbnail(data: any) {
      // console.log('Thumbnail: ' + JSON.stringify(data));
      const i = store.getters.getThumbnailByMd5(data);
      if (i) {
        return i.src;
      }
      return '';
    },
    save(evt: any) {
      console.log('Save clicked');
    },
    onSubmit(evt: any) {
      evt.preventDefault();
      albums.storeAlbums(this.Album);
    },
    onUpdate(evt: any) {
      evt.preventDefault();
      this.Album.Metadata.Thumbnail = this.Album.Pictures[0].Md5;
      albums.updateAlbums(this.Isn, this.Album);
      store.commit('CLEAR', '');
      store.dispatch('INIT_ALBUM', { nr: this.Isn, loadImage: false });
    },
    onReset(evt: any) {
      evt.preventDefault();
      // Reset our form values
      this.form.email = '';
      this.form.name = '';
      this.form.food = null;
      this.form.checked = [];
      // Trick to reset/clear native browser form validation state
      this.show = false;
      this.$nextTick(() => {
        this.show = true;
      });
    },
  },
};
</script>

<style>
</style>