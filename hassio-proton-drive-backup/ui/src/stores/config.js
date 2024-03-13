import { defineStore } from 'pinia'

export const useConfigStore = defineStore('config', {
    state: () => ({
        config: {

        },
    }),
    getters: {
    },
    actions: {
        async fetchConfig() {
            try {
                const response = await fetch('http://replaceme.homeassistant/api/config')
                if (!response.ok) {
                    throw new Error('Network response was not ok')
                }
                
                const data = await response.json()
                this.config = data
            } catch (error) {
                console.error(error)
            }
        }
    },
  })