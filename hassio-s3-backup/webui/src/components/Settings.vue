<template>
  <v-dialog v-model="dialog" width="1024">
    <template v-slot:activator="{ props }">
      <v-btn icon="mdi-cog" v-bind="props" class="mr-1 ml-1"></v-btn>
    </template>
    <v-card class="pa-2" color="secondary">
      <v-card-title class="text-white">
        <span class="text-h5">Settings</span>
      </v-card-title>
      <v-card-text class="text-white">
        <v-container>
          <v-row>
            <v-col cols="12">
              <v-text-field
                v-model="localConfig.backupNameFormat"
                class="mb-0"
                label="Name format"
                persistent-hint
                hint="Default: Full Backup {year}-{month}-{day} {hr24}:{min}:{sec}"
              ></v-text-field>
            </v-col>
            <v-col cols="12" md="6">
              <v-text-field
                v-model.number="localConfig.backupsInHA"
                type="number"
                class="mb-0"
                label="Number of backups to keep in Home Assistant"
                persistent-hint
                hint="The amount of backups to keep in Home Assistant. 0 disables house keeping"
              ></v-text-field>
            </v-col>
            <v-col cols="12" md="6">
              <v-text-field
                v-model.number="localConfig.backupsInStorage"
                type="number"
                class="mb-0"
                :label="`Number of backups to keep in S3`"
                persistent-hint
                :hint="`The amount of backups to keep in S3. 0 disables house keeping.`"
              ></v-text-field>
            </v-col>
          </v-row>
          <v-row>
            <v-col cols="12" md="6">
              <v-text-field
                v-model.number="localConfig.backupInterval"
                type="number"
                class="mb-0"
                label="Days between backups"
                persistent-hint
                hint="The amount of days between backups. Defaults to 3 days."
              ></v-text-field>
            </v-col>
          </v-row>
        </v-container>
      </v-card-text>
      <v-card-actions>
        <v-btn color="white" variant="outline" @click="revealResetData = true">
          Reset data
        </v-btn>
        <v-spacer></v-spacer>
        <v-btn color="white" variant="text" @click="dialog = false">
          Close
        </v-btn>
        <v-btn color="white" variant="text" @click="saveChanges"> Save </v-btn>
      </v-card-actions>
      <v-expand-transition>
        <v-card v-if="revealResetData" class="v-card--reveal" color="primary">
          <v-card-item>
            <v-card-title class="text-white text-heading-6"
              >Reset addon state</v-card-title
            >
          </v-card-item>
          <v-card-text style="height: 60px" class="pb-0">
            <p>
              Unexpected errors might cuase the addon state to become corrupt.
              Restting the addon state will remove all backups from the addon
              and trigger a sync which will re-add any backup found in Home
              Assistant and the remote storage. ⚠️ This is potentially
              destructive as any pinned backup will become unpinned and
              potentially removed. ⚠️
            </p>
          </v-card-text>
          <v-card-actions class="pb-0 align-end">
            <v-spacer></v-spacer>
            <v-btn
              density="comfortable"
              variant="text"
              color="white"
              @click="revealResetData = false"
            >
              Close
            </v-btn>
            <v-btn
              density="comfortable"
              variant="text"
              color="white"
              @click="resetData"
            >
              Accept
            </v-btn>
          </v-card-actions>
        </v-card>
      </v-expand-transition>
    </v-card>
  </v-dialog>
</template>
<script setup>
import { ref, watch } from "vue";
import { useConfigStore } from "@/stores/config";
import { useBackupsStore } from "@/stores/backups";
import { useSnackbarStore } from "@/stores/snackbar";

const cs = useConfigStore();
const bs = useBackupsStore();
const snackbar = useSnackbarStore();

const show = ref(false);
const dialog = ref(false);
const revealResetData = ref(false);
const localConfig = ref({});

watch(dialog, (newVal) => {
  if (newVal === true) {
    localConfig.value = JSON.parse(JSON.stringify(cs.config));
  }
});

function saveChanges() {
  cs.saveConfig(localConfig.value).then(({ success, error }) => {
    if (!success) {
      snackbar.show({ message: "⚠️ error.message" });
      return;
    }

    dialog.value = false;
    snackbar.show({ message: "🔥 Config updated" });
    emit("settingsUpdated");
  });
}

function resetData() {
  bs.resetData().then(({ success, error }) => {
    if (!success) {
      snackbar.show({ message: "⚠️ error.message" });
      return;
    }
    revealResetData.value = false;
    dialog.value = false;
    snackbar.show({ message: "Addon state has been reset" });
    emit("settingsUpdated");
  });
}
</script>
