import "vuetify/styles";
import "@mdi/font/css/materialdesignicons.css"; // Import icon styles
import { createVuetify } from "vuetify";

export default createVuetify({
  // Configure themes, i18n, etc. here
  theme: {
    defaultTheme: "dark",
  },
  defaults: {
    // VSelect owns its own transition prop; combobox/autocomplete pass it
    // through menuProps. VMenu keeps the anchored VDialogTransition.
    VSelect: { transition: "md-menu-pop" },
    VCombobox: { menuProps: { transition: "md-menu-pop" } },
    VAutocomplete: { menuProps: { transition: "md-menu-pop" } },
  },
});
