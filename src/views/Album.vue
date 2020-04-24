<template>
  <div class="home">
    <MyHeader></MyHeader>
    <AlbumView></AlbumView>
  </div>
</template>

<script lang="ts">
import { Component, Vue } from 'vue-property-decorator';
import MyHeader from '@/components/Header.vue';
import AlbumView from '@/components/AlbumView.vue';

@Component({
  components: {
    MyHeader,
    AlbumView,
  },
})
export default class Home extends Vue {}
</script>
