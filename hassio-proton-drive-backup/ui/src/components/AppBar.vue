<template>
  <v-app-bar color="secondary" :elevation="1">
    <v-img class="mx-2, ml-4" src="@/assets/Drive-logomark-noborder.svg" max-height="40" max-width="40" contain></v-img>

    <v-app-bar-title>
      Proton Drive Backup
    </v-app-bar-title>

    <v-spacer></v-spacer>
    <v-tooltip open-delay="1000" location="bottom" :text="date">
      <template v-slot:activator="{ props }">
        <p v-bind="props">Next backup in {{ roundedTimer }}</p>
      </template>
    </v-tooltip>
    <v-divider vertical class="border-opacity-50"
      style="height: 40px; margin-top: 12px; margin-left: 18px;"></v-divider>
    <Settings></Settings>

  </v-app-bar>
</template>
<script setup>
import { ref, onMounted, computed } from 'vue'
import Settings from '@/components/Settings.vue'

let milliseconds = ref(0)

// Fetch timer in milliseconds until next backup
const fetchTimer = async () => {
  try {
    const response = await fetch('http://replaceme.homeassistant/api/backups/timer')
    if (response.ok) {
      const data = await response.json()
      milliseconds.value = data.milliseconds;
    } else {
      console.error('Failed to fetch data')
    }
  } catch (error) {
    console.error(error)
  }
}

const roundedTimer = computed(() => {
  const seconds = Math.floor(milliseconds.value / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.ceil(hours / 24)

  if (days === 1) {
    return "1 day";
  } else if (days > 1) {
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
    return `${seconds % 60} seconds`;
  }
})

const date = computed(() => {
  const months = ["January", "February", "March", "April", "May", "June",
    "July", "August", "September", "October", "November", "December"]
  const suffixes = ["th", "st", "nd", "rd"]

  const date = new Date(Date.now() + milliseconds.value);
  const day = date.getDate()
  const daySuffix = suffixes[(day % 10) - 1] || suffixes[0]

  const month = months[date.getMonth()]
  const hours = date.getHours().toString().padStart(2, '0')
  const minutes = date.getMinutes().toString().padStart(2, '0')

  return `${month} ${day}${daySuffix}, ${hours}:${minutes}`
})

onMounted(() => {
  // Fetches timer from server of milliseconds until next backup
  fetchTimer()

  // Keep calculating the timer on the client side
  setInterval(() => {
    if (milliseconds.value > 0) {
      milliseconds.value -= 1000
    }
  }, 1000)
})
</script>
<style>
.v-tooltip .v-overlay__content {
  background: rgba(var(--v-theme-primary), 1) !important;
}
</style>