package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type FileTab struct {
	Path      string
	History   []string
	Entries   []os.DirEntry
	List      *widget.List
	Page      *fyne.Container
	LoadDir   func(string)
	TabButton *TabChip
}

type FileRow struct {
	widget.BaseWidget
	icon     *widget.Icon
	name     *widget.Label
	size     *widget.Label
	Entry    os.DirEntry
	Path     string
	Window   fyne.Window
	OnTapped func()
	OnRename func()
}

func (r *FileRow) Tapped(_ *fyne.PointEvent) {
	if r.OnTapped != nil {
		r.OnTapped()
	}
}

func (r *FileRow) TappedSecondary(e *fyne.PointEvent) {
	menu := fyne.NewMenu("",
		fyne.NewMenuItem("Copy path", func() {
			r.Window.Clipboard().SetContent(r.Path)
		}),
		fyne.NewMenuItem("Rename", func() {
			if r.OnRename != nil {
				r.OnRename()
			}
		}),
	)

	popup := widget.NewPopUpMenu(menu, r.Window.Canvas())
	popup.ShowAtPosition(e.AbsolutePosition)

}

func NewFileRow(win fyne.Window) *FileRow {
	r := &FileRow{
		Window: win,
		icon:   widget.NewIcon(theme.FileIcon()),
		name:   widget.NewLabel(""),
		size:   widget.NewLabel(""),
	}
	r.name.Truncation = fyne.TextTruncateEllipsis
	r.size.Importance = widget.LowImportance 
	r.size.Alignment = fyne.TextAlignTrailing
	r.ExtendBaseWidget(r)
	return r
}

func (r *FileRow) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(
		container.NewBorder(nil, nil, r.icon, r.size, r.name),
	)
}

func (r *FileRow) SetEntry(entry os.DirEntry, dir string) {
	r.Entry = entry
	r.Path = filepath.Join(dir, entry.Name())
	r.name.SetText(entry.Name())

	if entry.IsDir() {
		r.icon.SetResource(theme.NewPrimaryThemedResource(theme.FolderIcon()))
		r.size.SetText("")
		return
	}
	r.icon.SetResource(theme.FileIcon())
	r.size.SetText("")
	if info, err := entry.Info(); err == nil {
		r.size.SetText(formatSize(info.Size()))
	}
}

func formatSize(size int64) string {
	if size >= 1024*1024*1024 {
		return fmt.Sprintf("%.1f GB", float64(size)/(1024*1024*1024))
	}
	if size >= 1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}
	if size >= 1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	} else {
		return fmt.Sprintf("%d B", size)
	}
}

func flatButton(icon fyne.Resource, label string, tapped func()) *widget.Button {
	b := widget.NewButtonWithIcon(label, icon, tapped)
	b.Importance = widget.LowImportance 
	b.Alignment = widget.ButtonAlignLeading
	return b
}

func panel(content fyne.CanvasObject, fill color.Color, radius float32) fyne.CanvasObject {
	bg := canvas.NewRectangle(fill)
	bg.CornerRadius = radius
	return container.NewStack(bg, container.NewPadded(content))
}

func sectionLabel(text string) fyne.CanvasObject {
	l := widget.NewLabel(strings.ToUpper(text))
	l.Importance = widget.LowImportance
	l.TextStyle = fyne.TextStyle{Bold: true}
	return l
}


func spacer(w, h float32) fyne.CanvasObject {
	r := canvas.NewRectangle(color.Transparent)
	r.SetMinSize(fyne.NewSize(w, h))
	return r
}

func tabTitle(p string) string {
	b := filepath.Base(p)
	if b == "." || b == string(filepath.Separator) {
		return p
	}
	return b
}

func main() {
	myApp := app.New()
	myApp.Settings().SetTheme(newVaulticTheme())
	window := myApp.NewWindow("Vaultic - File Manager")
	window.Resize(fyne.NewSize(1000, 650))

	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	tabs := []*FileTab{}
	activeTab := 0

	tabBar := container.NewHBox()
	content := container.NewMax()

	var addTab func(string)
	var switchTab func(int)

	addTab = func(path string) {
		entries, err := os.ReadDir(path)
		if err != nil {
			return
		}

		tab := &FileTab{
			Path:    path,
			Entries: entries,
		}

		pathEntry := widget.NewEntry()
		pathEntry.SetText(path)

		pathEntry.OnSubmitted = func(path string) {
			info, err := os.Stat(path)
			if err == nil && info.IsDir() {
				tab.History = append(tab.History, tab.Path)
				tab.LoadDir(path)
			}
		}

		renameEntry := func(entry os.DirEntry) {
			input := widget.NewEntry()
			input.SetText(entry.Name())

			dialog.ShowForm("Rename", "Rename", "Cancel",
				[]*widget.FormItem{widget.NewFormItem("New name", input)},
				func(ok bool) {
					if !ok {
						return
					}
					newName := strings.TrimSpace(input.Text)
					if newName == "" || newName == entry.Name() {
						return
					}
					if strings.ContainsAny(newName, `/\`) {
						dialog.ShowError(fmt.Errorf("name cant contain slashes"), window)
						return
					}
					oldPath := filepath.Join(tab.Path, entry.Name())
					newPath := filepath.Join(tab.Path, newName)

					if !strings.EqualFold(newName, entry.Name()) {
						if _, err := os.Stat(newPath); err == nil {
							dialog.ShowError(fmt.Errorf("%q already exists", newName), window)
							return
						}
					}
					if err := os.Rename(oldPath, newPath); err != nil {
						dialog.ShowError(err, window)
						return
					}
					tab.LoadDir(tab.Path)
				}, window)
		}

		tab.List = widget.NewList(
			func() int {
				return len(tab.Entries)
			},
			func() fyne.CanvasObject {
				return NewFileRow(window)
			},
			func(id widget.ListItemID, item fyne.CanvasObject) {
				row := item.(*FileRow)
				row.SetEntry(tab.Entries[id], tab.Path)
				entry := tab.Entries[id]
				row.OnTapped = func() { tab.List.Select(id) }
				row.OnRename = func() { renameEntry(entry) }
			},
		)

		tab.LoadDir = func(newPath string) {
			items, err := os.ReadDir(newPath)
			if err != nil {
				return
			}

			tab.Path = newPath
			tab.Entries = items

			sort.Slice(items, func(i, j int) bool {
				a := items[i]
				b := items[j]

				if a.IsDir() && !b.IsDir() {
					return true
				}
				if !a.IsDir() && b.IsDir() {
					return false
				}

				return strings.ToLower(a.Name()) < strings.ToLower(b.Name())
			})

			tab.TabButton.SetText(tabTitle(newPath))
			pathEntry.SetText(newPath)
			tab.List.UnselectAll()
			tab.List.Refresh()
		}

		tab.List.OnSelected = func(id widget.ListItemID) {
			entry := tab.Entries[id]
			if !entry.IsDir() {
				filePath := filepath.Join(tab.Path, entry.Name())
				cmd := exec.Command("cmd", "/c", "start", "", filePath)
				_ = cmd.Start()

				tab.List.UnselectAll()
				return
			}

			tab.History = append(tab.History, tab.Path)
			tab.LoadDir(filepath.Join(tab.Path, entry.Name()))
			tab.List.UnselectAll()
		}

		backButton := flatButton(theme.NavigateBackIcon(), "", func() {
			if len(tab.History) == 0 {
				return
			}

			last := len(tab.History) - 1
			previousPath := tab.History[last]
			tab.History = tab.History[:last]
			tab.LoadDir(previousPath)
		})

		rootButton := flatButton(theme.MoveUpIcon(), "", func() {
			parent := filepath.Dir(tab.Path)
			if parent != tab.Path {
				tab.History = append(tab.History, tab.Path)
				tab.LoadDir(parent)
			}

		})

		refreshButton := flatButton(theme.ViewRefreshIcon(), "", func() {
			tab.LoadDir(tab.Path)
		})

		toolbar := container.NewBorder(
			nil, nil,
			container.NewHBox(backButton, rootButton, refreshButton),
			nil,
			pathEntry,
		)

		dim := func(text string, align fyne.TextAlign) *widget.Label {
			l := widget.NewLabel(text)
			l.Importance = widget.LowImportance
			l.Alignment = align
			return l
		}
		headerRow := container.NewBorder(nil, nil,
			spacer(theme.IconInlineSize()+theme.Padding(), 0), // lines up with the file icons
			dim("Size", fyne.TextAlignTrailing),
			dim("Name", fyne.TextAlignLeading),
		)

		tab.Page = container.NewBorder(
			container.NewVBox(toolbar, headerRow, widget.NewSeparator()),
			nil, nil, nil,
			tab.List,
		)
		index := len(tabs)
		tabs = append(tabs, tab)

		tab.TabButton = NewTabChip(tabTitle(path), func() {
			switchTab(index)
		})
		tabBar.Add(tab.TabButton)
		tab.LoadDir(path)
		switchTab(index)
	}

	switchTab = func(index int) {
		if index < 0 || index >= len(tabs) {
			return
		}

		activeTab = index

		for i, t:= range tabs{
			t.TabButton.SetActive(i==activeTab)
		}

		content.Objects = []fyne.CanvasObject{tabs[activeTab].Page}
		content.Refresh()
	}

	addButton := flatButton(theme.ContentAddIcon(), "", func() { addTab(home) })

	tabHeader := container.NewBorder(
		nil, nil, nil, addButton, tabBar,
	)
	goTo := func(path string) {
		tab := tabs[activeTab]
		tab.History = append(tab.History, tab.Path)
		tab.LoadDir(path)
	}

	sidebar := container.NewVBox(
		sectionLabel("Places"),
		flatButton(theme.HomeIcon(), "Home", func() { goTo(home) }),
		flatButton(theme.ComputerIcon(), "Desktop", func() { goTo(filepath.Join(home, "Desktop")) }),
		flatButton(theme.DownloadIcon(), "Downloads", func() { goTo(filepath.Join(home, "Downloads")) }),
		flatButton(theme.DocumentIcon(), "Documents", func() { goTo(filepath.Join(home, "Documents")) }),
	)
	sidebarPane := container.NewStack(spacer(120, 0), container.NewPadded(sidebar))

	card := panel(content, theme.Color(theme.ColorNameMenuBackground), 12)
	main := container.NewBorder(tabHeader, nil, nil, nil, card)

	root := container.NewBorder(nil, nil, sidebarPane, nil, main)
	window.SetContent(container.NewPadded(root)) // outer margin: makes the card "float"

	addTab(home)
	window.ShowAndRun()

	addTab(home)
	window.ShowAndRun()
}
