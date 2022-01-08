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
  <div>
    <b-alert show variant="success">
      <h5 class="alert-heading">{{ albumTitle }}</h5>
    </b-alert>
    <b-carousel
      id="carousel-1"
      :interval="interval"
      controls
      indicators
      background="#ababab"
      background-size:cover
      style="text-shadow: 1px 1px 2px #333"
      @sliding-start="onSlideStart"
      ref="imageCarousel"
    >
      <b-carousel-slide
        v-for="(r, index) in images"
        v-bind:key="r.title"
        :caption="r.title"
        :text="albumTitle"
      >
        <div
          v-if="r.MIMEType && r.MIMEType.startsWith('image/')"
          :class="'image slot img-fluid ' + displayClass(r.width, r.height)"
          slot="img"
        >
          <div
            class="fillHeight"
            :style="'background-image:url(' + r.pic + ')'"
          ></div>
        </div>
        <div v-else class="image slot text-center vh-100" slot="img">
          <video
            center
            controls
            :ref="'video' + index"
            :id="'video' + index"
            class="fill"
          >
            Your browser does not support the video tag.
          </video>
        </div>
      </b-carousel-slide>
    </b-carousel>
  </div>
</template>

<script lang="ts">
import Vue from 'vue';
import store from '../store';
import { streamVideo } from '../images';
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
      interval: 8000,
      storeImages: store.state.images,
      albumTitle: '',
      a: store.state.albumsData,
    };
  },
  watch: {
    storeImages(newVal: any, oldVal: any) {
      if (this.items) {
        this.items.forEach((i: any, index: number) => {
          // if (this.images[index].pic === '') {
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
          // }
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
    onSlideStart(slide: any) {
      const image = this.images[slide];
      const vidElement = this.$refs['video' + slide];
      if (
        !vidElement ||
        vidElement.length === 0 ||
        this.items[slide].MIMEType !== 'video/mp4'
      ) {
        this.$data.interval = 8000;
        return;
      }
      const carouselElement = this.$refs.imageCarousel;
      if (carouselElement) {
        this.$data.interval = 0;
      } else {
        console.log('Not pause, carousel reference not found');
        return;
      }
      streamVideo(this.items[slide].src, vidElement[0]).then(() => {
        vidElement[0].onended = () => {
          this.$data.interval = 1000;
        };
      });
    },
    displayClass(width: number, height: number) {
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
          const info = this.items[index];
          Vue.set(this.images, index, {
            title: info.title,
            fill: 'fillHeight',
            MIMEType: info.MIMEType,
            height: info.h,
            width: info.w,
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