package main

import (
	"fmt"
	"github.com/deepnoodle-ai/gooey"
)

type TableDemo struct {
	screen *gooey.Screen
	table  *gooey.Table
}

func (d *TableDemo) Draw(frame gooey.RenderFrame) {
	d.table.Draw(frame)

	// Info
	msg := fmt.Sprintf("Selected Row: %d", d.table.SelectedRow)
	frame.PrintStyled(2, d.table.Y+d.table.Height+1, msg, gooey.NewStyle())
	frame.PrintStyled(2, d.table.Y+d.table.Height+2, "Press Arrows to move, q to quit.", gooey.NewStyle().WithDim())
}

func (d *TableDemo) HandleKey(event gooey.KeyEvent) bool {
	if event.Rune == 'q' {
		d.screen.Stop()
		return true
	}
	return d.table.HandleKey(event)
}

func main() {
	term, err := gooey.NewTerminal()
	if err != nil {
		panic(err)
	}
	defer term.Close()

	screen := gooey.NewScreen(term)

	header := gooey.SimpleHeader("Table Demo", gooey.NewStyle().WithBold().WithBackground(gooey.ColorBlue))
	screen.SetLayout(gooey.NewLayout(term).SetHeader(header))

	width, height := term.Size()

	table := gooey.NewTable(2, 2, width-4, height-6)
	table.SetColumns([]gooey.Column{
		{Title: "ID", Width: 5},
		{Title: "Name", Width: 20},
		{Title: "Role", Width: 15},
		{Title: "Status", Width: 10},
	})

	// Generate dummy data
	rows := make([][]string, 50)
	for i := 0; i < 50; i++ {
		rows[i] = []string{
			fmt.Sprintf("%d", i+1),
			fmt.Sprintf("User %d", i+1),
			"Developer",
			"Active",
		}
	}
	table.SetRows(rows)

	demo := &TableDemo{
		screen: screen,
		table:  table,
	}

	screen.AddWidget(demo)

	if err := screen.Run(); err != nil {
		panic(err)
	}
}
