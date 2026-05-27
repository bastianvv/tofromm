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

This will sync all selected ROMS and saves, and place them in the configured directories.
ROMs will be downloaded if they don't exist locally, and saves will be synced by keeping the newest version.
Take into account that the current implementation only supports native Retroarch with save directories named by content directory.

To force upload of local saves, you can run:

```bash
./tofromm upload
```

# Status
ToFromm is currently in development, with a working TUI and a GUI: **only happy path cases tested, expect bugs**

# Roadmap
* [X] ROM downloading
* [X] Support for native Retroach
* [X] Support for save files
* [X] Conflict resolution for save files
* [X] Support for more Retroarch variants: flatpak and Retrodeck
* [X] Support for save states
* [ ] Standalone emulator support
  * [ ] Duckstation
    * [X] Games
    * [X] Memcards
    * [ ] Save states
  * [ ] PCSX2
    * [X] Games
    * [X] Memcards
    * [ ] Save states
  * [ ] RPCS3
    * [X] Games
    * [ ] Memcards
    * [ ] Save states
  * [ ] Dolphin
    * [X] Games
    * [ ] Memcards
    * [ ] Save states
  * ... More to come
* [ ] Libadwaita GUI
* [ ] Steam Deck plugin
