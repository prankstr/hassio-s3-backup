import { defineStore } from "pinia";

export const useConfigStore = defineStore("config", {
  state: () => ({
    config: {},
  }),
  getters: {},
  setters: {},
  actions: {
    async fetchConfig() {
      try {
        const response = await fetch(
          "http://replaceme.homeassistant/api/config",
        );
        if (!response.ok) {
          throw new Error("Network response was not ok");
        }

        const data = await response.json();
        this.config = data;
      } catch (error) {
        console.error(error);
      }
    },
    async saveConfig(config) {
      try {
        const response = await fetch(
          "http://replaceme.homeassistant/api/config/update",
          {
            method: "POST",
            body: JSON.stringify(config),
          },
        );

        if (response.status === 200) {
          this.config = config;
          return { success: true };
        } else {
          throw new Error("Failed to save configuration");
        }
      } catch (error) {
        console.error("Failed to save configuration:", error);
        return { success: false, error: error };
      }
    },
  },
});

