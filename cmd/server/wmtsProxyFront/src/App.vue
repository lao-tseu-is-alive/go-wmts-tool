<template>
  <v-app>
    <v-app-bar :elevation="2" density="compact" color="primary">
      <template v-slot:prepend>
        <v-app-bar-nav-icon></v-app-bar-nav-icon>
      </template>
      <v-img class="align-start" src="/logo.svg" alt="logo" max-width="32px" ></v-img>
      <v-app-bar-title>{{APP}}, v{{VERSION}}</v-app-bar-title>
    </v-app-bar>
    <v-main>
      <MapLausanne />
    </v-main>
  </v-app>
</template>

<script setup lang="ts">
import MapLausanne from "@/components/MapLausanne.vue"
import {APP, getLog, VERSION} from "@/config";
import {onMounted} from "vue";

const log = getLog(`${APP} v${VERSION}`, 4, 4);

onMounted(async () => {
  const mountedMsg = `🏠 mounted ${APP}:App.vue`;
  document.title = APP;
  log.t(mountedMsg);
});
</script>
