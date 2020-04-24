<template>
  <div>
    <b-alert show variant="success">
      <h5 class="alert-heading">{{albumTitle}}</h5>
    </b-alert>
    <b-carousel
      id="carousel-1"
      :interval="8000"
      controls
      indicators
      background="#ababab"
      background-size:cover
      style="text-shadow: 1px 1px 2px #333;"
    >
      <b-carousel-slide
        v-for="r in images"
        v-bind:key="r.title"
        :caption="r.title"
        :text="albumTitle"
      >
        <div
          v-if="r.MIMEType && r.MIMEType.startsWith('image/')"
          :class="'image slot img-fluid '+dispClass(r.width,r.height)"
          slot="img"
        >
          <div class="fillHeight" :style="'background-image:url('+r.pic+')'"></div>
          <!--img :src="r.pic" :alt="'image slot '+r.MIMEType" /-->
        </div>
        <div v-else class="image slot" slot="img">
          <!--video class="d-block vw-100 video-fluid" controls>
           <source :src="r.pic" type="video/mp4" />
          </video-->
          <video controls id="tribune" class="fillHeight">
            <source :src="r.pic" type="video/mp4" />Your browser does not support the video tag.
          </video>
        </div>
      </b-carousel-slide>
    </b-carousel>
  </div>
</template>

<script lang="ts">
import Vue from 'vue';
import { authHeader, jwtAuth } from '../auth-header';
import { userService } from '../user.service';
import { image } from '../images';
import { config } from '../config';
import store from '../store';
import {
  CarouselPlugin,
  ImagePlugin,
  AlertPlugin,
  BAlert,
  EmbedPlugin,
} from 'bootstrap-vue';
import 'bootstrap/dist/css/bootstrap.css';
import 'bootstrap-vue/dist/bootstrap-vue.css';

Vue.component('b-alert', BAlert);
Vue.use(AlertPlugin);
Vue.use(CarouselPlugin);
Vue.use(ImagePlugin);
Vue.use(EmbedPlugin);

export default {
  extends: Vue,
  props: {
    id: {
      type: String,
    },
  },
  data() {
    return {
      albums: null,
      items: [],
      images: [],
      storeImages: store.state.images,
      albumTitle: '',
      a: store.state.albumsData,
    };
  },
  watch: {
    storeImages: function(newVal: any, oldVal: any) {
      if (this.items) {
        this.items.forEach((i: any, index: number) => {
          if (this.images[index].pic === '') {
            const t = newVal.find((a: any) => a.md5 === i.src);
            if (t) {        
              Vue.set(this.images, index, {
                title: i.title,
                fill: t.fill,
                MIMEType: t.MIMEType,
                height: t.height,
                width: t.width,
                pic: t.src,
              });
            }
          }
        });
      }
    },
    a(newVal: any, oldVal: any) {
      if (this.albums === null) {
        this.albums = newVal.find((album: any) => album.id === this.id);
        if (this.albums && this.albums !== null) {
          this.items = this.albums.pictures;
          this.albumTitle = this.albums.Title;
          this.syncImages();
        }
      }
    },
  },
  methods: {
    dispClass(width: number, height: number) {
      // console.log('w:'+width+' h:'+height);
      // if (height > width) {
      return 'vh-100';
      // }
      // return 'vw-100';
    },
    syncImages() {
      this.items.forEach((element: any, index: number) => {
        const i = store.getters.getImageByMd5(element.src);
        if (i) {
          Vue.set(this.images, index, {
            title: element.title,
            fill: i.fill,
            MIMEType: i.MIMEType,
            height: i.height,
            width: i.width,
            pic: i.src,
          });
        } else {
          // store.dispatch('LOAD_IMAGE',element.src)
          // console.log('Image not loaded: ' + index);
          Vue.set(this.images, index, {
            title: '' + index,
                fill: 'fillHeight',
            MIMEType: '',
            height: 0,
            width: 0,
            pic: '',
          });
        }
      });
    },
  },
  created() {
    // console.log('Create album ' + this.id);
  },
  updated() {
    // console.log('Update album ' + this.id);
  },
  mounted() {
    const a = store.getters.getAlbumById(this.id);
    if (a) {
      this.albums = a;
      this.albumTitle = a.Title;
      this.items = this.albums.pictures;
      this.syncImages();
      return;
    }
    store.dispatch('INIT_ALBUM', { nr: this.id, loadImage: true });
  },
};
</script>

<style>
.box {
  height: 100vh;
  max-height: 75%; /* added */
  display: block;
}
video {
  height: 100%;
  width: auto;
  max-height: 75%; /* added */
}
.fill {
  height: auto;
  width: 100%;
  background-position: center;
  -webkit-background-size: cover;
  -moz-background-size: cover;
  background-size: contain;
  background-repeat: no-repeat;
  -o-background-size: cover;
}
.fillHeight {
  height: 100%;
  width: auto;
  background-position: center;
  -webkit-background-size: cover;
  -moz-background-size: cover;
  background-size: contain;
  background-repeat: no-repeat;
  -o-background-size: cover;
}
</style>