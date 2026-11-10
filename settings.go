package main

import (
	"strconv"
	"strings"

	"github.com/oneo-me/nsgo/core"
	"github.com/oneo-me/nsgo/ui"

	"github.com/oneo-me/nsgo-localsend/internal/localsend"
	"github.com/oneo-me/nsgo-localsend/locales"
)

// settingsPage is the app's preferences.
func (a *app) settingsPage(cx *core.Scope) core.Node {
	return core.Scroll{Style: ui.FillRest, TestID: "settings-page", Children: []core.Node{
		core.View{Style: core.Style{Padding: pagePadding, Gap: 16, MaxWidth: core.Pt(760)}, Children: []core.Node{
			a.identitySettings(cx),
			a.networkSettings(cx),
			a.receiveSettings(cx),
			a.appearanceSettings(cx),
			a.aboutSettings(cx),
			bottomGap(cx),
		}},
	}}
}

// settingsCard is a titled card of settings.
func (a *app) settingsCard(cx *core.Scope, title, detail string, children []core.Node) core.Node {
	colors := ui.ColorsOf(cx)
	var head []core.Node
	if title != "" {
		head = append(head, core.Text{Content: title, Style: core.Style{FontSize: 15, LineHeight: 20, FontWeight: ui.Semibold},
			StyleFunc: ui.Colored(colors.Text)})
	}
	if detail != "" {
		head = append(head, core.Text{Content: detail, MaxLines: 3, Style: ui.TypeCaption, StyleFunc: ui.Colored(colors.Muted)})
	}
	body := append(head, children...)
	return core.View{Style: core.Style{Padding: core.All(16), Gap: 12}, StyleFunc: func() core.Style { return ui.CardStyle(cx) }, Children: body}
}

// settingsField is a setting with its control below its label.
func (a *app) settingsField(cx *core.Scope, title, detail string, control core.Node) core.Node {
	colors := ui.ColorsOf(cx)
	var body []core.Node
	body = append(body, core.Text{Content: title, Style: ui.TypeLabel, StyleFunc: ui.Colored(colors.Text)})
	if detail != "" {
		body = append(body, core.Text{Content: detail, MaxLines: 3, Style: ui.TypeCaption, StyleFunc: ui.Colored(colors.Muted)})
	}
	body = append(body, control)
	return core.View{Style: core.Style{Gap: 5}, Children: body}
}

// settingsRow places a label and its control side by side.
func (a *app) settingsRow(cx *core.Scope, title, detail string, control core.Node) core.Node {
	colors := ui.ColorsOf(cx)
	var text []core.Node
	text = append(text, core.Text{Content: title, Style: ui.TypeLabel, StyleFunc: ui.Colored(colors.Text)})
	if detail != "" {
		text = append(text, core.Text{Content: detail, MaxLines: 3, Style: ui.TypeCaption, StyleFunc: ui.Colored(colors.Muted)})
	}
	return core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 14}, Children: []core.Node{
		core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0), Gap: 2}, Children: text},
		control,
	}}
}

// identitySettings names this device.
func (a *app) identitySettings(cx *core.Scope) core.Node {
	l := locales.For(cx)
	colors := ui.ColorsOf(cx)
	self := a.selfDevice()
	return a.settingsCard(cx, l.SettingsIdentity(), l.SettingsIdentityDetail(), []core.Node{
		a.settingsField(cx, l.Alias(), l.AliasPlaceholder(), ui.Input{
			Value: func() string { return a.settings.Get().Alias },
			OnChange: func(value string) {
				a.settings.Update(func(s settings) settings { s.Alias = value; return s })
			},
			Placeholder: l.AliasPlaceholder(), MaxLength: 64, TestID: "settings-alias",
		}),
		a.settingsRow(cx, l.DeviceType(), "", ui.Badge{Text: deviceTypeLabel(cx, self.DeviceType), Variant: ui.BadgeNeutral}),
		a.settingsRow(cx, l.Fingerprint(), "", core.Text{ContentFunc: func() string { return shortFingerprint(self.Fingerprint) },
			Style: core.Style{FontSize: 11, FontVariant: core.FontTabularNums}, StyleFunc: ui.Colored(colors.Muted)}),
	})
}

// networkSettings sets the receiver's port, encryption and PIN.
func (a *app) networkSettings(cx *core.Scope) core.Node {
	l := locales.For(cx)
	colors := ui.ColorsOf(cx)
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
	statusNode := core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 8}, Children: []core.Node{
		core.View{Style: core.Style{Width: core.Pt(7), Height: core.Pt(7), CornerRadius: 4}, StyleFunc: func() core.Style {
			switch {
			case a.offline, a.serverErr.Get() != "":
				return core.Style{Background: colors.Danger()}
			case !a.nodeReady.Get():
				return core.Style{Background: colors.Warn()}
			}
			return core.Style{Background: colors.Up()}
		}},
		core.Text{ContentFunc: status, Style: ui.TypeCaption, StyleFunc: ui.Colored(colors.Muted)},
		core.View{Style: core.Style{FlexGrow: 1}},
		core.Show{When: func() bool { return a.serverErr.Get() != "" }, Then: func(cx *core.Scope) core.Node {
			return ui.Button{Text: l.Retry(), Variant: ui.ButtonSecondary, Size: ui.SizeSmall, OnPress: a.restartNode, TestID: "retry-server"}
		}},
	}}
	return a.settingsCard(cx, l.SettingsNetwork(), l.SettingsNetworkDetail(), []core.Node{
		statusNode,
		a.settingsRow(cx, l.Port(), l.PortDetail(), core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 6}, Children: []core.Node{
			core.View{Style: core.Style{Width: core.Pt(96)}, Children: []core.Node{ui.Input{Value: a.portDraft.Get, OnChange: a.portDraft.Set,
				KeyboardType: core.KeyboardNumber, MaxLength: 5, Label: l.Port(), TestID: "settings-port",
				OnSubmit: func(value string) { a.applyPort(value) }}}},
			ui.Button{Text: l.Apply(), Variant: ui.ButtonSecondary, Size: ui.SizeSmall, OnPress: func() {
				a.applyPort(a.portDraft.Get())
			}, TestID: "apply-port"},
		}}),
		a.settingsRow(cx, l.Encryption(), l.EncryptionDetail(), ui.Select[localsend.Protocol]{
			Options: []ui.Option[localsend.Protocol]{
				{Value: localsend.ProtocolHTTPS, Label: l.EncryptionOn()},
				{Value: localsend.ProtocolHTTP, Label: l.EncryptionOff()},
			},
			Value: func() localsend.Protocol { return a.settings.Get().Protocol },
			OnChange: func(value localsend.Protocol) {
				a.settings.Update(func(s settings) settings { s.Protocol = value; return s })
				a.restartNode()
				a.toast(l.SettingsSaved())
			},
			Label: l.Encryption(), TestID: "settings-protocol",
			Style: core.Style{Width: core.Pt(170)},
		}),
		a.settingsField(cx, l.Pin(), l.PinDetail(), ui.Input{
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
	return a.settingsCard(cx, l.SettingsReceiving(), l.SettingsReceivingDetail(), []core.Node{
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
		a.settingsField(cx, l.ReceiveFolder(), "", core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 8,
			Wrap: core.Wrap}, Children: []core.Node{
			core.Text{ContentFunc: func() string {
				if dir := a.settings.Get().ReceiveDir; dir != "" {
					return dir
				}
				return defaultReceiveDir(a.cx)
			}, MaxLines: 2, Style: core.Style{FontSize: 12, FontFamily: ui.Mono, FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(120)},
				StyleFunc: ui.Colored(colors.Muted)},
			ui.Button{Text: l.ChooseFolderShort(), Variant: ui.ButtonSecondary, Size: ui.SizeSmall, OnPress: a.chooseReceiveDir, TestID: "settings-choose-folder"},
			ui.Button{Text: l.UseDefaultFolder(), Variant: ui.ButtonGhost, Size: ui.SizeSmall, Disabled: func() bool { return a.settings.Get().ReceiveDir == "" },
				OnPress: func() { a.settings.Update(func(s settings) settings { s.ReceiveDir = ""; return s }) }, TestID: "settings-reset-folder"},
		}, TestID: "settings-receive-dir"}),
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
	return a.settingsCard(cx, l.SettingsAppearance(), l.SettingsAppearanceDetail(), []core.Node{
		a.settingsRow(cx, l.Theme(), l.SettingsAppearanceDetail(), ui.Select[string]{
			Options: []ui.Option[string]{
				{Value: "", Label: l.ThemeSystem()},
				{Value: "light", Label: l.ThemeLight()},
				{Value: "dark", Label: l.ThemeDark()},
			},
			Value: func() string { return a.settings.Get().Appearance },
			OnChange: func(value string) {
				a.settings.Update(func(s settings) settings { s.Appearance = value; return s })
			},
			Label: l.Theme(), TestID: "settings-theme",
			Style: core.Style{Width: core.Pt(170)},
		}),
		a.settingsRow(cx, l.Language(), "", ui.Select[string]{
			Options: []ui.Option[string]{
				{Value: "", Label: l.LanguageSystem()},
				{Value: "en", Label: l.LanguageEnglish()},
				{Value: "zh-Hans", Label: l.LanguageChinese()},
			},
			Value: func() string { return a.settings.Get().Locale },
			OnChange: func(value string) {
				a.settings.Update(func(s settings) settings { s.Locale = value; return s })
			},
			Label: l.Language(), TestID: "settings-language",
			Style: core.Style{Width: core.Pt(170)},
		}),
	})
}

// aboutSettings names the app and its protocol.
func (a *app) aboutSettings(cx *core.Scope) core.Node {
	l := locales.For(cx)
	colors := ui.ColorsOf(cx)
	self := a.selfDevice()
	return a.settingsCard(cx, l.SettingsAbout(), "", []core.Node{
		core.Text{Content: l.AboutDetail(), Style: ui.TypeBody, StyleFunc: ui.Colored(colors.Muted)},
		a.settingsRow(cx, l.Version(), "", core.Text{Content: versionString(), Style: ui.TypeCaption,
			StyleFunc: ui.Colored(colors.Muted)}),
		a.settingsRow(cx, l.Protocol(), "", core.Text{Content: localsend.Version, Style: ui.TypeCaption,
			StyleFunc: ui.Colored(colors.Muted)}),
		a.settingsRow(cx, l.Fingerprint(), "", core.Text{ContentFunc: func() string { return shortFingerprint(self.Fingerprint) },
			Style: core.Style{FontSize: 11, FontVariant: core.FontTabularNums}, StyleFunc: ui.Colored(colors.Muted)}),
	})
}
