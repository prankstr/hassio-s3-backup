<template>
  <AppBar></AppBar>
  <v-container style="max-width: 1600px; min-width: 1100;" fluid class="fill-height">
    <v-responsive class="fill-height">

      <v-row class="align-start justify-center">
        <v-col class="pt-4" sm="11" md="4">
          <div class="d-flex flex-column align-center align-md-end text-end">
            <DriveStats class="mb-2" :backups="bs.backups"></DriveStats>
           <!-- <BackupStats class="mb-2" :backups="backupList" :config="config"></BackupStats>
            <v-divider class="border-opacity-50" color="wite" length="375" vertical></v-divider> -->
            <NewBackup></NewBackup>
          </div>
        </v-col>
        <v-col cols="auto" md="7" class="pt-4">
          <v-row class="justify-center justify-md-start">
            <v-col cols="11" md="6" v-for="(backup, i) in bs.nonPinnedBackups" :key="backup.id">
              <div class="d-flex flex-column align-center">
                <BackupCard @backupsChanges=bs.fetchBackups :backup=backup></BackupCard>
              </div>
            </v-col>
          </v-row>

          <v-row v-if="bs.pinnedBackups.length > 0" class="justify-center justify-md-start">
            <v-col cols="11" class="pb-0">
              <div class="d-flex flex-column align-center align-md-start">
                <h2>Pinned Backups</h2>
              </div>
            </v-col>
            <v-col cols="11" md="6" v-for="(backup, i) in bs.pinnedBackups" key="backup.id">
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
import DriveStats from '@/components/DriveStats.vue'
import BackupStats from '@/components/BackupStats.vue'
import { useConfigStore } from '@/stores/config'
import { useBackupsStore } from '@/stores/backups'

const cs = useConfigStore()
const bs = useBackupsStore()

onMounted(() => {
  cs.fetchConfig()
  bs.fetchBackups()
})

</script>
