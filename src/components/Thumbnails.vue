<!--
 * Copyright (c) 2020 Software AG (http://www.softwareag.com/)
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
  <div class="thumbnails">
    <b-container
      ><b-row><b-col>xxxx</b-col><b-col>yyyy</b-col></b-row
      ><b-row><b-col>xxxx</b-col><b-col>yyyy</b-col></b-row></b-container
    >
    <b-container class="mb-3 w-100">
      <b-row cols="2"
        ><b-col sm="6">AAAA</b-col><b-col sm="6">BBBB</b-col>
        <b-col sm="6">
          <!--b-table small ref="picTable" :items="records" :fields="fields">
            <template v-slot:cell(pic)="data">
              <img
                :src="data.item.pic"
                class="rounded"
                :alt="'Not available'"
              />
            </template>
          </b-table-->Test
        </b-col></b-row
      ><b-row
        ><b-col sm="6"
          >Test
          <!--b-table small ref="picTable2" :items="records" :fields="fields">
            <template v-slot:cell(pic)="data">
              <img
                :src="data.item.pic"
                class="rounded"
                :alt="'Not available'"
              />
            </template>
          </b-table-->
        </b-col></b-row
      >
    </b-container>
  </div>
</template>

<script lang="ts">
import { Component, Prop, PropSync, Vue, Watch } from 'vue-property-decorator';
import { authHeader } from '../auth-header';
import { config } from '../config';
import axios from 'axios';
import store from '../store';

@Component
export default class Thumbnails extends Vue {
  @Prop() private msg!: string;
  public data() {
    return {
      thumbnail: store.state.thumbnail,
      numberDbs: 0,
      numberMaps: 0,
      offset: 0,
      records: null,
      fields: ['ISN', 'pic'],
      numbers: [1, 2, 3, 4, 5],
    };
  }
  private created() {
    console.log('Create thumbnail component');
    const getConfig = {
      headers: authHeader('application/json'),
      useCredentails: true,
    };
    console.log('Init receiving Albums');
    axios
      .get(
        config.Url() +
          '/rest/map/PictureMetadata?limit=10&offset=' +
          this.$data.offset,
        getConfig,
      )
      .then((response: any) => {
        console.log('RESPONSE: ' + JSON.stringify(response));
        this.$data.records = response.data.Records;
        this.$data.records.forEach((element: any) => {
          this.callThumbnail(element);
        });
      });
  }
  private callThumbnail(element: any) {
    store.dispatch('LOAD_THUMB', element.Md5).then((t: any) => {
      if (t) {
        console.log('K ' + t.md5);
        element.pic = t.src;
        (this.$refs.picTable as any).refresh();
      }
    });
  }
  @Watch('thumbnail')
  private onThumbnailChanged(value: any, oldValue: any) {
    console.log('Changed thumbnail: ' + JSON.stringify(value));
    this.$data.records.forEach((element: any, index: number) => {
      if (!element.pic || element.pic === '') {
        const x = store.getters.getThumbnailByMd5(element.Md5);
        if (x) {
          console.log(index + ' X found ' + element.Md5);
          element.pic = x.src;
        } else {
          console.log(index + ' X not found ' + element.Md5);
        }
      }
    });
    (this.$refs.picTable as any).refresh();
  }
  private recordThumbs(index: number): string {
    console.log('Check ' + index + ' on ' + this.$data.records.length);
    if (this.$data.records && index < this.$data.records.length) {
      if (this.$data.records[index].pic) {
        return this.$data.records[index].pic;
      }
    }
    return '';
  }
}
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
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
</style>
