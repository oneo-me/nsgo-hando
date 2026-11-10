package main

import (
	"strings"

	"github.com/oneo-me/nsgo/core"
	"github.com/oneo-me/nsgo/icons/lucide"
	"github.com/oneo-me/nsgo/ui"

	"github.com/oneo-me/nsgo-localsend/internal/localsend"
	"github.com/oneo-me/nsgo-localsend/locales"
)

// pagePadding is the page's gutter and spacing.
var pagePadding = core.Edges{Top: 20, Right: 20, Bottom: 28, Left: 20}

// view is the app's shell under the window.
func (a *app) view(cx *core.Scope, wide bool) core.Node {
	pages := []core.Node{a.nearbyPage(cx), a.transfersPage(cx), a.settingsPage(cx)}
	var shell core.Node
	if wide {
		shell = a.desktopShell(cx, pages)
	} else {
		shell = a.mobileShell(cx, pages)
	}
	colors := ui.ColorsOf(cx)
	return core.View{Style: ui.FillRest, StyleFunc: func() core.Style {
		return core.Style{Background: colors.Background(), Color: colors.Text()}
	}, Children: []core.Node{shell, a.overlays(cx)}}
}

// desktopShell is the sidebar and page layout of a wide window.
func (a *app) desktopShell(cx *core.Scope, pages []core.Node) core.Node {
	l := locales.For(cx)
	return ui.SidebarShell{
		Sidebar:       []core.Node{ui.Sidebar{SectionsFunc: func() []ui.SidebarSection { return a.sidebarSections(cx) }, Selected: func() string { return a.page.Get() }, OnSelect: a.selectSidebar, TestID: "sidebar"}},
		SidebarFooter: []core.Node{a.sidebarFooter(cx)},
		Header:        []core.Node{a.headerTitle(cx)},
		Actions:       a.headerActions(cx),
		Content:       []core.Node{a.pageStack(pages)},
		Open:          a.sidebar.Get,
		OnOpenChange:  a.sidebar.Set,
		ToggleLabel:   l.ToggleSidebar(),
		Width:         a.sideWidth.Get,
		OnWidthChange: a.sideWidth.Set,
		DefaultWidth:  236,
		MinWidth:      200,
		MaxWidth:      320,
		ResizeLabel:   l.ResizeSidebar(),
		TestID:        "sidebar-shell",
	}
}

// mobileShell is the tab layout of a narrow window.
func (a *app) mobileShell(cx *core.Scope, pages []core.Node) core.Node {
	l := locales.For(cx)
	tabs := []ui.ShellTab{
		{Text: l.Nearby(), Icon: lucide.RadioTower, SystemIcon: "dot.radiowaves.left.and.right", Content: func(cx *core.Scope) core.Node {
			return a.mobilePage(cx, "nearby", pages[0])
		}, TestID: "tab-nearby"},
		{Text: l.Transfers(), Icon: lucide.ArrowUpDown, SystemIcon: "arrow.up.arrow.down", Content: func(cx *core.Scope) core.Node {
			return a.mobilePage(cx, "transfers", pages[1])
		}, TestID: "tab-transfers"},
		{Text: l.Settings(), Icon: lucide.Settings, SystemIcon: "gearshape", Content: func(cx *core.Scope) core.Node {
			return a.mobilePage(cx, "settings", pages[2])
		}, TestID: "tab-settings"},
	}
	// Android's Material bar draws its own lavender surface, which does
	// not follow the app's theme; the drawn bar keeps this app's look.
	if cx.Env().OS() == core.OSAndroid {
		for i := range tabs {
			tabs[i].SystemIcon = ""
		}
	}
	return ui.TabShell{Tabs: tabs, Selected: a.pageIndex, OnChange: a.setPageIndex, Label: l.AppName(), TestID: "tabs"}
}

// mobilePage is a page with its own header under the tab shell; on iOS
// its pageScroll leads with the title instead.
func (a *app) mobilePage(cx *core.Scope, page string, body core.Node) core.Node {
	if a.largeTitle(cx) {
		return core.View{Style: ui.FillRest, Children: []core.Node{body}, TestID: "page-" + page}
	}
	return ui.PlainShell{Title: a.pageTitle(cx), Actions: a.headerActions(cx), Content: []core.Node{body}, TestID: "page-" + page}
}

// largeTitle reports whether pages show iOS's large title in a row with
// their actions, at the top of what scrolls.
func (a *app) largeTitle(cx *core.Scope) bool {
	return cx.Env().OS() == core.OSIOS && !a.wide.GetUntracked()
}

// pageScroll is page's scroll view of content. With largeTitle the
// page's title and actions lead it, scrolling away together, and the
// content blurs under the status bar.
func (a *app) pageScroll(cx *core.Scope, page, testID string, content core.Node) core.Node {
	if !a.largeTitle(cx) {
		return core.Scroll{Style: ui.FillRest, TestID: testID, Children: []core.Node{content}}
	}
	scroll := &core.Ref{}
	bar := core.View{EdgeBar: core.EdgeBar{Scroll: scroll, Side: core.SideTop}, PointerEvents: core.PointerEventsNone,
		Style: core.Style{Position: core.Absolute, Top: core.Pt(0), Right: core.Pt(0), Left: core.Pt(0)},
		// The blur runs on past the bar; keep it off the title at rest.
		StyleFunc: func() core.Style { return core.Style{Height: core.Pt(max(0, core.Insets(cx).Top-24))} }}
	return core.View{Style: ui.FillRest, Children: []core.Node{
		core.Scroll{Ref: scroll, Style: ui.FillRest, TestID: testID, Children: []core.Node{a.titleRow(cx, page), content}},
		bar,
	}}
}

// titleRow is page's large title with its actions, as glass buttons, at
// the end.
func (a *app) titleRow(cx *core.Scope, page string) core.Node {
	l := locales.For(cx)
	colors := ui.ColorsOf(cx)
	children := []core.Node{core.Text{ContentFunc: func() string { return a.titleOf(cx, page) }, Role: core.RoleHeader, MaxLines: 1,
		Style: core.Style{FontSize: 34, LineHeight: 41, FontWeight: ui.Bold, FlexGrow: 1, FlexShrink: 1}, StyleFunc: ui.Colored(colors.Text)}}
	switch page {
	case "nearby":
		children = append(children, glassButton(cx, func() *core.SVG {
			if a.refreshing.Get() {
				return lucide.LoaderCircle()
			}
			return lucide.RefreshCw()
		}, l.Refresh(), a.refreshing.Get, a.refresh, "refresh"))
	case "transfers":
		children = append(children, core.Show{When: func() bool { return len(a.finishedTransfers()) > 0 }, Then: func(cx *core.Scope) core.Node {
			return glassButton(cx, lucide.Trash, l.ClearHistory(), nil, a.clearHistory, "clear-history")
		}})
	}
	return core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 12}, StyleFunc: func() core.Style {
		return core.Style{Padding: core.Edges{Top: core.Insets(cx).Top + 8, Right: pagePadding.Right, Left: pagePadding.Left}}
	}, Children: children}
}

// glassButton is a round button on Liquid Glass, as iOS's bar buttons.
func glassButton(cx *core.Scope, icon func() *core.SVG, label string, disabled func() bool, press func(), testID string) core.Node {
	colors := ui.ColorsOf(cx)
	if disabled == nil {
		disabled = func() bool { return false }
	}
	return ui.Press{OnPress: press, Disabled: disabled, Label: label, TestID: testID, Scale: 0.92,
		Style: core.Style{Width: core.Pt(44), Height: core.Pt(44), CornerRadius: 22, FlexShrink: 0, AlignItems: core.AlignCenter,
			JustifyContent: core.JustifyCenter, GlassEffect: core.GlassRegular, BackdropFilters: []core.Filter{core.Blur(24)}},
		StyleFunc: func() core.Style { return core.Style{Background: ui.Alpha(colors.Surface(), 0.55)} },
		Children:  []core.Node{ui.IconOf(icon, 20, colors.Text)}}
}

// pageStack keeps every desktop page mounted, showing the chosen one.
func (a *app) pageStack(pages []core.Node) core.Node {
	children := make([]core.Node, len(pages))
	for i := range pages {
		index := i
		children[i] = core.View{Children: []core.Node{pages[i]}, StyleFunc: func() core.Style {
			if a.pageIndex() != index {
				return core.Style{Display: core.DisplayNone}
			}
			return ui.FillRest
		}, TestID: "page-" + []string{"nearby", "transfers", "settings"}[index]}
	}
	return core.View{Style: ui.FillRest, Children: children}
}

// titleOf names page.
func (a *app) titleOf(cx *core.Scope, page string) string {
	l := locales.For(cx)
	switch page {
	case "transfers":
		return l.Transfers()
	case "settings":
		return l.Settings()
	}
	return l.Nearby()
}

// headerTitle is the page's name in the shell's header.
func (a *app) headerTitle(cx *core.Scope) core.Node {
	colors := ui.ColorsOf(cx)
	return core.Text{ContentFunc: func() string { return a.pageTitle(cx) }, Role: core.RoleHeader, MaxLines: 1,
		Style: ui.TypeTitle.Merge(core.Style{FlexShrink: 1}), StyleFunc: ui.Colored(colors.Text)}
}

// headerActions are the chosen page's buttons.
func (a *app) headerActions(cx *core.Scope) []core.Node {
	l := locales.For(cx)
	clear := core.Show{When: func() bool {
		if a.page.Get() != "transfers" {
			return false
		}
		return len(a.finishedTransfers()) > 0
	}, Then: func(cx *core.Scope) core.Node {
		colors := ui.ColorsOf(cx)
		return ui.IconButton{Icon: lucide.Trash, Label: l.ClearHistory(), Tint: colors.Muted, OnPress: a.clearHistory, TestID: "clear-history"}
	}}
	return []core.Node{
		core.Show{When: func() bool { return a.page.Get() == "nearby" }, Then: func(cx *core.Scope) core.Node {
			return core.Dynamic(func(cx *core.Scope) core.Node {
				if a.refreshing.Get() {
					return core.View{Style: core.Style{Width: core.Pt(28), Height: core.Pt(28), AlignItems: core.AlignCenter,
						JustifyContent: core.JustifyCenter}, Children: []core.Node{ui.Spinner{Size: 16, Label: l.Refresh(), TestID: "refreshing"}}}
				}
				return a.refreshButton(cx)
			})
		}},
		clear,
	}
}

// refreshButton searches the network.
func (a *app) refreshButton(cx *core.Scope) core.Node {
	l := locales.For(cx)
	colors := ui.ColorsOf(cx)
	return ui.IconButton{Icon: lucide.RefreshCw, IconFunc: func() *core.SVG {
		if a.refreshing.Get() {
			return lucide.LoaderCircle()
		}
		return lucide.RefreshCw()
	}, Label: l.Refresh(), Tint: colors.Muted, Disabled: a.refreshing.Get, OnPress: a.refresh, TestID: "refresh"}
}

// sidebarSections are the sidebar's destinations.
func (a *app) sidebarSections(cx *core.Scope) []ui.SidebarSection {
	l := locales.For(cx)
	var favorites []ui.SidebarItem
	for _, d := range a.mergedDevices() {
		if !d.Favorite {
			continue
		}
		device := d
		favorites = append(favorites, ui.SidebarItem{ID: "fav:" + device.Fingerprint, Text: device.Alias,
			Icon: deviceIcon(device), TestID: "favorite-" + shortFingerprint(device.Fingerprint)})
		_ = favorites
	}
	var sections []ui.SidebarSection
	if len(favorites) > 0 {
		sections = append(sections, ui.SidebarSection{Title: l.Favorites(), Items: favorites, TestID: "sidebar-favorites"})
	}
	online := 0
	for _, d := range a.devices.Get() {
		if !d.LastSeen.IsZero() {
			online++
		}
	}
	nav := func(id string, text string, icon func() *core.SVG, testID string) ui.SidebarItem {
		return ui.SidebarItem{ID: id, Text: text, Icon: icon, TestID: testID}
	}
	sections = append(sections,
		ui.SidebarSection{Title: l.NearbySection(), Trailing: ui.Badge{TextFunc: func() string { return countLabel(cx, online) }},
			Items: []ui.SidebarItem{nav("nearby", l.Nearby(), lucide.RadioTower, "nav-nearby")}, TestID: "sidebar-devices"},
		ui.SidebarSection{Title: l.TransfersSection(),
			Items: []ui.SidebarItem{nav("transfers", l.Transfers(), lucide.ArrowUpDown, "nav-transfers")}, TestID: "sidebar-activity"},
		ui.SidebarSection{Title: l.AppSection(),
			Items: []ui.SidebarItem{nav("settings", l.Settings(), lucide.Settings, "nav-settings")}, TestID: "sidebar-app"},
	)
	return sections
}

// selectSidebar opens a page or a favorite.
func (a *app) selectSidebar(id string) {
	if strings.HasPrefix(id, "fav:") {
		fingerprint := strings.TrimPrefix(id, "fav:")
		for _, d := range a.mergedDevices() {
			if strings.EqualFold(d.Fingerprint, fingerprint) {
				a.openSend(d)
				return
			}
		}
		return
	}
	a.page.Set(id)
}

// sidebarFooter shows this device above the sidebar's bottom.
func (a *app) sidebarFooter(cx *core.Scope) core.Node {
	colors := ui.ColorsOf(cx)
	status := func() string {
		switch {
		case a.offline:
			return locales.ServerStopped(cx)
		case a.serverErr.Get() != "":
			return locales.ServerErrorTitle(cx)
		case !a.nodeReady.Get():
			return locales.Searching(cx)
		}
		host := "localhost"
		if addr := a.hosts.Get(); len(addr) > 0 {
			host = addr[0]
		}
		return host + ":" + itoa(a.selfDevice().Port)
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
	return core.View{OnPress: func(core.PressEvent) { a.page.Set("settings") }, Role: core.RoleButton, Label: locales.Settings(cx), TestID: "sidebar-device",
		Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 10, FlexShrink: 0, Width: core.Pct(100), MinWidth: core.Pt(0),
			Padding: core.Symmetric(8, 10), CornerRadius: 10, BorderWidth: 1, BorderCurve: core.BorderCurveContinuous, Transition: cardMotion},
		StyleFunc: func() core.Style {
			return core.Style{Background: colors.Surface(), BorderColor: colors.Border(), Shadows: ui.FinOf(cx).Raised().Shadows,
				Hover:  &core.Style{BorderColor: ui.Mix(colors.Border(), colors.Accent(), 0.5)},
				Active: &core.Style{Background: colors.Hover().Over(colors.Surface())}}
		}, Children: []core.Node{
			deviceTile(cx, deviceIconForSelf(a.dtype), true, 34),
			core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0), Gap: 3}, Children: []core.Node{
				core.Text{ContentFunc: func() string { return a.settings.Get().Alias }, MaxLines: 1,
					Style: ui.TypeLabel.Merge(core.Style{FlexShrink: 1, MinWidth: core.Pt(0)}), StyleFunc: ui.Colored(colors.Text)},
				core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 6, MinWidth: core.Pt(0)}, Children: []core.Node{
					core.View{Style: core.Style{Width: core.Pt(6), Height: core.Pt(6), CornerRadius: 3, FlexShrink: 0}, StyleFunc: ui.Filled(statusColor)},
					core.Text{ContentFunc: status, MaxLines: 1, Style: core.Style{FontSize: 11, LineHeight: 14, FontVariant: core.FontTabularNums,
						FlexShrink: 1, MinWidth: core.Pt(0)}, StyleFunc: ui.Colored(colors.Muted)},
				}},
			}},
			ui.IconOf(lucide.ChevronRight, 14, colors.Faint),
		}}
}

// overlays are the dialogs and toasts above the shell.
func (a *app) overlays(cx *core.Scope) core.Node {
	return core.Group{
		a.sendDialog(cx),
		a.incomingDialog(cx),
		a.pinDialog(cx),
		a.toastView(cx),
	}
}

// toastView places the toasts over the window.
func (a *app) toastView(cx *core.Scope) core.Node {
	return core.Dynamic(func(cx *core.Scope) core.Node {
		wide := a.wide.Get()
		placement, inset := ui.ToastBottomEnd, core.Points(0)
		if !wide {
			placement, inset = ui.ToastBottom, 64
		}
		return ui.Toasts{Toaster: a.toaster, Placement: placement, Inset: inset, CloseLabel: locales.Close(cx), TestID: "toasts"}
	})
}

// countLabel says how many nearby devices answered.
func countLabel(cx *core.Scope, n int) string {
	if n == 0 {
		return "0"
	}
	return itoa(n)
}

// itoa formats a nonnegative integer.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits [20]byte
	i := len(digits)
	for n > 0 {
		i--
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	return string(digits[i:])
}

// deviceIconForSelf picks this device's glyph.
func deviceIconForSelf(t localsend.DeviceType) func() *core.SVG {
	switch t {
	case localsend.DeviceMobile:
		return lucide.Smartphone
	case localsend.DeviceWeb:
		return lucide.Globe
	case localsend.DeviceServer, localsend.DeviceHeadless:
		return lucide.Server
	}
	return lucide.Laptop
}

// deviceTile draws a rounded tile with a device glyph.
func deviceTile(cx *core.Scope, icon func() *core.SVG, accent bool, size core.Points) core.Node {
	if size == 0 {
		size = 38
	}
	return core.View{Style: core.Style{Width: core.Pt(size), Height: core.Pt(size), FlexShrink: 0, CornerRadius: size * 0.28,
		AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter}, StyleFunc: func() core.Style {
		colors := ui.ColorsOf(cx)
		if accent {
			return core.Style{Background: ui.TintsOf(cx).Accent()}
		}
		return core.Style{Background: colors.Inset()}
	}, Children: []core.Node{ui.IconOf(icon, size*0.5, func() core.Color {
		colors := ui.ColorsOf(cx)
		if accent {
			return colors.Accent()
		}
		return colors.Muted()
	})}}
}

// bottomGap keeps a page's last content above a floating tab bar.
func bottomGap(cx *core.Scope) core.Node {
	return core.View{StyleFunc: func() core.Style { return core.Style{Height: core.Pt(ui.TabBarInset(cx) + 8)} }}
}
