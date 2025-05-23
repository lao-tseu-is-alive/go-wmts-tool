export interface LayerDefaultValues {
  WMSBackendURL: string;
  WMSBackendPrefix: string;
  WMTSBBox: number[];
  WMTSURLPrefix: string;
  WMTSURLStyle: string;
  WMTSDimensionName: string;
  WMTSDimensionYear: string;
  WMTSMatrixSet: string;
  ImageExtension: string;
  ImageMIMEType: string;
  EmptyTileDetectionSize: number;
  EmptyTileDetectionMD5Hash: string;
}

export interface LayerConfig extends LayerDefaultValues {
  WMSLayers: string;
  Name: string;
  Title: string;
  Abstract: string;
}

export interface LayersInfo {
  [key: string]: LayerConfig;
}
