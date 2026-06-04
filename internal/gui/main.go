//go:build !nogui

package gui

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/bastianvv/tofromm/internal/client"
	"github.com/bastianvv/tofromm/internal/emulator"
	syncer "github.com/bastianvv/tofromm/internal/sync"
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	pixbuf "github.com/diamondburned/gotk4/pkg/gdkpixbuf/v2"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/spf13/viper"
)

func newMainPage(nav *adw.NavigationView, overlay *adw.ToastOverlay) *adw.NavigationPage {
	spinner := adw.NewSpinner()
	spinner.SetHAlign(gtk.AlignCenter)
	spinner.SetVAlign(gtk.AlignCenter)
	spinner.SetSizeRequest(48, 48)

	toolbar := adw.NewToolbarView()
	toolbar.AddTopBar(adw.NewHeaderBar())
	toolbar.SetContent(spinner)

	page := adw.NewNavigationPage(toolbar, "tofromm")

	go func() {
		c := newClientFromConfig()
		allPlatforms, err := c.GetPlatforms()
		if err != nil {
			glib.IdleAdd(func() {
				t := adw.NewToast("Failed to load platforms: " + err.Error())
				t.SetTimeout(5)
				overlay.AddToast(t)
			})
			return
		}

		rawEmulators := viper.GetStringMap("emulators")
		configuredSlugs := make(map[string]bool)
		for kind := range rawEmulators {
			sub := viper.Sub("emulators." + kind)
			if sub == nil {
				continue
			}
			var cfg emulator.Config
			if err := sub.Unmarshal(&cfg); err != nil {
				continue
			}
			for _, slug := range cfg.Platforms {
				configuredSlugs[slug] = true
			}
		}

		var allRommPlatforms []client.Platform
		for _, p := range allPlatforms {
			if p.RomCount > 0 {
				allRommPlatforms = append(allRommPlatforms, p)
			}
		}

		var sidebarPlatforms []client.Platform
		for _, p := range allRommPlatforms {
			if configuredSlugs[p.FsSlug] {
				sidebarPlatforms = append(sidebarPlatforms, p)
			}
		}

		glib.IdleAdd(func() {
			page.SetChild(buildSplitView(nav, overlay, c, sidebarPlatforms, allRommPlatforms))
		})
	}()
	return page
}

func buildSplitView(nav *adw.NavigationView, overlay *adw.ToastOverlay, c *client.Client, platforms []client.Platform, allPlatforms []client.Platform) *adw.NavigationSplitView {
	contentStack := gtk.NewStack()
	contentStack.SetTransitionType(gtk.StackTransitionTypeCrossfade)

	rawEmulators := viper.GetStringMap("emulators")
	emuConfigs := make(map[string]emulator.Config)
	for kind := range rawEmulators {
		sub := viper.Sub("emulators." + kind)
		if sub == nil {
			continue
		}
		var cfg emulator.Config
		sub.Unmarshal(&cfg)
		emuConfigs[kind] = cfg
	}

	var homeSync func()
	var doSync func([]client.Rom)

	syncBtn := gtk.NewButton()
	syncBtn.SetLabel("Sync")
	syncBtn.AddCSSClass("suggested-action")
	syncBtn.AddCSSClass("pill")
	syncBtn.SetVAlign(gtk.AlignCenter)
	syncBtn.SetSensitive(false)
	syncBtn.SetVisible(false)

	var currentSearchEntry *gtk.SearchEntry
	var currentListBox *gtk.ListBox

	searchBtn := gtk.NewToggleButton()
	searchBtn.SetIconName("system-search-symbolic")
	searchBtn.SetVAlign(gtk.AlignCenter)
	searchBtn.SetSensitive(false)
	searchBtn.SetVisible(false)

	searchBtn.ConnectToggled(func() {
		if currentSearchEntry == nil {
			return
		}
		if searchBtn.Active() {
			currentSearchEntry.SetVisible(true)
			currentSearchEntry.GrabFocus()
		} else {
			currentSearchEntry.SetVisible(false)
			currentSearchEntry.SetText("")
			if currentListBox != nil {
				currentListBox.InvalidateFilter()
			}
		}
	})

	home := buildHomePage(overlay, c, platforms, emuConfigs, func(roms []client.Rom) {
		homeSync = func() { doSync(roms) }
		if contentStack.VisibleChildName() == "home" {
			syncBtn.SetVisible(true)
			syncBtn.SetSensitive(true)
		}
	})

	contentStack.AddNamed(home, "home")
	contentStack.SetVisibleChildName("home")

	progressBar := gtk.NewProgressBar()
	progressBar.SetShowText(true)
	progressBar.SetHExpand(true)
	progressBar.SetMarginTop(8)
	progressBar.SetMarginBottom(8)
	progressBar.SetMarginStart(12)
	progressBar.SetMarginEnd(6)

	logToggle := gtk.NewToggleButton()
	logToggle.SetLabel("Log")
	logToggle.SetVAlign(gtk.AlignCenter)
	logToggle.SetMarginEnd(12)

	logView := gtk.NewTextView()
	logView.SetEditable(false)
	logView.SetMonospace(true)
	logView.SetWrapMode(gtk.WrapWord)
	logView.SetMarginTop(6)
	logView.SetMarginBottom(6)
	logView.SetMarginStart(12)
	logView.SetMarginEnd(12)
	logBuf := logView.Buffer()
	logEndMark := logBuf.CreateMark("end", logBuf.EndIter(), false)

	logScrolled := gtk.NewScrolledWindow()
	logScrolled.SetChild(logView)
	logScrolled.SetSizeRequest(-1, 160)

	logRevealer := gtk.NewRevealer()
	logRevealer.SetChild(logScrolled)
	logRevealer.SetRevealChild(false)
	logRevealer.SetTransitionType(gtk.RevealerTransitionTypeSlideUp)

	logToggle.ConnectToggled(func() {
		logRevealer.SetRevealChild(logToggle.Active())
	})

	progressRow := gtk.NewBox(gtk.OrientationHorizontal, 0)
	progressRow.Append(progressBar)
	progressRow.Append(logToggle)

	progressArea := gtk.NewBox(gtk.OrientationVertical, 0)
	progressArea.Append(progressRow)
	progressArea.Append(logRevealer)
	progressArea.SetVisible(false)

	contentHeader := adw.NewHeaderBar()
	menuBtn := gtk.NewMenuButton()
	menuBtn.SetIconName("open-menu-symbolic")
	menuBtn.SetVAlign(gtk.AlignCenter)

	m := gio.NewMenu()
	m.Append("Preferences", "tofromm.preferences")
	m.Append("Emulator Configuration", "tofromm.emulator-config")
	m.Append("About", "tofromm.about")
	menuBtn.SetMenuModel(m)

	prefsAction := gio.NewSimpleAction("preferences", nil)
	prefsAction.ConnectActivate(func(_ *glib.Variant) {
		nav.Push(newPreferencesPage(nav))
	})

	emuAction := gio.NewSimpleAction("emulator-config", nil)
	emuAction.ConnectActivate(func(_ *glib.Variant) {
		nav.Push(newEmulatorSetupPage(nav, overlay, allPlatforms))
	})

	aboutAction := gio.NewSimpleAction("about", nil)
	aboutAction.ConnectActivate(func(_ *glib.Variant) {
		d := adw.NewAboutDialog()
		d.SetApplicationName("Tofromm")
		d.SetApplicationIcon(appID)
		d.SetVersion("0.8")
		d.SetDeveloperName("bastianvv")
		d.SetDevelopers([]string{"bastianvv"})
		d.SetComments("Sync ROMs and saves between your Linux machine and a ROMM server.")
		d.SetWebsite("https://github.com/bastianvv/tofromm")
		d.SetIssueURL("https://github.com/bastianvv/tofromm/issues")
		d.SetCopyright("© 2026 bastianvv")
		d.SetLicenseType(gtk.LicenseMITX11)
		d.Present(menuBtn)
	})

	ag := gio.NewSimpleActionGroup()
	ag.Insert(emuAction)
	ag.Insert(aboutAction)
	ag.Insert(prefsAction)

	contentHeader.PackEnd(syncBtn)
	contentHeader.PackStart(searchBtn)

	contentToolbar := adw.NewToolbarView()
	contentToolbar.AddTopBar(contentHeader)
	contentToolbar.SetContent(contentStack)
	contentToolbar.AddBottomBar(progressArea)

	keyCtrl := gtk.NewEventControllerKey()
	keyCtrl.SetPropagationPhase(gtk.PhaseCapture)
	keyCtrl.ConnectKeyPressed(func(keyval, keycode uint, state gdk.ModifierType) bool {
		if keyval == gdk.KEY_f && state&gdk.ModifierType(4) != 0 {
			if searchBtn.IsSensitive() {
				searchBtn.SetActive(!searchBtn.Active())
				return true
			}
		}
		return false
	})

	contentPage := adw.NewNavigationPage(contentToolbar, "Home")

	var currentRoms []client.Rom
	var currentPlatformPage *gtk.Box
	var currentChecks []*gtk.CheckButton
	doSync = func(selected []client.Rom) {
		syncBtn.SetSensitive(false)
		logBuf.SetText("")
		progressBar.SetFraction(0)
		progressBar.SetText("Starting…")
		progressArea.SetVisible(true)

		go func() {
			result, err := syncer.Run(syncer.Options{
				Client:     c,
				EmuConfigs: emuConfigs,
				Selected:   selected,
				OnRomStart: func(current, total int) {
					glib.IdleAdd(func() {
						progressBar.SetFraction(float64(current-1) / float64(total))
						progressBar.SetText(fmt.Sprintf("%d / %d", current, total))
					})
				},
				OnProgress: func(msg string) {
					glib.IdleAdd(func() {
						end := logBuf.EndIter()
						logBuf.Insert(end, msg+"\n")
						logView.ScrollMarkOnscreen(logEndMark)
					})
				},
				OnConflict: func(romName, serverTime, reason string) bool {
					switch viper.GetString("conflict_resolution") {
					case "local":
						return false
					case "server":
						return true
					}

					ch := make(chan bool, 1)
					glib.IdleAdd(func() {
						dialog := adw.NewAlertDialog(
							"Save Conflict",
							fmt.Sprintf("%s\nServer: %s\nReason: %s", romName, serverTime, reason),
						)
						dialog.AddResponse("local", "Keep Local")
						dialog.AddResponse("server", "Keep Server")
						dialog.SetDefaultResponse("local")
						dialog.SetResponseAppearance("server", adw.ResponseSuggested)
						dialog.ConnectResponse(func(response string) {
							ch <- (response == "server")
						})
						dialog.Present(syncBtn)
					})
					return <-ch
				},
			})
			glib.IdleAdd(func() {
				syncBtn.SetSensitive(true)
				progressBar.SetFraction(1.0)
				if err != nil {
					progressBar.SetText("Sync failed: " + err.Error())
					t := adw.NewToast("Sync failed: " + err.Error())
					t.SetTimeout(5)
					overlay.AddToast(t)
					return
				}
				progressBar.SetText(fmt.Sprintf("Done — %d synced, %d failed", result.Completed, result.Failed))
			})
		}()
	}

	syncBtn.ConnectClicked(func() {
		if contentStack.VisibleChildName() == "home" {
			if homeSync != nil {
				homeSync()
			}
			return
		}
		var selected []client.Rom
		for i, check := range currentChecks {
			if check.Active() {
				selected = append(selected, currentRoms[i])
			}
		}
		if len(selected) == 0 {
			t := adw.NewToast("No ROMs selected")
			t.SetTimeout(3)
			overlay.AddToast(t)
			return
		}
		doSync(selected)
	})

	onSelect := func(p client.Platform) {
		contentPage.SetTitle(p.Name)
		syncBtn.SetSensitive(false)
		searchBtn.SetSensitive(false)
		syncBtn.SetVisible(true)
		searchBtn.SetActive(false)
		searchBtn.SetVisible(true)
		currentRoms = nil
		currentChecks = nil

		spinner := adw.NewSpinner()
		spinner.SetHAlign(gtk.AlignCenter)
		spinner.SetVAlign(gtk.AlignCenter)
		spinner.SetSizeRequest(48, 48)
		contentStack.AddNamed(spinner, "loading")
		contentStack.SetVisibleChildName("loading")

		go func() {
			roms, err := c.GetRomsByPlatform(p.ID)

			syncedIDs := map[int]bool{}
			if err == nil {
				romNameIndex := make(map[string]int, len(roms))
				for _, rom := range roms {
					romNameIndex[rom.FsNameNoExt] = rom.ID
				}
				for kind, cfg := range emuConfigs {
					for _, slug := range cfg.Platforms {
						if slug != p.FsSlug {
							continue
						}
						emu, e := emulator.New(kind, cfg)
						if e == nil {
							for _, s := range emulator.ScanSaves(emu, []string{p.FsSlug}, romNameIndex) {
								syncedIDs[s.RomID] = true
							}
						}
						break
					}
				}
			}
			sort.SliceStable(roms, func(i, j int) bool {
				return syncedIDs[roms[i].ID] && !syncedIDs[roms[j].ID]
			})

			glib.IdleAdd(func() {
				if old := contentStack.ChildByName("loading"); old != nil {
					contentStack.Remove(old)
				}
				if err != nil {
					t := adw.NewToast("Failed to load ROMs: " + err.Error())
					t.SetTimeout(4)
					overlay.AddToast(t)
					contentStack.SetVisibleChildName("home")
					return
				}

				listBox := gtk.NewListBox()
				listBox.SetSelectionMode(gtk.SelectionNone)
				listBox.AddCSSClass("boxed-list")
				listBox.SetMarginTop(12)
				listBox.SetMarginBottom(12)
				listBox.SetMarginStart(12)
				listBox.SetMarginEnd(12)

				searchEntry := gtk.NewSearchEntry()
				searchEntry.SetPlaceholderText("Search ROMs...")
				searchEntry.SetMarginTop(6)
				searchEntry.SetMarginBottom(6)
				searchEntry.SetMarginStart(12)
				searchEntry.SetMarginEnd(12)
				searchEntry.SetVisible(false)

				listBox.SetFilterFunc(func(row *gtk.ListBoxRow) bool {
					query := strings.ToLower(searchEntry.Text())
					if query == "" {
						return true
					}
					return strings.Contains(strings.ToLower(row.Name()), query)
				})

				searchEntry.ConnectSearchChanged(func() {
					listBox.InvalidateFilter()
				})

				currentSearchEntry = searchEntry
				currentListBox = listBox

				currentRoms = roms
				currentChecks = make([]*gtk.CheckButton, len(roms))

				for i, rom := range roms {
					i := i
					row := adw.NewActionRow()
					row.SetTitle(markupEscape(rom.DisplayName()))
					row.SetSubtitle(markupEscape(rom.FsName))
					row.SetName(strings.ToLower(rom.DisplayName() + " " + rom.FsName))

					check := gtk.NewCheckButton()
					if syncedIDs[rom.ID] {
						check.SetActive(true)
					}
					currentChecks[i] = check
					row.AddPrefix(check)
					row.SetActivatableWidget(check)

					listBox.Append(row)
				}

				scrolled := gtk.NewScrolledWindow()
				scrolled.SetChild(listBox)
				scrolled.SetVExpand(true)

				pageBox := gtk.NewBox(gtk.OrientationVertical, 0)
				pageBox.Append(searchEntry)
				pageBox.Append(scrolled)

				if currentPlatformPage != nil {
					contentStack.Remove(currentPlatformPage)
				}
				contentStack.AddNamed(pageBox, "platform")
				contentStack.SetVisibleChildName("platform")
				searchBtn.SetSensitive(true)
				currentPlatformPage = pageBox
				syncBtn.SetSensitive(true)
			})
		}()
	}

	splitView := adw.NewNavigationSplitView()
	splitView.InsertActionGroup("tofromm", ag)
	splitView.SetMinSidebarWidth(200)
	splitView.SetMaxSidebarWidth(280)
	splitView.SetSidebar(buildSidebar(c, platforms, onSelect, func() {
		contentPage.SetTitle("Home")
		syncBtn.SetVisible(homeSync != nil)
		syncBtn.SetSensitive(homeSync != nil)
		searchBtn.SetVisible(false)
		searchBtn.SetActive(false)
		contentStack.SetVisibleChildName("home")
	}, menuBtn))
	splitView.SetContent(contentPage)
	splitView.AddController(keyCtrl)

	return splitView
}

func buildSidebar(c *client.Client, platforms []client.Platform, onSelect func(client.Platform), onHome func(), menuBtn *gtk.MenuButton) *adw.NavigationPage {
	box := gtk.NewBox(gtk.OrientationVertical, 0)

	homeList := gtk.NewListBox()
	homeList.SetSelectionMode(gtk.SelectionNone)
	homeList.AddCSSClass("navigation-sidebar")

	homeRow := adw.NewActionRow()
	homeRow.SetTitle("Home")
	homeRow.SetActivatable(true)

	homeRow.SetIconName("go-home-symbolic")
	homeRow.ConnectActivated(func() { onHome() })

	homeList.Append(homeRow)
	box.Append(homeList)

	consolesLabel := gtk.NewLabel("Consoles")
	consolesLabel.AddCSSClass("heading")
	consolesLabel.SetXAlign(0)
	consolesLabel.SetMarginTop(18)
	consolesLabel.SetMarginBottom(6)
	consolesLabel.SetMarginStart(12)
	consolesLabel.SetMarginEnd(12)
	box.Append(consolesLabel)

	platformList := gtk.NewListBox()
	platformList.SetSelectionMode(gtk.SelectionNone)
	platformList.AddCSSClass("navigation-sidebar")

	for _, p := range platforms {
		p := p
		row := adw.NewActionRow()
		row.SetTitle(p.Name)

		img := gtk.NewImage()
		img.SetFromIconName("media-optical-symbolic")
		img.SetPixelSize(32)
		row.AddPrefix(img)

		go func() {
			iconURL := ""
			for _, ext := range []string{".svg", ".ico"} {
				url := c.BaseURL + "/assets/platforms/" + p.Slug + ext
				resp, err := http.Get(url)
				if resp != nil {
					resp.Body.Close()
				}
				if err == nil && resp.StatusCode == http.StatusOK {
					iconURL = url
					break
				}
			}
			if iconURL == "" {
				return
			}

			resp, err := http.Get(iconURL)
			if err != nil || resp.StatusCode != http.StatusOK {
				return
			}
			defer resp.Body.Close()
			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return
			}
			loader := pixbuf.NewPixbufLoader()
			loader.SetSize(32, 32)
			if err := loader.Write(data); err != nil {
				return
			}
			loader.Close()
			pb := loader.Pixbuf()
			if pb == nil {
				return
			}
			glib.IdleAdd(func() {
				img.SetFromPixbuf(pb)
			})
		}()

		row.SetActivatable(true)
		row.ConnectActivated(func() {
			onSelect(p)
		})
		platformList.Append(row)
	}
	box.Append(platformList)

	scrolled := gtk.NewScrolledWindow()
	scrolled.SetChild(box)
	scrolled.SetVExpand(true)

	sidebarHeader := adw.NewHeaderBar()
	sidebarHeader.PackEnd(menuBtn)
	toolbar := adw.NewToolbarView()
	toolbar.AddTopBar(sidebarHeader)
	toolbar.SetContent(scrolled)

	return adw.NewNavigationPage(toolbar, "tofromm")
}

func markupEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
