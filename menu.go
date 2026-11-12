package main

import (
	"slices"

	"github.com/oneo-me/nsgo/core"
	"github.com/oneo-me/nsgo/desktop"
	"github.com/oneo-me/nsgo/updater"

	"github.com/oneo-me/nsgo-localsend/locales"
)

// installMenu gives macOS its menu bar: the app with its update check,
// editing, the pages and the window. Windows and Linux keep their windows
// bare.
func (a *app) installMenu(cx *core.Scope) {
	if cx.Env().OS() != core.OSMacOS || !desktop.IsAvailable(cx) {
		return
	}
	cx.Effect(func() { desktop.SetApplicationMenu(cx, a.menu(cx)) })
}

// menu is the menu bar, rebuilt as the page, sidebar and language change.
func (a *app) menu(cx *core.Scope) []desktop.MenuItem {
	l := locales.For(cx)
	page := a.page.Get()
	wide := a.wide.Get()
	pageItem := func(id, label, key string) desktop.MenuItem {
		return desktop.MenuItem{Label: label, Shortcut: key, Checkable: true, Checked: page == id, When: core.MenuNotEditing,
			OnSelect: func() { a.page.Set(id) }}
	}
	appItems := []desktop.MenuItem{
		{Role: desktop.MenuAbout}, {Separator: true},
		{Label: l.MenuSettings(), Shortcut: "mod+,", OnSelect: func() { a.page.Set("settings") }}, {Separator: true},
		{Role: desktop.MenuHide}, {Role: desktop.MenuHideOthers}, {Role: desktop.MenuShowAll}, {Separator: true},
		{Role: desktop.MenuQuit},
	}
	if updatesShown(cx) {
		appItems = slices.Insert(appItems, 1, desktop.MenuItem{Label: l.CheckForUpdates(), OnSelect: func() { updater.Check(cx) }})
	}
	return []desktop.MenuItem{
		{Label: l.AppName(), Items: appItems},
		{Label: l.MenuEdit(), Items: []desktop.MenuItem{
			{Role: desktop.MenuUndo}, {Role: desktop.MenuRedo}, {Separator: true},
			{Role: desktop.MenuCut}, {Role: desktop.MenuCopy}, {Role: desktop.MenuPaste}, {Role: desktop.MenuSelectAll},
		}},
		{Label: l.MenuView(), Items: []desktop.MenuItem{
			{Label: l.MenuShowSidebar(), Shortcut: "ctrl+mod+s", Checkable: true, Checked: wide && a.sidebar.Get(), Disabled: !wide,
				OnSelect: func() { a.sidebar.Set(!a.sidebar.GetUntracked()) }},
			{Separator: true},
			pageItem("nearby", l.Nearby(), "mod+1"),
			pageItem("transfers", l.Transfers(), "mod+2"),
			pageItem("settings", l.Settings(), "mod+3"),
			{Separator: true},
			{Label: l.Refresh(), Shortcut: "mod+r", Disabled: a.offline || a.refreshing.Get() || !a.nodeReady.Get(), OnSelect: a.refresh},
		}},
		{Label: l.MenuWindow(), Items: []desktop.MenuItem{
			{Role: desktop.MenuMinimize}, {Role: desktop.MenuZoom}, {Role: desktop.MenuToggleFullScreen}, {Separator: true},
			{Role: desktop.MenuClose},
		}},
	}
}
