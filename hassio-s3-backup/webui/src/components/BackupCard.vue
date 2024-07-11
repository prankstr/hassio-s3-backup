<template>
  <v-card
    max-width="450"
    width="100%"
    class="pr-0"
    color="primary"
    variant="tonal"
    :loading="loading"
  >
    <template v-slot:loader="{ isActive }">
      <v-progress-linear
        :active="isActive"
        color="primary"
        height="4"
        indeterminate
      ></v-progress-linear>
    </template>

    <v-card-item>
      <v-card-title class="text-white text-heading-6">{{
        backup.name
      }}</v-card-title>
    </v-card-item>

    <v-card-text>
      <v-row class="pb-0">
        <v-col>
          <div v-if="backup.status == 'FAILED'" class="text-white text-body-1">
            {{ translateStatus(backup.status) }}
            <v-tooltip
              open-delay="400"
              location="bottom"
              :text="backup.errorMessage"
            >
              <v-icon
                icon="mdi-help-circle-outline"
                color="red"
                class="pb-1"
                size="20"
              ></v-icon>
            </v-tooltip>
          </div>
          <div v-else class="text-white text-body-1">
            {{ translateStatus(backup.status) }}
          </div>
          <div class="text-white text-body-1">
            {{ translateSize(backup.size) }}
          </div>
        </v-col>
      </v-row>
    </v-card-text>
    <v-card-actions>
      <v-tooltip
        v-if="!backup.pinned"
        open-delay="400"
        location="bottom"
        text="Pin this backup, keeping it indefinately in Home Assistant and S3"
      >
        <template v-slot:activator="{ props }">
          <v-btn
            v-bind="props"
            density="comfortable"
            color="white"
            variant="text"
            icon="mdi-pin"
            @click="pinBackup"
          ></v-btn>
        </template>
      </v-tooltip>
      <v-tooltip
        v-if="backup.pinned"
        open-delay="400"
        location="bottom"
        text="Unpin this backup"
      >
        <template v-slot:activator="{ props }">
          <v-btn
            v-bind="props"
            density="comfortable"
            color="green"
            variant="text"
            icon="mdi-pin"
            @click="unpinBackup"
          ></v-btn>
        </template>
      </v-tooltip>

      <v-spacer></v-spacer>

      <v-tooltip open-delay="400" location="bottom" text="Delete backup">
        <template v-slot:activator="{ props }">
          <v-btn
            v-bind="props"
            density="comfortable"
            color="white"
            variant="text"
            icon="mdi-delete"
            @click="revealDelete = true"
          ></v-btn>
        </template>
      </v-tooltip>
      <v-tooltip
        v-if="backup.status != 'S3ONLY'"
        open-delay="400"
        location="bottom"
        text="Restore to this backup"
      >
        <template v-slot:activator="{ props }">
          <v-btn
            v-bind="props"
            density="comfortable"
            color="white"
            variant="text"
            icon="mdi-backup-restore"
            @click="revealRestore = true"
          ></v-btn>
        </template>
      </v-tooltip>
      <v-tooltip
        v-if="backup.status === 'S3ONLY'"
        open-delay="400"
        location="bottom"
        text="Download backup to Home Assistant"
      >
        <template v-slot:activator="{ props }">
          <v-btn
            v-bind="props"
            density="comfortable"
            color="white"
            variant="text"
            icon="mdi-download"
            @click="revealDownload = true"
          ></v-btn>
        </template>
      </v-tooltip>
    </v-card-actions>
    <v-expand-transition>
      <v-card v-if="revealRestore" class="v-card--reveal" color="primary">
        <v-card-item>
          <v-card-title class="text-white text-heading-6"
            >Restore backup?</v-card-title
          >
        </v-card-item>
        <v-card-text style="height: 60px" class="pb-0">
          <p>
            This will do a full restore of Home Assistant to the backup "{{
              backup.name
            }}". For a partial restore please use the Home Assistant UI.
          </p>
        </v-card-text>
        <v-card-actions class="pb-0 align-end">
          <v-spacer></v-spacer>
          <v-btn
            density="comfortable"
            variant="text"
            color="white"
            @click="revealRestore = false"
          >
            Close
          </v-btn>
          <v-btn
            density="comfortable"
            variant="text"
            color="white"
            @click="restoreBackup"
          >
            Accept
          </v-btn>
        </v-card-actions>
      </v-card>
      <v-card v-if="revealDelete" class="v-card--reveal" color="primary">
        <v-card-item>
          <v-card-title class="text-white text-heading-6"
            >Delete backup?</v-card-title
          >
        </v-card-item>
        <v-card-text style="height: 60px" class="pb-0">
          <p>
            This will delete the backup "{{ backup.name }}" from Home Assistant
            and Proton Drive
          </p>
        </v-card-text>
        <v-card-actions class="pb-0 align-end">
          <v-spacer></v-spacer>
          <v-btn
            density="comfortable"
            variant="text"
            color="white"
            @click="revealDelete = false"
          >
            Close
          </v-btn>
          <v-btn
            density="comfortable"
            variant="text"
            color="white"
            @click="deleteBackup"
          >
            Accept
          </v-btn>
        </v-card-actions>
      </v-card>
      <v-card v-if="revealDownload" class="v-card--reveal" color="primary">
        <v-card-item>
          <v-card-title class="text-white text-heading-6"
            >Download Backup?</v-card-title
          >
        </v-card-item>
        <v-card-text style="height: 60px" class="pb-0">
          <p>Do you want to download this backup to Home Assistant?</p>
        </v-card-text>
        <v-card-actions class="pb-0 align-end">
          <v-spacer></v-spacer>
          <v-btn
            density="comfortable"
            variant="text"
            color="white"
            @click="revealDownload = false"
          >
            Close
          </v-btn>
          <v-btn
            density="comfortable"
            variant="text"
            color="white"
            @click="downloadBackup"
          >
            Accept
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-expand-transition>
  </v-card>
</template>
<script setup>
import { ref, watch, defineProps } from "vue";
import { useBackupsStore } from "@/stores/backups";
import { useSnackbarStore } from "@/stores/snackbar";

const bs = useBackupsStore();
const snackbar = useSnackbarStore();

const loading = ref(false);
const revealRestore = ref(false);
const revealDelete = ref(false);
const revealDownload = ref(false);
const emit = defineEmits(["change"]);

const props = defineProps({
  backup: Object,
});

const translateSize = (size) => {
  const roundedSize = Math.round((size < 1000 ? size : size / 1024) * 10) / 10;
  const suffix = size < 1000 ? "MB" : "GB";
  return `${roundedSize} ${suffix}`;
};

const translateStatus = (status) => {
  const statusMessages = {
    SYNCED: "Synced",
    HAONLY: "Only in HA",
    S3ONLY: "Only in S3",
    DELETING: "Deleting",
    RUNNING: "In Progress",
    SYNCING: "Uploading",
    FAILED: "Failed",
  };

  return statusMessages[status] || "Unknown";
};

watch(
  () => props.backup.status,
  (status) => {
    loading.value =
      status !== "SYNCED" &&
      status !== "FAILED" &&
      status !== "HAONLY" &&
      status !== "S3ONLY";
  },
  { immediate: true },
);

function deleteBackup() {
  revealDelete.value = false;
  loading.value = true;

  bs.deleteBackup(props.backup.id).then(({ success, error }) => {
    if (!success) {
      snackbar.show({ message: `⚠️ error: ${error}` });
      loading.value = false;
    }

    snackbar.show({ message: "Backup deleted" });
    return (loading.value = false);
  });
}

function restoreBackup() {
  revealRestore.value = false;
  loading.value = true;

  bs.restoreBackup(props.backup.id).then(({ success, error }) => {
    if (!success) {
      snackbar.show({ message: `⚠️ error: ${error}` });
      return (loading.value = false);
    }

    snackbar.show({ message: "Restore started" });
    return (loading.value = false);
  });
}

function downloadBackup() {
  revealDownload.value = false;
  loading.value = true;

  bs.downloadBackup(props.backup.id).then(({ success, error }) => {
    if (!success) {
      snackbar.show({ message: `⚠️ error: ${error}` });
      return (loading.value = false);
    }

    snackbar.show({ message: "Backup downloaded" });
    return (loading.value = false);
  });
}

function pinBackup() {
  bs.pinBackup(props.backup.id).then(({ success, error }) => {
    if (!success) {
      snackbar.show({ message: `⚠️ error: ${error}` });
      return;
    }

    snackbar.show({ message: "Backup pinned" });
  });
}

function unpinBackup() {
  bs.unpinBackup(props.backup.id).then(({ success, error }) => {
    if (!success) {
      snackbar.show({ message: `⚠️ error: ${error}` });
      return;
    }

    snackbar.show({ message: "Backup unpinned" });
  });
}
</script>
<style>
.v-card--reveal {
  bottom: 0;
  opacity: 1 !important;
  position: absolute;
  width: 100%;
  height: 100%;
}

.v-tooltip .v-overlay__content {
  background: rgba(var(--v-theme-primary), 1) !important;
}
</style>
