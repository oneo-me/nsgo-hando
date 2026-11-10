package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/oneo-me/nsgo/base"
	"github.com/oneo-me/nsgo/core"
	"github.com/oneo-me/nsgo/icons/lucide"
	"github.com/oneo-me/nsgo/ui"

	"github.com/oneo-me/nsgo-localsend/internal/localsend"
	"github.com/oneo-me/nsgo-localsend/locales"
)

// openSend shows the send dialog for a device.
func (a *app) openSend(d localsend.Device) {
	a.sendTarget.Set(&d)
}

// closeSend hides the send dialog, keeping what was chosen.
func (a *app) closeSend() { a.sendTarget.Set(nil) }

// sendDialog is the send sheet.
func (a *app) sendDialog(cx *core.Scope) core.Node {
	return core.Show{When: func() bool { return a.sendTarget.Get() != nil }, Then: func(cx *core.Scope) core.Node {
		target := *a.sendTarget.Get()
		return a.sendDialogBody(cx, target)
	}}
}

// sendDialogBody is the dialog once a device is chosen.
func (a *app) sendDialogBody(cx *core.Scope, target localsend.Device) core.Node {
	l := locales.For(cx)
	detail := target.DeviceModel
	if detail == "" {
		detail = deviceTypeLabel(cx, target.DeviceType)
	}
	return ui.Dialog{
		Open: func() bool { return a.sendTarget.Get() != nil },
		OnOpenChange: func(open bool, _ base.Reason) {
			if !open {
				a.closeSend()
			}
		},
		Title:       l.SendTo(target.Alias),
		Description: detail + " · " + target.IP,
		Children:    []core.Node{core.Dynamic(func(cx *core.Scope) core.Node { return a.sendForm(cx) })},
		Footer: []core.Node{
			ui.Button{Text: l.Cancel(), Variant: ui.ButtonSecondary, OnPress: a.closeSend, TestID: "send-cancel"},
			ui.Button{Text: l.Send(), Icon: lucide.Send, Disabled: func() bool {
				return a.scanning.Get() || (len(a.sendFiles.Get()) == 0 && strings.TrimSpace(a.sendMsg.Get()) == "")
			}, OnPress: a.send, TestID: "send-submit"},
		},
		TestID: "send-dialog",
	}
}

// sendForm is the dialog's body: chosen files and a message.
func (a *app) sendForm(cx *core.Scope) core.Node {
	colors := ui.ColorsOf(cx)
	l := locales.For(cx)
	files := a.sendFiles.Get()
	scanning := a.scanning.Get()
	var body []core.Node
	body = append(body, core.View{Style: core.Style{Direction: core.Row, Wrap: core.Wrap, Gap: 8}, Children: []core.Node{
		ui.Button{Text: l.ChooseFiles(), Icon: lucide.Plus, Variant: ui.ButtonSecondary, OnPress: a.pickFiles, TestID: "choose-files"},
		ui.Button{Text: l.ChooseFolder(), Icon: lucide.FolderOpen, Variant: ui.ButtonSecondary, OnPress: a.pickFolder, TestID: "choose-folder"},
		core.View{Style: core.Style{FlexGrow: 1}},
		ui.Button{Text: l.Clear(), Variant: ui.ButtonGhost, Size: ui.SizeSmall, Disabled: func() bool { return len(files) == 0 },
			OnPress: a.clearSendFiles, TestID: "clear-files"},
	}})
	if scanning {
		body = append(body, core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 8, Padding: core.Symmetric(8, 10),
			CornerRadius: 8}, StyleFunc: ui.Filled(colors.Inset), Children: []core.Node{
			ui.Spinner{Size: 16},
			core.Text{Content: l.Refreshing(), Style: ui.TypeCaption, StyleFunc: ui.Colored(colors.Muted)},
		}})
	}
	body = append(body, core.Show{When: func() bool {
		return len(a.sendFiles.Get()) == 0 && strings.TrimSpace(a.sendMsg.Get()) == "" && !a.scanning.Get()
	}, Then: func(cx *core.Scope) core.Node {
		return core.View{Style: core.Style{Padding: core.Symmetric(20, 16), Gap: 4, AlignItems: core.AlignCenter, CornerRadius: 10},
			StyleFunc: ui.Filled(colors.Inset), Children: []core.Node{
				core.Text{Content: l.NoFilesYet(), Style: ui.TypeLabel, StyleFunc: ui.Colored(colors.Text)},
				core.Text{Content: l.NoFilesYetDetail(), MaxLines: 2, Style: core.Style{FontSize: 12, TextAlign: core.TextAlignCenter},
					StyleFunc: ui.Colored(colors.Muted)},
			}}
	}})
	if len(files) > 0 {
		rows := make([]core.Node, 0, len(files))
		for _, f := range files {
			file := f
			rows = append(rows, a.sendFileRow(cx, file))
		}
		body = append(body, core.Scroll{Style: core.Style{MaxHeight: core.Pt(240)}, Children: []core.Node{
			core.View{Style: core.Style{Gap: 2, Padding: core.Edges{Right: 4}}, Children: rows},
		}})
	}
	body = append(body, core.View{Style: core.Style{Gap: 6}, Children: []core.Node{
		core.Text{Content: l.Message(), Style: ui.TypeCaption, StyleFunc: ui.Colored(colors.Muted)},
		ui.Input{Value: a.sendMsg.Get, OnChange: a.sendMsg.Set, Multiline: true, Placeholder: l.MessagePlaceholder(),
			MaxLength: 4000, Label: l.Message(), TestID: "send-message"},
	}})
	return core.View{Style: core.Style{Gap: 12, MinWidth: core.Pt(340), MaxWidth: core.Pt(560)}, Children: body}
}

// sendFileRow is one chosen file.
func (a *app) sendFileRow(cx *core.Scope, f sendFile) core.Node {
	colors := ui.ColorsOf(cx)
	name := f.Name
	if f.Text && f.Preview != "" {
		name = f.Preview
	}
	return core.View{TestID: "send-file-" + f.ID, Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 8, Padding: core.Symmetric(5, 6),
		CornerRadius: 8}, StyleFunc: ui.Filled(colors.Inset), Children: []core.Node{
		deviceTile(cx, fileIcon(f.Name, f.MIME), false, 30),
		core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0), Gap: 0}, Children: []core.Node{
			core.Text{Content: name, MaxLines: 1, Style: ui.TypeBody, StyleFunc: ui.Colored(colors.Text)},
			core.Text{Content: func() string {
				if f.Text {
					return locales.TextMessage(cx)
				}
				return formatBytes(f.Size)
			}(), MaxLines: 1, Style: core.Style{FontSize: 11}, StyleFunc: ui.Colored(colors.Muted)},
		}},
		ui.IconButton{Icon: lucide.X, Label: locales.Remove(cx), Size: 24, IconSize: 13, Tint: colors.Muted,
			OnPress: func() { a.removeSendFile(f.ID) }, TestID: "remove-file"},
	}}
}

// pickFiles chooses files to send.
func (a *app) pickFiles() {
	a.cx.PickFiles(core.PickFilesOptions{Multiple: true, Limit: 100}, func(files []core.File, err error) {
		if err != nil {
			a.toastError(locales.ChooseFiles(a.cx), err.Error())
			return
		}
		out := make([]sendFile, 0, len(files))
		for _, file := range files {
			f := file
			mime := f.MIME
			if mime == "" {
				mime = localsend.MIMEForName(f.Name)
			}
			item := sendFile{ID: localsend.NewID(8), Name: f.Name, Size: f.Size, MIME: mime, Modified: f.Modified,
				open: func() (io.ReadCloser, error) { return f.Open() }}
			if f.Temporary && f.Path != "" {
				path := f.Path
				item.cleanup = func() { _ = os.Remove(path) }
			}
			out = append(out, item)
		}
		a.sendFiles.Update(func(list []sendFile) []sendFile { return append(list, out...) })
	})
}

// pickFolder chooses folders and adds the files inside them.
func (a *app) pickFolder() {
	a.cx.PickDirectories(core.PickDirectoriesOptions{Multiple: true, Limit: 20}, func(dirs []core.Directory, err error) {
		if err != nil {
			a.toastError(locales.ChooseFolder(a.cx), err.Error())
			return
		}
		if len(dirs) == 0 {
			return
		}
		a.scanning.Set(true)
		go a.walkFolders(dirs)
	})
}

// walkFolders lists a folder's files off the engine and adds them.
func (a *app) walkFolders(dirs []core.Directory) {
	const limit = 5000
	var out []sendFile
	truncated := false
	for _, dir := range dirs {
		if len(out) >= limit {
			truncated = true
			break
		}
		folder := dir
		root := folder.Name
		if root == "" {
			root = "folder"
		}
		_ = fs.WalkDir(folder, ".", func(name string, entry fs.DirEntry, err error) error {
			if err != nil || entry == nil {
				return nil
			}
			if entry.IsDir() {
				if name != "." && strings.HasPrefix(entry.Name(), ".") {
					return fs.SkipDir
				}
				return nil
			}
			if entry.Type()&fs.ModeSymlink != 0 || skipName(entry.Name()) {
				return nil
			}
			if len(out) >= limit {
				truncated = true
				return fs.SkipAll
			}
			info, err := entry.Info()
			if err != nil {
				return nil
			}
			relative := name
			if relative == "." || relative == "" {
				relative = entry.Name()
			}
			display := path.Join(root, filepath.ToSlash(relative))
			inner := filepath.ToSlash(relative)
			out = append(out, sendFile{ID: localsend.NewID(8), Name: display, Size: info.Size(), MIME: localsend.MIMEForName(display),
				Modified: info.ModTime(), open: func() (io.ReadCloser, error) {
					return folder.Open(inner)
				}})
			return nil
		})
	}
	a.cx.Post(func() {
		a.scanning.Set(false)
		a.sendFiles.Update(func(list []sendFile) []sendFile { return append(list, out...) })
		if truncated {
			a.toast(locales.FilesSelected(a.cx, limit))
		}
	})
}

// skipName reports a file the user did not mean to send.
func skipName(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	switch strings.ToLower(name) {
	case "thumbs.db", "desktop.ini":
		return true
	}
	return false
}

// removeSendFile drops a chosen file.
func (a *app) removeSendFile(id string) {
	a.sendFiles.Update(func(list []sendFile) []sendFile {
		out := list[:0]
		for _, f := range list {
			if f.ID != id {
				out = append(out, f)
				continue
			}
			if f.cleanup != nil {
				f.cleanup()
			}
		}
		return out
	})
}

// clearSendFiles drops every chosen file.
func (a *app) clearSendFiles() {
	for _, f := range a.sendFiles.Get() {
		if f.cleanup != nil {
			f.cleanup()
		}
	}
	a.sendFiles.Set(nil)
}

// send starts a transfer to the dialog's device.
func (a *app) send() {
	target := a.sendTarget.Get()
	if target == nil {
		return
	}
	files := append([]sendFile(nil), a.sendFiles.Get()...)
	message := strings.TrimSpace(a.sendMsg.Get())
	if message != "" {
		content := []byte(message)
		files = append(files, sendFile{ID: localsend.NewID(8), Name: localsend.NewID(8) + ".txt", Size: int64(len(content)),
			MIME: "text/plain", Preview: message, Text: true,
			open: func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(content)), nil }})
		a.sendMsg.Set("")
	}
	if len(files) == 0 {
		a.toast(locales.NoFilesYet(a.cx))
		return
	}
	a.closeSend()
	a.startSend(*target, files)
}

// startSend makes a transfer and begins preparing it.
func (a *app) startSend(target localsend.Device, files []sendFile) {
	id := localsend.NewID(16)
	items := make([]transferFile, 0, len(files))
	var total int64
	for _, f := range files {
		items = append(items, transferFile{ID: f.ID, Name: f.Name, Size: f.Size, Preview: f.Preview, Status: "pending"})
		total += f.Size
	}
	t := &transfer{ID: id, Direction: "send", Peer: target, Status: statusWaiting, Files: items, Total: total, Started: time.Now()}
	a.addTransfer(t)
	job := sendJob{TransferID: id, Target: target, Files: files}
	a.jobs[id] = job
	ctx, cancel := context.WithCancel(context.Background())
	a.cancels[id] = cancel
	go a.runSend(ctx, job)
}

// runSend prepares and uploads a send job.
func (a *app) runSend(ctx context.Context, job sendJob) {
	node := a.currentNode()
	if node == nil {
		a.cx.Post(func() { a.finalizeSend(job.TransferID, statusFailed, "", locales.ServerStopped(a.cx)) })
		return
	}
	client := node.Client()
	self := node.Self()
	offered := make([]localsend.File, 0, len(job.Files))
	for _, f := range job.Files {
		file := localsend.File{ID: f.ID, FileName: f.Name, Size: f.Size, FileType: f.MIME, Preview: f.Preview}
		if !f.Modified.IsZero() {
			file.Metadata = &localsend.Metadata{Modified: f.Modified.UTC().Format(time.RFC3339Nano)}
		}
		offered = append(offered, file)
	}
	answer, err := client.PrepareUpload(ctx, job.Target, self, offered, job.PIN)
	if err != nil {
		a.cx.Post(func() { a.sendPrepareFailed(job, err) })
		return
	}
	a.cx.Post(func() { a.sendPrepared(job, answer) })
	for _, f := range job.Files {
		token, ok := answer.Files[f.ID]
		if !ok {
			continue
		}
		if ctx.Err() != nil {
			a.postSendDone(job.TransferID, statusCancelled, "cancelled")
			a.cancelRemote(client, job.Target, answer.SessionID)
			return
		}
		content, err := f.open()
		if err != nil {
			a.postSendDone(job.TransferID, statusFailed, err.Error())
			a.cancelRemote(client, job.Target, answer.SessionID)
			return
		}
		last := time.Time{}
		err = client.Upload(ctx, job.Target, answer.SessionID, f.ID, token, content, f.Size, func(n int64) {
			if n < f.Size && time.Since(last) < 140*time.Millisecond {
				return
			}
			last = time.Now()
			a.cx.Post(func() { a.sendFileProgress(job.TransferID, f.ID, n) })
		})
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, context.Canceled) {
				a.postSendDone(job.TransferID, statusCancelled, "cancelled")
			} else {
				a.postSendDone(job.TransferID, statusFailed, err.Error())
			}
			a.cancelRemote(client, job.Target, answer.SessionID)
			return
		}
		a.cx.Post(func() { a.sendFileProgress(job.TransferID, f.ID, f.Size) })
	}
	a.postSendDone(job.TransferID, statusCompleted, "")
}

// cancelRemote tells a receiver to drop a session.
func (a *app) cancelRemote(client *localsend.Client, target localsend.Device, sessionID string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = client.Cancel(ctx, target, sessionID)
	}()
}

// sendPrepareFailed handles a refused prepare-upload.
func (a *app) sendPrepareFailed(job sendJob, err error) {
	switch {
	case errors.Is(err, localsend.ErrPINRequired):
		a.updateTransfer(job.TransferID, func(t *transfer) {
			t.Status, t.NeedPIN, t.Reason, t.Err = statusWaiting, true, "pin", ""
		})
		a.pinValue.Set("")
		prompt := &pinPrompt{TransferID: job.TransferID, Target: job.Target, Wrong: job.PIN != ""}
		a.pinPrompt.Set(prompt)
	case errors.Is(err, localsend.ErrRejected):
		a.finalizeSend(job.TransferID, statusDeclined, "rejected", "")
	case errors.Is(err, localsend.ErrBusy):
		a.finalizeSend(job.TransferID, statusFailed, "busy", err.Error())
	case errors.Is(err, localsend.ErrNoFiles):
		a.updateTransfer(job.TransferID, func(t *transfer) {
			t.Total = 0
			for i := range t.Files {
				t.Files[i].Status = statusSkipped
			}
		})
		a.finalizeSend(job.TransferID, statusCompleted, "nofiles", "")
		a.toast(locales.NoFilesNeededDetail(a.cx))
	case errors.Is(err, context.Canceled):
		a.finalizeSend(job.TransferID, statusCancelled, "cancelled", "")
	default:
		a.finalizeSend(job.TransferID, statusFailed, "", err.Error())
	}
}

// sendPrepared marks accepted files and the session active.
func (a *app) sendPrepared(job sendJob, answer localsend.PrepareUploadResponse) {
	a.updateTransfer(job.TransferID, func(t *transfer) {
		t.Status, t.NeedPIN, t.Reason, t.Err = statusActive, false, "", ""
		t.Total = 0
		for i := range t.Files {
			file := &t.Files[i]
			if _, ok := answer.Files[file.ID]; ok {
				file.Status = "pending"
				t.Total += file.Size
			} else {
				file.Status = statusSkipped
			}
		}
	})
}

// sendFileProgress records bytes a file sent.
func (a *app) sendFileProgress(id, fileID string, n int64) {
	a.updateTransfer(id, func(t *transfer) {
		if finalStatus(t.Status) || t.CancelRequested {
			return
		}
		t.Status = statusActive
		for i := range t.Files {
			if t.Files[i].ID == fileID {
				t.Files[i].Done = n
				if n >= t.Files[i].Size {
					t.Files[i].Status = "done"
				}
				break
			}
		}
	})
}

// postSendDone finalizes a send from the upload goroutine.
func (a *app) postSendDone(id, status, message string) {
	a.cx.Post(func() { a.finalizeSend(id, status, "", message) })
}

// finalizeSend ends a transfer, cleans its files and saves its record.
func (a *app) finalizeSend(id, status, reason, message string) {
	if reason == "" {
		switch status {
		case statusCancelled:
			reason = "cancelled"
		}
	}
	t := a.updateTransfer(id, func(t *transfer) {
		t.Status, t.Reason, t.Err, t.NeedPIN = status, reason, message, false
		t.CancelRequested, t.Speed = false, 0
		t.Ended = time.Now()
		for i := range t.Files {
			file := &t.Files[i]
			switch status {
			case statusCompleted:
				if file.Status != statusSkipped {
					file.Status, file.Done = "done", file.Size
				}
			case statusCancelled, statusDeclined:
				if file.Status != "done" {
					file.Status = status
				}
			default:
				if file.Status != "done" {
					file.Status = status
					if file.Error == "" {
						file.Error = message
					}
				}
			}
		}
		if status == statusCompleted {
			t.Done = t.Total
		}
	})
	if t == nil {
		return
	}
	if job, ok := a.jobs[id]; ok {
		for _, f := range job.Files {
			if f.cleanup != nil {
				f.cleanup()
			}
		}
		delete(a.jobs, id)
	}
	if cancel := a.cancels[id]; cancel != nil {
		cancel()
		delete(a.cancels, id)
	}
	a.rememberTransfer(t)
	l := locales.For(a.cx)
	switch status {
	case statusCompleted:
		a.toastSuccess(l.TransferSent(t.Peer.Alias), formatBytes(t.Total))
	case statusDeclined:
		a.toastError(l.TransferSent(t.Peer.Alias), l.RejectedDetail())
	case statusCancelled:
		a.toast(l.Cancelled())
	case statusFailed:
		a.toastError(l.TransferSent(t.Peer.Alias), reasonText(a.cx, t))
	}
}

// pinDialog asks for the PIN a receiver wants.
func (a *app) pinDialog(cx *core.Scope) core.Node {
	return core.Show{When: func() bool { return a.pinPrompt.Get() != nil }, Then: func(cx *core.Scope) core.Node {
		prompt := *a.pinPrompt.Get()
		l := locales.For(cx)
		var wrong core.Node = core.Text{Content: l.PinWrong(), Style: core.Style{FontSize: 12}, StyleFunc: ui.Colored(ui.ColorsOf(cx).Danger)}
		if !prompt.Wrong {
			wrong = nil
		}
		return ui.Dialog{
			Open: func() bool { return a.pinPrompt.Get() != nil },
			OnOpenChange: func(open bool, _ base.Reason) {
				if !open {
					a.abandonPIN()
				}
			},
			Title:       l.PinTitle(),
			Description: l.PinDescription(prompt.Target.Alias),
			Children: []core.Node{
				ui.Input{Value: a.pinValue.Get, OnChange: a.pinValue.Set, Placeholder: l.PinPlaceholder(), Masked: true,
					KeyboardType: core.KeyboardNumber, MaxLength: 12, AutoFocus: true,
					OnSubmit: func(string) { a.submitPIN() }, Label: l.PinTitle(), TestID: "pin-input"},
				wrong,
			},
			Footer: []core.Node{
				ui.Button{Text: l.Cancel(), Variant: ui.ButtonSecondary, OnPress: a.abandonPIN, TestID: "pin-cancel"},
				ui.Button{Text: l.Send(), OnPress: a.submitPIN, TestID: "pin-submit"},
			},
			TestID: "pin-dialog",
		}
	}}
}

// submitPIN retries a send with the typed PIN.
func (a *app) submitPIN() {
	prompt := a.pinPrompt.Get()
	if prompt == nil {
		return
	}
	job, ok := a.jobs[prompt.TransferID]
	if !ok {
		a.pinPrompt.Set(nil)
		return
	}
	pin := strings.TrimSpace(a.pinValue.Get())
	if pin == "" {
		return
	}
	job.PIN = pin
	a.jobs[prompt.TransferID] = job
	a.pinPrompt.Set(nil)
	a.updateTransfer(prompt.TransferID, func(t *transfer) {
		t.Status, t.NeedPIN, t.Reason, t.Err = statusWaiting, false, "", ""
		t.Done = 0
		for i := range t.Files {
			if t.Files[i].Status != statusSkipped {
				t.Files[i].Status, t.Files[i].Done = "pending", 0
			}
		}
	})
	ctx, cancel := context.WithCancel(context.Background())
	a.cancels[prompt.TransferID] = cancel
	go a.runSend(ctx, job)
}

// abandonPIN cancels a send that waited for a PIN.
func (a *app) abandonPIN() {
	prompt := a.pinPrompt.Get()
	a.pinPrompt.Set(nil)
	if prompt == nil {
		return
	}
	if _, ok := a.jobs[prompt.TransferID]; !ok {
		return
	}
	a.finalizeSend(prompt.TransferID, statusCancelled, "cancelled", "")
}
