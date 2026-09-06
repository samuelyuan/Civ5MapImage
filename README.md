# Civ5MapImage

## Table of Contents

* [Introduction](#introduction)
* [Command-Line Usage](#command-line-usage)
* [Examples](#examples)
* [Running Tests](#running-tests)
* [File Format Documentation](#file-format-documentation)

## Introduction

Most custom maps designed for Civ 5 will usually provide screenshots of the map, but they will either only show a portion of the map in the game or a zoomed out image which shows all of the cities but not the terrain. This program is designed to provide you a detailed view of the entire map in one single image.

You have the option of generating a physical map or a political map. The physical map focuses on generating the terrain, while the political map shows the civilization boundaries and major cities. This program will convert a Civ 5 map with the file extension .Civ5Map to a PNG image.

## Command-Line Usage

The map filename can either be a .civ5map or .json file. To start using this application, you can use any of the map files in the maps/ folder or you can load a .civ5map in your game directory.

If you generated the map image and want to modify the map, you can export the .civ5map as a .json by providing an output filename with the file extension .json and reuse the exported json as the map filename.

```
./Civ5MapImage.exe -map=[map filename] -mode=[drawing mode (optional)] -output=[output filename (default is output.png)]
```

`-input` is also accepted as an older alias for `-map`.

### Generate Physical Map Image

The default map mode is physical, which shows the different types of terrain.
```
./Civ5MapImage.exe -map=earth.Civ5Map -output=earth.png
```

<div style="display:inline-block;">
<img src="https://raw.githubusercontent.com/samuelyuan/Civ5MapImage/master/screenshots/earth.png" alt="earth" width="550" height="300" />
</div>

### Generate Political Map Image

To generate a political map with the civilization and city state borders, you must pass in -mode=political to specify the drawing mode.
```
./Civ5MapImage.exe -map=maps/europe1939.json -mode=political -output=europe1939.png
```

<div style="display:inline-block;">
<img src="https://raw.githubusercontent.com/samuelyuan/Civ5MapImage/master/screenshots/europe1939.png" alt="europe" width="400" height="300" />
</div>

### Generate Replay

To generate a replay, provide the base map and the replay file of a game. Passing `-replay` alone implies `-mode=replay` and defaults `-output` to `output.gif`:
```
./Civ5MapImage.exe -map=[map filename] -replay=[replay filename]
```
which is shorthand for the fully explicit form:
```
./Civ5MapImage.exe -mode=replay -map=[map filename] -replay=[replay filename] -output=[gif filename]
```

| Flag | Expected extension(s) | Notes |
|------|------------------------|-------|
| `-map` | `.civ5map` or `.json` | The base map. Use the same map the replay was recorded on. |
| `-replay` | `.civ5replay` or `.json` | The replay event log. A `.json` here must be a replay previously exported with `-mode=exportjson` (either directly from a `.civ5replay`, or converted from a `.civ5save` — see [Extract Replay From Save File](#extract-replay-from-save-file) below). |
| `-output` | `.gif` | Replays are always GIF-encoded; any other extension is rejected before rendering. |

Any other extension on `-map` or `-replay` is also rejected immediately. Before rendering, the map and replay are cross-checked: grid dimensions, tile-by-tile geography, and whether `-map`'s filename matches the one the replay recorded (a mismatch there is only a warning). A geography mismatch fails fast with a descriptive error instead of rendering a wrong or partial animation.

### Extract Replay From Save File

To extract a replay from a save file, you will need to convert the save file into a json and use the new json as a replay file.

```
./Civ5MapImage.exe -mode=exportjson -map=[save filename] -output=[json filename]
```

### Convert .civ5map to .json

Set -mode=exportjson and output to have a filename ending in .json. No image will be generated.
```
./Civ5MapImage.exe -mode=exportjson -map=earth.Civ5Map -output=earth.json
```

## Examples

<div style="display:inline-block;">
<img src="https://raw.githubusercontent.com/samuelyuan/Civ5MapImage/master/screenshots/europe.png" alt="europe" width="200" height="150" />
<img src="https://raw.githubusercontent.com/samuelyuan/Civ5MapImage/master/screenshots/europe1914.png" alt="europe" width="200" height="150" />
<img src="https://raw.githubusercontent.com/samuelyuan/Civ5MapImage/master/screenshots/europe1939.png" alt="europe" width="200" height="150" />
</div>

<div style="display:inline-block;">
<img src="https://raw.githubusercontent.com/samuelyuan/Civ5MapImage/master/screenshots/europe2014.png" alt="europe" width="200" height="150" />
<img src="https://raw.githubusercontent.com/samuelyuan/Civ5MapImage/master/screenshots/world.png" alt="world" width="200" height="150" />
<img src="https://raw.githubusercontent.com/samuelyuan/Civ5MapImage/master/screenshots/earth1900.png" alt="earth 1900" width="200" height="150" />
</div>

<div style="display:inline-block;">
<img src="https://raw.githubusercontent.com/samuelyuan/Civ5MapImage/master/screenshots/earth1936.png" alt="earth 1936" width="200" height="150" />
<img src="https://raw.githubusercontent.com/samuelyuan/Civ5MapImage/master/screenshots/earth1939.png" alt="earth 1939" width="200" height="150" />
<img src="https://raw.githubusercontent.com/samuelyuan/Civ5MapImage/master/screenshots/earth1942.png" alt="earth 1942" width="200" height="150" />
</div>

<div style="display:inline-block;">
<img src="https://raw.githubusercontent.com/samuelyuan/Civ5MapImage/master/screenshots/earth2014a.png" alt="earth 2014 huge 1" width="200" height="150" />
<img src="https://raw.githubusercontent.com/samuelyuan/Civ5MapImage/master/screenshots/earth2014b.png" alt="earth 2014 huge 2" width="200" height="150" />
<img src="https://raw.githubusercontent.com/samuelyuan/Civ5MapImage/master/screenshots/earth2022.png" alt="earth 2022" width="200" height="150" />
</div>

<div style="display:inline-block;">
<img src="https://raw.githubusercontent.com/samuelyuan/Civ5MapImage/master/screenshots/mongol.png" alt="mongol" width="200" height="150" />
<img src="https://raw.githubusercontent.com/samuelyuan/Civ5MapImage/master/screenshots/india.png" alt="india" width="200" height="150" />
<img src="https://raw.githubusercontent.com/samuelyuan/Civ5MapImage/master/screenshots/stalingrad.png" alt="stalingrad" width="200" height="150" />
</div>

## Running Tests

This project has unit tests for the `fileio` and `graphics` packages. To run the full test suite:

```
go test ./...
```

To see individual test names as they run, add `-v`:

```
go test ./... -v
```

To check test coverage:

```
go test ./... -cover
```

## File Format Documentation

For detailed technical specifications of the Civ5 file formats, see [FORMAT.md](FORMAT.md).

This document covers:
- **Map File Format** (.civ5map) - Complete structure and data types
- **Replay File Format** (.civ5replay) - Event data and civilization information  
- **Save File Format** (.civ5save) - Compressed game state data

The format documentation is intended for developers who need to understand the binary structure of Civ5 files or want to extend the functionality of this tool.