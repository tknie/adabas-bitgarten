<template>
  <div class="registry">
    <MyHeader></MyHeader>
    <Thumbs></Thumbs>
  </div>
</template>

<script lang="ts">
import { Component, Vue } from 'vue-property-decorator';
import MyHeader from '@/components/Header.vue';
import Thumbs from '@/components/Thumbs.vue';

@Component({
  components: {
    MyHeader,
    Thumbs,
  },
})
export default class Registry extends Vue {}
</script>
