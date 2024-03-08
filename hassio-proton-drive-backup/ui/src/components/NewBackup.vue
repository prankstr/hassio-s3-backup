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
					v-bind="props">
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
import { ref } from 'vue'

const dialog = ref(false)
const loading = ref(false)
const snackbar = ref(false)
const backupName = ref("")
const snackbarMsg = ref("Awesome! New backup created")


function triggerBackup() {
	loading.value = true

	fetch('http://replaceme.homeassistant/api/backups/new/full', {
		method: 'POST',
		body: JSON.stringify({
			"name": backupName.value
		})
	})
		.then(response => {
			console.log(response)
			loading.value = false
			dialog.value = false
			snackbar.value = true
		})
		.catch(error => {
			snackbarMsg.value = "Error when creating backup: " + error
			console.log(error)
		});
}

</script>