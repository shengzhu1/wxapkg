import {createApp} from "vue";
import App from "./App.vue";
import PrimeVue from 'primevue/config';
import Aura from '@primeuix/themes/aura';
import Tooltip from 'primevue/tooltip';
import Button from 'primevue/button';
import ToastService from 'primevue/toastservice';

import 'primeicons/primeicons.css';

const app = createApp(App);
app.use(PrimeVue, {
    size: 'small',
    theme: {
        preset: Aura,
    },
});
app.use(ToastService);
app.directive('tooltip', Tooltip);
app.component('Button', Button)
app.mount("#app");