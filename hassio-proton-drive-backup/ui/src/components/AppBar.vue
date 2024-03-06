<template>
  <v-app-bar color="secondary" :elevation="1">
    <v-img class="mx-2, ml-4" src="@/assets/Drive-logomark-noborder.svg" max-height="40" max-width="40" contain></v-img>

    <v-app-bar-title>
      Proton Drive Backup
    </v-app-bar-title>

    <v-spacer></v-spacer>
    Next backup in {{timer}}
    <v-divider vertical class="border-opacity-50" style="height: 40px; margin-top: 12px; margin-left: 16px;"></v-divider>
    <Settings></Settings>

  </v-app-bar>
</template>
<script setup>
import { ref, onMounted } from 'vue'
import Settings from '@/components/Settings.vue';

let timer = ref(null)

// Function to fetch data and update backups ref
const fetchData = async () => {
  try {
    const response = await fetch('http://replaceme.homeassistant/api/backups/timer');
    if (response.ok) {
      const data = await response.json();
      timer.value = data.timer; 
    } else {
      console.error('Failed to fetch data');
    }
  } catch (error) {
    console.error(error);
  }
}

onMounted(() => {
  fetchData();
  setInterval(fetchData, 5000);
});
</script>