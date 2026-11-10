package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/oneo-me/nsgo/nsgotest"

	"github.com/oneo-me/nsgo-localsend/internal/localsend"
)

// The README's screenshots in assets/screenshots: the nearby page with a
// few devices, one test per language, since a phone shows one app per
// process. The Mac's come with its window frame:
//
//	go tool nsgo test -target macos -run TestScreenshot -env HANDO_WINDOW_SHOTS=$PWD/assets/screenshots
//
// The phones' are .nsgo/test/screenshots/<target>/nearby-<locale>.png
// from -target ios_simulator and android, saved as ios-<locale>.png and
// android-<locale>.png.

func TestScreenshotEnglish(t *testing.T) { screenshotNearby(t, "en") }

func TestScreenshotChinese(t *testing.T) { screenshotNearby(t, "zh-Hans") }

// screenshotPeers are the devices the screenshots show.
var screenshotPeers = []localsend.Device{
	{Alias: "Ada's iPhone", DeviceModel: "iPhone", DeviceType: localsend.DeviceMobile},
	{Alias: "Studio PC", DeviceModel: "Windows", DeviceType: localsend.DeviceDesktop},
	{Alias: "Home Server", DeviceModel: "Linux", DeviceType: localsend.DeviceServer},
}

func screenshotNearby(t *testing.T, locale string) {
	alias := map[string]string{"ios_simulator": "My iPhone", "android": "My Pixel"}[nsgotest.Target()]
	if alias == "" {
		alias = "My Mac"
	}
	port := localsend.DefaultPort
	if ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port)); err != nil {
		port = freePort(t)
	} else {
		ln.Close()
	}
	s := settings{Alias: alias, Port: port, Protocol: localsend.ProtocolHTTPS, ReceiveDir: t.TempDir(),
		VerifyChecksums: true, Locale: locale}
	data, _ := json.Marshal(s)
	app := nsgotest.Start(t, Root, nsgotest.Size(1000, 720), nsgotest.Storage(map[string]string{"settings": string(data)}))
	waitTLSServer(t, port)

	for i, d := range screenshotPeers {
		id, err := localsend.NewIdentity()
		if err != nil {
			t.Fatal(err)
		}
		d.Version, d.Protocol = localsend.Version, localsend.ProtocolHTTPS
		d.Fingerprint = id.ClaimedFingerprint(localsend.ProtocolHTTPS)
		server := localsend.NewServer(localsend.ServerConfig{Identity: id, Addr: "127.0.0.1:0", Info: func() localsend.Device { return d }})
		if err := server.Start(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { server.Stop(context.Background()) })
		d.Port = server.Port()
		client := localsend.NewClient(id)
		t.Cleanup(client.Close)
		at := localsend.Device{IP: "127.0.0.1", Port: port, Protocol: localsend.ProtocolHTTPS}
		if _, err := client.Register(context.Background(), at, d); err != nil {
			t.Fatal(err)
		}
		app.WaitFor("id:device-"+shortFingerprint(d.Fingerprint), 5*time.Second)
		if i == 0 {
			app.Press("id:device-" + shortFingerprint(d.Fingerprint) + " >> id:favorite-device")
		}
	}
	app.Screenshot("nearby-" + locale)
	if dir := os.Getenv("HANDO_WINDOW_SHOTS"); dir != "" && nsgotest.Target() == "macos" {
		windowShot(t, filepath.Join(dir, "macos-"+locale+".png"))
	}
}

// windowShotSource prints the number of the process's largest window,
// the app's own beside any the system made for it.
const windowShotSource = `import CoreGraphics
let pid = Int(CommandLine.arguments[1])!
let list = CGWindowListCopyWindowInfo([.optionOnScreenOnly], kCGNullWindowID) as! [[String: Any]]
var best = (0, 0.0)
for w in list where (w[kCGWindowOwnerPID as String] as? Int) == pid && (w[kCGWindowLayer as String] as? Int) == 0 {
	let b = w[kCGWindowBounds as String] as! [String: Double]
	let area = b["Width"]! * b["Height"]!
	if area > best.1 { best = (w[kCGWindowNumber as String] as! Int, area) }
}
if best.0 != 0 { print(best.0) }
`

// windowShot saves the app's macOS window with its frame and shadow, as
// the system's window capture shows it, which the app's own screenshot
// leaves out.
func windowShot(t *testing.T, path string) {
	t.Helper()
	src := filepath.Join(t.TempDir(), "window.swift")
	if err := os.WriteFile(src, []byte(windowShotSource), 0o644); err != nil {
		t.Fatal(err)
	}
	// A window in front shows its traffic lights in color.
	activate := fmt.Sprintf(`tell application "System Events" to set frontmost of (first process whose unix id is %d) to true`, os.Getpid())
	if out, err := exec.Command("osascript", "-e", activate).CombinedOutput(); err != nil {
		t.Logf("bringing the window to the front: %v %s", err, out)
	}
	time.Sleep(300 * time.Millisecond)
	// The window server may not have the window's image yet.
	var last string
	for range 10 {
		out, err := exec.Command("swift", src, fmt.Sprint(os.Getpid())).Output()
		id := strings.TrimSpace(string(out))
		if err != nil || id == "" {
			last = fmt.Sprintf("finding the window: %v %s", err, out)
		} else if out, err := exec.Command("screencapture", "-x", "-l"+id, path).CombinedOutput(); err != nil {
			last = fmt.Sprintf("screencapture: %v %s", err, out)
		} else {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal(last)
}

// waitTLSServer waits for the app's HTTPS receiver to answer.
func waitTLSServer(t *testing.T, port int) {
	t.Helper()
	id, err := localsend.NewIdentity()
	if err != nil {
		t.Fatal(err)
	}
	client := localsend.NewClient(id)
	defer client.Close()
	at := localsend.Device{IP: "127.0.0.1", Port: port, Protocol: localsend.ProtocolHTTPS}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		res, err := client.HTTP(at).Get(at.URLBase() + localsend.APIPrefix + "/info")
		if err == nil {
			res.Body.Close()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("the app's receiver did not come up")
}
