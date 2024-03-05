<template>
    <v-snackbar color="primary" multi-line :timeout="2500" v-model="snackbar">
        {{ snackbarMsg }}

        <template v-slot:actions>

            <v-btn color="white" variant="text" @click="snackbar = false">
                Close
            </v-btn>
        </template>
    </v-snackbar>
    <v-dialog v-model="dialog" width="1024">
        <template v-slot:activator="{ props }">
            <v-btn icon="mdi-cog" v-bind="props"></v-btn>
        </template>
        <v-card class="pa-2" color="secondary">
            <v-card-title class="text-white">
                <span class="text-h5">Settings</span>
            </v-card-title>
            <v-card-text class="text-white">
                <v-container>
                    <v-row>
                        <v-col cols="6">
                            <v-text-field v-model=backupsToKeep class="mb-0" label="Number of backups" persistent-hint
                                hint="The amount of backups to keep. Defaults to 4"></v-text-field>
                        </v-col>
                        <v-col cols="6">
                            <v-text-field v-model=backupInterval class="mb-0" label="Time between backups" persistent-hint
                                hint="The amount of time between backups. Defaults to 3 days."></v-text-field>
                        </v-col>
                    </v-row>
                </v-container>
            </v-card-text>
            <template v-slot:actions>
                <v-spacer></v-spacer>
                <v-btn color="white" variant="text" @click="dialog = false">
                    Close
                </v-btn>
                <v-btn color="white" variant="text" @click="updateConfig">
                    Save
                </v-btn>
            </template>
        </v-card>
    </v-dialog>
</template>
<script setup>
import { ref, onMounted, watch } from 'vue'

const dialog = ref(false)
const backupInterval = ref(0)
const backupsToKeep = ref(0)
const snackbar = ref(false)
const snackbarMsg = ref("Config updated")

onMounted(() => {
    getConfig()
})

watch(dialog, (open) => {
  if (open) {
    getConfig()
  }
})

function getConfig() {
    fetch('http://replaceme.homeassistant/api/config')
		.then(res => res.json())
		.then(data => {
            backupInterval.value = data.backupInterval
            backupsToKeep.value = data.backupsToKeep
        })
		.catch(err => console.log(err.message))
}

function updateConfig() {
    fetch('http://replaceme.homeassistant/api/config/update', {
        method: 'POST',
        body: JSON.stringify({
            "backupInterval": parseInt(backupInterval.value, 10),
            "backupsToKeep": parseInt(backupsToKeep.value, 10)
        })
    })
        .then(response => {
            dialog.value = false
            snackbar.value = true
        })
        .catch(error => {
            snackbarMsg.value = "Error when updating config: " + error
            console.log(error)
        });
}

</script>