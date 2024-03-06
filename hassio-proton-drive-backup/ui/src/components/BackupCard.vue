<template>
	<v-card max-width="450" width="100%" class="pr-0" color="primary" variant="tonal" :loading="loading">
		<template v-slot:loader="{ isActive }">
			<v-progress-linear :active="isActive" color="primary" height="4" indeterminate></v-progress-linear>
		</template>

		<v-card-item>
			<v-card-title class="text-white text-heading-6">{{ backup.name }}</v-card-title>
		</v-card-item>

		<v-card-text>
			<v-row class="pb-0">
				<v-col>
					<div class="text-white text-body-1">{{ translateStatus(backup.status) }}</div>
					<div class="text-white text-body-1">{{ translateSize(backup.size) }}</div>
				</v-col>
			</v-row>

		</v-card-text>
		<v-card-actions>
			<v-spacer></v-spacer>
			<v-tooltip open-delay="1000" location="bottom" text="Delete backup">
  				<template v-slot:activator="{ props }">
					<v-btn v-bind="props" density="comfortable" color="white" variant="text" icon="mdi-delete"
				@click="revealDelete = true"></v-btn>
				</template>
			</v-tooltip>
			<v-tooltip open-delay="1000" location="bottom" text="Restore to this backup">
  				<template v-slot:activator="{ props }">
					<v-btn v-bind="props" density="comfortable" color="white" variant="text" icon="mdi-backup-restore"
				@click="revealRestore = true"></v-btn>
				</template>
			</v-tooltip>
			<v-btn density="comfortable" color="white" variant="text" icon="mdi-download" @click="reveal = true"></v-btn>
		</v-card-actions>
		<v-expand-transition>
			<v-card v-if="revealRestore" class="v-card--reveal" color="primary">
				<v-card-item>
					<v-card-title class="text-white text-heading-6">Restore backup?</v-card-title>
				</v-card-item>
				<v-card-text style="height: 60px" class="pb-0">
					<p>This will do a full restore of Home Assistant to the backup "{{ backup.name }}"</p>
				</v-card-text>
				<v-card-actions class="pb-0 align-end">
					<v-spacer></v-spacer>
					<v-btn density="comfortable" variant="text" color="white" @click="revealRestore = false">
						Close
					</v-btn>
					<v-btn density="comfortable" variant="text"  color="white" @click="restoreBackup">
						Accept
					</v-btn>
				</v-card-actions>
			</v-card>
			<v-card v-if="revealDelete" class="v-card--reveal" color="primary">
				<v-card-item>
					<v-card-title class="text-white text-heading-6">Delete backup?</v-card-title>
				</v-card-item>
				<v-card-text style="height: 60px" class="pb-0">
					<p>This will delete the backup "{{ backup.name }}" from Home Assistant and Proton Drive</p>
				</v-card-text>
				<v-card-actions class="pb-0 align-end">
					<v-spacer></v-spacer>
					<v-btn density="comfortable" variant="text" color="white" @click="revealDelete = false">
						Close
					</v-btn>
					<v-btn density="comfortable" variant="text"  color="white" @click="deleteBackup">
						Accept
					</v-btn>
				</v-card-actions>
			</v-card>
		</v-expand-transition>
	</v-card>
</template>
<script setup>
import { ref, watch, defineProps } from 'vue'

const loading = ref(false)
const revealRestore = ref(false)
const revealDelete = ref(false)

const props = defineProps({
	backup: Object
})

const translateSize = size => {
  const roundedSize = Math.round((size < 1000 ? size : size / 1024) * 10) / 10
  const suffix = size < 1000 ? 'MB' : 'GB'
  return `${roundedSize} ${suffix}`
}

const translateStatus = (status) => {
	const statusMessages = {
		'COMPLETED': 'Completed',
		'DELETING': 'Deleting',
		'RUNNING': 'In Progress',
		'SYNCING': 'Uploading',
		'FAILED': 'Failed',
	}

	return statusMessages[status] || 'Unknown'
}

watch(() => props.backup.status, (status) => {
	console.log("Backup status changed to:", status)
	loading.value = status !== 'COMPLETED' && status !== 'FAILED'
}, { immediate: true })

function deleteBackup() {
	revealDelete.value = false
	loading.value = true

	fetch('http://replaceme.homeassistant/api/backups/delete', {
		method: 'POST',
		body: JSON.stringify({
			"id": props.backup.id
		})
	})
		.then(response => {
			console.log(response)
		})
		.catch(error => {
			console.log(error)
		})
}

function restoreBackup() {
	revealRestore.value = false
	loading.value = true

	fetch('http://replaceme.homeassistant/api/backups/restore', {
		method: 'POST',
		body: JSON.stringify({
			"slug": props.backup.slug
		})
	})
		.then(response => {
			console.log(response)
		})
		.catch(error => {
			console.log(error)
		})
}

</script>
<style>
.v-card--reveal {
	bottom: 0;
	opacity: 1 !important;
	position: absolute;
	width: 100%;
	height: 100%;
}

.v-tooltip .v-overlay__content {
    background: rgba(var(--v-theme-primary), 1) !important;
}
</style>