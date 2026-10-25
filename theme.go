package main

import(
	"image/color"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)


type vaulticTheme struct{
	fyne.Theme
}

func newVaulticTheme() fyne.Theme{
	return &vaulticTheme{theme.DefaultTheme()}
}

func rgba(r,g,b,a uint8) color.Color{
	return color.NRGBA{R:r,G:g,B:b,A:a}
}

func (t *vaulticTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return rgba(0x12, 0x12, 0x14, 0xff)
	case theme.ColorNameForeground:
		return rgba(0xe6, 0xe6, 0xe9, 0xff)
	case theme.ColorNamePlaceHolder, theme.ColorNameDisabled:
		return rgba(0x6b, 0x6b, 0x73, 0xff)
	case theme.ColorNamePrimary: // your single accent color
		return rgba(0x7c, 0x8c, 0xff, 0xff)
	case theme.ColorNameHover:
		return rgba(0xff, 0xff, 0xff, 0x0d)
	case theme.ColorNameSelection:
		return rgba(0x7c, 0x8c, 0xff, 0x26)
	case theme.ColorNameInputBackground:
		return rgba(0x1b, 0x1b, 0x1f, 0xff)
	case theme.ColorNameInputBorder, theme.ColorNameSeparator:
		return rgba(0x24, 0x24, 0x29, 0xff)
	case theme.ColorNameButton:
		return color.Transparent // flat buttons everywhere
	}
	// Anything we didn't override: default theme, forced to dark
	return t.Theme.Color(name, theme.VariantDark)
}

func (t *vaulticTheme) Size(name fyne.ThemeSizeName) float32{
	switch name{
	case theme.SizeNamePadding:
		return 6
	case theme.SizeNameInnerPadding:
		return 8
	case theme.SizeNameText:
		return 14
	case theme.SizeNameSeparatorThickness:
		return 1
	case theme.SizeNameInputRadius,theme.SizeNameSelectionRadius:
		return 8
	}

	return t.Theme.Size(name)
}