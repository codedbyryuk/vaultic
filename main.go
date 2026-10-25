package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	// "fyne.io/fyne/v2/layout"
)

type FileTab struct {
	Path      string
	History   []string
	Entries   []os.DirEntry
	List      *widget.List
	Page      *fyne.Container
	LoadDir   func(string)
	TabButton *widget.Button
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

func main() {
	myApp := app.New()
	window := myApp.NewWindow("Vaultic")
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

		tab.List = widget.NewList(
			func() int {
				return len(tab.Entries)
			},
			func() fyne.CanvasObject {
				return widget.NewLabel("File")
			},
			func(id widget.ListItemID, item fyne.CanvasObject) {
				entry := tab.Entries[id]
				info, err := entry.Info()
				size := formatSize(info.Size())
				if err != nil {
					return
				}
				name := entry.Name()
				if entry.IsDir() {
					name = "📁  " + name
				} else {
					name = "   " + name + " (" + size + ")"
				}
				item.(*widget.Label).SetText(name)
			},
		)

		tab.LoadDir = func(newPath string) {
			items, err := os.ReadDir(newPath)
			if err != nil {
				return
			}

			tab.Path = newPath
			tab.Entries = items

			sort.Slice(items,func (i,j int) bool {
				a := items[i]
				b := items[j]

				if a.IsDir() && !b.IsDir(){
					return true
				}
				if !a.IsDir() && b.IsDir(){
					return false
				}

				return strings.ToLower(a.Name()) < strings.ToLower(b.Name())
			})
			

			tab.TabButton.SetText(filepath.Base(newPath))
			pathEntry.SetText(newPath)
			tab.List.UnselectAll()
			tab.List.Refresh()
		}

		tab.List.OnSelected = func(id widget.ListItemID) {
			entry := tab.Entries[id]
			if !entry.IsDir() {
				tab.List.UnselectAll()
				return
			}

			tab.History = append(tab.History, tab.Path)
			tab.LoadDir(filepath.Join(tab.Path, entry.Name()))
			tab.List.UnselectAll()
		}

		backButton := widget.NewButton("←", func() {
			if len(tab.History) == 0 {
				return
			}

			last := len(tab.History) - 1
			previousPath := tab.History[last]
			tab.History = tab.History[:last]
			tab.LoadDir(previousPath)
		})

		rootButton := widget.NewButton("up", func() {
			parent := filepath.Dir(tab.Path)
			if parent != tab.Path {
				tab.History = append(tab.History, tab.Path)
				tab.LoadDir(parent)
			}

		})

		refreshButton := widget.NewButton("↻", func() {
			tab.LoadDir(tab.Path)
		})

		toolbar := container.NewBorder(
			nil, nil,
			container.NewHBox(backButton, rootButton, refreshButton),
			nil,
			pathEntry,
		)

		tab.Page = container.NewBorder(toolbar, nil, nil, nil, tab.List)

		index := len(tabs)
		tabs = append(tabs, tab)

		tab.TabButton = widget.NewButton(filepath.Base(path), func() {
			switchTab(index)
		})
		tabBar.Add(tab.TabButton)

		switchTab(index)
	}

	switchTab = func(index int) {
		if index < 0 || index >= len(tabs) {
			return
		}

		activeTab = index

		for i := 0; i < len(tabs); i++ {
			folderName := filepath.Base(tabs[i].Path)
			if i == activeTab {
				folderName = "● " + folderName
			}

			tabs[i].TabButton.SetText(folderName)
		}

		content.Objects = []fyne.CanvasObject{tabs[activeTab].Page}
		content.Refresh()
	}

	addButton := widget.NewButton("+", func() {
		addTab(home)
	})

	tabHeader := container.NewBorder(
		nil, nil, nil, addButton, tabBar,
	)
	mainContent := container.NewBorder(
		tabHeader, nil, nil, nil, content,
	)

	sidebar := container.NewVBox(
		widget.NewLabel("Places"),
		widget.NewSeparator(),
		widget.NewButton("Home", func() {
			tab := tabs[activeTab]
			tab.History = append(tab.History, tab.Path)
			tab.LoadDir(home)
		}),
		widget.NewButton("Desktop", func() {
			tab := tabs[activeTab]
			tab.History = append(tab.History, tab.Path)
			tab.LoadDir(filepath.Join(home, "Desktop"))
		}),
		widget.NewButton("Downloads", func() {
			tab := tabs[activeTab]
			tab.History = append(tab.History, tab.Path)
			tab.LoadDir(filepath.Join(home, "Downloads"))
		}),
		widget.NewButton("Documents", func() {
			tab := tabs[activeTab]
			tab.History = append(tab.History, tab.Path)
			tab.LoadDir(filepath.Join(home, "Documents"))
		}),
	)
	split := container.NewHSplit(sidebar, mainContent)
	split.SetOffset(0.2)

	window.SetContent(split)

	addTab(home)
	window.ShowAndRun()
}
