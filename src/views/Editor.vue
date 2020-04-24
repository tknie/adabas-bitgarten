<template>
  <div class="editor">
     <MyHeader></MyHeader>
     <MyEditView></MyEditView>
 </div>
</template>

<script lang="ts">
import { Component, Vue } from 'vue-property-decorator';
import MyHeader from '@/components/Header.vue';
import MyEditView from '@/components/EditView.vue';

@Component({
  components: {
    MyHeader,
    MyEditView,
  },
})
export default class Editor extends Vue {}
</script>
