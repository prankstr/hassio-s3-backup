<template>
  <AppBar></AppBar>
  <Snackbar></Snackbar>
  <v-container style="max-width: 1600px; min-width: 1100;" fluid class="fill-height">
    <v-responsive class="fill-height">

      <v-row class="align-start justify-center">
        <v-col class="pt-4" sm="11" md="4">
          <div class="d-flex flex-column align-center align-md-end text-end">
            <SummaryCard class="mb-2"></SummaryCard>
            <!-- <v-divider class="border-opacity-50" color="wite" length="375" vertical></v-divider> -->
            <NewBackup></NewBackup>
          </div>
        </v-col>
        <v-col class="pt-4" cols="auto" md="7">
          <v-row class="align-center align-md-start">
            <v-col cols="12" md="6" v-for="(backup, i) in bs.nonPinnedBackups" :key="backup.id">
              <div class="d-flex flex-column align-center">
                <BackupCard @backupsChanges=bs.fetchBackups :backup=backup></BackupCard>
              </div>
            </v-col>
          </v-row>

          <v-row v-if="bs.pinnedBackups.length > 0" class="justify-center justify-md-start">
            <v-col cols="12" md="11" class="pb-0">
              <div class="d-flex flex-column align-center align-md-start">
                <h2>Pinned Backups</h2>
              </div>
            </v-col>
            <v-col cols="12" md="6" v-for="(backup, i) in bs.pinnedBackups" key="backup.id">
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
import { onMounted } from 'vue'

import AppBar from '@/components/AppBar.vue'
import NewBackup from '@/components/NewBackup.vue'
import BackupCard from '@/components/BackupCard.vue'
import SummaryCard from '@/components/SummaryCard.vue'
import Snackbar from '@/components/Snackbar.vue'
import { useConfigStore } from '@/stores/config'
import { useBackupsStore } from '@/stores/backups'

const cs = useConfigStore()
const bs = useBackupsStore()

onMounted(() => {
  cs.fetchConfig()
  bs.fetchBackups()

  setInterval(() => {
    bs.fetchBackups()
  }, 5000)
})

</script>
