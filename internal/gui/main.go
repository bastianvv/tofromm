package gui

import (
	"fmt"
	"os"
	"strings"

	"github.com/bastianvv/tofromm/internal/client"
	"github.com/bastianvv/tofromm/internal/emulator"
	syncer "github.com/bastianvv/tofromm/internal/sync"
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
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
		var platforms []client.Platform
		for _, p := range allPlatforms {
			if p.RomCount > 0 {
				platforms = append(platforms, p)
			}
		}
		glib.IdleAdd(func() {
			page.SetChild(buildSplitView(nav, overlay, c, platforms))
		})
	}()
	return page
}

func buildSplitView(nav *adw.NavigationView, overlay *adw.ToastOverlay, c *client.Client, platforms []client.Platform) *adw.NavigationSplitView {
	contentStack := gtk.NewStack()
	contentStack.SetTransitionType(gtk.StackTransitionTypeCrossfade)

	home := adw.NewStatusPage()
	home.SetTitle("Coming Soon")
	home.SetDescription("Your home screen will show recently played games and collection stats.")
	home.SetIconName("view-dashboard-symbolic")

	contentStack.AddNamed(home, "home")
	contentStack.SetVisibleChildName("home")

	syncBtn := gtk.NewButton()
	syncBtn.SetLabel("Sync")
	syncBtn.AddCSSClass("suggested-action")
	syncBtn.AddCSSClass("pill")
	syncBtn.SetVAlign(gtk.AlignCenter)
	syncBtn.SetSensitive(false)

	contentHeader := adw.NewHeaderBar()
	menuBtn := gtk.NewMenuButton()
	menuBtn.SetIconName("open-menu-symbolic")
	menuBtn.SetVAlign(gtk.AlignCenter)

	m := gio.NewMenu()
	m.Append("Emulator Configuration", "tofromm.emulator-config")
	m.Append("About", "tofromm.about")
	menuBtn.SetMenuModel(m)

	emuAction := gio.NewSimpleAction("emulator-config", nil)
	emuAction.ConnectActivate(func(_ *glib.Variant) {
		nav.Push(newEmulatorSetupPage(nav, overlay, platforms))
	})

	aboutAction := gio.NewSimpleAction("about", nil)
	aboutAction.ConnectActivate(func(_ *glib.Variant) {
		d := adw.NewAboutDialog()
		d.SetApplicationName("Tofromm")
		d.SetApplicationIcon(appID)
		d.SetVersion("0.5")
		d.SetDeveloperName("bastianvv")
		d.SetDevelopers([]string{"bastianvv"})
		d.SetComments("Sync ROMs and saves between your Linux machine and a ROMM server.")
		d.SetWebsite("https://github.com/bastianvv/tofromm")
		d.SetIssueURL("https://github.com/bastianvv/tofromm/issues")
		d.SetCopyright("© 2025 bastianvv")
		d.SetLicenseType(gtk.LicenseMITX11)
		d.Present(menuBtn)
	})

	ag := gio.NewSimpleActionGroup()
	ag.Insert(emuAction)
	ag.Insert(aboutAction)

	contentHeader.PackEnd(syncBtn)

	contentToolbar := adw.NewToolbarView()
	contentToolbar.AddTopBar(contentHeader)
	contentToolbar.SetContent(contentStack)

	contentPage := adw.NewNavigationPage(contentToolbar, "tofromm")

	var currentRoms []client.Rom
	var currentChecks []*gtk.CheckButton

	syncBtn.ConnectClicked(func() {
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

		syncBtn.SetSensitive(false)
		t := adw.NewToast("Sync started…")
		t.SetTimeout(2)
		overlay.AddToast(t)

		go func() {
			result, err := syncer.Run(syncer.Options{
				Client:     c,
				EmuConfigs: emuConfigs,
				Selected:   selected,
				OnProgress: func(msg string) {
					fmt.Fprintln(os.Stderr, msg)
				},
				OnConflict: func(romName, serverTime, reason string) bool {
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
				if err != nil {
					t := adw.NewToast("Sync failed: " + err.Error())
					t.SetTimeout(5)
					overlay.AddToast(t)
					return
				}
				t := adw.NewToast(fmt.Sprintf("Done — %d succeeded, %d failed", result.Completed, result.Failed))
				t.SetTimeout(5)
				overlay.AddToast(t)
			})
		}()
	})

	onSelect := func(p client.Platform) {
		contentPage.SetTitle(p.Name)
		syncBtn.SetSensitive(false)
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

				currentRoms = roms
				currentChecks = make([]*gtk.CheckButton, len(roms))

				for i, rom := range roms {
					i := i
					row := adw.NewActionRow()
					row.SetTitle(markupEscape(rom.DisplayName()))
					row.SetSubtitle(markupEscape(rom.FsName))

					check := gtk.NewCheckButton()
					currentChecks[i] = check
					row.AddPrefix(check)
					row.SetActivatableWidget(check)

					listBox.Append(row)
				}

				scrolled := gtk.NewScrolledWindow()
				scrolled.SetChild(listBox)
				scrolled.SetVExpand(true)

				pageName := "platform-" + p.FsSlug
				if old := contentStack.ChildByName(pageName); old != nil {
					contentStack.Remove(old)
				}
				contentStack.AddNamed(scrolled, pageName)
				contentStack.SetVisibleChildName(pageName)
				syncBtn.SetSensitive(true)
			})
		}()
	}

	splitView := adw.NewNavigationSplitView()
	splitView.InsertActionGroup("tofromm", ag)
	splitView.SetMinSidebarWidth(200)
	splitView.SetMaxSidebarWidth(280)
	splitView.SetSidebar(buildSidebar(platforms, onSelect, func() { contentStack.SetVisibleChildName("home") }, menuBtn))
	splitView.SetContent(contentPage)

	return splitView
}

func buildSidebar(platforms []client.Platform, onSelect func(client.Platform), onHome func(), menuBtn *gtk.MenuButton) *adw.NavigationPage {
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
