import { createApp } from "vue";
import App from "./App.vue";
import vuetify from "./plugins/vuetify";
import router from "./router";
import i18n from "./plugins/i18n";

const app = createApp(App);

// 3. Chain-mount plugins
app.use(router).use(i18n).use(vuetify);

app.mount("#app");
