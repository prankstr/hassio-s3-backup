import { defineStore } from 'pinia'

export const useBackupsStore = defineStore('backups', {
    state: () => ({
        backups: [],
    }),
    getters: {
        pinnedBackups(state) {
            return state.backups.filter((backup) => backup.pinned)
        },
        nonPinnedBackups(state) {
            return state.backups.filter((backup) => !backup.pinned)
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
        },
        async createBackup(name) {
            try {
                const response = await fetch('http://replaceme.homeassistant/api/backups/new/full', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        "name": name
                    })
                })

                if (response.status === 202) {
                    this.fetchBackups()
                    return { success: true }
                } else {
                    throw new Error('Failed to create backup')
                }
            } catch (error) {
                console.error('Failed to create backup:', error)
                return { success: false, error: error }
            }
        },
        async deleteBackup(id) {
            try {
                const response = await fetch(`http://replaceme.homeassistant/api/backups/${id}`, {
                    method: 'DELETE'
                })

                if (response.status === 200) {
                    this.backups = this.backups.filter((backup) => backup.id !== id)
                    return { success: true }
                } else {
                    throw new Error('Failed to delete backup')
                } 
            } catch (error) {
                console.error('Failed to delete backup:', error)
                return { success: false, error: error }
            }
        },
        async pinBackup(id) {
            try {
                const response = await fetch(`http://replaceme.homeassistant/api/backups/${id}/pin`, {
                    method: 'POST'
                })

                if (response.status === 200) {
                    const backup = this.backups.find((backup) => backup.id === id)
                    backup.pinned = true
                    return { success: true }
                } else {
                    throw new Error('Failed to pin backup')
                }

            } catch (error) {
                console.error('Failed to pin backup:', error)
                return { success: false, error: error }
            }
        },
        async unpinBackup(id) {
            try {
                const response = await fetch(`http://replaceme.homeassistant/api/backups/${id}/unpin`, {
                    method: 'POST'
                })

                if (response.status === 200) {
                    const backup = this.backups.find((backup) => backup.id === id)
                    backup.pinned = false
                    return { success: true }
                } else {
                    throw new Error('Failed to unpin backup')
                }
            } catch (error) {
                console.error('Failed to unpin backup:', error)
                return { success: false, error: error }
            }
        },
        async resetData() {
            try {
                const response = await fetch('http://replaceme.homeassistant/api/backups/reset', {
                    method: 'POST'
                })

                if (response.status === 200) {
                    this.backup = [];
                    return { success: true }
                  } else {
                    throw new Error('Failed to reset data')
                  }
            } catch (error) {
                console.error('Failed to reset data:', error)
                return { success: false, error: error }
            }
        }
    },
})