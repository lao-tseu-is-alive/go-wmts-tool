/**
 * MapLausanne.ts
 * Created by CGil on 2025-02-15.
 * some functions to allow display an OpenLayers Map in Lausanne Switzerland
 * and handle some interactions with this map
 */
import axios from "axios";
import {BACKEND_URL, getLog, defaultAxiosTimeout} from "@/config";
import proj4 from "proj4";
import OlMap from "ol/Map";
import OlView from "ol/View";
import OlFeature from "ol/Feature";
import OlPoint from "ol/geom/Point";
import OlPolygon from "ol/geom/Polygon";
import OlMultiPoint from "ol/geom/MultiPoint";
import OlProjection from "ol/proj/Projection";
import OlLayer from "ol/layer/Layer";
import OlLayerTile from "ol/layer/Tile";
import OlLayerVector from "ol/layer/Vector";
import OlSourceVector from "ol/source/Vector";
import VectorSource from "ol/source/Vector";
import OlSourceWMTS, {optionsFromCapabilities} from "ol/source/WMTS";
import type { Options as WMTSOptions } from "ol/source/WMTS";
import OlFormatGeoJSON from "ol/format/GeoJSON";
import OlFormatWMTSCapabilities from "ol/format/WMTSCapabilities";
import {Fill, Icon, RegularShape, Stroke, Style} from "ol/style";
import OlStyle from "ol/style/Style";
import OlStroke from "ol/style/Stroke";
import OlCircle from "ol/style/Circle";
import OlFill from "ol/style/Fill";
import {register} from "ol/proj/proj4";
import LayerSwitcher from "ol-layerswitcher";
import {isNullOrUndefined} from "cgil-html-utils";
import type {Coordinate} from "ol/coordinate";
import type {LayersInfo} from "@/components/WmtsLayers";

const log = getLog("mapLausanne", 4, 2);
const urlLausanneMN95 = "https://tilesmn95.lausanne.ch/tiles/1.0.0/LausanneWMTS.xml";
const LausanneMaxExtent = [2532500, 1149000, 2545625, 1161000];
const SwissMaxExtent = [2420000, 1030000, 2900000, 1350000];
const lausanneGare: Coordinate2D = [2537968.5, 1152088.0];
export type Coordinate2D = [number, number]
export type BBox = [number, number, number, number]
export type baseLayerType = "orthophotos_ortho_lidar_2024" | "orthophotos_ortho_lidar_2016" | "fonds_geo_osm_bdcad_gris" | "fonds_geo_osm_bdcad_couleur";

const defaultBaseLayer:baseLayerType = "fonds_geo_osm_bdcad_couleur";

export interface mapFeatureInfo {
  id: string;
  feature: OlFeature;
  layer: string;
  data: object;
}

export interface mapClickInfo {
  x: number;
  y: number;
  features: mapFeatureInfo[];
  msg: string;
}

proj4.defs(
  "EPSG:2056",
  "+proj=somerc +lat_0=46.95240555555556 +lon_0=7.439583333333333 +k_0=1 +x_0=2600000 +y_0=1200000 +ellps=bessel +towgs84=674.374,15.056,405.346,0,0,0,0 +units=m +no_defs",
);
proj4.defs(
  "EPSG:21781",
  "+proj=somerc +lat_0=46.95240555555556 +lon_0=7.439583333333333 +k_0=1 +x_0=600000 +y_0=200000 +ellps=bessel +towgs84=674.4,15.1,405.3,0,0,0,0 +units=m +no_defs",
);

register(proj4);
const swissProjection = new OlProjection({
  code: "EPSG:2056",
  extent: SwissMaxExtent,
  units: "m",
});
const parser = new OlFormatWMTSCapabilities();

export interface IMarkerFeature {
  position: Coordinate;
  iconPath: string; // "/img/gomarker_star_red.png"
  itemTitle: string;
  itemId: string;
}

async function getWMTSCapabilitiesFromUrl(url: string) {
  const response = await fetch(url);
  if (!response.ok) {
    const message = `###!### ERROR in getWMTSCapabilitiesFromUrl when doing fetch(${url}: http status: ${response.status}`;
    throw new Error(message);
  }
  return await response.text();
}

function getWmtsSource(WMTSCapabilitiesParsed: object, layerName: string) {
  const localDebug = false;
  if (localDebug) log.t(`layerName: ${layerName}`);
  const WMTSOptions = optionsFromCapabilities(WMTSCapabilitiesParsed, {
    layer: layerName,
    matrixSet: "EPSG2056",
    format: "image/png",
    style: "default",
    crossOrigin: "anonymous",
  });
  return new OlSourceWMTS(<WMTSOptions>WMTSOptions);
}

function createBaseOlLayerTile(parsedWmtsCapabilities: object, title: string, layerName: string, visible = false) {
  const tempTileLayer = new OlLayerTile({
    visible,
    source: getWmtsSource(parsedWmtsCapabilities, layerName),
  });
  tempTileLayer.setProperties({ title: title, type: "base" });
  return tempTileLayer;
}

async function getWmtsBaseLayers(url: string, initialBaseLayer: string) {
  const arrWmtsLayers = [];
  try {
    const WMTSCapabilities = await getWMTSCapabilitiesFromUrl(url);

    const WMTSCapabilitiesParsed = parser.read(WMTSCapabilities);
    log.l(`## in getWmtsBaseLayers(${url} : WMTSCapabilitiesParsed : \n`, WMTSCapabilitiesParsed)
    arrWmtsLayers.push(
      createBaseOlLayerTile(
        WMTSCapabilitiesParsed,
        "Orthophoto 2016 (Lausanne)",
        "orthophotos_ortho_lidar_2016",
        initialBaseLayer === "orthophotos_ortho_lidar_2016",
      ),
    );
    arrWmtsLayers.push(
        createBaseOlLayerTile(
            WMTSCapabilitiesParsed,
            "Orthophoto 2024 (Lausanne)",
            "orthophotos_ortho_lidar_2024",
            initialBaseLayer === "orthophotos_ortho_lidar_2024",
        ),
    );

    arrWmtsLayers.push(
      createBaseOlLayerTile(
        WMTSCapabilitiesParsed,
        "Fond cadastral (Lausanne)",
        "fonds_geo_osm_bdcad_gris",
        initialBaseLayer === "fonds_geo_osm_bdcad_gris",
      ),
    );
    arrWmtsLayers.push(
      createBaseOlLayerTile(
        WMTSCapabilitiesParsed,
        "Plan ville (Lausanne)",
        "fonds_geo_osm_bdcad_couleur",
        initialBaseLayer === "fonds_geo_osm_bdcad_couleur",
      ),
    );
    return arrWmtsLayers;
  } catch (err) {
    const message = `###!### ERROR in getWmtsBaseLayers occured with url:${url}: error is: ${err}`;
    console.warn(message);
    return [];
  }
}

/**
 * getLayerByName retrieves the Ol layer having the given layerName or null if it does not exist
 * @param olMap to search for the layerName
 * @param layerName the name of the OlLayer to find
 */
export const getLayerByName = (olMap: OlMap, layerName: string): null | OlLayer => {
  log.t(`## in getLayerByName layerName: ${layerName} `);
  const localDebug = false;
  if (isNullOrUndefined(olMap)) {
    log.w("NO WAY : olMap is NULL");
    return null;
  }

  const allLayers = olMap.getAllLayers();
  for (const layer of allLayers) {
    if (localDebug) log.l(`## in getLayerByName : layer) : [${typeof layer}] ${layer}`, layer, layer.getProperties());
    if (layer.get("name") !== undefined && layer.get("name") === layerName) {
      return layer; // This now returns from getLayerByName
    }
  }
  log.l(`## in getLayerByName : the layer [${layerName}] was not found returning NULL `);
  return null;
};

export const addMarker2Layer = (olMap: OlMap, layerName: string, clearLayer = false, marker: IMarkerFeature) => {
  log.t("In addNewMarker markerPos:", marker.position);
  const iconFeature = new OlFeature({
    geometry: new OlPoint(marker.position),
    title: marker.itemTitle,
    id: marker.itemId,
  });
  const iconStyle = new Style({
    image: new Icon({
      anchor: [0.5, 46],
      anchorXUnits: "fraction",
      anchorYUnits: "pixels",
      src: marker.iconPath,
    }),
  });
  iconFeature.setStyle(iconStyle);
  const olLayer = getLayerByName(olMap, layerName);
  if (olLayer == null) {
    // layer was not yet created so let's create it with the brand new marker icon feature
    log.t(`In addNewMarker creating ${layerName}`);
    const vectorSource = new OlSourceVector({ features: [iconFeature] });
    // @ts-ignore
    const vectorLayer = new OlLayerVector({ source: vectorSource });
    vectorLayer.setProperties({ title: layerName, name: layerName });
    olMap.addLayer(vectorLayer);
  } else {
    log.t(`In addNewMarker adding feature to existing ${layerName}`);
    const vectorSource = olLayer.getSource() as OlSourceVector;
    if (vectorSource !== null) {
      if (clearLayer) {
        vectorSource.clear();
      }
      vectorSource.addFeature(iconFeature);
    }
  }
  return iconFeature;
};

export const getPointStyle = (feature: OlFeature, resolution: number) => {
  const localDebug = false;
  if (localDebug) log.t(`## Entering getPointStyle resolution : ${resolution}`, feature);
  const defaultIconPath = "/gomarker_star_red.png";
  if (!isNullOrUndefined(feature) && !isNullOrUndefined(feature.getProperties())) {
    const props = feature.getProperties();
    // const geomType = props.geometry.getType()
    // const type_id = isNullOrUndefined(props.type_id) ? 0 : props.type_id
    const iconPath = isNullOrUndefined(props.icon_path) ? defaultIconPath : props.icon_path;
    return new Style({
      image: new Icon({
        anchor: [0.5, 46],
        anchorXUnits: "fraction",
        anchorYUnits: "pixels",
        src: iconPath,
      }),
    });
  } else {
    return new Style({
      image: new Icon({
        anchor: [0.5, 46],
        anchorXUnits: "fraction",
        anchorYUnits: "pixels",
        src: defaultIconPath,
      }),
    });
  }
};

const getPointStyleLocal = (_feature: OlFeature) => {
  let color = "rgba(255, 0, 0, 0.9)";
  let colorCircle = "rgba(255, 255, 255, 0.3)";
  const style = [];
  // pour dessiner une croix a partir d' un polygone dégénéré de 2 sommets
  style.push(new Style({
    image: new RegularShape({
      fill: new Fill({ color: color }),
      stroke: new Stroke({ width: 1, color: color }),
      points: 2,
      radius: 20,
    }),
  }));
  style.push(new Style({
    image: new RegularShape({
      fill: new Fill({ color: color }),
      stroke: new Stroke({ width: 1, color: color }),
      points: 2,
      radius: 20,
      angle: Math.PI / 2,
    }),
  }));
  style.push(new Style({
    image: new OlCircle({
      fill: new Fill({ color: colorCircle }),
      stroke: new Stroke({ width: 1, color: color }),
      radius: 15,
    }),
  }));
  return style;
};

export interface PolygonWithVerticesStyleOptions {
  strokeColor?: string;
  strokeWidth?: number;
  fillColor?: string;
  vertexFillColor?: string;
  vertexRadius?: number;
}

const defaultPolygonWithVerticesStyleOptions: PolygonWithVerticesStyleOptions = {
  strokeColor: 'black',
  strokeWidth: 2,
  fillColor: 'rgba(0, 0, 255, 0.1)',
  vertexFillColor: 'yellow',
  vertexRadius: 3,
};

const getPolygonWithVerticesStyle = (options:PolygonWithVerticesStyleOptions) => {
  const mergedOptions: PolygonWithVerticesStyleOptions = { ...defaultPolygonWithVerticesStyleOptions, ...options };

  return [
    new OlStyle({
      stroke: new OlStroke({
        color: mergedOptions.strokeColor,
        width: mergedOptions.strokeWidth,
      }),
      fill: new OlFill({
        color: mergedOptions.fillColor,
      }),
    }),
    new OlStyle({
      image: new OlCircle({
        // @ts-ignore
        radius: mergedOptions.vertexRadius,
        fill: new OlFill({
          color: mergedOptions.vertexFillColor,
        }),
      }),
      geometry: function (feature) {
        if (feature && feature.getGeometry()) {
          try {
            // @ts-ignore  (TS often has issues with OpenLayers geometry types)
            const coordinates = feature.getGeometry().getCoordinates()[0];
            return new OlMultiPoint(coordinates);
          } catch (error) {
            console.error("Error getting geometry coordinates:", error);
            return undefined; // Or some other fallback geometry
          }
        }
        return undefined; // Return undefined if no geometry.  Handles null/undefined gracefully.
      },
    }),
  ];
}

export const redrawMarker = (olMap: OlMap, layerName: string, center: Coordinate2D) => {
  log.t(`In redrawMarker layerName: ${layerName} center: [${center[0]}, ${center[1]}]`);
  const olLayer = getLayerByName(olMap, layerName);
  if (olLayer !== null) {
    const vectorSource = olLayer.getSource() as OlSourceVector;
    if (vectorSource !== null) {
      const iconFeature = new OlFeature({
        geometry: new OlPoint(center),
      });
      iconFeature.setStyle(getPointStyleLocal(iconFeature));
      vectorSource.clear();
      vectorSource.addFeature(iconFeature);
    }
  } else {
    log.w(`In redrawMarker layer ${layerName} not found`);
  }
};

export const drawBBox = (
    olMap: OlMap,
    layerName: string,
    bbox: BBox,
    clearLayer:boolean=true,
    styleOptions:PolygonWithVerticesStyleOptions=defaultPolygonWithVerticesStyleOptions
) => {
  const mergedOptions: PolygonWithVerticesStyleOptions = { ...defaultPolygonWithVerticesStyleOptions, ...styleOptions };
  log.t(`In drawBBox layerName: ${layerName} bbox: [${bbox[0]}, ${bbox[1]}, ${bbox[2]}, ${bbox[3]}]`);
  const [x_min, y_min, x_max, y_max] = bbox;
  const rectCoordinates = [
    [x_min, y_min], // Bottom-left
    [x_max, y_min], // Bottom-right
    [x_max, y_max], // Top-right
    [x_min, y_max], // Top-left
    [x_min, y_min]  // Close the polygon
  ];
  let olLayer = getLayerByName(olMap, layerName);
  if (olLayer == null) {
    log.w(`In drawBBox layer ${layerName} not found, will create it`);
    olLayer = new OlLayerVector({
      source: new VectorSource(),
    });
    olLayer.setProperties({ title: layerName, name: layerName });
    olMap.addLayer(olLayer);
  }
  const vectorSource = olLayer.getSource() as OlSourceVector;
  if (vectorSource !== null) {
    const polygonFeature = new OlFeature({
      geometry: new OlPolygon([rectCoordinates]),
    });
    polygonFeature.setStyle(getPolygonWithVerticesStyle(mergedOptions));
    if (clearLayer) {
      vectorSource.clear();
    }
    vectorSource.addFeature(polygonFeature);
  }
};


export const getPolygonStyle = (feature: OlFeature, resolution: number) => {
  const options = {
    fill_color: "rgba(255, 0, 0, 0.8)",
    stroke_color: "#191aff",
    stroke_width: 5,
  };
  const localDebug = false;
  if (localDebug) log.t("## Entering getPolygonStyle with feature :", feature);
  if (localDebug) log.l(`resolution : ${resolution}`);

  let props = null;
  let theStyle = null;
  if (!isNullOrUndefined(feature) && !isNullOrUndefined(feature.getProperties())) {
    props = feature.getProperties();
    // const geomType = props.geometry.getType()
    const id = isNullOrUndefined(props.id) ? "#INCONNU#" : props.id;
    if (localDebug) log.l(`id : ${id}`);
    theStyle = new OlStyle({
      fill: new OlFill({
        color: isNullOrUndefined(props.fill_color) ? options.fill_color : props.fill_color,
      }),
      stroke: new OlStroke({
        color: isNullOrUndefined(props.stroke_color) ? options.stroke_color : props.stroke_color,
        width: isNullOrUndefined(props.stroke_width) ? options.stroke_width : props.stroke_width,
      }),
      image: new OlCircle({
        radius: isNullOrUndefined(props.stroke_width) ? options.stroke_width : props.stroke_width,
        fill: new OlFill({
          color: isNullOrUndefined(props.fill_color) ? options.fill_color : props.fill_color,
        }),
      }),
    });
  } else {
    theStyle = new OlStyle({
      fill: new OlFill({
        color: options.fill_color, // 'rgba(255, 0, 0, 0.8)',
      }),
      stroke: new OlStroke({
        color: options.stroke_color, // '#191aff',
        width: options.stroke_width,
      }),
      image: new OlCircle({
        radius: 9,
        fill: new OlFill({
          color: "#ffcc33",
        }),
      }),
    });
  }
  return theStyle;
};


const getVectorSourceGeoJson = (geoJsonData: object) => {
  log.t("## in getVectorSourceGeoJson ");
  return new OlSourceVector({
    format: new OlFormatGeoJSON({
      dataProjection: "EPSG:2056",
      featureProjection: "EPSG:2056",
    }),
    features: new OlFormatGeoJSON().readFeatures(geoJsonData),
  });
};
export const addGeoJsonLayer = (olMap: OlMap | null, layerName: string, geoJsonData: object) => {
  log.t(`> will trying creating/updating features in layer : ${layerName}...`);
  if (!isNullOrUndefined(olMap)) {
    if (!isNullOrUndefined(geoJsonData)) {
      const olLayer = getLayerByName(olMap as OlMap, layerName);
      if (olLayer == null) {
        // layer was not yet created so let's create it
        const vectorSource = getVectorSourceGeoJson(geoJsonData);
        const vectorLayer = new OlLayerVector({
          source: vectorSource,
          // @ts-expect-error it's what is in ol doc
          style: getPointStyle,
        });
        vectorLayer.setProperties({ title: layerName, name: layerName });
        // @ts-ignore
        olMap.addLayer(vectorLayer);
      } else {
        log.l(`In addGeoJsonLayer setting geoJson source to existing ${layerName}`);
        const oldVectorSource = olLayer.getSource() as OlSourceVector;
        if (oldVectorSource !== null) {
          oldVectorSource.clear();
        }
        olLayer.setSource(getVectorSourceGeoJson(geoJsonData));
      }
    } else {
      log.w("addGeoJsonLayer will do nothing because geoJsonData isNullOrUndefined");
    }
  } else {
    log.w("addGeoJsonLayer will do nothing because olMap isNullOrUndefined");
  }
};



/**
 * createLausanneMap will create a map in the given div
 * @param divOfMap the id of the div you want to draw a map
 * @param centerOfMap the position where you want to center map in MN95 Coordinates [x,y] array
 * @param zoomLevel
 * @param pointLayerName the name of the layer to draw points
 * @param baseLayer the base layer to use
 * @returns an instance of an OpenLayer Map
 */
export async function createLausanneMap(
  divOfMap: HTMLDivElement,
  centerOfMap: Coordinate2D = lausanneGare,
  zoomLevel = 16,
  pointLayerName: string = "PointLayer",
  baseLayer: string = defaultBaseLayer,
) {
  log.t(`createLausanneMap(x,y: [${centerOfMap[0]},${centerOfMap[1]}]  zoom:${zoomLevel})`);
  const arrBaseLayers = await getWmtsBaseLayers(urlLausanneMN95, baseLayer);

  if (arrBaseLayers === null || arrBaseLayers.length < 1) {
    log.w("arrBaseLayers cannot be null or empty to be able to see a nice map !");
    return null;
  }
  if (isNullOrUndefined(divOfMap)) {
    log.e("divOfMap cannot be null  i need a div the map !");
    return null;
  }

  const map = new OlMap({
    target: divOfMap,
    layers: arrBaseLayers,
    view: new OlView({
      projection: swissProjection,
      center: centerOfMap,
      zoom: zoomLevel,
      constrainResolution: true,
      extent: LausanneMaxExtent,
      resolutions: [50,20,10,5,2.5,1,0.5,0.25,0.1,0.05],
    }),
  });
  //create a center point feature
  const iconFeature = new OlFeature({
    geometry: new OlPoint(centerOfMap),
  });
  // create a new vector layer for markers
  const vectorLayer = new OlLayerVector({
    source: new VectorSource({ features: [iconFeature] }),
    // @ts-expect-error it's what is in ol doc
    style: getPointStyleLocal,
  });
  vectorLayer.setProperties({ title: pointLayerName, name: pointLayerName });
  map.addLayer(vectorLayer);


  const layerSwitcher = new LayerSwitcher({
    tipLabel: "Légende", // Optional label for button
    groupSelectStyle: "children", // Can be 'children' [default], 'group' or 'none'
  });
  map.addControl(layerSwitcher);
  return map;
}
export const getTileUrl = (layer: string, year:string,z: number, row: number, col: number) => `/${layer}/default/${year}/swissgrid_05/${z}/${row}/${col}.png`
export const getWmtsProxyTileUrl = (layer: string,year:string, z: number, row: number, col: number) => `/tiles/1.0.0/${layer}/default/${year}/swissgrid_05/${z}/${row}/${col}`

export interface tileInfo {
  "zoom": number,
  "col": number,
  "row": number,
  "wms_url": string,
  "bbox": [number, number, number, number],
}

export const getTileByXY = async (layer:string, z: number, x: number, y: number): Promise<tileInfo | null> => {
  const url = `${BACKEND_URL}/getTileByXY/${layer}/${z}/${x}/${y}`;
  log.l(`getTileByXY url:${url}`);
  try {
    const response = await axios.get(url, {timeout: defaultAxiosTimeout});
    log.l(`getTileByXY response:`, response);
    return response.data;
  } catch (error) {
    log.e(`getTileByXY error:`, error);
    return null;
  }
}

export const getWmtsLayersInfo = async ():Promise<LayersInfo[]> => {
  const url = `${BACKEND_URL}/layersInfo`;
  log.l(`url:${url}`);
  try {
    const response = await axios.get(url, {timeout: defaultAxiosTimeout});
    //log.l(`response:`, response)
    return response.data;
  } catch (error) {
    log.e(`error:`, error);
    return [];
  }
}
