package gui

import (
	"github.com/bastianvv/tofromm/internal/client"
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/spf13/viper"

	appconfig "github.com/bastianvv/tofromm/internal/config"
)

func newServerSetupPage(nav *adw.NavigationView, overlay *adw.ToastOverlay) *adw.NavigationPage {
	serverEntry := adw.NewEntryRow()
	serverEntry.SetTitle("Server URL")
	serverEntry.SetText(viper.GetString("server"))

	usernameEntry := adw.NewEntryRow()
	usernameEntry.SetTitle("Username")
	usernameEntry.SetText(viper.GetString("username"))

	passwordEntry := adw.NewPasswordEntryRow()
	passwordEntry.SetTitle("Password")

	group := adw.NewPreferencesGroup()
	group.SetTitle("ROMM Server")
	group.Add(serverEntry)
	group.Add(usernameEntry)
	group.Add(passwordEntry)

	connectBtn := gtk.NewButton()
	connectBtn.SetLabel("Connect")
	connectBtn.AddCSSClass("suggested-action")
	connectBtn.AddCSSClass("pill")

	header := adw.NewHeaderBar()
	header.SetShowBackButton(false)

	box := gtk.NewBox(gtk.OrientationVertical, 0)
	box.SetMarginTop(24)
	box.SetMarginBottom(24)
	box.SetMarginStart(12)
	box.SetMarginEnd(12)
	box.Append(group)

	btnBox := gtk.NewBox(gtk.OrientationHorizontal, 0)
	btnBox.SetHAlign(gtk.AlignCenter)
	btnBox.SetMarginTop(24)
	btnBox.Append(connectBtn)
	box.Append(btnBox)

	clamp := adw.NewClamp()
	clamp.SetMaximumSize(600)
	clamp.SetChild(box)

	scrolled := gtk.NewScrolledWindow()
	scrolled.SetVExpand(true)
	scrolled.SetChild(clamp)

	toolbar := adw.NewToolbarView()
	toolbar.AddTopBar(header)
	toolbar.SetContent(scrolled)

	doConnect := func() {
		serverURL := serverEntry.Text()
		username := usernameEntry.Text()
		password := passwordEntry.Text()

		connectBtn.SetSensitive(false)
		connectBtn.SetLabel("Connecting…")

		go func() {
			c := client.NewClient(serverURL, username, password)
			platforms, err := c.GetPlatforms()

			if err == nil {
				var withRoms []client.Platform
				for _, p := range platforms {
					if p.RomCount > 0 {
						withRoms = append(withRoms, p)
					}
				}
				platforms = withRoms
			}

			glib.IdleAdd(func() {
				if err != nil {
					connectBtn.SetSensitive(true)
					connectBtn.SetLabel("Connect")
					t := adw.NewToast("Connection failed: " + err.Error())
					t.SetTimeout(4)
					overlay.AddToast(t)
					return
				}
				viper.Set("server", serverURL)
				viper.Set("username", username)
				viper.Set("password", password)
				appconfig.EnsureDir()
				viper.WriteConfigAs(appconfig.FilePath())
				nav.Push(newEmulatorSetupPage(nav, overlay, platforms))
			})
		}()
	}

	connectBtn.ConnectClicked(func() { doConnect() })
	serverEntry.ConnectEntryActivated(func() { doConnect() })
	usernameEntry.ConnectEntryActivated(func() { doConnect() })
	passwordEntry.ConnectEntryActivated(func() { doConnect() })

	page := adw.NewNavigationPage(toolbar, "Connect to Romm")
	page.ConnectShown(func() {
		connectBtn.SetSensitive(true)
		connectBtn.SetLabel("Connect")
	})
	return page
}

var emuOptions = []string{
	"(skip)", "retroarch", "retroarch-flatpak", "duckstation",
	"pcsx2", "pcsx2-flatpak", "rpcs3", "rpcs3-flatpak", "dolphin",
}

var emuDefaults = map[string][3]string{
	"(skip)":            {"", "", ""},
	"retroarch":         {"~/ROMs", "~/.config/retroarch/saves", "~/.config/retroarch/states"},
	"retroarch-flatpak": {"~/ROMs", "~/.var/app/org.libretro.RetroArch/config/retroarch/saves", "~/.var/app/org.libretro.RetroArch/config/retroarch/states"},
	"duckstation":       {"~/ROMs/psx", "~/.local/share/duckstation/saves", "~/.local/share/duckstation/states"},
	"pcsx2":             {"~/ROMs/ps2", "~/.config/pcsx2/memcards", "~/.config/pcsx2/sstates"},
	"pcsx2-flatpak":     {"~/ROMs/ps2", "~/.var/app/org.pcsx2.pcsx2/config/pcsx2/memcards", "~/.var/app/org.pcsx2.pcsx2/config/pcsx2/sstates"},
	"rpcs3":             {"~/.config/rpcs3/games", "~/.config/rpcs3/saves", "~/.config/rpcs3/states"},
	"rpcs3-flatpak":     {"~/.var/app/net.rpcs3.RPCS3/config/rpcs3/games", "~/.var/app/net.rpcs3.RPCS3/config/rpcs3/saves", "~/.var/app/net.rpcs3.RPCS3/config/rpcs3/states"},
	"dolphin":           {"~/ROMs/dolphin", "~/.var/app/org.DolphinEmu.dolphin-emu/config/dolphin-emu/saves", "~/.var/app/org.DolphinEmu.dolphin-emu/data/dolphin-emu/StateSave"},
}

func newEmulatorSetupPage(nav *adw.NavigationView, overlay *adw.ToastOverlay, platforms []client.Platform) *adw.NavigationPage {
	type platState struct {
		combo       *adw.ComboRow
		romsEntry   *adw.EntryRow
		savesEntry  *adw.EntryRow
		statesEntry *adw.EntryRow
	}

	states := make([]platState, len(platforms))

	emuModel := gtk.NewStringList(emuOptions)

	type emuPaths struct{ romsDir, savesDir, statesDir string }
	platformToEmu := map[string]string{}
	emuToPaths := map[string]emuPaths{}

	for kind := range viper.GetStringMap("emulators") {
		sub := viper.Sub("emulators." + kind)
		if sub == nil {
			continue
		}
		for _, slug := range sub.GetStringSlice("platforms") {
			platformToEmu[slug] = kind
		}
		emuToPaths[kind] = emuPaths{
			romsDir:   sub.GetString("roms_dir"),
			savesDir:  sub.GetString("saves_dir"),
			statesDir: sub.GetString("states_dir"),
		}
	}

	prefsPage := adw.NewPreferencesPage()
	group := adw.NewPreferencesGroup()
	group.SetTitle("Assign Emulators")
	group.SetDescription("For each platform, choose the emulator you want to use")
	prefsPage.Add(group)

	for i, platform := range platforms {
		i, platform := i, platform

		expander := adw.NewExpanderRow()
		expander.SetTitle(platform.Name)
		expander.SetSubtitle(platform.FsSlug)

		combo := adw.NewComboRow()
		combo.SetTitle("Emulator")
		combo.SetModel(emuModel)

		romsEntry := adw.NewEntryRow()
		romsEntry.SetTitle("ROMs Directory")

		savesEntry := adw.NewEntryRow()
		savesEntry.SetTitle("Saves Directory")

		statesEntry := adw.NewEntryRow()
		statesEntry.SetTitle("States Directory")

		states[i] = platState{
			combo, romsEntry, savesEntry, statesEntry,
		}

		combo.Connect("notify::selected", func() {
			idx := int(combo.Selected())
			if idx < 0 || idx >= len(emuOptions) {
				return
			}
			defaults := emuDefaults[emuOptions[idx]]
			romsEntry.SetText(defaults[0])
			savesEntry.SetText(defaults[1])
			statesEntry.SetText(defaults[2])
		})

		existingKind := platformToEmu[platform.FsSlug]
		selectedIdx := 0
		for j, opt := range emuOptions {
			if opt == existingKind {
				selectedIdx = j
				break
			}
		}

		combo.SetSelected(uint(selectedIdx))
		if existingKind != "" {
			paths := emuToPaths[existingKind]
			romsEntry.SetText(paths.romsDir)
			savesEntry.SetText(paths.savesDir)
			statesEntry.SetText(paths.statesDir)
			expander.SetExpanded(true)
		}

		expander.AddRow(combo)
		expander.AddRow(romsEntry)
		expander.AddRow(savesEntry)
		expander.AddRow(statesEntry)

		group.Add(expander)
	}

	saveBtn := gtk.NewButton()
	saveBtn.SetLabel("Save & Continue")
	saveBtn.AddCSSClass("suggested-action")
	saveBtn.AddCSSClass("pill")
	saveBtn.SetVAlign(gtk.AlignCenter)

	header := adw.NewHeaderBar()
	header.PackEnd(saveBtn)

	toolbar := adw.NewToolbarView()
	toolbar.AddTopBar(header)
	toolbar.SetContent(prefsPage)

	saveBtn.ConnectClicked(func() {
		type emuEntry struct {
			platforms []string
			romsDir   string
			savesDir  string
			statesDir string
		}

		configs := map[string]*emuEntry{}
		for i, platform := range platforms {
			idx := int(states[i].combo.Selected())
			if idx <= 0 || idx >= len(emuOptions) {
				continue
			}
			name := emuOptions[idx]
			if existing, ok := configs[name]; ok {
				existing.platforms = append(existing.platforms, platform.FsSlug)
			} else {
				configs[name] = &emuEntry{
					platforms: []string{platform.FsSlug},
					romsDir:   states[i].romsEntry.Text(),
					savesDir:  states[i].savesEntry.Text(),
					statesDir: states[i].statesEntry.Text(),
				}
			}
		}

		if len(configs) == 0 {
			t := adw.NewToast("Select at least one emulator to continue")
			t.SetTimeout(3)
			overlay.AddToast(t)
			return
		}

		emulatorsMap := map[string]interface{}{}
		for kind, cfg := range configs {
			emulatorsMap[kind] = map[string]interface{}{
				"platforms":  cfg.platforms,
				"roms_dir":   cfg.romsDir,
				"saves_dir":  cfg.savesDir,
				"states_dir": cfg.statesDir,
			}
		}
		viper.Set("emulators", emulatorsMap)
		appconfig.EnsureDir()
		viper.WriteConfigAs(appconfig.FilePath())

		nav.Replace([]*adw.NavigationPage{newMainPage(nav, overlay)})
	})

	return adw.NewNavigationPage(toolbar, "Emulator Setup")
}
