package main

import (
	"fmt"
	"image/color"
	"context"
	"errors"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Clicks struct {
	OnTapped func()
	OnDouble func()
	OnMenu   func(pos fyne.Position)
}

func (c *Clicks) Tapped(*fyne.PointEvent) {
	if c.OnTapped != nil {
		c.OnTapped()
	}
}

func (c *Clicks) DoubleTapped(*fyne.PointEvent) {
	if c.OnDouble != nil {
		c.OnDouble()
	}
}

func (c *Clicks) TappedSecondary(e *fyne.PointEvent) {
	if c.OnMenu != nil {
		c.OnMenu(e.AbsolutePosition)
	}
}

func iconFor(e os.DirEntry) fyne.Resource {
	if e.IsDir() {
		return theme.NewPrimaryThemedResource(theme.FolderIcon())
	}
	return theme.FileIcon()
}

type Item struct {
	Entry os.DirEntry
	Dir   string
}

func (i Item) Path() string {
	return filepath.Join(i.Dir, i.Entry.Name())
}

type FileTab struct {
	Path       string
	StopSearch func()
	History    []string
	Entries    []Item
	All        []Item
	List       *widget.List
	Grid       *widget.GridWrap
	GridMode   bool
	Page       *fyne.Container
	LoadDir    func(string)
	TabButton  *TabChip
}

func (t *FileTab) Refresh() {
	t.List.Refresh()
	t.Grid.Refresh()
}

func (t *FileTab) ClearSelection() {
	t.List.UnselectAll()
	t.Grid.UnselectAll()
}

type FileRow struct {
	widget.BaseWidget
	Clicks
	icon *widget.Icon
	name *widget.Label
	loc  *widget.Label
	size *widget.Label
}
func NewFileRow() *FileRow {
	r := &FileRow{}
	r.icon = widget.NewIcon(theme.FileIcon())
	r.name = widget.NewLabel("")
	r.loc = widget.NewLabel("")
	r.size = widget.NewLabel("")
	r.name.Truncation = fyne.TextTruncateEllipsis
	r.size.Importance = widget.LowImportance
	r.size.Alignment = fyne.TextAlignTrailing
	r.loc.Importance = widget.LowImportance
	r.loc.Alignment = fyne.TextAlignTrailing
	r.ExtendBaseWidget(r)
	return r
}



func (r *FileRow) CreateRenderer() fyne.WidgetRenderer {
	right := container.NewHBox(r.loc, r.size)
	layout := container.NewBorder(nil, nil, r.icon, right, r.name)
	return widget.NewSimpleRenderer(layout)
}

func shorten(s string, limit int) string {
	letters := []rune(s)
	if len(letters) <= limit {
		return s
	}
	return "…" + string(letters[len(letters)-limit+1:])
}

func (r *FileRow) SetItem(item Item, base string) {
	r.name.SetText(item.Entry.Name())
	r.icon.SetResource(iconFor(item.Entry))

	r.loc.SetText("")
	if item.Dir != base {
		rel, err := filepath.Rel(base, item.Dir)
		if err == nil {
			r.loc.SetText(shorten(rel, 36))
		}
	}

	r.size.SetText("")
	if !item.Entry.IsDir() {
		info, err := item.Entry.Info()
		if err == nil {
			r.size.SetText(formatSize(info.Size()))
		}
	}
}

type FileCell struct {
	widget.BaseWidget
	Clicks
	icon *widget.Icon
	name *widget.Label
}

func NewFileCell() *FileCell {
	c := &FileCell{}
	c.icon = widget.NewIcon(theme.FileIcon())
	c.name = widget.NewLabel("")
	c.name.Alignment = fyne.TextAlignCenter
	c.name.Truncation = fyne.TextTruncateEllipsis
	c.ExtendBaseWidget(c)
	return c
}

func (c *FileCell) CreateRenderer() fyne.WidgetRenderer {
	iconBox := container.NewStack(spacer(112,60),container.NewPadded(c.icon))
	return widget.NewSimpleRenderer(container.NewVBox(iconBox,c.name))
	
}

func (c *FileCell) SetItem(item Item) {
	c.name.SetText(item.Entry.Name())
	c.icon.SetResource(iconFor(item.Entry))
}

type fileClip struct {
	Paths []string
	Cut   bool
}

func validName(name string) error {
	if name == "" {
		return errors.New("name can't be empty")
	}
	if strings.ContainsAny(name, `/\:*?"<>|`) {
		return errors.New(`name can't contain / \ : * ? " < > |`)
	}
	return nil
}


func uniquePath(dir, name string) string {
	path := filepath.Join(dir, name)
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return path
	}
 
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	n := 1
	for {
		var newName string
		if n == 1 {
			newName = base + " - Copy" + ext
		} else {
			newName = fmt.Sprintf("%s - Copy (%d)%s", base, n, ext)
		}
		path = filepath.Join(dir, newName)
		_, err = os.Stat(path)
		if os.IsNotExist(err) {
			return path
		}
		n++
	}
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
 
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
 
	_, err = io.Copy(out, in)
	closeErr := out.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
 
	if !info.IsDir() {
		return copyFile(src, dst, info.Mode().Perm())
	}
 
	err = os.MkdirAll(dst, 0o755)
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		from := filepath.Join(src, e.Name())
		to := filepath.Join(dst, e.Name())
		err = copyPath(from, to)
		if err != nil {
			return err
		}
	}
	return nil
}
func formatSize(size int64) string {
	kb := float64(1024)
	mb := kb * 1024
	gb := mb * 1024
	s := float64(size)
 
	if s >= gb {
		return fmt.Sprintf("%.1f GB", s/gb)
	} else if s >= mb {
		return fmt.Sprintf("%.1f MB", s/mb)
	} else if s >= kb {
		return fmt.Sprintf("%.1f KB", s/kb)
	}
	return fmt.Sprintf("%d B", size)
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



func spacer(w, h float32) fyne.CanvasObject {
	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(fyne.NewSize(w, h))
	return rect
}

func sectionLabel(text string) fyne.CanvasObject {
	label := widget.NewLabel(strings.ToUpper(text))
	label.Importance = widget.LowImportance
	label.TextStyle = fyne.TextStyle{Bold: true}
	return label
}
func tabTitle(path string) string {
	base := filepath.Base(path)
	if base == "." || base == string(filepath.Separator) {
		return path
	}
	return base
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
	var closeTab func(*FileTab)
	var clip fileClip
 
	indexOf := func(t *FileTab) int {
		for i := 0; i < len(tabs); i++ {
			if tabs[i] == t {
				return i
			}
		}
		return -1
	}
 
	addTab = func(path string) {
		_, err := os.ReadDir(path)
		if err != nil {
			return
		}
 
		tab := &FileTab{Path: path}
 
		pathEntry := widget.NewEntry()
		pathEntry.SetText(path)
 
		pathEntry.OnSubmitted = func(typed string) {
			info, err := os.Stat(typed)
			if err == nil && info.IsDir() {
				tab.History = append(tab.History, tab.Path)
				tab.LoadDir(typed)
			}
		}
 
		renameItem := func(item Item) {
			oldName := item.Entry.Name()
			input := widget.NewEntry()
			input.SetText(oldName)
 
			formItems := []*widget.FormItem{widget.NewFormItem("New name", input)}
 
			dialog.ShowForm("Rename", "Rename", "Cancel", formItems, func(ok bool) {
				if !ok {
					return
				}
				newName := strings.TrimSpace(input.Text)
				if newName == "" || newName == oldName {
					return
				}
				err := validName(newName)
				if err != nil {
					dialog.ShowError(err, window)
					return
				}
 
				oldPath := item.Path()
				newPath := filepath.Join(item.Dir, newName)
 
				if !strings.EqualFold(newName, oldName) {
					_, err := os.Stat(newPath)
					if err == nil {
						dialog.ShowError(fmt.Errorf("%q already exists", newName), window)
						return
					}
				}
 
				err = os.Rename(oldPath, newPath)
				if err != nil {
					dialog.ShowError(err, window)
					return
				}
				tab.LoadDir(tab.Path)
			}, window)
		}
 
		openItem := func(item Item) {
			if item.Entry.IsDir() {
				tab.History = append(tab.History, tab.Path)
				tab.LoadDir(item.Path())
				return
			}
			exec.Command("cmd", "/c", "start", "", item.Path()).Start()
		}
 
		selectIndex := func(id int) {
			if tab.GridMode {
				tab.Grid.Select(id)
			} else {
				tab.List.Select(id)
			}
		}
 
		deleteItem := func(item Item) {
			message := fmt.Sprintf("“%s” will be permanently deleted. This can't be undone.", item.Entry.Name())
			dialog.ShowConfirm("Delete permanently?", message, func(ok bool) {
				if !ok {
					return
				}
				err := os.RemoveAll(item.Path())
				if err != nil {
					dialog.ShowError(err, window)
					return
				}
				tab.LoadDir(tab.Path)
			}, window)
		}
 
		newFolder := func() {
			input := widget.NewEntry()
			input.SetText("New folder")
 
			formItems := []*widget.FormItem{widget.NewFormItem("Name", input)}
 
			dialog.ShowForm("New folder", "Create", "Cancel", formItems, func(ok bool) {
				if !ok {
					return
				}
				name := strings.TrimSpace(input.Text)
				err := validName(name)
				if err != nil {
					dialog.ShowError(err, window)
					return
				}
				err = os.Mkdir(filepath.Join(tab.Path, name), 0o755)
				if err != nil {
					dialog.ShowError(err, window)
					return
				}
				tab.LoadDir(tab.Path)
			}, window)
		}
 
		pasteHere := func() {
			if len(clip.Paths) == 0 {
				return
			}
			here := strings.ToLower(tab.Path) + string(filepath.Separator)
 
			for _, src := range clip.Paths {
				if clip.Cut && strings.EqualFold(filepath.Dir(src), tab.Path) {
					continue
				}
				if strings.HasPrefix(here, strings.ToLower(src)+string(filepath.Separator)) {
					dialog.ShowError(errors.New("can't paste a folder into itself"), window)
					return
				}
 
				dst := uniquePath(tab.Path, filepath.Base(src))
				var err error
				if clip.Cut {
					err = os.Rename(src, dst)
					if err != nil {
						err = copyPath(src, dst)
						if err == nil {
							err = os.RemoveAll(src)
						}
					}
				} else {
					err = copyPath(src, dst)
				}
				if err != nil {
					dialog.ShowError(err, window)
					break
				}
			}
 
			if clip.Cut {
				clip = fileClip{}
			}
			tab.LoadDir(tab.Path)
		}
 
		showMenu := func(item Item, pos fyne.Position) {
			pasteItem := fyne.NewMenuItem("Paste here", pasteHere)
			if len(clip.Paths) == 0 {
				pasteItem.Disabled = true
			}
 
			menuItems := []*fyne.MenuItem{}
			menuItems = append(menuItems, fyne.NewMenuItem("Open", func() { openItem(item) }))
			if item.Entry.IsDir() {
				menuItems = append(menuItems, fyne.NewMenuItem("Open in new tab", func() { addTab(item.Path()) }))
			}
			menuItems = append(menuItems, fyne.NewMenuItemSeparator())
			menuItems = append(menuItems, fyne.NewMenuItem("Copy", func() {
				clip = fileClip{Paths: []string{item.Path()}}
			}))
			menuItems = append(menuItems, fyne.NewMenuItem("Cut", func() {
				clip = fileClip{Paths: []string{item.Path()}, Cut: true}
			}))
			menuItems = append(menuItems, pasteItem)
			menuItems = append(menuItems, fyne.NewMenuItemSeparator())
			menuItems = append(menuItems, fyne.NewMenuItem("Rename", func() { renameItem(item) }))
			menuItems = append(menuItems, fyne.NewMenuItem("Delete", func() { deleteItem(item) }))
			menuItems = append(menuItems, fyne.NewMenuItemSeparator())
			menuItems = append(menuItems, fyne.NewMenuItem("Copy path", func() {
				window.Clipboard().SetContent(item.Path())
			}))
 
			menu := fyne.NewMenu("", menuItems...)
			widget.NewPopUpMenu(menu, window.Canvas()).ShowAtPosition(pos)
		}
 
		bind := func(c *Clicks, id int, item Item) {
			c.OnTapped = func() { selectIndex(id) }
			c.OnDouble = func() { openItem(item) }
			c.OnMenu = func(pos fyne.Position) {
				selectIndex(id)
				showMenu(item, pos)
			}
		}
 
		searchEntry := widget.NewEntry()
		searchEntry.SetPlaceHolder("Search this folder")
 
		clearBtn := widget.NewButtonWithIcon("", theme.ContentClearIcon(), func() {
			searchEntry.SetText("")
		})
		clearBtn.Importance = widget.LowImportance
		clearBtn.Hide()
		searchEntry.ActionItem = clearBtn
 
		emptyLabel := widget.NewLabel("")
		emptyLabel.Importance = widget.LowImportance
		emptyLabel.Alignment = fyne.TextAlignCenter
		emptyLabel.Hide()
 
		nameHeader := widget.NewLabel("Name")
		nameHeader.Importance = widget.LowImportance
 
		var cancelSearch context.CancelFunc
 
		applyFilter := func() {
			if cancelSearch != nil {
				cancelSearch()
				cancelSearch = nil
			}
 
			query := strings.ToLower(strings.TrimSpace(searchEntry.Text))
 
			if query == "" {
				tab.Entries = tab.All
				nameHeader.SetText("Name")
				if len(tab.Entries) == 0 {
					emptyLabel.SetText("This folder is empty")
					emptyLabel.Show()
				} else {
					emptyLabel.Hide()
				}
				tab.ClearSelection()
				tab.Refresh()
				return
			}
 
			ctx, cancel := context.WithCancel(context.Background())
			cancelSearch = cancel
			root := tab.Path
 
			tab.Entries = nil
			nameHeader.SetText("Searching…")
			emptyLabel.Hide()
			tab.List.UnselectAll()
			tab.List.Refresh()
 
			go func() {
				select {
				case <-time.After(200 * time.Millisecond):
				case <-ctx.Done():
					return
				}
 
				batch := []Item{}
				total := 0
				lastFlush := time.Now()
 
				flush := func(done bool) {
					found := batch
					batch = nil
					total = total + len(found)
					count := total
 
					fyne.Do(func() {
						if ctx.Err() != nil {
							return
						}
						tab.Entries = append(tab.Entries, found...)
 
						if !done {
							nameHeader.SetText(fmt.Sprintf("Searching… %d found", count))
						} else if count == 0 {
							nameHeader.SetText("Name")
							emptyLabel.SetText("No matches for “" + strings.TrimSpace(searchEntry.Text) + "”")
							emptyLabel.Show()
						} else {
							nameHeader.SetText(fmt.Sprintf("Name · %d results", count))
						}
						tab.List.Refresh()
					})
				}
 
				filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
					if ctx.Err() != nil {
						return ctx.Err()
					}
					if err != nil {
						if d != nil && d.IsDir() {
							return fs.SkipDir
						}
						return nil
					}
					if p == root {
						return nil
					}
 
					if strings.Contains(strings.ToLower(d.Name()), query) {
						batch = append(batch, Item{Entry: d, Dir: filepath.Dir(p)})
					}
					if len(batch) > 0 && time.Since(lastFlush) > 150*time.Millisecond {
						flush(false)
						lastFlush = time.Now()
					}
					return nil
				})
 
				if ctx.Err() == nil {
					flush(true)
				}
			}()
		}
 
		tab.StopSearch = func() {
			if cancelSearch != nil {
				cancelSearch()
			}
		}
 
		searchEntry.OnChanged = func(s string) {
			if s == "" {
				clearBtn.Hide()
			} else {
				clearBtn.Show()
			}
			applyFilter()
		}
 
		tab.List = widget.NewList(
			func() int { return len(tab.Entries) },
			func() fyne.CanvasObject { return NewFileRow() },
			func(id widget.ListItemID, o fyne.CanvasObject) {
				row := o.(*FileRow)
				item := tab.Entries[id]
				row.SetItem(item, tab.Path)
				bind(&row.Clicks, id, item)
			},
		)
 
		tab.Grid = widget.NewGridWrap(
			func() int { return len(tab.Entries) },
			func() fyne.CanvasObject { return NewFileCell() },
			func(id widget.GridWrapItemID, o fyne.CanvasObject) {
				cell := o.(*FileCell)
				item := tab.Entries[id]
				cell.SetItem(item)
				bind(&cell.Clicks, id, item)
			},
		)
		tab.Grid.Hide()
 
		tab.LoadDir = func(newPath string) {
			items, err := os.ReadDir(newPath)
			if err != nil {
				return
			}
 
			sort.Slice(items, func(i, j int) bool {
				if items[i].IsDir() != items[j].IsDir() {
					return items[i].IsDir()
				}
				return strings.ToLower(items[i].Name()) < strings.ToLower(items[j].Name())
			})
 
			all := []Item{}
			for _, e := range items {
				all = append(all, Item{Entry: e, Dir: newPath})
			}
 
			navigated := newPath != tab.Path
			tab.Path = newPath
			tab.All = all
 
			tab.TabButton.SetText(tabTitle(newPath))
			pathEntry.SetText(newPath)
			if navigated {
				searchEntry.SetText("")
			}
			applyFilter()
		}
 
		backButton := flatButton(theme.NavigateBackIcon(), "", func() {
			if len(tab.History) == 0 {
				return
			}
			last := len(tab.History) - 1
			previous := tab.History[last]
			tab.History = tab.History[:last]
			tab.LoadDir(previous)
		})
 
		upButton := flatButton(theme.MoveUpIcon(), "", func() {
			parent := filepath.Dir(tab.Path)
			if parent != tab.Path {
				tab.History = append(tab.History, tab.Path)
				tab.LoadDir(parent)
			}
		})
 
		refreshButton := flatButton(theme.ViewRefreshIcon(), "", func() {
			tab.LoadDir(tab.Path)
		})
 
		newFolderBtn := flatButton(theme.FolderNewIcon(), "", newFolder)
		pasteBtn := flatButton(theme.ContentPasteIcon(), "", pasteHere)
		viewBtn := flatButton(theme.GridIcon(), "", nil)
 
		leftButtons := container.NewHBox(backButton, upButton, refreshButton)
		rightButtons := container.NewHBox(newFolderBtn, pasteBtn, viewBtn, container.NewStack(spacer(220, 0), searchEntry))
		toolbar := container.NewBorder(nil, nil, leftButtons, rightButtons, pathEntry)
 
		sizeHeader := widget.NewLabel("Size")
		sizeHeader.Importance = widget.LowImportance
		sizeHeader.Alignment = fyne.TextAlignTrailing
 
		headerRow := container.NewBorder(nil, nil,
			spacer(theme.IconInlineSize()+theme.Padding(), 0),
			sizeHeader,
			nameHeader,
		)
 
		top := container.NewVBox(toolbar, headerRow, widget.NewSeparator())
		middle := container.NewStack(tab.List, tab.Grid, container.NewCenter(emptyLabel))
		tab.Page = container.NewBorder(top, nil, nil, nil, middle)
 
		viewBtn.OnTapped = func() {
			tab.GridMode = !tab.GridMode
			if tab.GridMode {
				tab.List.Hide()
				tab.Grid.Show()
				headerRow.Hide()
				viewBtn.SetIcon(theme.ListIcon())
			} else {
				tab.Grid.Hide()
				tab.List.Show()
				headerRow.Show()
				viewBtn.SetIcon(theme.GridIcon())
			}
			tab.ClearSelection()
			tab.Refresh()
		}
 
		tabs = append(tabs, tab)
 
		tab.TabButton = NewTabChip(tabTitle(path),
			func() { switchTab(indexOf(tab)) },
			func() { closeTab(tab) },
		)
		tabBar.Add(tab.TabButton)
		tab.LoadDir(path)
		switchTab(len(tabs) - 1)
	}
 
	closeTab = func(tab *FileTab) {
		if len(tabs) <= 1 {
			return
		}
		i := indexOf(tab)
		if i < 0 {
			return
		}
		if tab.StopSearch != nil {
			tab.StopSearch()
		}
 
		tabs = append(tabs[:i], tabs[i+1:]...)
		tabBar.Remove(tab.TabButton)
 
		if i < activeTab {
			activeTab--
		} else if i == activeTab {
			if i > len(tabs)-1 {
				activeTab = len(tabs) - 1
			} else {
				activeTab = i
			}
		}
		switchTab(activeTab)
	}
 
	switchTab = func(index int) {
		if index < 0 || index >= len(tabs) {
			return
		}
 
		activeTab = index
 
		for i, t := range tabs {
			t.TabButton.SetActive(i == activeTab)
			t.TabButton.SetClosable(len(tabs) > 1)
		}
 
		content.Objects = []fyne.CanvasObject{tabs[activeTab].Page}
		content.Refresh()
	}
 
	addButton := flatButton(theme.ContentAddIcon(), "", func() { addTab(home) })
 
	tabHeader := container.NewBorder(nil, nil, nil, addButton, tabBar)
 
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
	mainArea := container.NewBorder(tabHeader, nil, nil, nil, card)
 
	root := container.NewBorder(nil, nil, sidebarPane, nil, mainArea)
	window.SetContent(container.NewPadded(root))
 
	addTab(home)
	window.ShowAndRun()
}
