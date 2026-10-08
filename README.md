# go-wmts-tool
Allow to retrieve [WMTS](https://en.wikipedia.org/wiki/Web_Map_Tile_Service) tiles in swiss grid from an existing wms server 
and store them in a classical directory tree.

## saveWmtsTiles

Command line tool that pre-generates all the png tiles of a layer for one or several zoom levels.
To reduce the number of WMS requests, it asks the WMS server for a *meta-tile* (by default 4x4 tiles)
and splits it into individual tiles, saved in the directory tree
`{cache folder}/{wmts_url_prefix}/{layer}/{style}/{year}/{matrix_set}/{zoom}/{row}/{col}.png`.

```bash
./saveWmtsTiles -config saveWmtsTiles-config.yaml -layer fonds_geo_osm_bdcad_gris -zoom 9 -workers 2
```

### Options

| Option | Default | Description |
|---|---|---|
| `-config` | `wmtsConfig.yaml` | YAML config file with the cache folder and the layers definition |
| `-layer` | `fonds_geo_osm_bdcad_couleur` | name of the layer (key under `layers:` in the config file) |
| `-zoom` | `3` | zoom level to generate |
| `-minZoom`, `-maxZoom` | | range of zoom levels to generate, used only when **both** are given (then `-zoom` is ignored) |
| `-workers` | `4` | number of concurrent WMS requests |
| `-metatile` | `4` | number of tiles per side of a meta-tile (4 means one WMS request for 4x4 tiles) |
| `-buffer` | `50` (or `BUFFER_SIZE`) | buffer in pixels requested around each meta-tile and cropped afterwards, avoids cut labels at tile borders (0-256) |
| `-ClientTimeOut` | `30` | HTTP timeout in seconds for one WMS request |
| `-retries` | `3` | number of retries of a failed WMS request, see [Error handling](#error-handling) |
| `-skipExisting` | `false` | skip the meta-tiles whose tiles all already exist **and** are younger than `-maxTileAge` |
| `-maxTileAge` | `24h` | with `-skipExisting`, max age of an existing tile to be kept, older tiles are considered outdated and fetched again. Go duration syntax: `30m`, `12h`, `168h` (days are not supported) |
| `-retryFile` | | only process the meta-tiles listed in this file, written by a previous run (zoom options are ignored) |
| `-failedFile` | `failed_<layer>_<timestamp>.csv` | file where the meta-tiles still failing at the end of the run are written |
| `-verbose` | `false` | print the layers details and every saved meta-tile |

### Environment variables

| Variable | Default | Description |
|---|---|---|
| `LOG_LEVEL` | `warn` | `debug`, `info`, `warn`, `error` or `fatal` |
| `LOG_FILE` | `stderr` | `stderr`, `stdout`, `DISCARD` or a file name (opened in append mode) |
| `BUFFER_SIZE` | `50` | default value of `-buffer` |

### Error handling

WMS servers behind a proxy sometimes answer with a transient error (e.g. a `502 Bad Gateway`
when nginx loses its connection to the upstream). The tool handles them in three levels:

1. **Retry**: network errors and the HTTP status `408`, `429`, `500`, `502`, `503`, `504` are retried
   up to `-retries` times, waiting 1s, 2s, 4s... (plus some random jitter) between attempts.
   Other errors (e.g. `400`, or a `200` answer containing an XML ServiceException instead of an image)
   are not retried.
2. **Second pass**: the meta-tiles that still failed are tried again with a single worker,
   15 seconds after the end of the first pass.
3. **Failed file**: the meta-tiles failing after the second pass are written in a csv file,
   and the program exits with status **1** (status 0 means that all tiles were saved).
   To fetch only them later, run again with the same `-config` and `-layer`:

   ```bash
   ./saveWmtsTiles -config saveWmtsTiles-config.yaml -layer fonds_geo_osm_bdcad_gris -retryFile failed_fonds_geo_osm_bdcad_gris_20261006-133516.csv
   ```

Tiles are written atomically (temporary file then rename), so an interrupted run never leaves
a truncated png. An interrupted or partially failed run can also simply be restarted with
`-skipExisting`: the meta-tiles saved recently are skipped and only the missing or outdated ones are fetched.

Each retry warning and each failure names the tiles of the meta-tile concerned
(e.g. `zoom:9 rows 15432-15435 cols 9192-9195`). At the end of each zoom level, a summary gives
the counts of meta-tiles and of png really written by this run, so there is no need to count the files:

```
Processing tiles for layer fonds_geo_osm_bdcad_gris, zoom 9: 1848 meta-tiles, 29568 png expected
  saved   :   1846 meta-tiles,    29536 png written
  skipped :      0 meta-tiles,        0 png fresh
  failed  :      2 meta-tiles,       32 png missing
💥 2 meta-tiles (32 png) are still missing:
   zoom:9 rows 15432-15435 cols 9192-9195
   zoom:9 rows 15436-15439 cols 9224-9227
   they are listed in failed_fonds_geo_osm_bdcad_gris_20261008-100153.csv
   to fetch only them, run again with the same -config and -layer and: -retryFile failed_fonds_geo_osm_bdcad_gris_20261008-100153.csv
```

The expected count covers whole meta-tiles, so it can be slightly larger than the bbox at its edges.
