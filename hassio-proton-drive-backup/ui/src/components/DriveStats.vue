<template>
	<v-card max-width="450" width="100%" class="ml-0 text-left" color="primary" variant="tonal">
		<v-card-item>
			<v-card-title class="text-white text-h4">Drive</v-card-title>
		</v-card-item>

		<v-card-text>
			<v-row class="pb-0">
				<v-col cols="4" class="pr-0">
					<div class="text-white text-subtitle-1">Used Space:</div>
				</v-col>
				<v-col class="py-5">
					<v-progress-linear :width="1" model-value="100" color="white" :height="12" rounded
						v-model="usedSpacePercent">
						<strong class="text-white">{{ usedSpacePercent }}%</strong>
					</v-progress-linear>
				</v-col>
			</v-row>
			<v-row class="pt-0">
				<v-col cols="5" class="pr-0 pt-0">
					<div class="text-white text-subtitle-1">Backups:</div>
				</v-col>
				<v-col class="pr-0 pt-0">
					<div class="text-white text-subtitle-1">{{ usedByBackups }}</div>
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
		</v-card-text>
	</v-card>
</template>
<script setup>
import { ref, onMounted, computed, defineProps } from 'vue'

const about = ref({})

const props = defineProps({
	backups: Array
})

onMounted(() => {
	fetch('http://replaceme.homeassistant/api/drive/about')
		.then(res => res.json())
		.then(data => about.value = data)
		.catch(err => console.log(err.message))
})

const usedByBackups = computed(() => {
	const roundedSize = props.backups.reduce((totalSize, backup) => totalSize + backup.size, 0)
  	const suffix = roundedSize < 1000 ? 'MB' : 'GB'
	return `${roundedSize} ${suffix}`
})

const usedSpacePercent = computed(() => {
	let n = about.value.UsedSpace / about.value.MaxSpace * 100
	return n.toFixed(2)
})

const availableGb = computed(() => {
	let n = (about.value.MaxSpace - about.value.UsedSpace) / 1024 / 1024 / 1024
	return n.toFixed(2)
})
</script>