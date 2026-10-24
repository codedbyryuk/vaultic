package main

import(
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func main(){

	myApp := app.New()

	window := myApp.NewWindow("Vaultic")
	window.Resize(fyne.NewSize(900,900))

	title := widget.NewLabel("Vaultic - File Manager")
	window.SetContent(container.NewCenter(title))

	window.ShowAndRun()
}