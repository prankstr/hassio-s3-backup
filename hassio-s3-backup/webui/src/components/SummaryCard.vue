<template>
  <v-card
    max-width="450"
    width="100%"
    class="ml-0 text-left"
    color="primary"
    variant="tonal"
  >
    <v-card-item>
      <v-card-title class="text-white text-h4 pb-4">Summary</v-card-title>
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
      <v-row class="pt-0 mt-0">
        <v-col cols="auto" class="pr-1">
          <div :class="['text-subtitle-1', status.textColor]">
            {{ status.message }}
          </div>
        </v-col>
        <v-col cols="1" class="pl-0">
          <v-icon
            :icon="status.icon"
            :color="status.iconColor"
            size="20"
            class="pt-1"
          ></v-icon>
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
  };

  if (
    bs.s3BackupsCount != cs.config.backupsInS3 ||
    bs.haBackupsCount != cs.config.backupsInHA
  ) {
    if (bs.backupsInS3 < 1) {
      return {
        icon: "mdi-alert-decagram-outline",
        message: "No backups in S3",
        iconColor: "red",
        textColor: "text-red",
      };
    } else {
      return {
        icon: "mdi-alert-circle-outline",
        message: "Mismatch",
        iconColor: "orange",
        textColor: "text-orange",
      };
    }
  }

  return statusObj;
});
</script>
