import { createRouter, createWebHashHistory } from "vue-router";
import Home from "../views/Home.vue";
import Download from "../views/Download.vue";
import Manage from "../views/Manage.vue";
import Unpin from "../views/Unpin.vue";
import Settings from "../views/Settings.vue";

const routes = [
  { path: "/", name: "Home", component: Home },
  { path: "/download", name: "Download", component: Download },
  { path: "/manage", name: "Manage", component: Manage },
  { path: "/unpin", name: "Unpin", component: Unpin },
  { path: "/settings", name: "Settings", component: Settings },
  { path: "/:pathMatch(.*)*", redirect: "/" },
];

const router = createRouter({
  history: createWebHashHistory(),
  routes,
});

export default router;
