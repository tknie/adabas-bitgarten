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
  <div class="albumView">
    <b-table
      striped
      hover
      borderless
      small
      selectable
      sort-by="Date"
      :sort-desc="sortDesc"
      select-mode="single"
      @row-selected="selected"
      :fields="fields"
      :items="items"
      :busy="isBusy"
    >
      <template v-slot:table-busy>
        <div class="text-center text-danger my-2">
          <b-spinner class="align-middle"></b-spinner>
          <strong>Loading...</strong>
        </div>
      </template>

      <!-- Optional default data cell scoped slot -->
      <template v-slot:cell(Date)="data">
        <b>{{ data.value }}</b> <br />
        <h4>{{ data.item.Title }}</h4>
      </template>
      <template v-slot:cell()="data">
        <i>{{ data.value }}</i>
      </template>
      <template v-slot:cell(Thumbnail)="data">
        <b-img :src="Thumbnail(data)" fluid rounded='true' alt="Loading..." />
      </template>
    </b-table>
  </div>
</template>

<script lang='ts'>
import { Component, Prop, Watch, Vue } from 'vue-property-decorator';
import store from '../store';

@Component({
  store,
})
export default class AlbumView extends Vue {
  @Prop() private msg!: string;
  private itemx = store.state.albums;
  private busy = true;
  public data() {
    return {
      isBusy: this.busy,
      items: store.state.albums,
      thumbnail: store.state.thumbnail,
      sortDesc: true,
      images: [],
      fields: [
        // A column that needs custom formatting
        // { key: 'ISN', label: 'Index' },
        {
          key: 'Date',
          label: 'Informationen',
          sortable: true,
          formatter: (dt: any) => {
            if (dt === null) {
              return '';
            }
            const DD = ('0' + dt.getDate()).slice(-2);
            const MM = ('0' + (dt.getMonth() + 1)).slice(-2);
            const YYYY = dt.getFullYear();
            return DD + '.' + MM + '.' + YYYY;
          },
        },
        // 'Title',
        // { key: 'Title', label: 'Titel' },
        { key: 'Thumbnail', label: '' },
      ],
    };
  }
  private mounted() {
    const items = this.getItems();
    if (items.length < 2) {
      // Firefox need some time to omit NS_BINDING_ABORTED
      this.sleep(1000).then(() => {
        store.dispatch('INIT_ALBUMS', '');
      });
    }
    if (items) {
      this.syncImages();
    } else {
      store.dispatch('INIT_ALBUMS', '');
    }
  }
  private sleep(ms: any) {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }
  private Thumbnail(data: any) {
    const i = store.getters.getThumbnailByMd5(data.item.Thumbnail);
    if (i) {
      return i.src;
    }
    // if (this.images) {
    //   console.log('LEN: '+this.images.length)
    //   this.images.forEach((a,index)=>{if (!a) { console.log(index+'. '+a)}})
    //   const t = this.images.find((a) => a.md5 === data.item.Thumbnail);
    //   if (t) {
    //     return t.src
    //   }
    // }
    return '';
  }
  private getItems() {
    return this.itemx;
  }
  private setBusy(busy: boolean) {
    this.busy = busy;
  }
  private selected(item: any) {
    this.$router.push({ path: `/pictures/${item[0].ISN}` });
  }
  private syncImages() {
    this.setBusy(false);
    // this.items.forEach((element,index) => {
    //   let i = store.getters.getThumbnailByMd5(element.Thumbnail);
    //   if (i) {
    //     // Vue.set(this.images,index,i);
    //   }
    // });
  }
}
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style>
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
.preview-img-list {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
}
.preview-img-item {
  margin: 5px;
  max-width: 100px;
  max-height: 100px;
}
.pswp__caption .pswp__caption__center {
  font-family: Garamond;
  background: rgba(0, 0, 0, 0.3);
  font-size: 26px;
  padding-bottom: 30px;
}
.pswp__caption .pswp__caption__center small {
  font-size: 20px;
}
</style>

