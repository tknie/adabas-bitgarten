<template>
  <div class="registry">
    <MyHeader></MyHeader>
    <Thumbnails></Thumbnails>
  </div>
</template>

<script lang="ts">
import { Component, Vue } from 'vue-property-decorator';
import MyHeader from '@/components/Header.vue';
import Thumbnails from '@/components/Thumbnails.vue';

@Component({
  components: {
    MyHeader,
    Thumbnails,
  },
})
export default class Registry extends Vue {}
</script>
