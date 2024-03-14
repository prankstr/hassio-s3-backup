<template>
	<v-card max-width="450" width="100%" class="ml-0 mb-5 text-left" color="primary" variant="tonal">
		<v-card-item>
			<v-card-title class="text-white text-h6">Backups</v-card-title>
		</v-card-item>

		<v-card-text>
			<v-col>
				<v-row class="pt-0">
					<div class="text-white text-subtitle-1">Drive: {{ onDrive.length }} / {{ config.backupsOnDrive }}
					</div>
				</v-row>
				<v-row class="pt-0">
					<div class="text-white text-subtitle-1">HA: {{ inHA.length }} / {{ config.backupsInHA }}</div>
				</v-row>
			</v-col>
		</v-card-text>
	</v-card>
</template>
<script setup>
import { computed, defineProps } from 'vue'

const props = defineProps({
	backups: Array,
	config: Object
})

const onDrive = computed(() => {
	return props.backups.filter(backup => backup.status === "DRIVEONLY" || backup.status === "SYNCED")
})

const inHA = computed(() => {
	return props.backups.filter(backup => backup.status === "HAONLY" || backup.status === "SYNCED")
})

const usedByBackups = computed(() => {
	const totalSizeMB = props.backups.reduce((totalSize, backup) => totalSize + backup.size, 0)

	let displaySize
	let suffix

	if (totalSizeMB < 1000) {
		displaySize = totalSizeMB.toFixed(1)
		suffix = 'MB'
	} else {
		displaySize = (totalSizeMB / 1024).toFixed(1) // Convert MB to GB
		suffix = 'GB'
	}

	return `${displaySize} ${suffix}`
})
</script>