package main

import (
	"sort"
	"strings"
	"time"

	"github.com/oneo-me/nsgo/core"
	"github.com/oneo-me/nsgo/icons/lucide"
	"github.com/oneo-me/nsgo/ui"

	"github.com/oneo-me/nsgo-localsend/locales"
)

// transfersPage lists active transfers and the recent history.
func (a *app) transfersPage(cx *core.Scope) core.Node {
	return a.pageScroll(cx, "transfers", "transfers-page",
		core.Dynamic(func(cx *core.Scope) core.Node {
			active := a.activeTransfers()
			done := a.finishedTransfers()
			if len(active) == 0 && len(done) == 0 {
				return core.View{Style: core.Style{Padding: pagePadding, Gap: 18}, Children: []core.Node{a.emptyTransfers(cx), bottomGap(cx)}}
			}
			var body []core.Node
			if len(active) > 0 {
				cards := make([]core.Node, 0, len(active))
				for _, t := range active {
					cards = append(cards, a.transferCard(cx, t))
				}
				body = append(body, a.sectionTitle(cx, locales.For(cx).ActiveTransfers(), itoa(len(active))), core.View{Style: core.Style{Gap: 10}, Children: cards})
			}
			if len(done) > 0 {
				cards := make([]core.Node, 0, len(done))
				for _, t := range done {
					cards = append(cards, a.transferCard(cx, t))
				}
				body = append(body, a.sectionTitle(cx, locales.For(cx).History(), itoa(len(done))), core.View{Style: core.Style{Gap: 10}, Children: cards})
			}
			body = append(body, bottomGap(cx))
			return core.View{Style: core.Style{Padding: pagePadding, Gap: 18}, Children: body}
		}))
}

// sectionTitle heads a list with a count.
func (a *app) sectionTitle(cx *core.Scope, title string, count string) core.Node {
	colors := ui.ColorsOf(cx)
	return core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 8, Padding: core.Edges{Top: 2}}, Children: []core.Node{
		core.Text{Content: title, Style: core.Style{FontSize: 15, LineHeight: 20, FontWeight: ui.Semibold}, StyleFunc: ui.Colored(colors.Text)},
		ui.Badge{Text: count},
		core.View{Style: core.Style{FlexGrow: 1}},
	}}
}

// activeTransfers are sessions still running.
func (a *app) activeTransfers() []*transfer {
	var out []*transfer
	for _, t := range a.transfers.Get() {
		if t.Status == statusWaiting || t.Status == statusActive {
			out = append(out, t)
		}
	}
	return out
}

// finishedTransfers are finished sessions of this launch and the saved
// history, by end time, each once.
func (a *app) finishedTransfers() []*transfer {
	var out []*transfer
	seen := map[string]bool{}
	for _, t := range a.transfers.Get() {
		if t.Status == statusWaiting || t.Status == statusActive {
			continue
		}
		copy := *t
		out = append(out, &copy)
		seen[t.ID] = true
	}
	for _, h := range a.history.Get() {
		if seen[h.ID] {
			continue
		}
		copy := h
		out = append(out, &copy)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Ended.After(out[j].Ended) })
	return out
}

// transferCard is one session's row.
func (a *app) transferCard(cx *core.Scope, t *transfer) core.Node {
	colors := ui.ColorsOf(cx)
	l := locales.For(cx)
	title := l.TransferSent(t.Peer.Alias)
	if t.Direction == "receive" {
		title = l.TransferReceived(t.Peer.Alias)
	}
	icon := lucide.Upload
	if t.Direction == "receive" {
		icon = lucide.Download
	}
	running := t.Status == statusWaiting || t.Status == statusActive
	size := formatBytes(t.Total)
	detail := l.FilesSelected(len(t.Files)) + " · " + size
	if len(t.Files) == 1 && t.Files[0].Preview != "" {
		detail = oneLine(t.Files[0].Preview, 80) + " · " + size
	}
	if running && t.Speed > 0 {
		detail += " · " + formatSpeed(t.Speed)
	}
	status := statusLabel(cx, t)
	reason := reasonText(cx, t)
	var actions []core.Node
	if running && !t.CancelRequested {
		actions = append(actions, ui.IconButton{Icon: lucide.X, Label: l.Cancel(), Size: 26, IconSize: 15, Tint: colors.Muted,
			OnPress: func() { a.cancelTransfer(t) }, TestID: "cancel-transfer"})
	}
	if !running && t.Direction == "receive" && t.Status == statusCompleted {
		if received := receivedFiles(t); len(received) == 1 {
			actions = append(actions, ui.Button{Text: l.Open(), Variant: ui.ButtonSecondary, Size: ui.SizeSmall,
				Icon: lucide.ExternalLink, OnPress: func() { a.openReceived(t) }, TestID: "open-received"})
		} else if len(received) > 1 && t.SavedDir != "" && hasFileManager(cx) {
			actions = append(actions, ui.Button{Text: l.OpenFolder(), Variant: ui.ButtonSecondary, Size: ui.SizeSmall,
				Icon: lucide.FolderOpen, OnPress: func() { a.openReceivedFolder(t) }, TestID: "open-received-folder"})
		} else if len(received) > 1 && canShare(cx) {
			actions = append(actions, ui.Button{Text: l.Share(), Variant: ui.ButtonSecondary, Size: ui.SizeSmall,
				Icon: lucide.Share, OnPress: func() { a.shareReceived(t) }, TestID: "share-received"})
		}
	}
	var body []core.Node
	body = append(body, core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 10}, Children: []core.Node{
		deviceTile(cx, icon, t.Direction == "receive", 38),
		core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0), Gap: 1}, Children: []core.Node{
			core.Text{Content: title, MaxLines: 1, Style: ui.TypeLabel, StyleFunc: ui.Colored(colors.Text)},
			core.Text{Content: detail, MaxLines: 1, Style: core.Style{FontSize: 11}, StyleFunc: ui.Colored(colors.Muted)},
		}},
		ui.Badge{Text: status, Variant: statusVariant(t.Status)},
	}})
	if running {
		body = append(body, core.View{Style: core.Style{Gap: 6}, Children: []core.Node{
			ui.Progress{Value: func() float32 { return percent(t.Done, t.Total) }, Label: title, TestID: "transfer-progress"},
			core.View{Style: core.Style{Direction: core.Row, Gap: 8}, Children: []core.Node{
				core.Text{ContentFunc: func() string { return formatBytes(t.Done) + " / " + formatBytes(t.Total) },
					Style: core.Style{FontSize: 11, FontVariant: core.FontTabularNums}, StyleFunc: ui.Colored(colors.Muted)},
				core.View{Style: core.Style{FlexGrow: 1}},
				core.Text{ContentFunc: func() string { return formatSpeed(t.Speed) }, Style: core.Style{FontSize: 11},
					StyleFunc: ui.Colored(colors.Muted)},
			}},
		}})
	} else if t.Status == statusCompleted && t.Direction == "receive" && t.SavedDir != "" {
		body = append(body, core.Text{Content: l.SavedTo(t.SavedDir), MaxLines: 2, Style: core.Style{FontSize: 11},
			StyleFunc: ui.Colored(colors.Faint)})
	}
	if reason != "" && t.Status != statusCompleted {
		body = append(body, core.Text{Content: reason, MaxLines: 2, Style: core.Style{FontSize: 11}, StyleFunc: ui.Colored(colors.Danger)})
	}
	body = append(body, core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 6}, Children: []core.Node{
		core.Text{Content: timestamp(t.Ended, t.Started), Style: core.Style{FontSize: 11, FontVariant: core.FontTabularNums},
			StyleFunc: ui.Colored(colors.Faint)},
		core.View{Style: core.Style{FlexGrow: 1}},
		core.Group(actions),
	}})
	return core.View{TestID: "transfer-" + t.ID, Style: core.Style{Padding: core.All(14), Gap: 10, BorderWidth: 1, CornerRadius: 12},
		StyleFunc: func() core.Style { return core.Style{Background: colors.Surface(), BorderColor: colors.Border()} }, Children: body}
}

// finalStatus reports a status no later event changes.
func finalStatus(status string) bool {
	switch status {
	case statusCompleted, statusFailed, statusCancelled, statusDeclined:
		return true
	}
	return false
}

// statusVariant colors a transfer's status.
func statusVariant(status string) ui.BadgeVariant {
	switch status {
	case statusCompleted:
		return ui.BadgeAccent
	case statusFailed, statusDeclined:
		return ui.BadgeDanger
	case statusCancelled:
		return ui.BadgeNeutral
	case statusWaiting:
		return ui.BadgeWarn
	case statusActive:
		return ui.BadgeInfo
	}
	return ui.BadgeNeutral
}

// oneLine is s on one line, at most max runes.
func oneLine(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	runes := []rune(s)
	if len(runes) > max {
		return string(runes[:max-1]) + "…"
	}
	return s
}

// timestamp is when a transfer ended, or when it began.
func timestamp(ended, started time.Time) string {
	at := ended
	if at.IsZero() {
		at = started
	}
	if at.IsZero() {
		return ""
	}
	now := time.Now()
	if at.Year() == now.Year() && at.YearDay() == now.YearDay() {
		return at.Format("15:04")
	}
	return at.Format("01-02 15:04")
}

// emptyTransfers is what shows before any transfer.
func (a *app) emptyTransfers(cx *core.Scope) core.Node {
	colors := ui.ColorsOf(cx)
	l := locales.For(cx)
	return core.View{TestID: "transfers-empty", Style: core.Style{Padding: core.Symmetric(34, 20), Gap: 12, AlignItems: core.AlignCenter,
		CornerRadius: 14, BorderWidth: 1}, StyleFunc: func() core.Style {
		return core.Style{Background: colors.Inset(), BorderColor: colors.Border()}
	}, Children: []core.Node{
		deviceTile(cx, lucide.ArrowUpDown, false, 52),
		core.Text{Content: l.NoTransfers(), Style: ui.TypeTitle.Merge(core.Style{TextAlign: core.TextAlignCenter}), StyleFunc: ui.Colored(colors.Text)},
		core.Text{Content: l.NoTransfersDetail(), MaxLines: 3, Style: ui.TypeBody.Merge(core.Style{TextAlign: core.TextAlignCenter, MaxWidth: core.Pt(360)}),
			StyleFunc: ui.Colored(colors.Muted)},
		ui.Button{Text: l.Nearby(), Icon: lucide.RadioTower, Variant: ui.ButtonSecondary, OnPress: func() { a.page.Set("nearby") }, TestID: "empty-nearby"},
	}}
}

// cancelTransfer stops a running session.
func (a *app) cancelTransfer(t *transfer) {
	if t.CancelRequested {
		return
	}
	a.updateTransfer(t.ID, func(x *transfer) { x.CancelRequested = true })
	if t.Direction == "send" {
		if cancel := a.cancels[t.ID]; cancel != nil {
			cancel()
		} else {
			a.finalizeSend(t.ID, statusCancelled, "cancelled", "")
		}
		return
	}
	if node := a.currentNode(); node != nil {
		node.CancelReceive(t.ID)
	}
}

// receivedFiles are a transfer's files that reached the receive folder.
func receivedFiles(t *transfer) []core.SharedFile {
	var out []core.SharedFile
	for _, f := range t.Files {
		if f.Path != "" {
			out = append(out, core.SharedFile{Path: f.Path, Name: f.Name})
		}
	}
	return out
}

// hasFileManager reports whether the platform opens a folder in a file
// manager.
func hasFileManager(cx *core.Scope) bool {
	switch cx.Env().OS() {
	case core.OSMacOS, core.OSWindows, core.OSLinux:
		return true
	}
	return false
}

// canShare reports whether the platform has a share sheet. A received
// folder has no other way out of a phone app's storage, since files open
// one at a time.
func canShare(cx *core.Scope) bool {
	switch cx.Env().OS() {
	case core.OSIOS, core.OSAndroid:
		return true
	}
	return false
}

// openReceived opens a received file.
func (a *app) openReceived(t *transfer) {
	received := receivedFiles(t)
	if len(received) != 1 {
		return
	}
	file := received[0]
	a.cx.OpenFile(file, func(opened bool, err error) {
		if err != nil {
			a.toastError(locales.Open(a.cx), err.Error())
		} else if !opened {
			a.toastError(locales.Open(a.cx), file.Path)
		}
	})
}

// openReceivedFolder shows a received folder in the file manager.
func (a *app) openReceivedFolder(t *transfer) {
	a.cx.OpenFile(core.SharedFile{Path: t.SavedDir}, func(opened bool, err error) {
		if err != nil {
			a.toastError(locales.OpenFolder(a.cx), err.Error())
		} else if !opened {
			a.toastError(locales.OpenFolder(a.cx), t.SavedDir)
		}
	})
}

// shareReceived hands a received folder's files to the share sheet, where
// they can be saved to the system's files or opened in another app.
func (a *app) shareReceived(t *transfer) {
	a.cx.Share(core.ShareOptions{Files: receivedFiles(t)}, func(shown bool, err error) {
		if err != nil {
			a.toastError(locales.Share(a.cx), err.Error())
		}
	})
}

// clearHistory forgets finished transfers.
func (a *app) clearHistory() {
	a.history.Set(nil)
	a.transfers.Update(func(list []*transfer) []*transfer {
		out := list[:0]
		for _, t := range list {
			if t.Status == statusWaiting || t.Status == statusActive {
				out = append(out, t)
			}
		}
		return out
	})
	a.toast(locales.HistoryCleared(a.cx))
}

// addTransfer puts a new session at the top.
func (a *app) addTransfer(t *transfer) {
	if t.Started.IsZero() {
		t.Started = time.Now()
	}
	a.transfers.Update(func(list []*transfer) []*transfer {
		return append([]*transfer{t}, list...)
	})
}

// updateTransfer changes a session and reports its progress.
func (a *app) updateTransfer(id string, fn func(*transfer)) *transfer {
	var found *transfer
	a.transfers.Update(func(list []*transfer) []*transfer {
		out := append([]*transfer(nil), list...)
		for _, t := range out {
			if t.ID != id {
				continue
			}
			fn(t)
			t.Done = transferDone(t)
			sampleSpeed(t)
			found = t
			break
		}
		return out
	})
	return found
}

// transferDone adds the files' progress.
func transferDone(t *transfer) int64 {
	var done int64
	for _, f := range t.Files {
		done += f.Done
	}
	if t.Status == statusCompleted {
		done = t.Total
	}
	return done
}

// sampleSpeed keeps a running transfer's bytes per second, sampled no
// more than a few times a second.
func sampleSpeed(t *transfer) {
	now := time.Now()
	if t.Status != statusWaiting && t.Status != statusActive {
		t.Speed = 0
		t.lastAt, t.lastDone = time.Time{}, 0
		return
	}
	if t.lastAt.IsZero() {
		t.lastAt, t.lastDone = now, t.Done
		return
	}
	if elapsed := now.Sub(t.lastAt); elapsed >= 400*time.Millisecond {
		if delta := t.Done - t.lastDone; delta > 0 {
			t.Speed = float64(delta) / elapsed.Seconds()
		}
		t.lastAt, t.lastDone = now, t.Done
	}
}

// rememberTransfer adds a finished transfer to the saved history.
func (a *app) rememberTransfer(t *transfer) {
	if t.Ended.IsZero() {
		t.Ended = time.Now()
	}
	copy := *t
	copy.Speed, copy.CancelRequested, copy.NeedPIN = 0, false, false
	copy.Files = append([]transferFile(nil), t.Files...)
	a.history.Update(func(list []transfer) []transfer {
		out := append([]transfer{copy}, list...)
		for i := 1; i < len(out); i++ {
			if out[i].ID == copy.ID {
				out = append(out[:i], out[i+1:]...)
				break
			}
		}
		if len(out) > 50 {
			out = out[:50]
		}
		return out
	})
}
