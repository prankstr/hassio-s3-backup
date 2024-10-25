<template>
  <v-card
    max-width="450"
    width="100%"
    class="ml-0 text-left"
    color="primary"
    variant="tonal"
  >
    <v-card-item>
      <v-card-title class="text-white text-h4 mb-4 mt-2">Summary</v-card-title>
    </v-card-item>

    <v-card-text>
      <v-row class="pl-3">
        <div class="text-white font-weight-bold text-subtitle-1">S3:</div>
      </v-row>
      <v-row class="mt-0">
        <v-col cols="3" class="pr-0">
          <div class="text-white text-body-2 pl-3">Backups:</div>
          <div
            v-if="bs.pinnedS3BackupsCount > 0"
            class="text-white text-body-2 pl-3"
          >
            Pinned:
          </div>
          <div
            v-if="bs.pinnedS3BackupsCount > 0"
            class="text-white text-body-2 pl-3"
          >
            Total:
          </div>
        </v-col>
        <v-col cols="2" class="pl-0 pr-0">
          <div class="text-white text-body-2">
            {{ bs.s3BackupsCount }} / {{ backupsInS3 }}
          </div>
          <div
            v-if="bs.pinnedS3BackupsCount > 0"
            class="text-white text-body-2"
          >
            {{ bs.pinnedS3BackupsCount }}
          </div>
          <div
            v-if="bs.pinnedS3BackupsCount > 0"
            class="text-white text-body-2"
          >
            {{ bs.totalS3BackupsCount }}
          </div>
        </v-col>
        <v-col class="pl-0">
          <div class="text-white text-body-2">({{ bs.s3BackupsSize }})</div>
          <div
            v-if="bs.pinnedS3BackupsCount > 0"
            class="text-white text-body-2"
          >
            ({{ bs.pinnedS3BackupsSize }})
          </div>

          <div
            v-if="bs.pinnedS3BackupsCount > 0"
            class="text-white text-body-2"
          >
            ({{ bs.totalS3BackupsSize }})
          </div>
        </v-col>
      </v-row>
      <v-row class="pl-3">
        <div class="text-white font-weight-bold text-subtitle-1">
          Home Assistant:
        </div>
      </v-row>
      <v-row class="mt-0">
        <v-col cols="3" class="pr-0">
          <div class="text-white text-body-2 pl-3">Backups:</div>
          <div
            v-if="bs.pinnedHABackupsCount > 0"
            class="text-white text-body-2 pl-3"
          >
            Pinned:
          </div>
          <div
            v-if="bs.pinnedHABackupsCount > 0"
            class="text-white text-body-2 pl-3"
          >
            Total:
          </div>
        </v-col>
        <v-col cols="2" class="pl-0 pr-0">
          <div class="text-white text-body-2">
            {{ bs.haBackupsCount }} / {{ backupsInHA }}
          </div>
          <div
            v-if="bs.pinnedHABackupsCount > 0"
            class="text-white text-body-2"
          >
            {{ bs.pinnedHABackupsCount }}
          </div>
          <div
            v-if="bs.pinnedHABackupsCount > 0"
            class="text-white text-body-2"
          >
            {{ bs.totalHABackupsCount }}
          </div>
        </v-col>
        <v-col class="pl-0">
          <div class="text-white text-body-2">({{ bs.haBackupsSize }})</div>
          <div
            v-if="bs.pinnedHABackupsCount > 0"
            class="text-white text-body-2"
          >
            ({{ bs.pinnedHABackupsSize }})
          </div>

          <div
            v-if="bs.pinnedHABackupsCount > 0"
            class="text-white text-body-2"
          >
            ({{ bs.totalHABackupsSize }})
          </div>
        </v-col>
      </v-row>
      <v-row v-if="backupsInHA > 0 || backupsInS3 > 0" class="pt-0 mt-0">
        <v-col cols="auto" class="pr-1">
          <div
            v-tooltip:right.contained="status.tooltip"
            :class="['text-subtitle-1', status.textColor]"
          >
            {{ status.message }}

            <v-icon
              :icon="status.icon"
              :color="status.iconColor"
              class="pb-1"
              size="20"
            ></v-icon>
          </div>
        </v-col>
      </v-row>
    </v-card-text>
  </v-card>
</template>
<script setup>
import { ref, onMounted, computed, defineProps } from "vue";
import { useBackupsStore } from "@/stores/backups";
import { useConfigStore } from "@/stores/config";

const bs = useBackupsStore();
const cs = useConfigStore();

const about = ref({});

const props = defineProps({
  backups: Array,
});

onMounted(() => {
  fetch("http://replaceme.homeassistant/api/drive/about")
    .then((res) => res.json())
    .then((data) => (about.value = data))
    .catch((err) => console.log(err.message));
});

const backupsInS3 = computed(() => {
  if (cs.config.backupsInS3 == 0) {
    return "∞";
  }

  return cs.config.backupsInS3;
});

const backupsInHA = computed(() => {
  if (cs.config.backupsInHA == 0) {
    return "∞";
  }

  return cs.config.backupsInHA;
});

const status = computed(() => {
  let statusObj = {
    icon: "mdi-check-circle-outline",
    message: "All good",
    iconColor: "green",
    textColor: "text-green",
    tooltip:
      "The amount of backups in S3 and Home Assistant match the configured amount",
  };

  if (cs.config.backupsInHA > 0 && cs.config.backupsInS3 > 0) {
    if (
      bs.s3BackupsCount != cs.config.backupsInS3 ||
      bs.haBackupsCount != cs.config.backupsInHA
    ) {
      if (bs.s3BackupsCount < 1) {
        return {
          icon: "mdi-alert-decagram-outline",
          message: "No backups in S3",
          iconColor: "red",
          textColor: "text-red",
          tooltip: "No backups in S3, please create one",
        };
      } else {
        return {
          icon: "mdi-alert-circle-outline",
          message: "Mismatch",
          iconColor: "orange",
          textColor: "text-orange",
          tooltip:
            "If you just enabled this addon or increased the amount of backups, the mismatch will resolve itself when the new backups are created",
        };
      }
    }
  }

  return statusObj;
});
</script>
