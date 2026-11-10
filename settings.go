package main

import (
	"strconv"
	"strings"

	"github.com/oneo-me/nsgo/base"
	"github.com/oneo-me/nsgo/core"
	"github.com/oneo-me/nsgo/icons/lucide"
	"github.com/oneo-me/nsgo/ui"

	"github.com/oneo-me/nsgo-localsend/internal/localsend"
	"github.com/oneo-me/nsgo-localsend/locales"
)

// settingsPage is the app's preferences as grouped lists, the way iOS's
// Settings lays them out.
func (a *app) settingsPage(cx *core.Scope) core.Node {
	return a.pageScroll(cx, "settings", "settings-page",
		core.View{Style: core.Style{Padding: pagePadding, Gap: 28, MaxWidth: core.Pt(760)}, Children: []core.Node{
			a.identitySettings(cx),
			a.networkSettings(cx),
			a.receiveSettings(cx),
			a.appearanceSettings(cx),
			a.aboutSettings(cx),
			bottomGap(cx),
		}})
}

// settingsType is the settings' type scale: larger on touch screens,
// where rows take a finger as iOS's do.
type settingsType struct {
	title, detail, note core.Points
	row                 core.Points
}

func settingsTypeOf(cx *core.Scope) settingsType {
	if ui.Touch(cx) {
		return settingsType{title: 17, detail: 13, note: 13, row: 52}
	}
	return settingsType{title: 13, detail: 11.5, note: 12, row: 40}
}

// settingsRadius rounds a group of rows as iOS 26's inset grouped lists.
const settingsRadius core.Points = 22

// settingsSection is rows on one rounded surface, split by inset lines,
// under a title and over a note.
func (a *app) settingsSection(cx *core.Scope, title, note string, rows []core.Node) core.Node {
	colors := ui.ColorsOf(cx)
	ts := settingsTypeOf(cx)
	var body []core.Node
	for i, row := range rows {
		if i > 0 {
			body = append(body, core.View{Style: core.Style{Height: core.Pt(1), Margin: core.Edges{Left: 16}}, StyleFunc: ui.Filled(colors.Line)})
		}
		body = append(body, row)
	}
	var children []core.Node
	if title != "" {
		children = append(children, core.Text{Content: title, Role: core.RoleHeader,
			Style:     core.Style{FontSize: ts.note, LineHeight: ts.note + 5, FontWeight: ui.Medium, Padding: core.Symmetric(0, 16)},
			StyleFunc: ui.Colored(colors.Muted)})
	}
	children = append(children, core.View{Style: core.Style{CornerRadius: settingsRadius, BorderCurve: core.BorderCurveContinuous, Overflow: core.Hidden},
		StyleFunc: ui.Filled(colors.Surface), Children: body})
	if note != "" {
		children = append(children, core.Text{Content: note, Style: core.Style{FontSize: ts.note, LineHeight: ts.note + 5, Padding: core.Symmetric(0, 16)},
			StyleFunc: ui.Colored(colors.Muted)})
	}
	return core.View{Style: core.Style{Gap: 7}, Children: children}
}

// settingsText is a row's title over its detail.
func (a *app) settingsText(cx *core.Scope, title, detail string) core.Node {
	colors := ui.ColorsOf(cx)
	ts := settingsTypeOf(cx)
	text := []core.Node{core.Text{Content: title, Style: core.Style{FontSize: ts.title, LineHeight: ts.title + 5}, StyleFunc: ui.Colored(colors.Text)}}
	if detail != "" {
		text = append(text, core.Text{Content: detail, MaxLines: 3, Style: core.Style{FontSize: ts.detail, LineHeight: ts.detail + 4},
			StyleFunc: ui.Colored(colors.Muted)})
	}
	return core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0), Gap: 2}, Children: text}
}

// settingsRowStyle is a row's box: its height, gutter and layout.
func settingsRowStyle(cx *core.Scope) core.Style {
	return core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 12, MinHeight: core.Pt(settingsTypeOf(cx).row),
		Padding: core.Symmetric(8, 16)}
}

// settingsRow is a title and detail with a control at the row's end.
func (a *app) settingsRow(cx *core.Scope, title, detail string, control core.Node) core.Node {
	children := []core.Node{a.settingsText(cx, title, detail)}
	if control != nil {
		children = append(children, core.View{Style: core.Style{FlexShrink: 0}, Children: []core.Node{control}})
	}
	return core.View{Style: settingsRowStyle(cx), Children: children}
}

// settingsValue is a row's value at its end, in the secondary color.
func (a *app) settingsValue(cx *core.Scope, value func() string) core.Node {
	colors := ui.ColorsOf(cx)
	ts := settingsTypeOf(cx)
	return core.Text{ContentFunc: value, MaxLines: 1, Style: core.Style{FontSize: ts.title, FontVariant: core.FontTabularNums},
		StyleFunc: ui.Colored(colors.Muted)}
}

// settingsField is a row whose title is followed by a borderless field
// that fills the rest, its text at the end, as iOS's forms edit.
func (a *app) settingsField(cx *core.Scope, title string, field base.Input, trailing ...core.Node) core.Node {
	colors := ui.ColorsOf(cx)
	ts := settingsTypeOf(cx)
	field.Style = core.Style{FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(60), FontSize: ts.title, TextAlign: core.TextAlignEnd}
	field.StyleFunc = func() core.Style {
		return core.Style{Color: colors.Muted(), PlaceholderColor: colors.Faint(), CaretColor: colors.Accent(), SelectionColor: cx.Palette().Selection}
	}
	children := []core.Node{
		core.Text{Content: title, MaxLines: 1, Style: core.Style{FontSize: ts.title, FlexShrink: 0}, StyleFunc: ui.Colored(colors.Text)},
		field,
	}
	return core.View{Style: settingsRowStyle(cx), Children: append(children, trailing...)}
}

// settingsChoice is a row that picks one of options from a menu, which
// its value and a pair of chevrons at the row's end open.
func settingsChoice[T comparable](a *app, cx *core.Scope, title, detail string, options []ui.Option[T], value func() T, change func(T), testID string) core.Node {
	colors := ui.ColorsOf(cx)
	ts := settingsTypeOf(cx)
	label := func() string {
		v := value()
		for _, o := range options {
			if o.Value == v {
				return o.Label
			}
		}
		return ""
	}
	var items []core.Node
	for i, o := range options {
		items = append(items, ui.MenuItem{Text: o.Label, Checked: func() bool { return value() == o.Value }, OnSelect: func() { change(o.Value) },
			TestID: testID + "-" + strconv.Itoa(i)})
	}
	face := core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 4}, Children: []core.Node{
		core.Text{ContentFunc: label, MaxLines: 1, Style: core.Style{FontSize: ts.title}, StyleFunc: ui.Colored(colors.Muted)},
		ui.IconOf(lucide.ChevronsUpDown, ts.title-3, colors.Muted),
	}}
	return a.settingsRow(cx, title, detail, ui.Menu{Trigger: ui.Button{Variant: ui.ButtonGhost}, Face: face, Align: core.AlignEnd,
		Children: items, Label: title, TestID: testID})
}

// settingsAction is a row that does what its accent-colored text says,
// and none while disabled says so.
func (a *app) settingsAction(cx *core.Scope, text string, disabled func() bool, press func(), testID string) core.Node {
	colors := ui.ColorsOf(cx)
	ts := settingsTypeOf(cx)
	return core.View{Role: core.RoleButton, Label: text, TestID: testID, Disabled: disabled,
		OnPress: func(core.PressEvent) {
			if !disabled() {
				press()
			}
		},
		Style: settingsRowStyle(cx),
		StyleFunc: func() core.Style {
			if disabled() {
				return core.Style{}
			}
			return core.Style{Active: &core.Style{Background: colors.Pressed()}}
		},
		Children: []core.Node{core.Text{Content: text, Style: core.Style{FontSize: ts.title}, StyleFunc: func() core.Style {
			if disabled() {
				return core.Style{Color: colors.Faint()}
			}
			return core.Style{Color: colors.Accent()}
		}}}}
}

// identitySettings names this device.
func (a *app) identitySettings(cx *core.Scope) core.Node {
	l := locales.For(cx)
	self := a.selfDevice()
	return a.settingsSection(cx, l.SettingsIdentity(), l.SettingsIdentityDetail(), []core.Node{
		a.settingsField(cx, l.Alias(), base.Input{
			Value: func() string { return a.settings.Get().Alias },
			OnChange: func(value string) {
				a.settings.Update(func(s settings) settings { s.Alias = value; return s })
			},
			Placeholder: l.AliasPlaceholder(), MaxLength: 64, Label: l.Alias(), TestID: "settings-alias",
		}),
		a.settingsRow(cx, l.DeviceType(), "", a.settingsValue(cx, func() string { return deviceTypeLabel(cx, self.DeviceType) })),
	})
}

// networkSettings sets the receiver's port, encryption and PIN.
func (a *app) networkSettings(cx *core.Scope) core.Node {
	l := locales.For(cx)
	colors := ui.ColorsOf(cx)
	ts := settingsTypeOf(cx)
	status := func() string {
		switch {
		case a.offline:
			return l.ServerStopped()
		case a.serverErr.Get() != "":
			return l.ServerErrorTitle()
		case !a.nodeReady.Get():
			return l.Searching()
		}
		return l.ServerRunning("0.0.0.0:" + strconv.Itoa(a.selfDevice().Port))
	}
	statusRow := core.View{Style: settingsRowStyle(cx), Children: []core.Node{
		core.View{Style: core.Style{Width: core.Pt(8), Height: core.Pt(8), CornerRadius: 4, FlexShrink: 0}, StyleFunc: func() core.Style {
			switch {
			case a.offline, a.serverErr.Get() != "":
				return core.Style{Background: colors.Danger()}
			case !a.nodeReady.Get():
				return core.Style{Background: colors.Warn()}
			}
			return core.Style{Background: colors.Up()}
		}},
		core.Text{ContentFunc: status, MaxLines: 2, Style: core.Style{FontSize: ts.title, FlexGrow: 1, FlexShrink: 1}, StyleFunc: ui.Colored(colors.Text)},
		core.Show{When: func() bool { return a.serverErr.Get() != "" }, Then: func(cx *core.Scope) core.Node {
			return ui.Button{Text: l.Retry(), Variant: ui.ButtonSecondary, Size: ui.SizeSmall, OnPress: a.restartNode, TestID: "retry-server"}
		}},
	}}
	return a.settingsSection(cx, l.SettingsNetwork(), l.PortDetail(), []core.Node{
		statusRow,
		a.settingsField(cx, l.Port(), base.Input{Value: a.portDraft.Get, OnChange: a.portDraft.Set,
			KeyboardType: core.KeyboardNumber, MaxLength: 5, Label: l.Port(), TestID: "settings-port",
			OnSubmit: func(value string) { a.applyPort(value) }},
			core.Show{When: func() bool { return a.portDraft.Get() != strconv.Itoa(a.settings.Get().Port) }, Then: func(cx *core.Scope) core.Node {
				return ui.Button{Text: l.Apply(), Variant: ui.ButtonSecondary, Size: ui.SizeSmall, OnPress: func() {
					a.applyPort(a.portDraft.Get())
				}, TestID: "apply-port"}
			}}),
		settingsChoice(a, cx, l.Encryption(), l.EncryptionDetail(), []ui.Option[localsend.Protocol]{
			{Value: localsend.ProtocolHTTPS, Label: l.EncryptionOn()},
			{Value: localsend.ProtocolHTTP, Label: l.EncryptionOff()},
		}, func() localsend.Protocol { return a.settings.Get().Protocol }, func(value localsend.Protocol) {
			if value == a.settings.Get().Protocol {
				return
			}
			a.settings.Update(func(s settings) settings { s.Protocol = value; return s })
			a.restartNode()
			a.toast(l.SettingsSaved())
		}, "settings-protocol"),
		a.settingsField(cx, l.Pin(), base.Input{
			Value: func() string { return a.settings.Get().ReceivePIN },
			OnChange: func(value string) {
				a.settings.Update(func(s settings) settings { s.ReceivePIN = sanitizePIN(value); return s })
			},
			Placeholder: l.ReceivePinPlaceholder(), Masked: true, KeyboardType: core.KeyboardNumber, MaxLength: 12,
			Label: l.Pin(), TestID: "settings-pin",
		}),
	})
}

// applyPort restarts the receiver on a new port.
func (a *app) applyPort(value string) {
	port := parsePort(value)
	if port == 0 {
		a.toastError(locales.Port(a.cx), locales.PortDetail(a.cx))
		return
	}
	a.portDraft.Set(strconv.Itoa(port))
	if port == a.settings.Get().Port {
		return
	}
	a.settings.Update(func(s settings) settings { s.Port = port; return s })
	a.restartNode()
	a.toast(locales.SettingsSaved(a.cx))
}

// receiveSettings chooses where received files go and how they are handled.
func (a *app) receiveSettings(cx *core.Scope) core.Node {
	l := locales.For(cx)
	colors := ui.ColorsOf(cx)
	ts := settingsTypeOf(cx)
	folder := core.View{Role: core.RoleButton, Label: l.ReceiveFolder(), TestID: "settings-receive-dir",
		OnPress: func(core.PressEvent) { a.chooseReceiveDir() },
		Style:   settingsRowStyle(cx), StyleFunc: func() core.Style { return core.Style{Active: &core.Style{Background: colors.Pressed()}} },
		Children: []core.Node{
			core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0), Gap: 2}, Children: []core.Node{
				core.Text{Content: l.ReceiveFolder(), Style: core.Style{FontSize: ts.title, LineHeight: ts.title + 5}, StyleFunc: ui.Colored(colors.Text)},
				core.Text{ContentFunc: func() string {
					if dir := a.settings.Get().ReceiveDir; dir != "" {
						return dir
					}
					return defaultReceiveDir(a.cx)
				}, MaxLines: 2, Style: core.Style{FontSize: ts.detail, LineHeight: ts.detail + 4, FontFamily: ui.Mono}, StyleFunc: ui.Colored(colors.Muted)},
			}},
			core.Text{Content: l.ChooseFolderShort(), MaxLines: 1, TestID: "settings-choose-folder", Style: core.Style{FontSize: ts.title, FlexShrink: 0},
				StyleFunc: ui.Colored(colors.Muted)},
			ui.IconOf(lucide.ChevronRight, ts.title-2, colors.Faint),
		}}
	return a.settingsSection(cx, l.SettingsReceiving(), l.SettingsReceivingDetail(), []core.Node{
		a.settingsRow(cx, l.QuickSave(), l.QuickSaveDetail(), ui.Switch{
			Checked: func() bool { return a.settings.Get().QuickSave },
			OnChange: func(on bool) {
				a.settings.Update(func(s settings) settings { s.QuickSave = on; return s })
			},
			Label: l.QuickSave(),
		}),
		a.settingsRow(cx, l.VerifyChecksums(), l.VerifyChecksumsDetail(), ui.Switch{
			Checked: func() bool { return a.settings.Get().VerifyChecksums },
			OnChange: func(on bool) {
				a.settings.Update(func(s settings) settings { s.VerifyChecksums = on; return s })
				a.restartNode()
			},
			Label: l.VerifyChecksums(),
		}),
		folder,
		a.settingsAction(cx, l.UseDefaultFolder(), func() bool { return a.settings.Get().ReceiveDir == "" },
			func() { a.settings.Update(func(s settings) settings { s.ReceiveDir = ""; return s }) }, "settings-reset-folder"),
	})
}

// chooseReceiveDir lets the user pick where received files go.
func (a *app) chooseReceiveDir() {
	a.cx.PickDirectories(core.PickDirectoriesOptions{Limit: 1}, func(dirs []core.Directory, err error) {
		if err != nil {
			a.toastError(locales.ReceiveFolder(a.cx), err.Error())
			return
		}
		if len(dirs) == 0 {
			return
		}
		if dirs[0].Path == "" {
			a.toastError(locales.ReceiveFolder(a.cx), locales.ChooseFolderUnavailable(a.cx))
			return
		}
		dir := dirs[0].Path
		a.settings.Update(func(s settings) settings { s.ReceiveDir = dir; return s })
		a.toast(locales.SettingsSaved(a.cx))
	})
}

// sanitizePIN keeps a receive PIN numeric.
func sanitizePIN(value string) string {
	var b strings.Builder
	for _, r := range value {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if len(out) > 12 {
		out = out[:12]
	}
	return out
}

// appearanceSettings chooses the look and language.
func (a *app) appearanceSettings(cx *core.Scope) core.Node {
	l := locales.For(cx)
	return a.settingsSection(cx, l.SettingsAppearance(), l.SettingsAppearanceDetail(), []core.Node{
		settingsChoice(a, cx, l.Theme(), "", []ui.Option[string]{
			{Value: "", Label: l.ThemeSystem()},
			{Value: "light", Label: l.ThemeLight()},
			{Value: "dark", Label: l.ThemeDark()},
		}, func() string { return a.settings.Get().Appearance }, func(value string) {
			a.settings.Update(func(s settings) settings { s.Appearance = value; return s })
		}, "settings-theme"),
		settingsChoice(a, cx, l.Language(), "", []ui.Option[string]{
			{Value: "", Label: l.LanguageSystem()},
			{Value: "en", Label: l.LanguageEnglish()},
			{Value: "zh-Hans", Label: l.LanguageChinese()},
		}, func() string { return a.settings.Get().Locale }, func(value string) {
			a.settings.Update(func(s settings) settings { s.Locale = value; return s })
		}, "settings-language"),
	})
}

// aboutSettings names the app and its protocol.
func (a *app) aboutSettings(cx *core.Scope) core.Node {
	l := locales.For(cx)
	self := a.selfDevice()
	return a.settingsSection(cx, l.SettingsAbout(), l.AboutDetail(), []core.Node{
		a.settingsRow(cx, l.Version(), "", a.settingsValue(cx, versionString)),
		a.settingsRow(cx, l.Protocol(), "", a.settingsValue(cx, func() string { return localsend.Version })),
		a.settingsRow(cx, l.Fingerprint(), "", a.settingsValue(cx, func() string { return shortFingerprint(self.Fingerprint) })),
	})
}
