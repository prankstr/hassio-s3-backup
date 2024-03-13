import { defineStore } from 'pinia'

export const useBackupsStore = defineStore('backups', {
    state: () => ({
        backups: [],
    }),
    getters: {
        pinnedBackups(state) {
            console.log(state.backups)
            return state.backups.filter((backup) => backup.pinned)
        },
        driveBackups(state) {
            return state.backups.filter((backup) => backup.status === "DRIVEONLY" || backup.status === "SYNCED")
        },
        haBackups(state) {
            return state.backups.filter((backup) => backup.status === "HAONLY" || backup.status === "SYNCED")
        },
        driveBackupsCount() {
            return this.driveBackups.length
        },
        haBackupsCount() {
            return this.haBackups.length
        },
    },
    actions: {
        async fetchBackups() {
            try {
                const response = await fetch('http://replaceme.homeassistant/api/backups')
                if (!response.ok) {
                    throw new Error('Network response was not ok')
                }
                
                const data = await response.json()
                this.backups = data
            } catch (error) {
                console.error(error)
            }
        }
    },
})