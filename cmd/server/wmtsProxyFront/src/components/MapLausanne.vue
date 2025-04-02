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
      class=" justify-center fill-height mx-1"
      max-width="1000"
    >
      <v-row>
        <v-col cols="12">
          <div class="map" ref="divMap">
          </div>
        </v-col>
      </v-row>
      <v-row>
        <v-col cols="12">
          <v-chip>{{center}}</v-chip>
        </v-col>
      </v-row>
    </v-responsive>
  </v-container>
</template>

<script setup lang="ts">
import {computed, onMounted, reactive, ref} from "vue";
import { getLog, BACKEND_URL } from "@/config";
import {
  type baseLayerType,
  createLausanneMap,
  drawBBox, getTileByXY, getTileUrl, getWmtsProxyTileUrl,
  type PolygonWithVerticesStyleOptions,
  redrawMarker
} from "@/components/mapLausanne";

type coordinate2dArray = [number, number]
const moduleName = "MapLausanne.vue";
const log = getLog(`${moduleName}`, 4, 4);

const myPointLayerName = "GoelandPointLayer";
const myBBoxLayerName = "GoelandBBoxLayer";
const goeland = [2537612, 1152611] as coordinate2dArray;
const defaultZoom = 6
const defaultBaseLayer: baseLayerType = "fonds_geo_osm_bdcad_couleur"
const getBaseTileUrl = "https://tilesmn95.lausanne.ch/tiles/1.0.0"
const zoom= ref(defaultZoom)
const center = ref(goeland)
const baseLayer = ref(defaultBaseLayer)
const debugMsg = ref("")
const divMap =  ref<HTMLDivElement | null>(null);



//// COMPUTED SECTION
const getCurrentBackgroundLayer = computed(() => {
  return `${baseLayer}`;
});
//// FUNCTIONS SECTION


onMounted(async () => {
  const mountedMsg = `🏠 mounted ${moduleName} `;
  log.t(mountedMsg);
  try {
    const myOlMap = await createLausanneMap(
      divMap.value as HTMLDivElement,
      center.value,
      zoom.value,
      myPointLayerName,
      baseLayer.value);
    if (myOlMap !== null) {
      log.l(`✅ map createLausanneMap returned a valid map`);
      myOlMap.getView().setCenter(center.value);
      myOlMap.getView().setZoom(zoom.value);
      /* draw a bbox
      const imgBBox=[2537000.0,1152000.025,2537999.975,1153000.0];
      const imgBBoxPolygonWithVerticesStyleOptions: PolygonWithVerticesStyleOptions = {
          strokeColor: 'blue',
          strokeWidth: 2,
          fillColor: 'rgba(255, 0, 0, 0.1)',
          vertexFillColor: 'yellow',
          vertexRadius: 3,
      };

      drawBBox(myOlMap, myBBoxLayerName, imgBBox as [number, number, number, number], false, imgBBoxPolygonWithVerticesStyleOptions);
      const tileGridBBox=[2532640.0,1145200.0,2548000.0,1158000.0];
      const tileGridBBoxPolygonWithVerticesStyleOptions: PolygonWithVerticesStyleOptions = {
          strokeColor: 'black',
          strokeWidth: 2,
          fillColor: 'rgba(0, 0, 0, 0.7)',
          vertexFillColor: 'yellow',
          vertexRadius: 3,
      };
      drawBBox(myOlMap, myBBoxLayerName, tileGridBBox as [number, number, number, number], false, tileGridBBoxPolygonWithVerticesStyleOptions);

       */
      const tileBBoxWithVerticesStyleOptions: PolygonWithVerticesStyleOptions = {
        strokeColor: 'red',
        strokeWidth: 2,
        fillColor: 'rgba(255, 0, 250, 0.1)',
        vertexFillColor: 'yellow',
        vertexRadius: 3,
      };
      myOlMap.on("click", async (evt) => {
        log.t(`map click event`, evt);
        const x = +Number(evt.coordinate[0]).toFixed(2);
        const y = +Number(evt.coordinate[1]).toFixed(2);
        const msg = `map click at [${x},${y}]`;
        log.l(msg);
        debugMsg.value = `map click at [${x},${y}]`;
        center.value = [x, y]
        myOlMap.getView().setCenter(center.value);
        const currentZoom = Number(zoom.value)
        redrawMarker(myOlMap, myPointLayerName, [x, y]);
        const res = await getTileByXY(defaultBaseLayer, currentZoom, x, y);
        log.l(`getTileByXY response:`, res);
        if (res !== null) {
          const tileUrl = getTileUrl(baseLayer.value as baseLayerType , res.zoom, res.row, res.col)
          const tileSrc = `${getBaseTileUrl}${tileUrl}`;
          const wmtsProxyTileUrl = getWmtsProxyTileUrl(baseLayer.value as baseLayerType , res.zoom, res.row, res.col)
          const wmtsProxyTileSrc = `${BACKEND_URL}${wmtsProxyTileUrl}`;
          debugMsg.value = `map click at [${x},${y}]\n tileSrc:${tileSrc},\n wms_url:${res.wms_url}`;
          /*
          tileImage.innerHTML = `<img src="${tileSrc}" alt="tile image"/>`;
          tileInfoUrl.innerHTML = `${tileUrl}`;
          wmsImage.innerHTML = `<img src="${res.wms_url}" alt="wms image"/>`;
          wmsImageFromWmtsProxy.innerHTML = `<img src="${wmtsProxyTileSrc}" alt="wmts-proxy image"/>`;
          wmsInfoUrl.innerHTML = `WMS bbox:${res.bbox}`;

           */
          drawBBox(myOlMap, myBBoxLayerName, res.bbox, true, tileBBoxWithVerticesStyleOptions);
        }

      });
      myOlMap.on("moveend", () => {
        log.t(`map moveend event`);
        const newCenter = myOlMap.getView().getCenter() || goeland;
        const realZoom = myOlMap.getView().getZoom() || defaultZoom;
        log.l(`real zoom: ${realZoom}`);
        const newZoom = Math.round(realZoom);
        const x = newCenter[0].toFixed(2);
        const y = newCenter[1].toFixed(2);
        center.value = [x, y]
        zoom.value = newZoom;
        myOlMap.getView().setZoom(newZoom);
        const msg = `map view changed to [${newCenter[0].toFixed(2)},${newCenter[1].toFixed(2)}] zoom:${newZoom}`;
        log.l(msg);
        debugMsg.value = msg;
      });
    }

  } catch (error) {
    log.e(`event [map-error]💥💥 map initialization error: ${error}`);
  }

});
</script>
