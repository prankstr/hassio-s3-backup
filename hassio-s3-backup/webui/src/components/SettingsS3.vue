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
        label="Number of backups to keep remote"
        persistent-hint
        hint="The amount of backups to keep in the remote storage. 0 disables housekeeping."
      ></v-text-field>
    </v-col>
  </v-row>
  <v-row>
    <v-col cols="12" md="6">
      <v-text-field
        v-model.number="localConfig.backupInterval"
        type="number"
        class="mb-0"
        label="Time between backups"
        persistent-hint
        hint="The amount of time between backups. Defaults to 3 days."
      ></v-text-field>
    </v-col>
  </v-row>
</v-container>
<script setup>
import { ref, watch } from "vue";
import { useConfigStore } from "@/stores/config";
import { useSnackbarStore } from "@/stores/snackbar";

const cs = useConfigStore();
const snackbar = useSnackbarStore();

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
</script>
