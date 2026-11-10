package main

import (
	"strings"
	"testing"
	"time"

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
