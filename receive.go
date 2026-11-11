package main

import (
	"strings"
	"time"

	"github.com/oneo-me/nsgo/base"
	"github.com/oneo-me/nsgo/core"
	"github.com/oneo-me/nsgo/icons/lucide"
	"github.com/oneo-me/nsgo/ui"

	"github.com/oneo-me/nsgo-localsend/internal/localsend"
	"github.com/oneo-me/nsgo-localsend/locales"
)

// prepare handles a sender's upload request: quick saves a favorite, and
// asks about everyone else.
func (a *app) prepare(req *localsend.IncomingRequest) {
	if a.incoming.Get() != nil {
		req.Decline()
		return
	}
	if a.settings.GetUntracked().QuickSave && a.favorite(req.Sender.Fingerprint) {
		a.acceptIncomingRequest(req, nil)
		return
	}
	selected := make(map[string]bool, len(req.Files))
	for _, f := range req.Files {
		selected[f.ID] = true
	}
	a.incoming.Set(&incomingRequest{Req: req, Selected: selected})
}

// incomingDialog asks whether to receive a sender's files.
func (a *app) incomingDialog(cx *core.Scope) core.Node {
	return core.Show{When: func() bool { return a.incoming.Get() != nil }, Then: func(cx *core.Scope) core.Node {
		inc := a.incoming.Get()
		if inc == nil {
			return nil
		}
		req := inc.Req
		l := locales.For(cx)
		return ui.Dialog{
			Open: func() bool { return a.incoming.Get() != nil },
			OnOpenChange: func(open bool, _ base.Reason) {
				if !open {
					a.declineIncoming()
				}
			},
			NoCloseOnEscape:        true,
			NoCloseOnBackdropPress: true,
			Title:                  l.IncomingTitle(),
			Description:            l.IncomingFrom(req.Sender.Alias) + " · " + l.IncomingDetail(len(req.Files), formatBytes(req.Total)),
			Children:               []core.Node{core.Dynamic(func(cx *core.Scope) core.Node { return a.incomingBody(cx) })},
			Footer: []core.Node{
				ui.Button{Text: l.Decline(), Variant: ui.ButtonSecondary, OnPress: a.declineIncoming, TestID: "incoming-decline"},
				ui.Button{Text: l.AcceptSelected(), Icon: lucide.Download, Disabled: func() bool { return a.selectedIncomingCount() == 0 },
					OnPress: a.acceptIncoming, TestID: "incoming-accept"},
			},
			TestID: "incoming-dialog",
		}
	}}
}

// incomingBody is the sender and the files with their check boxes.
func (a *app) incomingBody(cx *core.Scope) core.Node {
	inc := a.incoming.Get()
	if inc == nil {
		return nil
	}
	colors := ui.ColorsOf(cx)
	l := locales.For(cx)
	req := inc.Req
	sender := core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 10, Padding: core.Symmetric(8, 10),
		CornerRadius: 10}, StyleFunc: ui.Filled(colors.Inset), Children: []core.Node{
		deviceTile(cx, deviceIcon(req.Sender), true, 36),
		core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0), Gap: 1}, Children: []core.Node{
			core.Text{Content: req.Sender.Alias, MaxLines: 1, Style: ui.TypeLabel, StyleFunc: ui.Colored(colors.Text)},
			core.Text{Content: deviceTypeLabel(cx, req.Sender.DeviceType) + " · " + req.Sender.IP, MaxLines: 1, Style: core.Style{FontSize: 11},
				StyleFunc: ui.Colored(colors.Muted)},
		}},
		ui.IconButton{Icon: lucide.Star, Tint: func() core.Color {
			if a.favoriteTracked(req.Sender.Fingerprint) {
				return colors.Accent()
			}
			return colors.Faint()
		}, Size: 26, IconSize: 15, LabelFunc: func() string {
			if a.favoriteTracked(req.Sender.Fingerprint) {
				return l.Unfavorite()
			}
			return l.Favorite()
		}, OnPress: func() { a.toggleFavorite(req.Sender) }, TestID: "incoming-favorite"},
	}}
	rows := make([]core.Node, 0, len(req.Files))
	for _, f := range req.Files {
		file := f
		on := inc.Selected[file.ID]
		detail := formatBytes(file.Size)
		if file.Preview != "" && strings.HasPrefix(file.FileType, "text/") {
			detail = file.Preview
		}
		rows = append(rows, core.View{TestID: "incoming-file-" + file.ID, Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 8,
			Padding: core.Symmetric(4, 2)}, Children: []core.Node{
			ui.Checkbox{Checked: func() bool { return on }, OnChange: func(checked bool) { a.toggleIncoming(file.ID, checked) },
				Label: file.FileName, TestID: "incoming-check"},
			deviceTile(cx, fileIcon(file.FileName, file.FileType), false, 28),
			core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0), Gap: 0}, Children: []core.Node{
				core.Text{Content: file.FileName, MaxLines: 1, Style: ui.TypeBody, StyleFunc: ui.Colored(colors.Text)},
				core.Text{Content: detail, MaxLines: 1, Style: core.Style{FontSize: 11}, StyleFunc: ui.Colored(colors.Muted)},
			}},
		}})
	}
	return core.View{Style: core.Style{Gap: 10, MinWidth: core.Pt(340), MaxWidth: core.Pt(560)}, Children: []core.Node{
		sender,
		core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 8}, Children: []core.Node{
			core.Text{Content: l.IncomingQuestion(), Style: ui.TypeCaption, StyleFunc: ui.Colored(colors.Muted)},
			core.View{Style: core.Style{FlexGrow: 1}},
			core.Text{ContentFunc: func() string { return l.FilesSelected(a.selectedIncomingCount()) }, Style: ui.TypeCaption,
				StyleFunc: ui.Colored(colors.Muted)},
		}},
		core.Scroll{Style: core.Style{MaxHeight: core.Pt(280)}, Children: []core.Node{
			core.View{Style: core.Style{Gap: 2, Padding: core.Edges{Right: 4}}, Children: rows},
		}},
	}}
}

// toggleIncoming checks or unchecks a file.
func (a *app) toggleIncoming(id string, checked bool) {
	inc := a.incoming.Get()
	if inc == nil {
		return
	}
	selected := make(map[string]bool, len(inc.Selected))
	for key, value := range inc.Selected {
		selected[key] = value
	}
	if checked {
		selected[id] = true
	} else {
		delete(selected, id)
	}
	a.incoming.Set(&incomingRequest{Req: inc.Req, Selected: selected})
}

// selectedIncomingCount is how many files are checked.
func (a *app) selectedIncomingCount() int {
	inc := a.incoming.Get()
	if inc == nil {
		return 0
	}
	n := 0
	for _, on := range inc.Selected {
		if on {
			n++
		}
	}
	return n
}

// selectedIncomingIDs are the checked files.
func (a *app) selectedIncomingIDs() []string {
	inc := a.incoming.Get()
	if inc == nil {
		return nil
	}
	var ids []string
	for _, f := range inc.Req.Files {
		if inc.Selected[f.ID] {
			ids = append(ids, f.ID)
		}
	}
	return ids
}

// acceptIncoming answers the open request with its checked files.
func (a *app) acceptIncoming() {
	inc := a.incoming.Get()
	if inc == nil {
		return
	}
	a.incoming.Set(nil)
	a.acceptIncomingRequest(inc.Req, a.selectedIncomingIDs())
}

// declineIncoming rejects the open request.
func (a *app) declineIncoming() {
	inc := a.incoming.Get()
	a.incoming.Set(nil)
	if inc != nil {
		inc.Req.Decline()
	}
}

// acceptIncomingRequest saves the request's chosen files (all when ids
// is empty) and records the receive transfer.
func (a *app) acceptIncomingRequest(req *localsend.IncomingRequest, ids []string) {
	dir := a.receiveDir()
	chosen := make(map[string]bool, len(req.Files))
	for _, f := range req.Files {
		chosen[f.ID] = true
	}
	if len(ids) > 0 {
		chosen = make(map[string]bool, len(ids))
		for _, id := range ids {
			chosen[id] = true
		}
	}
	var items []transferFile
	var total int64
	for _, f := range req.Files {
		if !chosen[f.ID] {
			continue
		}
		items = append(items, transferFile{ID: f.ID, Name: f.FileName, Size: f.Size, Preview: f.Preview, Status: "pending"})
		total += f.Size
	}
	if len(items) == 0 {
		req.Decline()
		return
	}
	if !req.Accept(dir, ids) {
		a.toastError(locales.IncomingTitle(a.cx), locales.CancelledDetail(a.cx))
		return
	}
	t := &transfer{ID: req.SessionID, Direction: "receive", Peer: req.Sender, Status: statusActive,
		Files: items, Total: total, Started: time.Now(), SavedDir: dir}
	a.addTransfer(t)
	a.toast(locales.For(a.cx).IncomingDetail(len(items), formatBytes(total)))
}

// receiveProgress records incoming bytes.
func (a *app) receiveProgress(p localsend.Progress) {
	a.updateTransfer(p.SessionID, func(t *transfer) {
		if finalStatus(t.Status) {
			return
		}
		t.Status = statusActive
		for i := range t.Files {
			if t.Files[i].ID == p.FileID {
				t.Files[i].Done = p.Received
				if p.Done {
					t.Files[i].Status = "done"
				}
				break
			}
		}
	})
}

// receiveDone ends a receive transfer with the server's result.
func (a *app) receiveDone(result localsend.SessionResult) {
	status := result.Status
	if status == localsend.StatusCompleted {
		status = statusCompleted
	} else if status == localsend.StatusCancelled {
		status = statusCancelled
	} else {
		status = statusFailed
	}
	// iOS opens a folder only as an archive, so a folder send is packed
	// when it ends and the transfer opens the zip.
	archiving := status == statusCompleted && a.cx.Env().OS() == core.OSIOS && hasReceivedFolders(result.Files)
	t := a.updateTransfer(result.SessionID, func(t *transfer) {
		t.Status, t.Ended, t.CancelRequested, t.Speed = status, time.Now(), false, 0
		if archiving {
			t.Status = statusArchiving
		}
		byID := make(map[string]localsend.ReceivedFile, len(result.Files))
		for _, f := range result.Files {
			byID[f.ID] = f
		}
		for i := range t.Files {
			file := &t.Files[i]
			received, ok := byID[file.ID]
			if !ok {
				continue
			}
			file.Path = received.Path
			file.Error = received.Error
			switch received.Status {
			case "done":
				file.Status, file.Done = "done", file.Size
			case "failed":
				file.Status = "failed"
				if file.Error != "" && t.Err == "" {
					t.Err = file.Error
				}
			default:
				file.Status = "failed"
			}
		}
		if status == statusCompleted {
			t.Done = t.Total
		}
	})
	if t == nil {
		return
	}
	if archiving {
		id, dir, files := t.ID, t.SavedDir, append([]transferFile(nil), t.Files...)
		go func() {
			path, err := archiveReceivedFolders(dir, files)
			a.cx.Post(func() { a.receiveArchived(id, path, err) })
		}()
		return
	}
	a.rememberTransfer(t)
	l := locales.For(a.cx)
	switch status {
	case statusCompleted:
		detail := t.SavedDir
		if detail == "" {
			detail = formatBytes(t.Total)
		}
		a.toastSuccess(l.TransferReceived(t.Peer.Alias), l.SavedTo(detail))
	case statusCancelled:
		a.toast(l.Cancelled())
	default:
		a.toastError(l.TransferReceived(t.Peer.Alias), reasonText(a.cx, t))
	}
}
