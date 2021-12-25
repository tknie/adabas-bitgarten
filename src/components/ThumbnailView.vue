<template>
  <div class="thumbnailView">
    <div>
      <div>
        <b-alert show variant="success">
          <h5 class="alert-heading">Thumbnails view ...</h5>
        </b-alert>
      </div>
      <div>
        <b-alert show variant="success">Bitte Album auswählen:</b-alert>
        <b-form-select
          variant="outline-success"
          v-model="selectedItem"
          @change="fetchAlbumData(selectedItem)"
          ><option value="null">Bitte Album auswählen</option>
          <option
            v-for="(item, index) in items"
            :key="item.DateTime"
            :value="index + 1"
          >
            {{
              index +
              1 +
              ". " +
              item.Title +
              " - " +
              new Date(item.DateTime * 1000).toUTCString()
            }}
          </option>
        </b-form-select>
      </div>
      <div></div>
    </div>
    <div>
      <b-alert show variant="success">{{ selectedTitle() }}</b-alert>
      <b-container fluid class="bv-example-row mb-3">
        <b-modal centered size="xl" id="modal-image" title="Image" ok-only
          ><b-img center fluid :src="currentPic" class="vh-100" />
          <b-alert show class="text-center" variant="success">{{
            selectedDescription
          }}</b-alert>
          <b-alert
            class="w-50 pb-2 d-inline-block"
            size="sm"
            id="notice"
            show
            variant="danger"
            >{{ selectedTitle() }}</b-alert
          >
          <b-alert
            class="w-50 pb-2 d-inline-block text-right"
            size="sm"
            id="download"
            show
            variant="danger"
          >
            <a
              :download="'custom-' + currentMd5 + '.jpg'"
              :href="currentPic"
              title="ImageName"
            >
              &gt;Download Bild&lt;
            </a>
          </b-alert>
        </b-modal>
        <b-modal centered size="xl" id="modal-video" title="Video" ok-only
          ><video center controls id="tribune" class="vh-100 fillHeight">
            <source :src="currentPic" type="video/mp4" />
            Your browser does not support the video tag.
          </video></b-modal
        >
        <b-row align-h="around">
          <div v-for="(p, index) in Album.Pictures" v-bind:key="p.Md5">
            <b-col align-v="center" style="display: inline-block">
              <b-button
                variant="outline-primary"
                v-if="p.MIMEType.startsWith('image')"
                v-b-modal.modal-image
                v-on:click="loadImage(p.Md5)"
                >{{ index + 1 }}
                <b-img
                  class="rounded"
                  thumbnail
                  fluid
                  :src="Thumbnail(p.Md5)"
                  alt="Error loading" /></b-button
              ><b-button
                variant="outline-primary"
                v-else
                v-b-modal.modal-video
                v-on:click="loadVideo(p.Md5)"
                >{{ index + 1 }} Video Movie</b-button
              ></b-col
            >
            <div class="w-100" v-if="(index + 1) % 5 === 0"></div>
          </div> </b-row
      ></b-container>
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
  ButtonPlugin,
  LayoutPlugin,
} from 'bootstrap-vue';
import store from '../store';
import { image } from '../images';
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
Vue.use(LayoutPlugin);
Vue.use(ButtonPlugin);

@Component({
  components: {
    CreateAlbum,
  },
})
export default class ThumbnailView extends Vue {
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
      images: store.state.images,
      items: store.state.albums,
      selectedItem: null,
      selectedPicBaseItem: '',
      fields: [{ key: 'Md5' }],
      selectedDescription: '',
      currentPic: '',
      currentMd5: '',
    };
  }
  @Watch('a')
  public changeAlbum(newVal: any, oldVal: any) {
    // console.log('Change Album');
    const id = this.$data.items[this.$data.selectedItem - 1].ISN;
    this.$data.albums = newVal.find((album: any) => album.id === id);
    if (this.$data.albums && this.$data.albums !== null) {
      this.adaptAlbum(this.$data.albums);
    }
  }
  @Watch('images')
  public changeImage(newVal: any, oldVal: any) {
    const entry = newVal.find((x: any) => x.md5 === this.$data.currentMd5);
    /*console.log('Got: ' + this.$data.currentMd5 + '<' + entry + '>');
    newVal.forEach((e:any) => {
      console.log('MD5: <'+e.md5+'> check <'+this.$data.currentMd5+'>'+(e.md5 === this.$data.currentMd5));
    });*/
    if (entry) {
      // console.log('Found: <'+entry.md5+'>')
      this.$data.currentPic = entry.src;
    }
  }
  public created() {
    // console.log('Create editor');
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
      (error: any) => {
        console.log('Load error: ' + error);
      },
    );
  }
  public selectedTitle() {
    if (this.$data.selectedItem === null) {
      this.$data.selectedItem = 1;
      return this.$data.items[0].Title;
    }
    if (
      this.$data.selectedItem < 1 ||
      this.$data.items.length < this.$data.selectedItem
    ) {
      return '';
    }
    return this.$data.items[this.$data.selectedItem - 1].Title;
  }
  public getItems() {
    return store.state.albums;
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
    if (this.$data.selectedItem === null || !this.$data.selectedItem) {
      return;
    }
    const id = this.$data.items[this.$data.selectedItem - 1].ISN;
    // console.log('Get album fetch '+this.$data.selectedItem+' id='+id);
    const a = store.getters.getAlbumById(id);
    if (a) {
      // console.log('GOT: ' + JSON.stringify(a));
      this.adaptAlbum(a);
      return;
    } else {
      console.log('Not in cache: ' + this.$data.selectedItem);
    }
    store.dispatch('INIT_ALBUM', {
      nr: id,
      loadImage: false,
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
  private loadImage(data: any) {
    // console.log('Load Image ' + data);
    this.$data.currentMd5 = data;
    const p = this.$data.Album.Pictures.find((x: any) => x.Md5 === data);
    this.$data.selectedDescription = p.Description;
    store.dispatch('LOAD_IMAGE', data);
    const i = store.getters.getImageByMd5(data);
    if (i) {
      this.$data.currentPic = i.src;
      // console.log('Found Image ' + data);
      return i.src;
    }
    return '';
  }
  private loadVideo(data: any) {
    // console.log('Load Video ' + data);
    this.$data.currentMd5 = data;
    store.dispatch('LOAD_VIDEO', data);
    const i = store.getters.getImageByMd5(data);
    if (i) {
      this.$data.currentPic = i.src;
      // console.log('Found Video ' + data);
      return i.src;
    }
    return '';
  }
}
</script>

<style>
#notice #download {
  font-size: 12px;
}
</style>