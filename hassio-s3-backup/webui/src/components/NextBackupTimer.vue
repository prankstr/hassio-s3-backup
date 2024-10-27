<template>
  <v-tooltip
    v-model="showTooltip"
    open-delay="400"
    location="bottom"
    :text="date"
  >
    <template v-slot:activator="{ props }">
      <p
        @click="showTooltip = !showTooltip"
        v-if="milliseconds > 0"
        v-bind="props"
      >
        Next backup in {{ roundedTimer }}
      </p>

      <!-- Just assuming everything is going alright here.. :) -->
      <p v-else v-bind="props">Backup in progress</p>
    </template>
  </v-tooltip>
</template>
<script setup>
import { ref, onMounted, computed, watch } from "vue";
import { useConfigStore } from "@/stores/config";
import { useBackupsStore } from "@/stores/backups";

const cs = useConfigStore();
const bs = useBackupsStore();

const milliseconds = ref(0);
const showTooltip = ref(false);

// Fetch timer in milliseconds until next backup
const fetchTimer = async () => {
  try {
    const response = await fetch(
      "http://replaceme.homeassistant/api/backups/timer",
    );
    if (response.ok) {
      const data = await response.json();
      milliseconds.value = data.milliseconds;
    } else {
      console.error("Failed to fetch data");
    }
  } catch (error) {
    console.error(error);
  }
};

const roundedTimer = computed(() => {
  const seconds = Math.floor(milliseconds.value / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.ceil(hours / 24);

  if (days > 1) {
    return `${days} days`;
  } else if (hours % 24 === 1) {
    return "1 hour";
  } else if (hours % 24 > 1) {
    return `${hours % 24} hours`;
  } else if (minutes % 60 === 1) {
    return "1 minute";
  } else if (minutes % 60 > 1) {
    return `${minutes % 60} minutes`;
  } else if (seconds % 60 === 1) {
    return "1 second";
  } else {
    return `now`;
  }
});

const date = computed(() => {
  const months = [
    "January",
    "February",
    "March",
    "April",
    "May",
    "June",
    "July",
    "August",
    "September",
    "October",
    "November",
    "December",
  ];

  function getDaySuffix(day) {
    if (day % 100 >= 11 && day % 100 <= 13) {
      return "th";
    }
    switch (day % 10) {
      case 1:
        return "st";
      case 2:
        return "nd";
      case 3:
        return "rd";
      default:
        return "th";
    }
  }

  const currentDate = new Date(Date.now() + milliseconds.value);
  const day = currentDate.getDate();
  const daySuffix = getDaySuffix(day);

  const month = months[currentDate.getMonth()];
  const hours = currentDate.getHours().toString().padStart(2, "0");
  const minutes = currentDate.getMinutes().toString().padStart(2, "0");

  return `${month} ${day}${daySuffix}, ${hours}:${minutes}`;
});

watch(() => cs.config.backupInterval, fetchTimer);
watch(() => bs.backups.length, fetchTimer);

onMounted(() => {
  fetchTimer();
});
</script>

<style>
.v-tooltip .v-overlay__content {
  background: rgba(var(--v-theme-primary), 1) !important;
}
</style>
