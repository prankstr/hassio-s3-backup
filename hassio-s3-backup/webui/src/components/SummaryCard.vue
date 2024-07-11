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
      <v-row class="pt-0">
        <v-col cols="5" class="pr-0 pt-0">
          <div class="text-white text-subtitle-1">S3</div>
        </v-col>
        <v-col class="pr-0 pt-0">
          <div class="text-white text-subtitle-1">
            {{ bs.s3BackupsCount }} / {{ backupsInS3 }} ({{ bs.s3BackupsSize }})
          </div>
        </v-col>
      </v-row>
      <v-row class="pt-0 mt-0">
        <v-col cols="5" class="pr-0 pt-0">
          <div class="text-white text-subtitle-1">Home Assistant:</div>
        </v-col>
        <v-col class="pr-0 pt-0">
          <div class="text-white text-subtitle-1">
            {{ bs.haBackupsCount }} / {{ backupsInHA }} ({{ bs.haBackupsSize }})
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
