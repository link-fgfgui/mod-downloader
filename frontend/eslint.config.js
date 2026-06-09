import pluginVue from "eslint-plugin-vue";

export default [
  {
    ignores: ["node_modules/", "dist/", "public/"],
  },
  ...pluginVue.configs["flat/recommended"],
];
