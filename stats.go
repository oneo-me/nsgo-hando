package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/oneo-me/nsgo/core"
)

// The app counts its use in Umami under a site of its own: a pageview for
// each page shown, tagged with the platform and carrying a random ID kept
// in the storage, so that one install is one visitor. Builds of nsgo dev
// (statsBuild), test binaries and the web report nothing, and the settings
// turn it off. Nothing about devices, files or transfers is sent.
const (
	statsURL     = "https://umami.oneo.me/api/send"
	statsWebsite = "c49d1ed4-6be7-42b4-a188-d44fe96d0b50"
	statsHost    = "hando.oneo.me"
)

// installKey is the storage key of the install's ID.
const installKey = "installID"

// Umami measures a visit from its first event to its last, so while a page
// is read the app sends a heartbeat each statsBeat, for statsIdle after the
// last page shown: a window left open does not count as use.
const (
	statsBeat = time.Minute
	statsIdle = 5 * time.Minute
)

// statsShown reports whether the app counts its use where it runs, and so
// shows the settings' switch: everywhere but the web.
func statsShown(cx *core.Scope) bool {
	return cx.Env().OS() != core.OSWeb
}

// statsPayload is an event of Umami's /api/send: a pageview without a
// name.
type statsPayload struct {
	Website  string `json:"website"`
	Hostname string `json:"hostname"`
	URL      string `json:"url"`
	Title    string `json:"title"`
	Language string `json:"language,omitempty"`
	Screen   string `json:"screen,omitempty"`
	Tag      string `json:"tag"`
	ID       string `json:"id"`
	Name     string `json:"name,omitempty"`
}

// reportUsage sends a pageview whenever the page shown changes, and the
// heartbeats while it is read. Events are made on the engine's goroutine
// and sent from another, oldest first; one that fails is dropped.
func (a *app) reportUsage(cx *core.Scope) {
	if !statsBuild || testing.Testing() || !statsShown(cx) {
		return
	}
	store := cx.Storage()
	id, ok := store.GetItem(installKey)
	if !ok {
		b := make([]byte, 16)
		rand.Read(b)
		id = hex.EncodeToString(b)
		store.SetItem(installKey, id)
	}
	env := cx.Env()
	agent := statsAgent(env.OS())
	queue := make(chan statsPayload, 32)
	go sendStats(queue, agent)

	var shown string
	var since time.Time
	send := func(name string) {
		if a.settings.Get().StatsOff {
			return
		}
		w := env.Window()
		p := statsPayload{Website: statsWebsite, Hostname: statsHost, URL: "/" + shown, Title: shown,
			Language: env.Locale(), Screen: fmt.Sprintf("%dx%d", int(w.Width), int(w.Height)),
			Tag: string(env.OS()), ID: id, Name: name}
		select {
		case queue <- p:
		default:
		}
	}
	cx.Effect(func() {
		page := a.page.Get()
		core.Untracked(cx, func() bool {
			if page != shown || since.IsZero() {
				shown, since = page, time.Now()
				send("")
			}
			return true
		})
	})
	var beat func()
	beat = func() {
		if time.Since(since) < statsIdle {
			core.Untracked(cx, func() bool { send("heartbeat"); return true })
		}
		cx.AfterFunc(statsBeat, beat)
	}
	cx.AfterFunc(statsBeat, beat)
}

// sendStats posts the queue's events to Umami until it closes.
func sendStats(queue <-chan statsPayload, agent string) {
	client := &http.Client{Timeout: 10 * time.Second}
	for p := range queue {
		body, _ := json.Marshal(map[string]any{"type": "event", "payload": p})
		req, err := http.NewRequest(http.MethodPost, statsURL, bytes.NewReader(body))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		if agent != "" {
			req.Header.Set("User-Agent", agent)
		}
		if resp, err := client.Do(req); err == nil {
			resp.Body.Close()
		}
	}
}

// statsAgent is the User-Agent the app sends, as a browser on the platform
// would say it, for Umami to tell the system and not take it for a bot as
// it does Go's own.
func statsAgent(os core.OS) string {
	platforms := map[core.OS]string{
		core.OSMacOS:   "Macintosh; Intel Mac OS X 10_15_7",
		core.OSWindows: "Windows NT 10.0; Win64; x64",
		core.OSLinux:   "X11; Linux x86_64",
		core.OSIOS:     "iPhone; CPU iPhone OS 18_0 like Mac OS X",
		core.OSAndroid: "Linux; Android 15",
	}
	p, ok := platforms[os]
	if !ok {
		return ""
	}
	return "Mozilla/5.0 (" + p + ") Hando/" + version
}
