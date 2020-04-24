<template>
  <div class="pictures">
     <MyHeader></MyHeader>
     <Carousel :id="$route.params.id" />
 </div>
</template>

<script lang="ts">
import { Component, Vue } from 'vue-property-decorator';
import Carousel from '@/components/Carousel.vue'; // @ is an alias to /src
import MyHeader from '@/components/Header.vue';

@Component({
  components: {
    Carousel,
    MyHeader,
  },
})
export default class Pictures extends Vue {}
</script>
