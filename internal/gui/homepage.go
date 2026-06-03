package gui

import (
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/bastianvv/tofromm/internal/client"
	"github.com/bastianvv/tofromm/internal/emulator"
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	pixbuf "github.com/diamondburned/gotk4/pkg/gdkpixbuf/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func buildHomePage(overlay *adw.ToastOverlay, c *client.Client, platforms []client.Platform, emuConfigs map[string]emulator.Config, onSync func([]client.Rom)) *gtk.Box {
	outer := gtk.NewBox(gtk.OrientationVertical, 0)

	spinner := adw.NewSpinner()
	spinner.SetHAlign(gtk.AlignCenter)
	spinner.SetVAlign(gtk.AlignCenter)
	spinner.SetSizeRequest(48, 48)
	spinner.SetVExpand(true)
	outer.Append(spinner)

	go func() {
		romsByID := map[int]client.Rom{}
		romNameIndex := map[string]int{}

		for _, platform := range platforms {
			roms, err := c.GetRomsByPlatform(platform.ID)
			if err != nil {
				continue
			}
			for _, rom := range roms {
				romsByID[rom.ID] = rom
				romNameIndex[rom.FsNameNoExt] = rom.ID
			}
		}

		latestSave := map[int]time.Time{}
		for kind, cfg := range emuConfigs {
			emu, err := emulator.New(kind, cfg)
			if err != nil {
				continue
			}
			for _, s := range emulator.ScanSaves(emu, cfg.Platforms, romNameIndex) {
				if existing, ok := latestSave[s.RomID]; !ok || s.ModTime.After(existing) {
					latestSave[s.RomID] = s.ModTime
				}
			}
		}

		type romWithTime struct {
			rom     client.Rom
			modTime time.Time
		}
		var withSaves []romWithTime
		for romID, t := range latestSave {
			rom, ok := romsByID[romID]
			if !ok {
				continue
			}
			withSaves = append(withSaves, romWithTime{rom, t})
		}
		sort.Slice(withSaves, func(i, j int) bool {
			return withSaves[i].modTime.After(withSaves[j].modTime)
		})

		cutoff := time.Now().AddDate(0, 0, -30)
		var currentlyPlaying, giveAnotherTry []client.Rom
		for _, rs := range withSaves {
			if rs.modTime.After(cutoff) {
				currentlyPlaying = append(currentlyPlaying, rs.rom)
			} else {
				giveAnotherTry = append(giveAnotherTry, rs.rom)
			}
		}

		seen := map[int]bool{}
		var onPlaylist []client.Rom
		for kind, cfg := range emuConfigs {
			emu, err := emulator.New(kind, cfg)
			if err != nil {
				continue
			}
			for _, rom := range romsByID {
				if seen[rom.ID] {
					continue
				}
				if _, hasSave := latestSave[rom.ID]; hasSave {
					continue
				}
				if emulator.RomExists(emu, rom.PlatformFsSlug, rom.FsName) {
					onPlaylist = append(onPlaylist, rom)
					seen[rom.ID] = true
				}
			}
		}

		glib.IdleAdd(func() {
			outer.Remove(spinner)

			if len(currentlyPlaying) == 0 && len(onPlaylist) == 0 && len(giveAnotherTry) == 0 {
				status := adw.NewStatusPage()
				status.SetTitle("Nothing Here Yet")
				status.SetDescription("Sync some games to get started.")
				status.SetIconName("media-playback-start-symbolic")
				status.SetVExpand(true)
				outer.Append(status)
				return
			}

			var allHomeRoms []client.Rom
			for _, rs := range withSaves {
				allHomeRoms = append(allHomeRoms, rs.rom)
			}
			allHomeRoms = append(allHomeRoms, onPlaylist...)

			content := gtk.NewBox(gtk.OrientationVertical, 24)
			content.SetMarginTop(24)
			content.SetMarginBottom(24)
			content.SetMarginStart(18)
			content.SetMarginEnd(18)

			if len(allHomeRoms) > 0 {
				onSync(allHomeRoms)
			}

			if len(currentlyPlaying) > 0 {
				content.Append(buildSection(c, "Currently Playing", currentlyPlaying))
			}
			if len(onPlaylist) > 0 {
				content.Append(buildSection(c, "On Your Playlist", onPlaylist))
			}
			if len(giveAnotherTry) > 0 {
				content.Append(buildSection(c, "Give Another Try…", giveAnotherTry))
			}

			scrolled := gtk.NewScrolledWindow()
			scrolled.SetVExpand(true)
			scrolled.SetChild(content)
			outer.Append(scrolled)
		})
	}()

	return outer
}

func buildSection(c *client.Client, title string, roms []client.Rom) *gtk.Box {
	box := gtk.NewBox(gtk.OrientationVertical, 12)

	label := gtk.NewLabel(title)
	label.AddCSSClass("title-2")
	label.SetXAlign(0)
	box.Append(label)

	row := gtk.NewBox(gtk.OrientationHorizontal, 12)

	for _, rom := range roms {
		row.Append(buildCard(c, rom))
	}

	scroll := gtk.NewScrolledWindow()
	scroll.SetPolicy(gtk.PolicyAutomatic, gtk.PolicyNever)
	scroll.SetChild(row)
	box.Append(scroll)

	return box
}

func buildCard(c *client.Client, rom client.Rom) *gtk.Box {
	card := gtk.NewBox(gtk.OrientationVertical, 6)
	card.SetSizeRequest(150, -1)

	pic := gtk.NewPicture()
	pic.SetContentFit(gtk.ContentFitFill)
	pic.SetOverflow(gtk.OverflowHidden)
	pic.AddCSSClass("cover-art")

	frame := gtk.NewAspectFrame(0.5, 0.5, float32(2)/float32(3), false)
	frame.SetSizeRequest(150, 225)
	frame.SetChild(pic)
	card.Append(frame)

	label := gtk.NewLabel(rom.DisplayName())
	label.SetWrap(true)
	label.SetLines(2)
	label.SetMaxWidthChars(14)
	label.SetJustify(gtk.JustifyCenter)
	card.Append(label)

	if rom.PathCoverSmall != "" {
		coverURL := c.BaseURL + strings.SplitN(rom.PathCoverSmall, "?", 2)[0]
		go func() {
			resp, err := http.Get(coverURL)
			if err != nil || resp.StatusCode != http.StatusOK {
				return
			}
			defer resp.Body.Close()
			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return
			}
			loader := pixbuf.NewPixbufLoader()
			if err := loader.Write(data); err != nil {
				return
			}
			loader.Close()
			pb := loader.Pixbuf()
			if pb == nil {
				return
			}
			glib.IdleAdd(func() {
				pic.SetPixbuf(pb)
			})
		}()
	}

	return card
}
