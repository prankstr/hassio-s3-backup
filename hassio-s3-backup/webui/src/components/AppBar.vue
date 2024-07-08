<template>
  <v-app-bar color="secondary" :elevation="1">
    <v-app-bar-title> S3 Backup </v-app-bar-title>

    <v-spacer></v-spacer>

    <NextBackupTimer></NextBackupTimer>

    <v-divider
      vertical
      class="border-opacity-50"
      style="height: 40px; margin-top: 12px; margin-left: 18px"
    ></v-divider>

    <Settings></Settings>
  </v-app-bar>
</template>

<script setup>
import Settings from "@/components/Settings.vue";
import NextBackupTimer from "@/components/NextBackupTimer.vue";
</script>
