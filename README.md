# ToFromm

## Description 
Tofromm syncs your ROMs and save files between a self-hosted [RomM](https://github.com/rommapp/romm) server and local emulators such as RetroArch, Duckstation, PCSX2, and more.
Pick games from your library, download them with their saves, and upload saves back when you're done.

The following emulators are currently supported:
* Retroarch (Native, Flatpak, and Retrodeck)
* Duckstation (AppImage, Flatpak, and Retrodeck)
* PCSX2 (AppImage, Flatpak, and Retrodeck)
* RPCS3 (Flatpak and Retrodeck)
* Dolphin (Flatpak and Retrodeck)

The current distribution is through a Flatpak, check out the [Releases](https://github.com/bastianvv/tofromm/releases/latest) section for the latest release.

# Status
ToFromm is currently in development, with a working TUI and a GUI: **only happy path cases tested, expect bugs**

Be warned: the sync process currently resolves conflicts automatically, using the newest version of each file. This can lead to unexpected save loss if you have modified a file locally and are syncing with a version from the server.

# Roadmap
* [X] ROM downloading
* [X] Support for native Retroach
* [X] Support for save files
* [ ] Conflict resolution for save files
  * [X] Automatic conflict resolution
  * [ ] Manual conflict resolution
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
* [X] Libadwaita GUI
* [ ] Background sync service
* [ ] Steam Deck plugin

# Building

To build ToFromm, clone the repository and run `go build`.
