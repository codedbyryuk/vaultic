package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type TabChip struct {
	widget.BaseWidget
	label     *widget.Label
	closeBtn *widget.Button
	bg        *canvas.Rectangle
	underline *canvas.Rectangle
	active    bool
	OnTapped  func()
	OnClose func()
}

func NewTabChip(text string, onTap,onClose func()) *TabChip {
	t := &TabChip{
		label:     widget.NewLabel(""),
		bg:        canvas.NewRectangle(color.Transparent),
		underline: canvas.NewRectangle(color.Transparent),
		OnTapped:  onTap,
		OnClose: onClose,
	}
	t.closeBtn = widget.NewButtonWithIcon("",theme.ContentClearIcon(),func() {
		if t.OnClose != nil{
			t.OnClose()
		}
	})
	t.closeBtn.Importance = widget.LowImportance
	t.bg.CornerRadius = 8
	t.underline.CornerRadius = 1
	t.underline.SetMinSize(fyne.NewSize(0, 2))
	t.label.Alignment = fyne.TextAlignLeading

	t.SetText(text)
	t.SetActive(false)
	t.ExtendBaseWidget(t)
	return t
}

func (t *TabChip) CreateRenderer() fyne.WidgetRenderer {
	inset := container.New(layout.NewCustomPaddedLayout(-12, 2, 12, 12), t.underline)
	body := container.NewBorder(nil, inset, nil, t.closeBtn, container.NewPadded(t.label))
	return widget.NewSimpleRenderer(container.NewStack(t.bg, body))
}

func (t *TabChip) SetText(text string) {
	if r := []rune(text); len(r) > 18 {
		text = string(r[:17]) + "…"}
	t.label.SetText(text)
}

func (t *TabChip) SetActive(active bool) {
	t.active = active
	if active {
		t.underline.FillColor = theme.Color(theme.ColorNamePrimary)
		t.label.Importance = widget.MediumImportance
		t.label.TextStyle = fyne.TextStyle{Bold: true}
	} else {
		t.underline.FillColor = color.Transparent
		t.label.Importance = widget.LowImportance
		t.label.TextStyle = fyne.TextStyle{}
	}
	t.underline.Refresh()
	t.label.Refresh()
}




func (t *TabChip) Tapped(*fyne.PointEvent) {
	if t.OnTapped != nil {
		t.OnTapped()
	}
}
func (t *TabChip) MouseIn(*desktop.MouseEvent) {
	if !t.active {
		t.bg.FillColor = theme.Color(theme.ColorNameHover)
		t.bg.Refresh()
	}
}
func (t *TabChip) MouseMoved(*desktop.MouseEvent) {}

func (t *TabChip) MouseOut() {
	t.bg.FillColor = color.Transparent
	t.bg.Refresh()
}

func (t *TabChip) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

func (t *TabChip) SetClosable(c bool){
	if c{
		t.closeBtn.Show()
	}else{
		t.closeBtn.Hide()
	}
	t.Refresh()
}