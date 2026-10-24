
package main

import (
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type FileTab struct {
	Path    string
	History []string
	Entries []os.DirEntry
	List    *widget.List
	Page    *fyne.Container
	LoadDir func(string)
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

		pathLabel := widget.NewLabel(path)

		tab.List = widget.NewList(
			func() int {
				return len(tab.Entries)
			},
			func() fyne.CanvasObject {
				return widget.NewLabel("File")
			},
			func(id widget.ListItemID, item fyne.CanvasObject) {
				entry := tab.Entries[id]
				name := entry.Name()
				if entry.IsDir() {
					name = "📁  " + name
				} else {
					name = "   " + name
				}
				item.(*widget.Label).SetText(name)
			},
		)

		loadDirectory := func(newPath string) {
			items, err := os.ReadDir(newPath)
			if err != nil {
				return
			}
			tab.Path = newPath
			tab.Entries = items
			pathLabel.SetText(newPath)
			tab.List.UnselectAll()
			tab.List.Refresh()
		}
		tab.LoadDir = loadDirectory

		tab.List.OnSelected = func(id widget.ListItemID) {
			entry := tab.Entries[id]
			if !entry.IsDir() {
				tab.List.UnselectAll()
				return
			}

			tab.History = append(tab.History, tab.Path)
			loadDirectory(filepath.Join(tab.Path, entry.Name()))
			tab.List.UnselectAll()
		}

		backButton := widget.NewButton("←", func() {
			if len(tab.History) == 0 {
				return
			}
			
			last := len(tab.History) - 1
			previousPath := tab.History[last]
			tab.History = tab.History[:last]
			loadDirectory(previousPath)
		})

		rootButton := widget.NewButton("up", func ()  {
			parent :=filepath.Dir(tab.Path)
			if parent != tab.Path{
				tab.History = append(tab.History, tab.Path)
				tab.LoadDir(parent)
			}
		


		})

		refreshButton := widget.NewButton("↻", func() {
			loadDirectory(tab.Path)
		})

		toolbar := container.NewBorder(
			nil, nil,
			container.NewHBox(backButton,rootButton, refreshButton),
			nil,
			pathLabel,
		)

		tab.Page = container.NewBorder(toolbar, nil, nil, nil, tab.List)

		index := len(tabs)
		tabs = append(tabs, tab)

		tabButton := widget.NewButton(filepath.Base(path), func() {
			switchTab(index)
		})
		tabBar.Add(tabButton)

		switchTab(index)
	}

	switchTab = func(index int) {
		if index < 0 || index >= len(tabs) {
			return
		}

		activeTab = index
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
		tabHeader,nil,nil,nil,content,
	)

	sidebar := container.NewVBox(
		widget.NewButton("Home",func() {
			tab := tabs[activeTab]
			tab.History  =append(tab.History, tab.Path)
			tab.LoadDir(home)
		}),
		widget.NewButton("Desktop",func() {
			tab := tabs[activeTab]
			tab.History = append(tab.History, tab.Path)
			tab.LoadDir(filepath.Join(home,"Desktop"))
		}),
		widget.NewButton("Downloads",func() {
			tab := tabs[activeTab]
			tab.History = append(tab.History, tab.Path)
			tab.LoadDir(filepath.Join(home,"Downloads"))
		}),
		widget.NewButton("Documents",func() {
			tab := tabs[activeTab]
			tab.History = append(tab.History, tab.Path)
			tab.LoadDir(filepath.Join(home,"Documents"))
		}),
	)

	window.SetContent(container.NewBorder(
		nil,nil,sidebar,nil,mainContent,
	))

	addTab(home)
	window.ShowAndRun()
}
