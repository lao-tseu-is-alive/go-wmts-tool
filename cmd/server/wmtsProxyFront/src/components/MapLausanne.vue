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
  width: 100%;
  min-width: 300px;
  height: 500px;
  min-height: 400px;
  background-color: #ffffff;
  color: #bbbbbb;
}
</style>
<template>
  <v-container class="fluid">
      <!--next 2 rows is map-->
      <v-row  class="ma-0 pa-0 ">
        <v-col cols="12" class="ma-0 pa-0">
          <div class="map" ref="divMap">
          </div>
        </v-col>
      </v-row>
      <v-row class="ma-0 pa-0">
        <v-col cols="12" class="ma-0 pa-0">
          <v-sheet color="primary-darken-1" border  class="no-margin">
            <p class="text-right ma-0 pa-0" >
              {{getFormattedCenter}}, zoom:{{zoom}} &nbsp;
            </p>
          </v-sheet>
        </v-col>
      </v-row>
      <!--end of map begin wmts tiles-->
      <v-row justify="space-evenly" align-items="center"  class="ma-1 pa-0 " >
        <v-col align="center" cols="4" class="ma-0 pa-0">
          <v-img :src="wmtsTileSrc" width="256px"></v-img>
        </v-col>
        <v-col align="center" cols="4" class="ma-0 pa-0">
          <v-img :src="wmsUrl" width="256px"></v-img>
        </v-col>
        <v-col align="center" cols="4" class="ma-0 pa-0">
          <v-img :src="wmtsProxyTileSrc" width="256px"></v-img>
        </v-col>
      </v-row>
      <v-row class="ma-0 pa-0">
        <v-col cols="12" class="ma-0 pa-0 ">
          <v-sheet color="primary-lighten-1" border  class="no-margin">
            <p class="ma-0 pa-0 ">
              {{ debugMsg }}
            </p>
          </v-sheet>
        </v-col>
      </v-row>
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
const wmtsTileSrc = ref("")
const wmsUrl = ref("")
const wmtsProxyTileSrc =  ref("")
const zoom= ref(defaultZoom)
const center = ref(goeland)
const baseLayer = ref(defaultBaseLayer)
const debugMsg = ref("click on the map to display the wmts tiles")
const divMap =  ref<HTMLDivElement | null>(null);



//// COMPUTED SECTION
const getCurrentBackgroundLayer = computed(() => {
  return `${baseLayer}`;
});

const getFormattedCenter = computed(() => {
  const x = center.value[0].toFixed(2);
  const y = center.value[1].toFixed(2);
  return `[ ${x}, ${y} ]`;
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
        const x = +Number(evt.coordinate[0]).toFixed(2);
        const y = +Number(evt.coordinate[1]).toFixed(2);
        const msg = `⚡⚡ Event map click at [${x},${y}]`;
        log.l(msg);
        debugMsg.value = msg;
        center.value = [x, y]
        myOlMap.getView().setCenter(center.value);
        const currentZoom = Number(zoom.value)
        redrawMarker(myOlMap, myPointLayerName, [x, y]);
        const res = await getTileByXY(defaultBaseLayer, currentZoom, x, y);
        log.l(`getTileByXY response:`, res);
        if (res !== null) {
          const tileUrl = getTileUrl(baseLayer.value as baseLayerType , res.zoom, res.row, res.col)
          wmtsTileSrc.value = `${getBaseTileUrl}${tileUrl}`;
          const wmtsProxyTileUrl = getWmtsProxyTileUrl(baseLayer.value as baseLayerType , res.zoom, res.row, res.col)
          wmtsProxyTileSrc.value = `${BACKEND_URL}${wmtsProxyTileUrl}`;
          wmsUrl.value = res.wms_url
          debugMsg.value = `tileUrl:${tileUrl}, wmtsProxyTileUrl:${wmtsProxyTileUrl}`;
          drawBBox(myOlMap, myBBoxLayerName, res.bbox, true, tileBBoxWithVerticesStyleOptions);
        }

      });
      myOlMap.on("moveend", () => {
        log.t(`⚡⚡ Event map moveend `);
        const newCenter = myOlMap.getView().getCenter() || goeland;
        const realZoom = myOlMap.getView().getZoom() || defaultZoom;
        log.l(`real zoom: ${realZoom}`);
        const newZoom = Math.round(realZoom);
        center.value = newCenter as coordinate2dArray;
        zoom.value = newZoom;
        myOlMap.getView().setZoom(newZoom);
        const msg = `map view changed to [${newCenter[0].toFixed(2)},${newCenter[1].toFixed(2)}] zoom:${newZoom}`;
        log.l(msg);
      });
    }

  } catch (error) {
    const errMsg = `event [map-error]💥💥 map initialization error: ${error}`
    log.e(errMsg);
    debugMsg.value = errMsg;
  }

});
</script>
