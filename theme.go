package main

import "github.com/oneo-me/nsgo/core"

// appTheme is LocalSend's look: a calm green page, white cards and a
// teal accent, in light and dark.
func appTheme() *core.Theme {
	t := core.DefaultTheme()
	t.Radius = 10
	light, dark := &t.Light, &t.Dark

	ink := core.RGB(0x14201b)
	light.Background = core.RGB(0xf5f7f6)
	light.Surface = core.RGB(0xffffff)
	light.SurfaceHover = ink.Faded(0.05)
	light.SurfacePressed = ink.Faded(0.09)
	light.Border = ink.Faded(0.13)
	light.Text = ink
	light.TextMuted = core.RGB(0x62736b)
	light.Accent = core.RGB(0x0e9f7c)
	light.AccentHover = core.RGB(0x0b8a6b)
	light.AccentPressed = core.RGB(0x09765b)
	light.AccentText = core.RGB(0xffffff)
	light.Danger = core.RGB(0xc4434b)
	light.Warning = core.RGB(0x9a5a00)
	light.Success = core.RGB(0x0e9f7c)
	light.Selection = core.RGBA(0x0e9f7c, 0.22)
	light.Focus = core.RGBA(0x0e9f7c, 0.55)
	light.Scrollbar = ink.Faded(0.28)
	light.SetColor("inset", core.RGB(0xedf2ef))
	light.SetColor("raised", core.RGB(0xffffff))
	light.SetColor("field", core.RGB(0xffffff))
	light.SetColor("fieldBorder", ink.Faded(0.16))
	light.SetColor("line", ink.Faded(0.10))
	light.SetColor("faint", ink.Faded(0.40))
	light.SetColor("accentFill", light.Accent)
	light.SetColor("light", ink)
	light.SetColor("onLight", core.RGB(0xffffff))
	light.SetColor("track", ink.Faded(0.22))
	light.SetColor("knob", core.RGB(0xffffff))
	light.SetColor("code", core.RGB(0x24352d))
	light.SetColor("info", core.RGB(0x0b7a9b))
	light.SetColor("down", light.Danger)

	mint := core.RGB(0xe8f0ec)
	dark.Background = core.RGB(0x0f1512)
	dark.Surface = core.RGB(0x17201c)
	dark.SurfaceHover = mint.Faded(0.06)
	dark.SurfacePressed = mint.Faded(0.10)
	dark.Border = mint.Faded(0.12)
	dark.Text = mint
	dark.TextMuted = mint.Faded(0.58)
	dark.Accent = core.RGB(0x2dd4a7)
	dark.AccentHover = core.RGB(0x4ce0b4)
	dark.AccentPressed = core.RGB(0x28c296)
	dark.AccentText = core.RGB(0x06231a)
	dark.Danger = core.RGB(0xf07e86)
	dark.Warning = core.RGB(0xe6b567)
	dark.Success = core.RGB(0x2dd4a7)
	dark.Selection = core.RGBA(0x2dd4a7, 0.24)
	dark.Focus = core.RGBA(0x2dd4a7, 0.80)
	dark.Scrollbar = mint.Faded(0.32)
	dark.SetColor("inset", core.RGB(0x1b2621))
	dark.SetColor("raised", core.RGB(0x223029))
	dark.SetColor("field", core.RGB(0x141d19))
	dark.SetColor("fieldBorder", mint.Faded(0.16))
	dark.SetColor("line", mint.Faded(0.08))
	dark.SetColor("faint", mint.Faded(0.40))
	dark.SetColor("accentFill", dark.Accent)
	dark.SetColor("light", mint)
	dark.SetColor("onLight", dark.Background)
	dark.SetColor("track", mint.Faded(0.26))
	dark.SetColor("knob", core.RGB(0xf5f5f5))
	dark.SetColor("code", core.RGB(0xd9e8e0))
	dark.SetColor("info", core.RGB(0x64d2ff))
	dark.SetColor("down", dark.Danger)
	return t
}
