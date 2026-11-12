package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/oneo-me/nsgo/core"
	"github.com/oneo-me/nsgo/i18n"
	"github.com/oneo-me/nsgo/icons/lucide"
	"github.com/oneo-me/nsgo/ui"

	"github.com/oneo-me/nsgo-localsend/internal/localsend"
	"github.com/oneo-me/nsgo-localsend/locales"
)

// desktopWidth is where the window takes the sidebar layout.
const desktopWidth core.Points = 760

// settings are the app's preferences, kept as one JSON value.
type settings struct {
	Alias           string             `json:"alias"`
	Port            int                `json:"port"`
	Protocol        localsend.Protocol `json:"protocol"`
	ReceivePIN      string             `json:"receivePin"`
	QuickSave       bool               `json:"quickSave"`
	ReceiveDir      string             `json:"receiveDir"`
	VerifyChecksums bool               `json:"verifyChecksums"`
	Locale          string             `json:"locale"`
	Appearance      string             `json:"appearance"`
	StatsOff        bool               `json:"statsOff"`
}

// transferFile is one file of a transfer.
type transferFile struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	Done    int64  `json:"done"`
	Status  string `json:"status"`
	Path    string `json:"path,omitempty"`
	Preview string `json:"preview,omitempty"`
	Error   string `json:"error,omitempty"`
}

// transfer is one send or receive session, active or finished.
type transfer struct {
	ID        string           `json:"id"`
	Direction string           `json:"direction"` // send or receive
	Peer      localsend.Device `json:"peer"`
	Status    string           `json:"status"`
	Files     []transferFile   `json:"files"`
	Total     int64            `json:"total"`
	Done      int64            `json:"done"`
	Speed     float64          `json:"-"`
	Started   time.Time        `json:"started"`
	Ended     time.Time        `json:"ended,omitempty"`
	Reason    string           `json:"reason,omitempty"`
	Err       string           `json:"error,omitempty"`
	SavedDir  string           `json:"savedDir,omitempty"`
	// Archive is the zip a received folder arrived as, when the platform
	// cannot open a folder.
	Archive         string `json:"archive,omitempty"`
	NeedPIN         bool   `json:"-"`
	CancelRequested bool   `json:"-"`

	lastAt   time.Time
	lastDone int64
}

// transfer statuses.
const (
	statusWaiting   = "waiting"
	statusActive    = "active"
	statusArchiving = "archiving"
	statusCompleted = "completed"
	statusFailed    = "failed"
	statusCancelled = "cancelled"
	statusDeclined  = "declined"
	statusSkipped   = "skipped"
)

// sendFile is a file chosen for sending.
type sendFile struct {
	ID       string
	Name     string
	Size     int64
	MIME     string
	Modified time.Time
	Preview  string
	Text     bool
	open     func() (io.ReadCloser, error)
	cleanup  func()
}

// sendJob keeps a send that may wait for a PIN.
type sendJob struct {
	TransferID string
	Target     localsend.Device
	Files      []sendFile
	PIN        string
}

// pinPrompt asks for the PIN a receiver wants.
type pinPrompt struct {
	TransferID string
	Target     localsend.Device
	Wrong      bool
}

// incomingRequest is a sender's offer waiting for the app's answer.
type incomingRequest struct {
	Req      *localsend.IncomingRequest
	Selected map[string]bool
}

// app is the running app's state. Components read its signals; background
// work reaches them through cx.Post.
type app struct {
	cx      *core.Scope
	toaster *ui.Toaster

	settings  *core.Signal[settings]
	favorites *core.Signal[[]localsend.Device]
	devices   *core.Signal[[]localsend.Device]
	nearby    *core.Memo[[]localsend.Device]
	wide      *core.Memo[bool]
	hosts     *core.Memo[[]string]
	transfers *core.Signal[[]*transfer]
	history   *core.Signal[[]transfer]
	page      *core.Signal[string]

	refreshing *core.Signal[bool]
	nodeReady  *core.Signal[bool]
	serverErr  *core.Signal[string]
	networkErr *core.Signal[string]

	incoming   *core.Signal[*incomingRequest]
	pinPrompt  *core.Signal[*pinPrompt]
	pinValue   *core.Signal[string]
	sendTarget *core.Signal[*localsend.Device]
	sendFiles  *core.Signal[[]sendFile]
	sendMsg    *core.Signal[string]
	scanning   *core.Signal[bool]
	sidebar    *core.Signal[bool]
	sideWidth  *core.Signal[core.Points]
	portDraft  *core.Signal[string]

	model       string
	dtype       localsend.DeviceType
	offline     bool
	identityDir string
	tempDir     string

	alias atomic.Value // string
	pin   atomic.Value // string

	nodeMu sync.Mutex
	node   *localsend.Node

	jobs    map[string]sendJob
	cancels map[string]context.CancelFunc

	lastLocale     string
	lastAppearance string
}

// Root is the app.
func Root(cx *core.Scope) core.Node {
	a := newApp(cx)
	return core.Dynamic(func(cx *core.Scope) core.Node {
		i18n.Language(cx)
		// Only crossing the breakpoint rebuilds the shell; every other
		// resize lays the same tree out again.
		wide := a.wide.Get()
		// The shell mounts outside the dynamic's own reads, so a keystroke
		// that changes a signal read while building it (this device's
		// name, for one) restyles the controls in place rather than
		// rebuilding the tree under the user's caret.
		return core.Func(func(cx *core.Scope) core.Node { return a.view(cx, wide) })
	})
}

// newApp loads the app's state and starts its receiver.
func newApp(cx *core.Scope) *app {
	a := &app{cx: cx, toaster: ui.NewToaster(cx)}
	a.model, a.dtype = deviceProfile(cx)
	a.offline = os.Getenv("LOCALSEND_OFFLINE") == "1" || cx.Env().OS() == core.OSWeb
	s := a.loadSettings(cx)
	a.settings = core.NewSignal(cx, s)
	a.favorites = core.NewSignalFunc(cx, loadFavorites(cx), nil)
	a.devices = core.NewSignalFunc(cx, []localsend.Device(nil), nil)
	a.nearby = core.NewMemoFunc(cx, a.mergeDevices, sameDevices)
	a.wide = core.NewMemo(cx, func() bool { return cx.Env().Window().Width >= desktopWidth })
	a.transfers = core.NewSignalFunc(cx, []*transfer(nil), nil)
	a.history = core.NewSignalFunc(cx, loadHistory(cx), nil)
	a.page = core.NewSignal(cx, "nearby")
	a.refreshing = core.NewSignal(cx, false)
	a.nodeReady = core.NewSignal(cx, false)
	a.serverErr = core.NewSignal(cx, "")
	a.networkErr = core.NewSignal(cx, "")
	a.hosts = core.NewMemoFunc(cx, func() []string {
		// Listing the interfaces is a system call per interface; do it once
		// each time the receiver comes up rather than on every read.
		_ = a.nodeReady.Get()
		return a.localAddresses()
	}, slices.Equal[[]string])
	a.incoming = core.NewSignal[*incomingRequest](cx, nil)
	a.pinPrompt = core.NewSignal[*pinPrompt](cx, nil)
	a.pinValue = core.NewSignal(cx, "")
	a.sendTarget = core.NewSignal[*localsend.Device](cx, nil)
	a.sendFiles = core.NewSignalFunc(cx, []sendFile(nil), nil)
	a.sendMsg = core.NewSignal(cx, "")
	a.scanning = core.NewSignal(cx, false)
	a.sidebar = core.NewSignal(cx, true)
	a.sideWidth = core.NewSignal(cx, core.Points(236))
	a.portDraft = core.NewSignal(cx, strconv.Itoa(s.Port))
	a.jobs = map[string]sendJob{}
	a.cancels = map[string]context.CancelFunc{}
	a.alias.Store(s.Alias)
	a.pin.Store(s.ReceivePIN)
	a.identityDir = filepath.Join(cx.DataDir(), "identity")
	if cx.DataDir() == "" {
		// Tests and hosts without a data folder get a temporary identity.
		dir, err := os.MkdirTemp("", "hando-identity-*")
		if err == nil {
			a.identityDir, a.tempDir = dir, dir
		}
	}

	cx.SetTheme(appTheme())
	a.applyPreferences(cx, s)
	a.installMenu(cx)
	cx.Effect(func() {
		next := a.settings.Get()
		a.alias.Store(next.Alias)
		a.pin.Store(next.ReceivePIN)
		saveJSON(cx.Storage(), "settings", next)
		a.applyPreferences(cx, next)
	})
	cx.Effect(func() { saveJSON(cx.Storage(), "favorites", a.favorites.Get()) })
	cx.Effect(func() { saveJSON(cx.Storage(), "history", a.history.Get()) })
	a.reportUsage(cx)
	cx.OnExit(func(done func()) {
		a.cancelAllSends()
		if a.tempDir != "" {
			defer os.RemoveAll(a.tempDir)
		}
		node := a.currentNode()
		if node == nil {
			done()
			return
		}
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			node.Stop(ctx)
			done()
		}()
	})
	if a.offline {
		a.nodeReady.Set(false)
		return a
	}
	a.restartNode()
	return a
}

// applyPreferences applies the language and appearance, and restarts the
// receiver when its protocol or checksum preference changed.
func (a *app) applyPreferences(cx *core.Scope, s settings) {
	if s.Locale != a.lastLocale {
		cx.SetLocale(s.Locale)
		a.lastLocale = s.Locale
	}
	if s.Appearance != a.lastAppearance {
		a.lastAppearance = s.Appearance
		switch s.Appearance {
		case "light":
			cx.SetAppearance(core.AppearanceLight)
		case "dark":
			cx.SetAppearance(core.AppearanceDark)
		default:
			cx.SetAppearance(core.AppearanceSystem)
		}
	}
}

// loadSettings reads the saved settings, filling defaults.
func (a *app) loadSettings(cx *core.Scope) settings {
	s := settings{
		Alias:           defaultAlias(cx),
		Port:            localsend.DefaultPort,
		Protocol:        localsend.ProtocolHTTPS,
		VerifyChecksums: true,
	}
	if raw, ok := cx.Storage().GetItem("settings"); ok {
		_ = json.Unmarshal([]byte(raw), &s)
	}
	if strings.TrimSpace(s.Alias) == "" {
		s.Alias = defaultAlias(cx)
	}
	if s.Port < 1 || s.Port > 65535 {
		s.Port = localsend.DefaultPort
	}
	if s.Protocol != localsend.ProtocolHTTP && s.Protocol != localsend.ProtocolHTTPS {
		s.Protocol = localsend.ProtocolHTTPS
	}
	if s.Appearance != "light" && s.Appearance != "dark" {
		s.Appearance = ""
	}
	return s
}

// defaultAlias names this device before the user chooses one.
func defaultAlias(cx *core.Scope) string {
	switch cx.Env().OS() {
	case core.OSMacOS:
		return "My Mac"
	case core.OSWindows:
		return "My PC"
	case core.OSLinux:
		return "My Linux"
	case core.OSIOS:
		return "My iPhone"
	case core.OSAndroid:
		return "My Android"
	case core.OSWeb:
		return "Browser"
	}
	if host, err := os.Hostname(); err == nil && host != "" && host != "localhost" {
		return strings.TrimSuffix(host, ".local")
	}
	return "My device"
}

// deviceProfile is what this platform calls its model and type.
func deviceProfile(cx *core.Scope) (string, localsend.DeviceType) {
	switch cx.Env().OS() {
	case core.OSMacOS:
		return "macOS", localsend.DeviceDesktop
	case core.OSWindows:
		return "Windows", localsend.DeviceDesktop
	case core.OSLinux:
		return "Linux", localsend.DeviceDesktop
	case core.OSIOS:
		return "iOS", localsend.DeviceMobile
	case core.OSAndroid:
		return "Android", localsend.DeviceMobile
	case core.OSWeb:
		return "Web", localsend.DeviceWeb
	}
	if runtime.GOOS == "windows" {
		return "Windows", localsend.DeviceDesktop
	}
	return runtime.GOOS, localsend.DeviceDesktop
}

// defaultReceiveDir is where received files go before the user picks a
// folder: Downloads/Hando on desktops, the app's folder elsewhere.
func defaultReceiveDir(cx *core.Scope) string {
	switch cx.Env().OS() {
	case core.OSMacOS, core.OSWindows, core.OSLinux:
		if home, err := os.UserHomeDir(); err == nil {
			downloads := filepath.Join(home, "Downloads")
			if info, err := os.Stat(downloads); err == nil && info.IsDir() {
				return filepath.Join(downloads, "Hando")
			}
		}
	}
	if dir := cx.DataDir(); dir != "" {
		return filepath.Join(dir, "Received")
	}
	return "Received"
}

// receiveDir is the configured folder, or the default.
func (a *app) receiveDir() string {
	s := a.settings.GetUntracked()
	if s.ReceiveDir != "" {
		return s.ReceiveDir
	}
	return defaultReceiveDir(a.cx)
}

// restartNode stops the receiver and starts it with the current settings.
func (a *app) restartNode() {
	if a.offline {
		return
	}
	favorites := append([]localsend.Device(nil), a.favorites.GetUntracked()...)
	s := a.settings.GetUntracked()
	a.cancelAllSends()
	a.serverErr.Set("")
	a.nodeReady.Set(false)
	a.nodeMu.Lock()
	old := a.node
	a.node = nil
	a.nodeMu.Unlock()
	go func() {
		if old != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			old.Stop(ctx)
			cancel()
		}
		identity, err := localsend.LoadIdentity(a.identityDir)
		if err != nil {
			a.cx.Post(func() { a.serverErr.Set(err.Error()) })
			return
		}
		node, err := localsend.StartNode(localsend.NodeConfig{
			Identity:        identity,
			Alias:           a.aliasName,
			Port:            s.Port,
			Protocol:        s.Protocol,
			DeviceModel:     a.model,
			DeviceType:      a.dtype,
			PIN:             a.pinCode,
			VerifyChecksums: s.VerifyChecksums,
			OnRegister:      func(d localsend.Device) { a.cx.Post(func() { a.addDevice(d) }) },
			OnDevice:        func(d localsend.Device) { a.cx.Post(func() { a.addDevice(d) }) },
			OnPrepare:       func(req *localsend.IncomingRequest) { a.cx.Post(func() { a.prepare(req) }) },
			OnProgress:      func(p localsend.Progress) { a.cx.Post(func() { a.receiveProgress(p) }) },
			OnSessionEnd: func(r localsend.SessionResult) {
				a.cx.Post(func() { a.receiveDone(r) })
			},
			OnError: func(err error) { a.cx.Post(func() { a.networkErr.Set(err.Error()) }) },
		})
		if err != nil {
			a.cx.Post(func() { a.serverErr.Set(err.Error()) })
			return
		}
		a.cx.Post(func() {
			a.nodeMu.Lock()
			a.node = node
			a.nodeMu.Unlock()
			a.nodeReady.Set(true)
			if node.DiscoveryErr != nil {
				a.networkErr.Set(node.DiscoveryErr.Error())
			}
			go a.discover(node, favorites)
		})
	}()
}

// aliasName is the alias the receiver reports, safe to read on any
// goroutine.
func (a *app) aliasName() string {
	name, _ := a.alias.Load().(string)
	return name
}

// pinCode is the receive PIN, safe to read on any goroutine.
func (a *app) pinCode() string {
	pin, _ := a.pin.Load().(string)
	return pin
}

// currentNode is the running receiver, nil while starting or failed.
func (a *app) currentNode() *localsend.Node {
	a.nodeMu.Lock()
	defer a.nodeMu.Unlock()
	return a.node
}

// discover announces this device, probes favorites and, when nothing
// answered, scans the local networks once.
func (a *app) discover(node *localsend.Node, favorites []localsend.Device) {
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()
	node.Announce(ctx)
	for _, f := range favorites {
		if f.IP == "" || ctx.Err() != nil {
			continue
		}
		probeCtx, stop := context.WithTimeout(ctx, 3*time.Second)
		_, _ = node.Probe(probeCtx, f.IP, f.Port, f.Protocol)
		stop()
	}
	// Give peers a moment to answer the announcement; a scan of every
	// address is slow and only needed when nobody did.
	select {
	case <-ctx.Done():
		return
	case <-time.After(3 * time.Second):
	}
	a.cx.Post(func() {
		if len(a.devices.GetUntracked()) == 0 && !a.refreshing.GetUntracked() {
			a.refresh()
		}
	})
}

// refresh announces, probes favorites and scans the local networks.
func (a *app) refresh() {
	if a.refreshing.Get() {
		return
	}
	node := a.currentNode()
	if node == nil {
		a.toast(locales.ServerStopped(a.cx))
		return
	}
	favorites := append([]localsend.Device(nil), a.favorites.Get()...)
	a.refreshing.Set(true)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
		defer cancel()
		node.Announce(ctx)
		for _, f := range favorites {
			if f.IP == "" || ctx.Err() != nil {
				continue
			}
			probeCtx, stop := context.WithTimeout(ctx, 3*time.Second)
			_, _ = node.Probe(probeCtx, f.IP, f.Port, f.Protocol)
			stop()
		}
		node.Scan(ctx)
		a.cx.Post(func() { a.refreshing.Set(false) })
	}()
}

// addDevice records a peer discovery heard.
func (a *app) addDevice(d localsend.Device) {
	if d.Fingerprint == "" || d.IP == "" || a.isSelf(d) {
		return
	}
	if d.LastSeen.IsZero() {
		d.LastSeen = time.Now()
	}
	// Peers answer every announcement and scan; one that says nothing new
	// leaves the list alone, so the screen does not rebuild for it.
	list := a.devices.GetUntracked()
	for i := range list {
		if strings.EqualFold(list[i].Fingerprint, d.Fingerprint) {
			if sameDevice(list[i], d) {
				return
			}
			next := append([]localsend.Device(nil), list...)
			next[i] = d
			a.devices.Set(next)
			return
		}
	}
	a.devices.Set(append(append([]localsend.Device(nil), list...), d))
}

// sameDevice reports whether two records of a device show the same.
func sameDevice(x, y localsend.Device) bool {
	if online(x) != online(y) {
		return false
	}
	x.LastSeen, y.LastSeen = time.Time{}, time.Time{}
	return x == y
}

// sameDevices reports whether two device lists show the same.
func sameDevices(x, y []localsend.Device) bool {
	if len(x) != len(y) {
		return false
	}
	for i := range x {
		if !sameDevice(x[i], y[i]) {
			return false
		}
	}
	return true
}

// isSelf reports whether d is this device.
func (a *app) isSelf(d localsend.Device) bool {
	node := a.currentNode()
	if node == nil {
		return false
	}
	self := node.Self()
	return d.Fingerprint != "" && (strings.EqualFold(d.Fingerprint, self.Fingerprint) ||
		strings.EqualFold(d.Fingerprint, node.Identity().HTTPFingerprint))
}

// favorite reports whether the fingerprint is a favorite.
func (a *app) favorite(fingerprint string) bool {
	for _, f := range a.favorites.GetUntracked() {
		if strings.EqualFold(f.Fingerprint, fingerprint) {
			return true
		}
	}
	return false
}

// favoriteTracked is favorite in a binding that follows the set.
func (a *app) favoriteTracked(fingerprint string) bool {
	for _, f := range a.favorites.Get() {
		if strings.EqualFold(f.Fingerprint, fingerprint) {
			return true
		}
	}
	return false
}

// toggleFavorite pins or unpins a device.
func (a *app) toggleFavorite(d localsend.Device) {
	found := false
	list := append([]localsend.Device(nil), a.favorites.Get()...)
	out := list[:0]
	for _, f := range list {
		if strings.EqualFold(f.Fingerprint, d.Fingerprint) {
			found = true
			continue
		}
		out = append(out, f)
	}
	if !found {
		out = append(out, d)
	}
	a.favorites.Set(out)
}

// mergedDevices are favorites and discovered devices for the nearby list,
// recomputed only when either changes.
func (a *app) mergedDevices() []localsend.Device { return a.nearby.Get() }

// mergeDevices merges favorites and discovered devices: favorites first,
// online before offline, by name.
func (a *app) mergeDevices() []localsend.Device {
	discovered := a.devices.Get()
	byFP := make(map[string]localsend.Device, len(discovered))
	for _, d := range discovered {
		byFP[strings.ToUpper(d.Fingerprint)] = d
	}
	var out []localsend.Device
	seen := map[string]bool{}
	for _, f := range a.favorites.Get() {
		key := strings.ToUpper(f.Fingerprint)
		device, online := byFP[key]
		if !online {
			device = f
		}
		device.Favorite = true
		out = append(out, device)
		seen[key] = true
	}
	for _, d := range discovered {
		key := strings.ToUpper(d.Fingerprint)
		if seen[key] {
			continue
		}
		out = append(out, d)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Favorite != out[j].Favorite {
			return out[i].Favorite
		}
		if online(out[i]) != online(out[j]) {
			return online(out[i])
		}
		return strings.ToLower(out[i].Alias) < strings.ToLower(out[j].Alias)
	})
	return out
}

// online reports whether a device is in the discovered set.
func online(d localsend.Device) bool { return !d.LastSeen.IsZero() }

// selfDevice is this device as the list and settings show it.
func (a *app) selfDevice() localsend.Device {
	if node := a.currentNode(); node != nil {
		return node.Self()
	}
	s := a.settings.Get()
	return localsend.Device{Alias: s.Alias, Version: localsend.Version, DeviceModel: a.model,
		DeviceType: a.dtype, Port: s.Port, Protocol: s.Protocol}
}

// localAddresses are the addresses peers can reach this device at.
func (a *app) localAddresses() []string {
	if node := a.currentNode(); node != nil {
		return node.LocalAddresses()
	}
	return nil
}

// toast shows a message.
func (a *app) toast(title string) {
	a.toaster.Show(ui.ToastContent{Type: ui.ToastInfo, Title: title})
}

// toastSuccess shows a success.
func (a *app) toastSuccess(title, description string) {
	a.toaster.Show(ui.ToastContent{Type: ui.ToastSuccess, Title: title, Description: description})
}

// toastError shows a failure.
func (a *app) toastError(title, description string) {
	a.toaster.Show(ui.ToastContent{Type: ui.ToastError, Title: title, Description: description})
}

// cancelAllSends cancels running sends.
func (a *app) cancelAllSends() {
	for _, cancel := range a.cancels {
		cancel()
	}
}

// loadFavorites reads the pinned devices.
func loadFavorites(cx *core.Scope) []localsend.Device {
	var out []localsend.Device
	if raw, ok := cx.Storage().GetItem("favorites"); ok {
		_ = json.Unmarshal([]byte(raw), &out)
	}
	return out
}

// loadHistory reads recent finished transfers.
func loadHistory(cx *core.Scope) []transfer {
	var out []transfer
	if raw, ok := cx.Storage().GetItem("history"); ok {
		_ = json.Unmarshal([]byte(raw), &out)
	}
	if len(out) > 50 {
		out = out[:50]
	}
	return out
}

// saveJSON stores v under key, dropping a value too large for storage.
func saveJSON(store core.Storage, key string, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	_ = store.SetItem(key, string(data))
}

// statusLabel is the localized label of a transfer status.
func statusLabel(cx *core.Scope, t *transfer) string {
	l := locales.For(cx)
	switch t.Status {
	case statusWaiting:
		if t.NeedPIN {
			return l.PinTitle()
		}
		return l.WaitingShort()
	case statusActive:
		if t.Direction == "receive" {
			return l.Receiving()
		}
		return l.Sending()
	case statusArchiving:
		return l.Compressing()
	case statusCompleted:
		return l.Completed()
	case statusFailed:
		return l.Failed()
	case statusCancelled:
		return l.Cancelled()
	case statusDeclined:
		return l.Declined()
	}
	return t.Status
}

// reasonText says why a transfer failed, in the app's language.
func reasonText(cx *core.Scope, t *transfer) string {
	l := locales.For(cx)
	switch t.Reason {
	case "pin":
		return l.PinRequiredDetail()
	case "busy":
		return l.BusyDetail()
	case "rejected":
		return l.RejectedDetail()
	case "nofiles":
		return l.NoFilesNeededDetail()
	case "cancelled":
		return l.CancelledDetail()
	}
	return t.Err
}

// formatBytes is a byte count in B, KB, MB and GB.
func formatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit && exp < 3; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGT"[exp])
}

// formatSpeed is bytes per second in the same units, per second.
func formatSpeed(bytesPerSecond float64) string {
	if bytesPerSecond < 1 {
		return ""
	}
	return formatBytes(int64(bytesPerSecond)) + "/s"
}

// percent is how much of a transfer is done, for a progress bar.
func percent(done, total int64) float32 {
	if total <= 0 {
		return 0
	}
	v := float32(done) / float32(total) * 100
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

// shortFingerprint shows a fingerprint in groups.
func shortFingerprint(fingerprint string) string {
	fp := strings.ToUpper(fingerprint)
	if len(fp) > 16 {
		fp = fp[:16]
	}
	var parts []string
	for len(fp) > 4 {
		parts = append(parts, fp[:4])
		fp = fp[4:]
	}
	if fp != "" {
		parts = append(parts, fp)
	}
	return strings.Join(parts, " ")
}

// fileIcon picks a glyph by a file's type.
func fileIcon(name, mime string) func() *core.SVG {
	switch {
	case strings.HasPrefix(mime, "image/"):
		return lucide.Image
	case strings.HasPrefix(mime, "video/"):
		return lucide.Film
	case strings.HasPrefix(mime, "audio/"):
		return lucide.Music
	case strings.HasPrefix(mime, "text/"), mime == "application/pdf", mime == "application/json":
		return lucide.FileText
	case strings.Contains(mime, "zip"), strings.Contains(mime, "tar"), strings.Contains(mime, "compressed"), strings.Contains(mime, "archive"):
		return lucide.FileArchive
	case mime == "application/vnd.android.package-archive":
		return lucide.Package
	}
	return lucide.File
}

// deviceIcon picks a glyph by a device's type.
func deviceIcon(d localsend.Device) func() *core.SVG {
	switch d.DeviceType {
	case localsend.DeviceMobile:
		return lucide.Smartphone
	case localsend.DeviceWeb:
		return lucide.Globe
	case localsend.DeviceServer, localsend.DeviceHeadless:
		return lucide.Server
	case localsend.DeviceDesktop:
		if strings.Contains(strings.ToLower(d.DeviceModel), "mac") {
			return lucide.Laptop
		}
		return lucide.Monitor
	}
	return lucide.Monitor
}

// deviceTypeLabel names a device type.
func deviceTypeLabel(cx *core.Scope, typ localsend.DeviceType) string {
	l := locales.For(cx)
	switch typ {
	case localsend.DeviceMobile:
		return l.DeviceTypeMobile()
	case localsend.DeviceWeb:
		return l.DeviceTypeWeb()
	case localsend.DeviceServer, localsend.DeviceHeadless:
		return l.DeviceTypeServer()
	}
	return l.DeviceTypeDesktop()
}

// pageTitle names the page id.
func (a *app) pageTitle(cx *core.Scope) string {
	l := locales.For(cx)
	switch a.page.Get() {
	case "transfers":
		return l.Transfers()
	case "settings":
		return l.Settings()
	}
	return l.Nearby()
}

// pageIndex is the page as a tab index.
func (a *app) pageIndex() int {
	switch a.page.Get() {
	case "transfers":
		return 1
	case "settings":
		return 2
	}
	return 0
}

// setPageIndex shows the tab at index.
func (a *app) setPageIndex(index int) {
	switch index {
	case 1:
		a.page.Set("transfers")
	case 2:
		a.page.Set("settings")
	default:
		a.page.Set("nearby")
	}
}

// versionString is the app's version, as nsgo.yaml names it.
func versionString() string { return core.App().Version }

// parsePort keeps a typed port valid.
func parsePort(s string) int {
	port, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || port < 1 || port > 65535 {
		return 0
	}
	return port
}
