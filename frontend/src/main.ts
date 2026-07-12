import { createApp } from "vue";
import { createPinia } from "pinia";
import App from "./App.vue";
import vuetify from "./plugins/vuetify";
import router from "./router";
import i18n, { applyLanguagePreference } from "./plugins/i18n";
import { GetPreferences } from "../wailsjs/go/main/App";
import "./styles/animations.css";

async function bootstrap() {
    try {
        const preferences = await GetPreferences();
        applyLanguagePreference(preferences?.language);
    } catch {
        applyLanguagePreference("system");
    }

    const app = createApp(App);
    const pinia = createPinia();

    app.use(pinia).use(router).use(i18n).use(vuetify);
    app.mount("#app");
}

void bootstrap();
