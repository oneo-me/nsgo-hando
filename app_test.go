package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/oneo-me/nsgo-localsend/internal/localsend"
	"github.com/oneo-me/nsgo/core"
	"github.com/oneo-me/nsgo/nsgotest"
)

func TestMain(m *testing.M) { nsgotest.Main(m) }

// skipIfNoSidebar skips a test that needs the sidebar on a target whose
// screen caps the window to a phone's width.
func skipIfNoSidebar(t *testing.T, app *nsgotest.App) {
	t.Helper()
	if !app.Exists("id:nav-settings") {
		t.Skip("the target caps the window to the phone layout")
	}
}

// skipIfNoTabs skips a test that presses the drawn tab bar where the
// target shows the system's own, whose items a test cannot press.
func skipIfNoTabs(t *testing.T, app *nsgotest.App) {
	t.Helper()
	if !app.Exists("id:tab-bar >> role:tab") {
		t.Skip("the target shows the system's tab bar, which this test cannot press")
	}
}

func TestDesktopLayout(t *testing.T) {
	t.Setenv("LOCALSEND_OFFLINE", "1")
	app := nsgotest.Start(t, Root, nsgotest.Size(1100, 760))
	skipIfNoSidebar(t, app)
	if !app.Exists("id:this-device") {
		t.Fatal("this device card is missing")
	}
	if !app.Exists("id:nearby-empty") {
		t.Fatal("the empty nearby state is missing")
	}
	app.Press("id:nav-settings")
	if !app.Exists("id:page-settings") {
		t.Fatal("settings page did not open")
	}
	app.Type("id:settings-alias", "Ada")
	if got := app.Find("id:settings-alias").Text; !strings.Contains(got, "Ada") {
		t.Fatalf("alias field = %q", got)
	}
	app.Press("id:nav-transfers")
	if !app.Exists("id:transfers-empty") {
		t.Fatal("the empty transfers state is missing")
	}
	app.Press("id:nav-nearby")
	if !app.Exists("id:nearby-empty") {
		t.Fatal("nearby did not come back")
	}
}

func TestNarrowLayout(t *testing.T) {
	t.Setenv("LOCALSEND_OFFLINE", "1")
	app := nsgotest.Start(t, Root, nsgotest.Size(420, 820))
	skipIfNoTabs(t, app)
	if !app.Exists("id:tab-nearby") {
		t.Fatal("tab bar is missing")
	}
	app.Press("id:tab-bar >> role:tab >> nth=2")
	if !app.Exists("id:settings-page") {
		t.Fatal("settings tab did not open")
	}
	app.Press("id:tab-bar >> role:tab >> nth=1")
	if !app.Exists("id:transfers-empty") {
		t.Fatal("transfers tab did not open")
	}
}

func TestLanguageSwitch(t *testing.T) {
	t.Setenv("LOCALSEND_OFFLINE", "1")
	app := nsgotest.Start(t, Root, nsgotest.Size(1100, 760))
	skipIfNoSidebar(t, app)
	app.Press("id:nav-settings")
	app.Press("id:settings-language")
	app.Press("id:settings-language-2")
	app.WaitFor("text:设备身份", 2*time.Second)
	if !app.Exists("id:settings-page") {
		t.Fatal("settings page closed on language change")
	}
}

func TestMacMenu(t *testing.T) {
	t.Setenv("LOCALSEND_OFFLINE", "1")
	app := nsgotest.Start(t, Root, nsgotest.Size(1100, 760), nsgotest.Storage(map[string]string{"settings": `{"locale":"en"}`}))
	if len(app.Menu) == 0 {
		t.Skip("only macOS shows the app's menu bar")
	}
	app.ChooseMenu("View", "Transfers")
	if !app.Exists("id:transfers-empty") {
		t.Fatal("the View menu did not open transfers")
	}
	app.ChooseMenu("Hando", "Settings…")
	if !app.Exists("id:settings-alias") {
		t.Fatal("Settings… did not open settings")
	}
	app.ChooseMenu("View", "Show Sidebar")
	if app.Exists("id:nav-settings") {
		t.Fatal("the sidebar is still shown")
	}
}

// TestSettingsSwitch turns quick save on with the platform's own switch
// on macOS, and with ui's elsewhere.
func TestSettingsSwitch(t *testing.T) {
	t.Setenv("LOCALSEND_OFFLINE", "1")
	opts := []nsgotest.Option{nsgotest.Size(1100, 760), nsgotest.Storage(map[string]string{"settings": `{"locale":"en"}`})}
	if nsgotest.Target() == "" {
		opts = append(opts, nsgotest.Environment(core.Environment{OS: core.OSMacOS, Locale: "en-US"}))
	}
	app := nsgotest.Start(t, Root, opts...)
	skipIfNoSidebar(t, app)
	app.Press("id:nav-settings")
	const sw = "id:settings-quick-save"
	on := func() bool {
		v := app.FindMounted(sw)
		if v.Component != "" {
			return v.NativeProps["checked"] == true
		}
		return v.Props.Checked
	}
	if on() {
		t.Fatal("quick save starts on")
	}
	native := app.FindMounted(sw).Component != ""
	if apple := nsgotest.Target() == "" || nsgotest.Target() == "macos"; native != apple {
		t.Fatalf("system switch shown: %v, want %v", native, apple)
	}
	if native && nsgotest.Target() == "" {
		app.Emit(sw, "change", true)
	} else {
		app.Press(sw)
	}
	if !on() {
		t.Fatal("quick save stayed off")
	}
}

func TestSidebarFooterOpensSettings(t *testing.T) {
	t.Setenv("LOCALSEND_OFFLINE", "1")
	app := nsgotest.Start(t, Root, nsgotest.Size(1100, 760))
	skipIfNoSidebar(t, app)
	app.Press("id:sidebar-device")
	if !app.Exists("id:settings-alias") {
		t.Fatal("the sidebar's device did not open settings")
	}
}

// TestOpenReceivedFolder opens a received folder in the file manager on
// a desktop, and offers its files to the share sheet on a phone, which has
// no other way to reach them.
func TestOpenReceivedFolder(t *testing.T) {
	t.Setenv("LOCALSEND_OFFLINE", "1")
	dir := filepath.Join(t.TempDir(), "photos")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	var files []transferFile
	for _, name := range []string{"a.jpg", "b.jpg"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
		files = append(files, transferFile{ID: name, Name: "photos/" + name, Size: 5, Done: 5, Status: "done", Path: path})
	}
	history, _ := json.Marshal([]transfer{{ID: "s1", Direction: "receive", Peer: localsend.Device{Alias: "Phone"},
		Status: statusCompleted, Files: files, Total: 10, Done: 10, Started: time.Now(), Ended: time.Now(), SavedDir: dir}})
	start := func(t *testing.T, os core.OS) *nsgotest.App {
		app := nsgotest.Start(t, Root, nsgotest.Size(1100, 760),
			nsgotest.Environment(core.Environment{OS: os, Locale: "en-US"}),
			nsgotest.Storage(map[string]string{"history": string(history)}))
		skipIfNoSidebar(t, app)
		app.Press("id:nav-transfers")
		app.WaitFor("text:Completed", 2*time.Second)
		if app.Exists("id:open-received") {
			t.Fatal("a received folder offered to open as one file")
		}
		return app
	}
	t.Run("desktop", func(t *testing.T) {
		app := start(t, core.OSMacOS)
		if !app.Exists("id:open-received-folder") {
			t.Skip("the target has no file manager")
		}
		app.Press("id:open-received-folder")
		app.Advance(time.Second)
		if len(app.OpenedFiles) != 1 || app.OpenedFiles[0].Path != dir {
			t.Fatalf("opened = %+v", app.OpenedFiles)
		}
	})
	t.Run("phone", func(t *testing.T) {
		app := start(t, core.OSIOS)
		if !app.Exists("id:share-received") {
			t.Skip("the target has no share sheet")
		}
		app.Press("id:share-received")
		app.Advance(time.Second)
		if len(app.Shares) != 1 || len(app.Shares[0].Files) != 2 {
			t.Fatalf("shares = %+v", app.Shares)
		}
	})
}
