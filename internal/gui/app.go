//go:build !nogui

package gui

import (
	"os"

	"github.com/bastianvv/tofromm/internal/client"
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/spf13/viper"
)

const appID = "io.github.bastianvv.tofromm"

var serviceFileContent []byte

func Run(serviceFile []byte) {
	serviceFileContent = serviceFile
	app := adw.NewApplication(appID, gio.ApplicationFlagsNone)
	app.Connect("activate", func() {
		buildWindow(app)
	})
	os.Exit(app.Run(os.Args[:1]))
}

func buildWindow(app *adw.Application) {
	win := adw.NewApplicationWindow(&app.Application)
	win.SetTitle("Tofromm")
	win.SetDefaultSize(900, 600)

	nav := adw.NewNavigationView()
	overlay := adw.NewToastOverlay()
	overlay.SetChild(nav)

	if isFirstRun() {
		nav.Add(newServerSetupPage(nav, overlay))
	} else {
		nav.Add(newMainPage(nav, overlay))
	}

	win.SetContent(overlay)

	provider := gtk.NewCSSProvider()
	provider.LoadFromString(`.cover-art { border-radius: 12px; }`)
	gtk.StyleContextAddProviderForDisplay(gdk.DisplayGetDefault(), provider, 600)

	win.Present()
}

func isFirstRun() bool {
	return viper.GetString("server") == ""
}

func newClientFromConfig() *client.Client {
	return client.NewClient(
		viper.GetString("server"),
		viper.GetString("username"),
		viper.GetString("password"),
	)
}
