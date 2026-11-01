package main

import(
	"image/color"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)



type vaulticTheme struct {
	fyne.Theme
}
 
func newVaulticTheme() fyne.Theme {
	t := &vaulticTheme{}
	t.Theme = theme.DefaultTheme()
	return t
}
func rgba(r, g, b, a uint8) color.Color {
	return color.NRGBA{R: r, G: g, B: b, A: a}
}
 
func (t *vaulticTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if name == theme.ColorNameBackground {
		return rgba(0x12, 0x12, 0x14, 0xff)
	}
	if name == theme.ColorNameForeground {
		return rgba(0xe6, 0xe6, 0xe9, 0xff)
	}
	if name == theme.ColorNamePlaceHolder || name == theme.ColorNameDisabled {
		return rgba(0x6b, 0x6b, 0x73, 0xff)
	}
	if name == theme.ColorNamePrimary {
		return rgba(0x7c, 0x8c, 0xff, 0xff)
	}
	if name == theme.ColorNameHover {
		return rgba(0xff, 0xff, 0xff, 0x0d)
	}
	if name == theme.ColorNameSelection {
		return rgba(0x7c, 0x8c, 0xff, 0x26)
	}
	if name == theme.ColorNameInputBackground {
		return rgba(0x1b, 0x1b, 0x1f, 0xff)
	}
	if name == theme.ColorNameInputBorder || name == theme.ColorNameSeparator {
		return rgba(0x24, 0x24, 0x29, 0xff)
	}
	if name == theme.ColorNameButton {
		return color.Transparent
	}
 
	return t.Theme.Color(name, theme.VariantDark)
}
 
func (t *vaulticTheme) Size(name fyne.ThemeSizeName) float32 {
	if name == theme.SizeNamePadding {
		return 6
	}
	if name == theme.SizeNameInnerPadding {
		return 8
	}
	if name == theme.SizeNameText {
		return 14
	}
	if name == theme.SizeNameSeparatorThickness {
		return 1
	}
	if name == theme.SizeNameInputRadius || name == theme.SizeNameSelectionRadius {
		return 8
	}
 
	return t.Theme.Size(name)
}