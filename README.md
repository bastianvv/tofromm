# ToFromm

## Description 
A tool for syncing ROMs and saves with a [Romm](https://github.com/rommapp/romm) server.

## Requirements

- [Go](https://golang.org/)
- [Romm](https://github.com/rommapp/romm) v4.9+

## Building

To build ToFromm, you need to have [Go](https://golang.org/) installed.

First create a config.yaml using config.yaml.example as a template.

```bash
go build -o tofromm
```

# How to Use
Run 

```bash
./tofromm sync 
```

This will open the TUI with all your ROMs of the configured platforms.
You can select the ROMs you want to sync using the Space key. Once ready, press Enter to sync.

This will download all selected ROMS and saves, and place them in the configured directories.
Take into account that the current implementation only supports native Retroarch with save directories named by content directory.

After playing, you can upload the saves by running:

```bash
./tofromm upload
```

# Status
ToFromm is currently in development, in a working POC.

# Roadmap
* Conflict resolution for save files
* Support for save states
* Support for more platforms: Retroarch flatpak, Retrodeck, and some standalone emulators
* Gtk GUI
* Steam Deck plugin
