<template>
  <AppBar></AppBar>
  <v-container style="max-width: 1600px; min-width: 1100;" fluid class="fill-height">
    <v-responsive class="fill-height">

      <v-row class="align-start justify-center">
        <v-col class="pt-4" sm="11" md="4">
          <div class="d-flex flex-column align-center align-md-end text-end">
            <DriveStats class="mb-2"></DriveStats>
            <v-divider class="border-opacity-50" color="wite" length="375" vertical></v-divider>
            <NewBackup></NewBackup>
          </div>
        </v-col>
        <!-- <v-divider class="border-opacity-50" color="wite" vertical></v-divider> -->
        <v-col cols="auto" md="7" class="pt-4">
          <v-row class="justify-center justify-md-start">
            <v-col cols="11" md="6" v-for="(backup, i) in backups" :key="backup.id">
              <div class="d-flex flex-column align-center">
                <BackupCard :backup=backup></BackupCard>
              </div>
            </v-col>
          </v-row>
        </v-col>
      </v-row>

    </v-responsive>
  </v-container>

  <v-footer :app=true color="secondary">
    <strong>{{ new Date().getFullYear() }} © David Nilsson</strong>
  </v-footer>
</template>

<script setup>
import { ref, onMounted, reactive } from 'vue';

import AppBar from '@/components/AppBar.vue'
import DriveStats from '@/components/DriveStats.vue'
import NewBackup from '@/components/NewBackup.vue'
import BackupCard from '@/components/BackupCard.vue'

let backups = ref([])

// Function to fetch data and update backups ref
const fetchData = async () => {
  try {
    const response = await fetch('http://replaceme.homeassistant/api/backups');
    if (response.ok) {
      const data = await response.json();
      backups.value = data.map(backup => reactive(backup)); // Making each backup object reactive
    } else {
      console.error('Failed to fetch data');
    }
  } catch (error) {
    console.error(error);
  }
}

onMounted(() => {
  fetchData();
  setInterval(fetchData, 5000);
});
</script>
