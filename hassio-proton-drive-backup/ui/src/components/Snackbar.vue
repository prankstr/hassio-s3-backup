<template>
    <v-snackbar v-model="snackbar.visible" :color="snackbar.color" :timeout="snackbar.timeout" :multi-line="snackbar.multiLine">
        {{ snackbar.message }}

        <template v-slot:actions>
            <v-btn color="white" variant="text" @click="snackbar.hideMessage()">
                Close
            </v-btn>
        </template>
    </v-snackbar>
</template>

<script setup>
import { useSnackbarStore } from '@/stores/snackbar'

const snackbar = useSnackbarStore()
</script>