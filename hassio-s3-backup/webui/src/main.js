/**
 * main.js
 *
 * Bootstraps Vuetify and other plugins then mounts the App`
 */

// Composables
import { createApp } from "vue";
import { createPinia } from "pinia";

// Plugins
import { registerPlugins } from "@/plugins";

// Components
import App from "./App.vue";

const app = createApp(App);
const pinia = createPinia();

registerPlugins(app);
app.use(pinia);

app.mount("#app");
