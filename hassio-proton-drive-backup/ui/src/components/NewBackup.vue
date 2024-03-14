<template>
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
				<v-btn color="white" variant="text" @click="triggerBackup">
					Create
				</v-btn>
			</template>
		</v-card>
	</v-dialog>
</template>
<script setup>
import { ref } from 'vue'
import { useConfigStore } from '@/stores/config'
import { useBackupsStore } from '@/stores/backups'
import { useSnackbarStore } from '@/stores/snackbar'

const dialog = ref(false)
const backupName = ref("")
const emit = defineEmits(['backupCreated'])

const cs = useConfigStore()
const bs = useBackupsStore()
const snackbar = useSnackbarStore()

function generateBackupName() {
  let format = cs.config.backupNameFormat || 'Full Backup {year}-{month}-{day} {hr24}:{min}:{sec}'
  const now = new Date() // Uses the system's local timezone

  const replacements = {
    '{year}': String(now.getFullYear()),
    '{month}': String(now.getMonth() + 1).padStart(2, '0'), // Months are 0-indexed
    '{day}': String(now.getDate()).padStart(2, '0'),
    '{hr24}': String(now.getHours()).padStart(2, '0'),
    '{min}': String(now.getMinutes()).padStart(2, '0'),
    '{sec}': String(now.getSeconds()).padStart(2, '0'),
  }

  for (const [placeholder, value] of Object.entries(replacements)) {
    const placeholderRegex = new RegExp(placeholder, 'g')
    format = format.replace(placeholderRegex, value) // Accumulate replacements in format
  }

  backupName.value = format // Assign the final value to the reactive reference
}

function triggerBackup() {
	bs.createBackup(backupName.value).then(({ success, error }) => {
        if (!success) {
			snackbar.show({message: "⚠️ error.message"})
        }
        
		dialog.value = false
		snackbar.show({message: "🚀 Awesome! New backup created"})
    })
}

</script>