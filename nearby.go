package main

import (
	"strings"
	"time"

	"github.com/oneo-me/nsgo/core"
	"github.com/oneo-me/nsgo/icons/lucide"
	"github.com/oneo-me/nsgo/ui"

	"github.com/oneo-me/nsgo-localsend/internal/localsend"
	"github.com/oneo-me/nsgo-localsend/locales"
)

// nearbyPage shows this device, the network's state and nearby devices.
func (a *app) nearbyPage(cx *core.Scope) core.Node {
	return a.pageScroll(cx, "nearby", "nearby-page",
		core.View{Style: core.Style{Padding: pagePadding, Gap: 18}, Children: []core.Node{
			a.ownCard(cx),
			a.networkBanners(cx),
			a.devicesHeader(cx),
			a.deviceGrid(cx),
			bottomGap(cx),
		}})
}

// ownCard describes this device and its receiver.
func (a *app) ownCard(cx *core.Scope) core.Node {
	return core.Dynamic(func(cx *core.Scope) core.Node { return a.ownCardBody(cx) })
}

// ownCardBody builds the card as the receiver and settings change.
func (a *app) ownCardBody(cx *core.Scope) core.Node {
	// The card rebuilds when the receiver comes up, so its addresses and
	// fingerprint are there.
	_ = a.nodeReady.Get()
	_ = a.serverErr.Get()
	colors := ui.ColorsOf(cx)
	l := locales.For(cx)
	status := func() string {
		switch {
		case a.offline:
			return l.ServerStopped()
		case a.serverErr.Get() != "":
			return l.ServerErrorTitle()
		case !a.nodeReady.Get():
			return l.Searching()
		}
		addresses := a.hosts.Get()
		host := "localhost"
		if len(addresses) > 0 {
			host = addresses[0]
		}
		return l.ServerRunning(host + ":" + itoa(a.selfDevice().Port))
	}
	statusColor := func() core.Color {
		switch {
		case a.offline, a.serverErr.Get() != "":
			return colors.Danger()
		case !a.nodeReady.Get():
			return colors.Warn()
		}
		return colors.Up()
	}
	self := a.selfDevice()
	return core.View{TestID: "this-device", Style: core.Style{Padding: core.All(18), Gap: 14}, StyleFunc: func() core.Style {
		return ui.CardStyle(cx).Merge(core.Style{Shadows: ui.FinOf(cx).Raised().Shadows})
	},
		Children: []core.Node{
			core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 12}, Children: []core.Node{
				deviceTile(cx, deviceIconForSelf(a.dtype), true, 46),
				core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0), Gap: 2}, Children: []core.Node{
					core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 8}, Children: []core.Node{
						core.Text{ContentFunc: func() string { return a.settings.Get().Alias }, MaxLines: 1,
							Style: ui.TypeTitle.Merge(core.Style{FlexShrink: 1}), StyleFunc: ui.Colored(colors.Text)},
						ui.Badge{Text: l.You(), Variant: ui.BadgeAccent},
					}},
					core.Text{ContentFunc: func() string {
						model := self.DeviceModel
						if model == "" {
							model = deviceTypeLabel(cx, self.DeviceType)
						}
						return model
					}, MaxLines: 1, Style: ui.TypeBody, StyleFunc: ui.Colored(colors.Muted)},
				}},
				ui.IconButton{Icon: lucide.Settings, Label: l.Settings(), Tint: colors.Muted, OnPress: func() { a.page.Set("settings") }, TestID: "open-settings"},
			}},
			core.View{Style: core.Style{Direction: core.Row, Wrap: core.Wrap, AlignItems: core.AlignCenter, Gap: 8}, Children: a.addresses(cx)},
			core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 8}, Children: []core.Node{
				statusDot(cx, statusColor, func() bool { return !a.offline && a.serverErr.Get() == "" && a.nodeReady.Get() }),
				core.Text{ContentFunc: status, MaxLines: 1, Style: ui.TypeCaption.Merge(core.Style{FlexShrink: 1}), StyleFunc: ui.Colored(colors.Muted)},
				core.View{Style: core.Style{FlexGrow: 1}},
				ui.Badge{TextFunc: func() string {
					if a.settings.Get().Protocol == localsend.ProtocolHTTPS {
						return l.EncryptionOn()
					}
					return l.EncryptionOff()
				}, VariantFunc: func() ui.BadgeVariant {
					if a.settings.Get().Protocol == localsend.ProtocolHTTPS {
						return ui.BadgeAccent
					}
					return ui.BadgeWarn
				}},
			}},
		}}
}

// addresses are this device's reachable addresses.
func (a *app) addresses(cx *core.Scope) []core.Node {
	colors := ui.ColorsOf(cx)
	self := a.selfDevice()
	addresses := append([]string(nil), a.hosts.Get()...)
	if len(addresses) == 0 {
		addresses = []string{"localhost"}
	}
	var out []core.Node
	for _, host := range addresses {
		out = append(out, core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 5, Padding: core.Symmetric(3, 8),
			CornerRadius: 7, BorderWidth: 1}, StyleFunc: func() core.Style {
			return core.Style{Background: colors.Inset(), BorderColor: colors.Border()}
		}, Children: []core.Node{
			ui.IconOf(lucide.Wifi, 12, colors.Muted),
			core.Text{Content: host + ":" + itoa(self.Port), Style: core.Style{FontSize: 11, FontVariant: core.FontTabularNums},
				StyleFunc: ui.Colored(colors.Muted)},
		}})
	}
	out = append(out, core.Text{Content: shortFingerprint(self.Fingerprint), MaxLines: 1,
		Style: core.Style{FontSize: 11, FontVariant: core.FontTabularNums, FlexShrink: 1}, StyleFunc: ui.Colored(colors.Faint)})
	return out
}

// networkBanners show receiver and discovery problems.
func (a *app) networkBanners(cx *core.Scope) core.Node {
	l := locales.For(cx)
	return core.Group{
		core.Show{When: func() bool { return a.cx.Env().OS() == core.OSWeb }, Then: func(cx *core.Scope) core.Node {
			return a.banner(cx, ui.TintsOf(cx).Info(), ui.ColorsOf(cx).Info, lucide.Globe, l.WebTitle(),
				l.WebDetail(), "", nil, "web-notice")
		}},
		core.Show{When: func() bool { return a.serverErr.Get() != "" }, Then: func(cx *core.Scope) core.Node {
			return a.banner(cx, ui.TintsOf(cx).Danger(), ui.ColorsOf(cx).Danger, lucide.CircleAlert, l.ServerErrorTitle(),
				a.serverErr.Get(), l.Retry(), a.restartNode, "server-error")
		}},
		core.Show{When: func() bool { return a.networkErr.Get() != "" && a.serverErr.Get() == "" }, Then: func(cx *core.Scope) core.Node {
			return a.banner(cx, ui.TintsOf(cx).Warn(), ui.ColorsOf(cx).Warn, lucide.TriangleAlert, l.LinkLocal(),
				a.networkErr.Get(), "", nil, "network-warning")
		}},
	}
}

// banner is a tinted notice with an optional action.
func (a *app) banner(cx *core.Scope, tint core.Color, fg func() core.Color, icon func() *core.SVG, title, detail, action string, onAction func(), testID string) core.Node {
	colors := ui.ColorsOf(cx)
	var trailing []core.Node
	if action != "" && onAction != nil {
		trailing = append(trailing, ui.Button{Text: action, Variant: ui.ButtonSecondary, Size: ui.SizeSmall, OnPress: onAction})
	}
	if testID == "network-warning" {
		trailing = append(trailing, ui.IconButton{Icon: lucide.X, Label: locales.Close(cx), Size: 24, Tint: colors.Muted,
			OnPress: func() { a.networkErr.Set("") }})
	}
	return core.View{TestID: testID, Style: core.Style{Padding: core.Symmetric(10, 12), Gap: 10, Direction: core.Row, AlignItems: core.AlignStart,
		CornerRadius: 10, BorderWidth: 1}, StyleFunc: func() core.Style {
		return core.Style{Background: tint, BorderColor: ui.Mix(tint, colors.Border(), 0.6)}
	}, Children: []core.Node{
		core.View{Style: core.Style{Padding: core.All(4), Margin: core.Edges{Top: 2}}, Children: []core.Node{ui.IconOf(icon, 16, fg)}},
		core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0), Gap: 2}, Children: []core.Node{
			core.Text{Content: title, Style: ui.TypeLabel, StyleFunc: ui.Colored(colors.Text)},
			core.Text{Content: detail, Style: ui.TypeCaption, StyleFunc: ui.Colored(fg)},
		}},
		core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 4}, Children: trailing},
	}}
}

// devicesHeader heads the device grid.
func (a *app) devicesHeader(cx *core.Scope) core.Node {
	colors := ui.ColorsOf(cx)
	l := locales.For(cx)
	return core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 10, Padding: core.Edges{Top: 4}}, Children: []core.Node{
		core.Text{Content: l.NearbySection(), Style: core.Style{FontSize: 17, LineHeight: 22, FontWeight: ui.Semibold}, StyleFunc: ui.Colored(colors.Text)},
		core.View{Style: core.Style{FlexGrow: 1}},
		core.Text{ContentFunc: func() string { return l.OnlineCount(onlineCount(a)) }, MaxLines: 1, Style: ui.TypeCaption,
			StyleFunc: ui.Colored(colors.Muted)},
	}}
}

// onlineCount is how many discovered devices answered.
func onlineCount(a *app) int {
	n := 0
	for _, d := range a.devices.Get() {
		if !d.LastSeen.IsZero() {
			n++
		}
	}
	return n
}

// deviceGrid lays the devices out, or shows what to do when there are
// none. Cards are keyed by fingerprint, so a device that answers again
// rebuilds only its own card, and only when what it shows changed.
func (a *app) deviceGrid(cx *core.Scope) core.Node {
	grid := core.View{Style: core.Style{Display: core.DisplayGrid, GridTemplateColumns: "repeat(auto-fill, minmax(240px, 1fr))", Gap: 12},
		Children: []core.Node{core.Each(a.mergedDevices, func(d localsend.Device) string { return strings.ToUpper(d.Fingerprint) },
			func(cx *core.Scope, item func() localsend.Device, _ func() int) core.Node {
				device := core.NewMemoFunc(cx, item, sameDevice)
				return core.Dynamic(func(cx *core.Scope) core.Node { return a.deviceCard(cx, device.Get()) })
			})}}
	return core.Show{When: func() bool { return len(a.mergedDevices()) > 0 },
		Then: func(*core.Scope) core.Node { return grid },
		Else: func(cx *core.Scope) core.Node { return a.emptyNearby(cx) }}
}

// cardMotion eases a card's fill, edge, shadow and scale between states.
var cardMotion = core.Transition{Duration: 160 * time.Millisecond, Easing: core.EaseOut,
	Properties: core.TransitionBackground | core.TransitionBorder | core.TransitionShadow | core.TransitionTransform}

// deviceCard is one nearby device, pressed to send files to it.
func (a *app) deviceCard(cx *core.Scope, d localsend.Device) core.Node {
	colors := ui.ColorsOf(cx)
	l := locales.For(cx)
	device := d
	model := d.DeviceModel
	if model == "" {
		model = deviceTypeLabel(cx, d.DeviceType)
	}
	up := online(d)
	return core.View{OnPress: func(core.PressEvent) { a.openSend(device) }, Role: core.RoleButton, Label: device.Alias, TestID: "device-" + shortFingerprint(d.Fingerprint),
		Style: core.Style{Padding: core.All(14), Gap: 12, BorderWidth: 1, CornerRadius: 12, BorderCurve: core.BorderCurveContinuous, Transition: cardMotion},
		StyleFunc: func() core.Style {
			lift := ui.FinOf(cx).Raised().Shadows
			return core.Style{Background: colors.Surface(), BorderColor: colors.Border(), Shadows: lift,
				Hover: &core.Style{BorderColor: ui.Mix(colors.Border(), colors.Accent(), 0.5),
					Shadows: []core.BoxShadow{ui.Shadow(0, 6, 18, -4, ui.Black(0.14))}},
				Active: &core.Style{Background: colors.Hover().Over(colors.Surface()), Transform: core.Transform{Scale: 0.985}, Shadows: lift}}
		}, Children: []core.Node{
			core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 10}, Children: []core.Node{
				deviceTile(cx, deviceIcon(d), up, 40),
				core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0), Gap: 1}, Children: []core.Node{
					core.Text{Content: device.Alias, MaxLines: 1, Style: ui.TypeLabel, StyleFunc: ui.Colored(colors.Text)},
					core.Text{Content: model + " · " + deviceTypeLabel(cx, d.DeviceType), MaxLines: 1, Style: core.Style{FontSize: 11},
						StyleFunc: ui.Colored(colors.Muted)},
				}},
				ui.IconButton{Icon: lucide.Star, Tint: func() core.Color {
					if a.favoriteTracked(device.Fingerprint) {
						return colors.Accent()
					}
					return colors.Faint()
				}, Size: 26, IconSize: 15, LabelFunc: func() string {
					if a.favoriteTracked(device.Fingerprint) {
						return l.Unfavorite()
					}
					return l.Favorite()
				}, OnPress: func() { a.toggleFavorite(device) }, TestID: "favorite-device"},
			}},
			core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 6}, Children: []core.Node{
				core.View{Style: core.Style{Width: core.Pt(7), Height: core.Pt(7), CornerRadius: 4}, StyleFunc: func() core.Style {
					if up {
						return core.Style{Background: colors.Up()}
					}
					return core.Style{Background: colors.Faint()}
				}},
				core.Text{ContentFunc: func() string {
					if a.favorite(device.Fingerprint) && !up {
						return l.Offline()
					}
					return l.Online()
				}, Style: core.Style{FontSize: 11}, StyleFunc: ui.Colored(colors.Muted)},
				ui.Badge{Text: strings.ToUpper(string(d.Protocol)), VariantFunc: func() ui.BadgeVariant {
					if d.Protocol == localsend.ProtocolHTTP {
						return ui.BadgeWarn
					}
					return ui.BadgeNeutral
				}},
				core.View{Style: core.Style{FlexGrow: 1}},
				ui.IconOf(lucide.ChevronRight, 15, colors.Faint),
			}},
		}}
}

// statusDot is a status color's dot, ringed by a slow ping while live.
func statusDot(cx *core.Scope, color func() core.Color, live func() bool) core.Node {
	dot := core.Style{Width: core.Pt(8), Height: core.Pt(8), CornerRadius: 4}
	ring := core.View{Style: dot.Merge(core.Style{Position: core.Absolute, Top: core.Pt(0), Left: core.Pt(0)}), StyleFunc: func() core.Style {
		if !live() || ui.ReduceMotion(cx) {
			return core.Style{Display: core.DisplayNone}
		}
		return core.Style{Background: color(), Animation: core.KeyframeAnimation{Keyframes: core.KeyframesPing, Duration: 2 * time.Second}}
	}}
	return core.View{Style: dot, StyleFunc: ui.Filled(color), Children: []core.Node{ring}}
}

// emptyNearby is what shows before a device answers.
func (a *app) emptyNearby(cx *core.Scope) core.Node {
	colors := ui.ColorsOf(cx)
	l := locales.For(cx)
	return core.View{TestID: "nearby-empty", Style: core.Style{Padding: core.Symmetric(34, 20), Gap: 12, AlignItems: core.AlignCenter,
		CornerRadius: 14, BorderWidth: 1}, StyleFunc: func() core.Style {
		return core.Style{Background: colors.Inset(), BorderColor: colors.Border()}
	}, Children: []core.Node{
		deviceTile(cx, lucide.RadioTower, true, 52),
		core.Text{Content: l.NearbyEmpty(), Style: ui.TypeTitle.Merge(core.Style{TextAlign: core.TextAlignCenter}), StyleFunc: ui.Colored(colors.Text)},
		core.Text{Content: l.NearbyEmptyDetail(), MaxLines: 3, Style: ui.TypeBody.Merge(core.Style{TextAlign: core.TextAlignCenter, MaxWidth: core.Pt(360)}),
			StyleFunc: ui.Colored(colors.Muted)},
		ui.Button{Text: l.Refresh(), Icon: lucide.RefreshCw, Variant: ui.ButtonSecondary, OnPress: a.refresh, TestID: "empty-refresh"},
	}}
}
