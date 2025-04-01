/* eslint-disable no-underscore-dangle */
/* eslint-disable  no-console */
/**
 * Log library for javascript
 * Created by CGil on 15/12/2017
 */
export enum levelLog {
  none,
  err,
  warn,
  trace,
  info,
}

const getLogColor = function (level: levelLog): string {
  switch (level) {
    case levelLog.info:
      //return "background: #1b1b1b; color: #b3e5fc"
      return "color: #009900"
    case levelLog.trace:
      return "color: #0000ff"
    case levelLog.warn:
      return "color: #ff9900"
    case levelLog.err:
      return "background: #ff2325; color: #bfff1a"
  }
  return "background: #dddddd; color: #000"
}

const _getCallerName = function (stackTrace: string | undefined): string {
  if (!(typeof stackTrace === "undefined" || stackTrace === null)) {
    let callerName = stackTrace.replace(/^Error\s+/, "") // Sanitize Chrome
    callerName = callerName.split("\n")[1] // 1st item is this, 2nd item is caller
    callerName = callerName.replace(/^\s+at Object./, "") // Sanitize Chrome
    callerName = callerName.replace(/ \(.+\)$/, "") // Sanitize Chrome
    callerName = callerName.replace(/\@.+/, "") // Sanitize Firefox
    callerName = callerName.replace("at ", "").trim()
    callerName = callerName.replace("VueComponent.", "").trim()
    return callerName
  } else {
    return ""
  }
}
const _log = function (moduleName: string, callerName: string, msg: string, logtype: levelLog, ...args: any[]) {
  let prefix: string
  if (callerName.length > 1) {
    prefix = `${moduleName}::${callerName}()`
  } else {
    prefix = `${moduleName}::`
  }
  switch (logtype) {
    case levelLog.err:
      console.error(`%c ${prefix} ${msg}`, getLogColor(logtype))
      console.trace()
      break
    case levelLog.warn:
      console.warn(`%c ${prefix} ${msg}`, getLogColor(logtype))
      break
    default:
      console.log(`%c ${prefix} ${msg}`, getLogColor(logtype))
      break
  }
  if (args.length > 0) {
    args.forEach((v) => console.log(v))
  }
}

export class Log {
  private readonly _moduleName: string = ""
  private readonly _logLevel: levelLog = levelLog.info

  constructor(moduleName = "", logLevel: levelLog = levelLog.info) {
    this._moduleName = moduleName
    this._logLevel = logLevel
  }

  l(msg: string, ...args: any[]): void {
    if (this._logLevel >= levelLog.info) {
      const callerName = _getCallerName(new Error().stack) // Only tested in latest FF and Chrome
      _log(this._moduleName, callerName, msg, levelLog.info, ...args)
    }
  }

  t(msg: string, ...args: any[]): void {
    if (this._logLevel >= levelLog.trace) {
      const callerName = _getCallerName(new Error().stack) // Only tested in latest FF and Chrome
      _log(this._moduleName, callerName, msg, levelLog.trace, ...args)
    }
  }

  w(msg: string, ...args: any[]): void {
    if (this._logLevel >= levelLog.warn) {
      const callerName = _getCallerName(new Error().stack) // Only tested in latest FF and Chrome
      _log(this._moduleName, callerName, msg, levelLog.warn, ...args)
    }
  }

  e(msg: string, ...args: any[]): void {
    if (this._logLevel >= levelLog.err) {
      const callerName = _getCallerName(new Error().stack) // Only tested in latest FF and Chrome
      _log(this._moduleName, callerName, msg, levelLog.err, ...args)
    }
  }
}
