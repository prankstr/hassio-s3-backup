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
          <div class="text-white text-subtitle-1">Drive:</div>
        </v-col>
        <v-col class="pr-0 pt-0">
          <div class="text-white text-subtitle-1">
            {{ bs.driveBackupsCount }} / {{ cs.config.backupsOnDrive }} ({{
              bs.driveBackupsSize
            }})
          </div>
        </v-col>
      </v-row>
      <v-row class="pt-0">
        <v-col cols="5" class="pr-0 pt-0">
          <div class="text-white text-subtitle-1">Home Assistant:</div>
        </v-col>
        <v-col class="pr-0 pt-0">
          <div class="text-white text-subtitle-1">
            {{ bs.haBackupsCount }} / {{ cs.config.backupsInHA }} ({{
              bs.haBackupsSize
            }})
          </div>
        </v-col>
      </v-row>
      <v-row class="pt-0">
        <v-col cols="5" class="pr-0 pt-0">
          <div class="text-white text-subtitle-1">Available Space:</div>
        </v-col>
        <v-col class="pr-0 pt-0">
          <div class="text-white text-subtitle-1">{{ availableGb }} GB</div>
        </v-col>
      </v-row>
      <v-row class="pt-0">
        <v-col cols="4" class="pr-0 pt-0">
          <div class="text-white text-subtitle-1">Used Space:</div>
        </v-col>
        <v-col class="py-2">
          <v-progress-linear
            :width="1"
            model-value="100"
            color="white"
            :height="12"
            rounded
            v-model="usedSpacePercent"
          >
            <strong class="text-white">{{ usedSpacePercent }}%</strong>
          </v-progress-linear>
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

const usedSpacePercent = computed(() => {
  let n = (about.value.UsedSpace / about.value.MaxSpace) * 100;
  return n.toFixed(2);
});

const availableGb = computed(() => {
  let n = (about.value.MaxSpace - about.value.UsedSpace) / 1024 / 1024 / 1024;
  return n.toFixed(2);
});
</script>

