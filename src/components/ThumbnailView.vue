<template>
  <div class='thumbnailView'>
    <div>
      <div>
        <b-alert show variant='success'>
          <h5 class='alert-heading'>Thumbnails view ...</h5>
        </b-alert>
      </div>
      <div>
        <b-button pill variant='outline-primary' @click='refreshAlbums'
          >Refresh</b-button
        >
        <b-form-select
          v-model='selectedItem'
          @change='fetchAlbumData(selectedItem)'
        >
          <option
            v-for='(item, index) in items'
            :key='item.Title'
            :value='index + 1'
          >
            {{ index + 1 + '. ' + item.Title + ' - ' + item.Date }}
          </option>
        </b-form-select>
      </div>
      <div></div>
    </div>
    <div>
      <b-alert show variant='success'>{{ selectedTitle() }}</b-alert>
      <b-container fluid class='bv-example-row mb-3'>
        <b-modal centered size='xl' id='modal-image' title='Image' ok-only
          ><b-img fluid :src='currentPic'
        /></b-modal>
        <b-modal centered size='xl' id='modal-video' title='Video' ok-only
          ><video controls id='tribune' slot='img' class='fillHeight'>
            <source :src='currentPic' type='video/mp4' />
            Your browser does not support the video tag.
          </video></b-modal
        >
        <b-row align-h='around'>
          <b-col
            align-v='center'
            style='width: 10%; display: inline-block'
            v-for='(p, index) in Album.Pictures'
            v-bind:key='p.Md5'
          >
            <b-button
              variant='outline-primary'
              v-if='p.MIMEType.startsWith("image")'
              v-b-modal.modal-image
              v-on:click='loadImage(p.Md5)'
              >{{ index + 1 }}
              <b-img
                class='rounded'
                thumbnail
                fluid
                :src='Thumbnail(p.Md5)'
                alt='Error loading' /></b-button
            ><b-button
              variant='outline-primary'
              v-else
              v-b-modal.modal-video
              v-on:click='loadImage(p.Md5)'
              >{{ index + 1 }} Video Movie</b-button
            >
            <!--video v-else controls id='tribune' slot='img' class='fillHeight'>
              <source :src='p.pic' type='video/mp4' />
              Your browser does not support the video tag.
            </video-->
            <div class='w-100' v-if='(index + 1) % 5 === 0' /> </b-col></b-row
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
      selectedItem: '',
      selectedPicBaseItem: '',
      fields: [{ key: 'Md5' }],
      currentPic: '',
      currentMd5: '',
    };
  }
  @Watch('a')
  public changeAlbum(newVal: any, oldVal: any) {
    // console.log('Change Album');
    this.$data.albums = newVal.find(
      (album: any) => album.id === this.$data.selectedItem    );
    if (this.$data.albums && this.$data.albums !== null) {
      this.adaptAlbum(this.$data.albums);
    }
  }
  @Watch('images')
  public changeImage(newVal: any, oldVal: any) {
    const entry = newVal.find((x: any) => x.md5 === this.$data.currentMd5);
    // console.log('Got: <'+entry+'>')
    /*newVal.forEach((e:any) => {
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
      (error: any) => console.log('Load error: ' + error),
    );
  }
  public changeOrder(from: any, to: any) {
    // console.log('Change ' + from + ' to ' + to);
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
  public selectedTitle() {
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
  public deleteRecord() {
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
  public refreshAlbums() {
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
    store.dispatch('INIT_ALBUM', {
      nr: this.$data.selectedItem,
      loadImage: false,
    });
  }
  private loadPictureBase() {
    // console.log('Load picture base');
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
  private loadImage(data: any) {
    console.log('Request Images ' + data);
    this.$data.currentMd5 = data;
    store.dispatch('LOAD_IMAGE', data);
    const i = store.getters.getImageByMd5(data);
    if (i) {
      this.$data.currentPic = i.src;
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
    store.dispatch('INIT_ALBUM', { nr: this.$data.Isn, loadImage: false });
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
