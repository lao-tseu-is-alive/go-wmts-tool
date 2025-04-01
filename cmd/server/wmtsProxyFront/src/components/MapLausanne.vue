<style>
@import "ol/ol.css";
@import "ol-layerswitcher/dist/ol-layerswitcher.css";
.map-container {
  background-color: #f8dada;
  overflow: hidden;
}

.log-container {
  background-color: #ffff0a;
  overflow: hidden;
}
.zoom-button {
  font-size: 1.2em !important;
  text-align: center;
  padding: 0;
}
.no-margin{
  margin-left: 0 !important;
}
.map {
  position: relative;
  margin: auto;
  padding: 0;
  top: 0;
  left: 0;
  width: 100%;
  min-width: 300px;
  height: 610px;
  min-height: 600px;
  background-color: #ffffff;
  color: #bbbbbb;
}
</style>
<template>
  <v-container class="fill-height">
    <v-responsive
      class="align-center justify-center fill-height mx-auto"
      max-width="1000"
    >
      <v-row>
        <v-col cols="12">
          <div class="map-container">
          <map-lausanne ref="mymap"
                        :zoom="mapZoom"
                        :center="mapCenter"
                        :base-layer="baseLayerSelected"
                        :layers-visible="layersVisibility"
                        @map-click="handleMapClickEvent"
                        @map-error="handleMapErrorEvent"
                        @map-ready="handleMapReadyEvent"
          />
          </div>
        </v-col>
      </v-row>
    </v-responsive>
  </v-container>
</template>

<script setup lang="ts">
import MapLausanne from "ol-map-lausanne";
import { getFloatParam, getIntegerParam, getStringParam } from "cgil-html-utils";
import {computed, onMounted, reactive, ref} from "vue";
import { getLog } from "@/config";

type coordinate2dArray = [number, number]
const moduleName = "MapLausanne.vue";
const log = getLog(`${moduleName}`, 4, 4);
const goeland = [2537612, 1152611] as coordinate2dArray;
const posX = getFloatParam("x", goeland[0]);
const posY = getFloatParam("y", goeland[1]);
const zoom = getIntegerParam("zoom", 9);
const baseLayerSelected = getStringParam("baselayer", "fonds_geo_osm_bdcad_gris");
const layersVisibility = getStringParam("lvisibles", "05");

const initialPosition: coordinate2dArray = [posX, posY];
const coordinateX = ref(posX);
const coordinateY = ref(posY);
const mapZoom = ref(zoom);
const baseLayer = ref(baseLayerSelected);
const mapCenter: coordinate2dArray = initialPosition;
const logMessages: string[] = reactive(["📣📣 this is just a log message in App.vue 📣📣"]);

//// COMPUTED SECTION
const getCurrentBackgroundLayer = computed(() => {
  return `${baseLayer}`;
});
//// FUNCTIONS SECTION
const handleMapClickEvent = (e) => {
  log.t(`map-click event x: ${e.x}, y: ${e.y}`);
  coordinateX.value = e.x;
  coordinateY.value = e.y;
  mapCenter[0] = e.x;
  mapCenter[1] = e.y;
  logMessages.push(`map-click event x,y: [${coordinateX.value}, ${coordinateY.value}], msg: '${e.msg}'`);
};

const handleMapErrorEvent = (e) => {
  log.t(`map-error event: ${e}`);
  logMessages.push(`map-error event e: ${e}`);
};

const handleMapReadyEvent = (e) => {
  log.t(`map-ready event: ${e}`);
  logMessages.push(`map-ready event e: ${e}`);
};

onMounted(async () => {
  const mountedMsg = `🏠 mounted App.vue`;
  log.t(mountedMsg);
  logMessages.push(mountedMsg);
  log.l(`pos:[${mapCenter[0]}, ${mapCenter[1]}] zoom: ${mapZoom.value} layersVisibility: ${layersVisibility}`);
});
</script>
