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
      <v-row class="pt-0">
        <v-col cols="5" class="pr-0 pt-0">
          <div class="text-white text-subtitle-1">Home Assistant:</div>
        </v-col>
        <v-col class="pr-0 pt-0">
          <div class="text-white text-subtitle-1">
            {{ bs.haBackupsCount }} / {{ backupsInHA }} ({{ bs.haBackupsSize }})
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
</script>
