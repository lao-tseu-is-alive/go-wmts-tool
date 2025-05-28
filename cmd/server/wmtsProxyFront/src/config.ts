import {levelLog, Log} from "./log"

export const APP = "wmtsProxyFront"
export const APP_TITLE = "Carte de Lausanne"
export const APP_URL = "https://github.com/lao-tseu-is-alive/go-wmts-tool"
export const VERSION = "0.2.3"
export const BUILD_DATE = "2025-05-28"
// eslint-disable-next-line no-undef
export const DEV = process.env.NODE_ENV === "development"
export const HOME = DEV ? "http://localhost:8000/" : "/"
// eslint-disable-next-line no-restricted-globals
const url = new URL(location.toString())
export const BACKEND_URL = DEV ? "http://localhost:8000" : url.origin
export const getLog = (ModuleName: string, verbosityDev: levelLog, verbosityProd: levelLog) =>
  DEV ? new Log(ModuleName, verbosityDev) : new Log(ModuleName, verbosityProd)

export const defaultAxiosTimeout = 10000 // 10 sec
