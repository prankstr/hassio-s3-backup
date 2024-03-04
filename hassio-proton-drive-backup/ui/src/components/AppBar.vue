<template>
  <v-app-bar color="secondary" :elevation="1">
    <v-img class="mx-2" src="@/assets/Drive-logomark-noborder.svg" max-height="40" max-width="40" contain></v-img>

    <v-app-bar-title>
      Proton Drive Backup
    </v-app-bar-title>

    <v-spacer></v-spacer>

    <Settings></Settings>

  </v-app-bar>
</template>
<script setup>
import Settings from '@/components/Settings.vue';
</script>