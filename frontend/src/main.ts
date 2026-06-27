import { createApp } from "vue";
import { createPinia } from "pinia";
import App from "./App.vue";
import vuetify from "./plugins/vuetify";
import router from "./router";
import i18n from "./plugins/i18n";
import "./styles/animations.css";

const app = createApp(App);
const pinia = createPinia();

app.use(pinia).use(router).use(i18n).use(vuetify);

app.mount("#app");
