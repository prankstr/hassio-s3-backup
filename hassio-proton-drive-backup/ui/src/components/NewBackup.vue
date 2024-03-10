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
			<v-btn style="max-width: 450px; width: 100%" class="mt-2" color="primary" append-icon="mdi-plus"
				v-bind="props" @click="generateBackupName">
				New
				<v-tooltip open-delay="400" location="bottom" activator="parent">Create new backup</v-tooltip>
			</v-btn>
		</template>
		<v-card class="pa-2" color="secondary">
			<v-card-title class="text-white">
				<span class="text-h5">New Backup</span>
			</v-card-title>
			<v-card-text class="text-white">
				<v-container>
					<v-row>
						<v-col cols="12">
							<v-text-field v-model=backupName class="mb-0" label="Backup Name" persistent-hint
								hint="The name of the backup. Falls back to the global naming schema if you leave it empty."></v-text-field>
						</v-col>
					</v-row>
				</v-container>
			</v-card-text>
			<template v-slot:actions>
				<v-spacer></v-spacer>
				<v-btn color="white" variant="text" @click="dialog = false">
					Exit
				</v-btn>
				<v-btn color="white" variant="text" @click="triggerBackup" :loading="loading">
					Create
				</v-btn>
			</template>
		</v-card>
	</v-dialog>
</template>
<script setup>
import { ref, onMounted } from 'vue'

const dialog = ref(false)
const loading = ref(false)
const snackbar = ref(false)
const backupName = ref("")
const backupNameFormat = ref("")
const emit = defineEmits(['backupCreated'])
const snackbarMsg = ref("Awesome! New backup created 🚀")

onMounted(() => {
	getConfig()
})

function generateBackupName() {
  let format = backupNameFormat.value || 'Full Backup {year}-{month}-{day} {hr24}:{min}:{sec}';
  const now = new Date(); // Uses the system's local timezone

  const replacements = {
    '{year}': String(now.getFullYear()),
    '{month}': String(now.getMonth() + 1).padStart(2, '0'), // Months are 0-indexed
    '{day}': String(now.getDate()).padStart(2, '0'),
    '{hr24}': String(now.getHours()).padStart(2, '0'),
    '{min}': String(now.getMinutes()).padStart(2, '0'),
    '{sec}': String(now.getSeconds()).padStart(2, '0'),
  };

  for (const [placeholder, value] of Object.entries(replacements)) {
    const placeholderRegex = new RegExp(placeholder, 'g');
    format = format.replace(placeholderRegex, value); // Accumulate replacements in format
  }

  backupName.value = format; // Assign the final value to the reactive reference
}

function getConfig() {
	fetch('http://replaceme.homeassistant/api/config')
		.then(res => res.json())
		.then(data => {
			backupNameFormat.value = data.backupNameFormat
		})
		.catch(err => console.log(err.message))
}

function triggerBackup() {
	loading.value = true

	fetch('http://replaceme.homeassistant/api/backups/new/full', {
		method: 'POST',
		body: JSON.stringify({
			"name": backupName.value
		})
	})
    	.then(response => {
    	    loading.value = false
    	    if (!response.ok) {
    	        return response.text().then(text => { throw new Error(text || 'Server returned an error') })
    	    } 
    	})
		.then(() =>{
			dialog.value = false
			snackbar.value = true
			emit("backupCreated")
		})
		.catch(error => {
			snackbarMsg.value = "⚠️ " + error
			snackbar.value = true
		});
}

</script>