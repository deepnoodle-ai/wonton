// Example: paletteer - Terminal color palette designer
//
// A TUI for designing and previewing terminal color palettes. Create gradients,
// pick colors with RGB/HSL sliders, and export to various formats.
//
// Run with:
//
//	go run ./examples/paletteer
//	go run ./examples/paletteer --preset rainbow  # Start with rainbow preset
//	go run ./examples/paletteer --preset warm     # Start with warm preset
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/deepnoodle-ai/wonton/cli"
	"github.com/deepnoodle-ai/wonton/clipboard"
	"github.com/deepnoodle-ai/wonton/color"
	"github.com/deepnoodle-ai/wonton/tui"
)

// PaletteColor represents a color in the palette
type PaletteColor struct {
	R, G, B uint8
	Name    string
}

// ToRGB converts to color.RGB
func (c PaletteColor) ToRGB() color.RGB {
	return color.NewRGB(c.R, c.G, c.B)
}

// Hex returns the hex string
func (c PaletteColor) Hex() string {
	return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
}

// ViewMode determines what we're editing
type ViewMode int

const (
	ViewPalette ViewMode = iota
	ViewGradient
	ViewPreview
	ViewExport
)

// PaletteerApp is the TUI application
type PaletteerApp struct {
	// Palette colors
	colors   []PaletteColor
	selected int

	// Current color being edited (RGB)
	editingR, editingG, editingB uint8
	editingComponent             int // 0=R, 1=G, 2=B

	// Gradient
	gradientSteps  int
	gradientColors []color.RGB

	// View state
	mode      ViewMode
	width     int
	height    int
	statusMsg string

	// Export format
	exportFormat int // 0=hex, 1=rgb, 2=hsl, 3=ansi
}

// Preset palettes
var presets = map[string][]PaletteColor{
	"rainbow": {
		{255, 0, 0, "Red"},
		{255, 127, 0, "Orange"},
		{255, 255, 0, "Yellow"},
		{0, 255, 0, "Green"},
		{0, 0, 255, "Blue"},
		{75, 0, 130, "Indigo"},
		{148, 0, 211, "Violet"},
	},
	"warm": {
		{255, 87, 51, "Coral"},
		{255, 195, 0, "Amber"},
		{255, 128, 0, "Orange"},
		{220, 20, 60, "Crimson"},
		{178, 34, 34, "Firebrick"},
	},
	"cool": {
		{0, 191, 255, "DeepSkyBlue"},
		{0, 255, 255, "Cyan"},
		{0, 128, 128, "Teal"},
		{65, 105, 225, "RoyalBlue"},
		{138, 43, 226, "BlueViolet"},
	},
	"mono": {
		{0, 0, 0, "Black"},
		{64, 64, 64, "DarkGray"},
		{128, 128, 128, "Gray"},
		{192, 192, 192, "LightGray"},
		{255, 255, 255, "White"},
	},
	"neon": {
		{255, 0, 255, "Magenta"},
		{0, 255, 255, "Cyan"},
		{255, 255, 0, "Yellow"},
		{0, 255, 0, "Lime"},
		{255, 0, 128, "HotPink"},
	},
}

func main() {
	app := cli.New("paletteer").
		Description("Terminal color palette designer").
		Version("1.0.0")

	app.Main().
		Flags(
			cli.String("preset", "p").
				Default("rainbow").
				Enum("rainbow", "warm", "cool", "mono", "neon").
				Help("Starting palette preset"),
		).
		Run(func(ctx *cli.Context) error {
			preset := ctx.String("preset")

			tuiApp := &PaletteerApp{
				colors:        append([]PaletteColor{}, presets[preset]...),
				gradientSteps: 10,
				statusMsg:     "←→ select | ↑↓ adjust | Tab component | a add | d delete | g gradient | e export | q quit",
			}

			// Initialize editing with first color
			if len(tuiApp.colors) > 0 {
				tuiApp.editingR = tuiApp.colors[0].R
				tuiApp.editingG = tuiApp.colors[0].G
				tuiApp.editingB = tuiApp.colors[0].B
			}

			tuiApp.updateGradient()

			return tui.Run(tuiApp)
		})

	if err := app.Execute(); err != nil {
		if cli.IsHelpRequested(err) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(cli.GetExitCode(err))
	}
}

func (app *PaletteerApp) updateGradient() {
	if len(app.colors) < 2 {
		app.gradientColors = nil
		return
	}

	// Convert to RGB
	stops := make([]color.RGB, len(app.colors))
	for i, c := range app.colors {
		stops[i] = c.ToRGB()
	}

	app.gradientColors = color.MultiGradient(stops, app.gradientSteps*len(app.colors))
}

func (app *PaletteerApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.ResizeEvent:
		app.width = e.Width
		app.height = e.Height

	case tui.KeyEvent:
		// Global quit
		if e.Key == tui.KeyCtrlC {
			return []tui.Cmd{tui.Quit()}
		}

		switch app.mode {
		case ViewPalette:
			return app.handlePaletteKey(e)
		case ViewGradient:
			return app.handleGradientKey(e)
		case ViewExport:
			return app.handleExportKey(e)
		case ViewPreview:
			return app.handlePreviewKey(e)
		}
	}

	return nil
}

func (app *PaletteerApp) handlePaletteKey(e tui.KeyEvent) []tui.Cmd {
	if e.Rune == 'q' || e.Rune == 'Q' || e.Key == tui.KeyEscape {
		return []tui.Cmd{tui.Quit()}
	}

	switch e.Key {
	case tui.KeyArrowLeft:
		if app.selected > 0 {
			app.selected--
			app.loadSelectedColor()
		}
	case tui.KeyArrowRight:
		if app.selected < len(app.colors)-1 {
			app.selected++
			app.loadSelectedColor()
		}
	case tui.KeyArrowUp:
		app.adjustComponent(5)
	case tui.KeyArrowDown:
		app.adjustComponent(-5)
	case tui.KeyTab:
		app.editingComponent = (app.editingComponent + 1) % 3
	case tui.KeyEnter:
		app.applyEditingColor()
	}

	switch e.Rune {
	case 'r', 'R':
		app.editingComponent = 0
	case 'g', 'G':
		if e.Rune == 'g' {
			app.mode = ViewGradient
			app.statusMsg = "←→ adjust steps | p preview | Esc back"
		} else {
			app.editingComponent = 1
		}
	case 'b', 'B':
		app.editingComponent = 2
	case 'a', 'A':
		// Add new color
		newColor := PaletteColor{
			R:    app.editingR,
			G:    app.editingG,
			B:    app.editingB,
			Name: fmt.Sprintf("Color%d", len(app.colors)+1),
		}
		app.colors = append(app.colors, newColor)
		app.selected = len(app.colors) - 1
		app.updateGradient()
		app.statusMsg = "✓ Color added"
	case 'd', 'D':
		if len(app.colors) > 1 {
			app.colors = append(app.colors[:app.selected], app.colors[app.selected+1:]...)
			if app.selected >= len(app.colors) {
				app.selected = len(app.colors) - 1
			}
			app.loadSelectedColor()
			app.updateGradient()
			app.statusMsg = "✓ Color deleted"
		}
	case 'e', 'E':
		app.mode = ViewExport
		app.statusMsg = "↑↓ format | c copy | Esc back"
	case 'p', 'P':
		app.mode = ViewPreview
		app.statusMsg = "View color combinations | Esc back"
	case 'c', 'C':
		// Copy current color
		hex := app.colors[app.selected].Hex()
		if err := clipboard.Write(hex); err == nil {
			app.statusMsg = fmt.Sprintf("✓ Copied %s", hex)
		}
	case '1':
		app.colors = append([]PaletteColor{}, presets["rainbow"]...)
		app.selected = 0
		app.loadSelectedColor()
		app.updateGradient()
	case '2':
		app.colors = append([]PaletteColor{}, presets["warm"]...)
		app.selected = 0
		app.loadSelectedColor()
		app.updateGradient()
	case '3':
		app.colors = append([]PaletteColor{}, presets["cool"]...)
		app.selected = 0
		app.loadSelectedColor()
		app.updateGradient()
	case '4':
		app.colors = append([]PaletteColor{}, presets["mono"]...)
		app.selected = 0
		app.loadSelectedColor()
		app.updateGradient()
	case '5':
		app.colors = append([]PaletteColor{}, presets["neon"]...)
		app.selected = 0
		app.loadSelectedColor()
		app.updateGradient()
	}

	return nil
}

func (app *PaletteerApp) handleGradientKey(e tui.KeyEvent) []tui.Cmd {
	if e.Key == tui.KeyEscape || e.Rune == 'q' {
		app.mode = ViewPalette
		app.statusMsg = "←→ select | ↑↓ adjust | Tab component | a add | d delete | g gradient | e export | q quit"
		return nil
	}

	switch e.Key {
	case tui.KeyArrowLeft:
		if app.gradientSteps > 2 {
			app.gradientSteps--
			app.updateGradient()
		}
	case tui.KeyArrowRight:
		if app.gradientSteps < 50 {
			app.gradientSteps++
			app.updateGradient()
		}
	}

	switch e.Rune {
	case 'p', 'P':
		app.mode = ViewPreview
		app.statusMsg = "View color combinations | Esc back"
	case 'c', 'C':
		// Copy gradient as CSS
		css := app.gradientToCSS()
		if err := clipboard.Write(css); err == nil {
			app.statusMsg = "✓ Copied CSS gradient"
		}
	}

	return nil
}

func (app *PaletteerApp) handleExportKey(e tui.KeyEvent) []tui.Cmd {
	if e.Key == tui.KeyEscape || e.Rune == 'q' {
		app.mode = ViewPalette
		app.statusMsg = "←→ select | ↑↓ adjust | Tab component | a add | d delete | g gradient | e export | q quit"
		return nil
	}

	switch e.Key {
	case tui.KeyArrowUp:
		if app.exportFormat > 0 {
			app.exportFormat--
		}
	case tui.KeyArrowDown:
		if app.exportFormat < 3 {
			app.exportFormat++
		}
	}

	switch e.Rune {
	case 'c', 'C':
		// Copy in selected format
		var output string
		switch app.exportFormat {
		case 0: // Hex
			var hexes []string
			for _, c := range app.colors {
				hexes = append(hexes, c.Hex())
			}
			output = strings.Join(hexes, "\n")
		case 1: // RGB
			var rgbs []string
			for _, c := range app.colors {
				rgbs = append(rgbs, fmt.Sprintf("rgb(%d, %d, %d)", c.R, c.G, c.B))
			}
			output = strings.Join(rgbs, "\n")
		case 2: // CSS variables
			var vars []string
			for i, c := range app.colors {
				vars = append(vars, fmt.Sprintf("  --color-%d: %s;", i+1, c.Hex()))
			}
			output = ":root {\n" + strings.Join(vars, "\n") + "\n}"
		case 3: // ANSI
			var ansis []string
			for _, c := range app.colors {
				ansis = append(ansis, fmt.Sprintf("\\033[38;2;%d;%d;%dm", c.R, c.G, c.B))
			}
			output = strings.Join(ansis, "\n")
		}

		if err := clipboard.Write(output); err == nil {
			app.statusMsg = "✓ Copied to clipboard"
		}
	}

	return nil
}

func (app *PaletteerApp) handlePreviewKey(e tui.KeyEvent) []tui.Cmd {
	if e.Key == tui.KeyEscape || e.Rune == 'q' {
		app.mode = ViewPalette
		app.statusMsg = "←→ select | ↑↓ adjust | Tab component | a add | d delete | g gradient | e export | q quit"
	}
	return nil
}

func (app *PaletteerApp) loadSelectedColor() {
	if app.selected >= 0 && app.selected < len(app.colors) {
		c := app.colors[app.selected]
		app.editingR = c.R
		app.editingG = c.G
		app.editingB = c.B
	}
}

func (app *PaletteerApp) adjustComponent(delta int) {
	switch app.editingComponent {
	case 0:
		app.editingR = clampUint8(int(app.editingR) + delta)
	case 1:
		app.editingG = clampUint8(int(app.editingG) + delta)
	case 2:
		app.editingB = clampUint8(int(app.editingB) + delta)
	}
	app.applyEditingColor()
}

func (app *PaletteerApp) applyEditingColor() {
	if app.selected >= 0 && app.selected < len(app.colors) {
		app.colors[app.selected].R = app.editingR
		app.colors[app.selected].G = app.editingG
		app.colors[app.selected].B = app.editingB
		app.updateGradient()
	}
}

func (app *PaletteerApp) gradientToCSS() string {
	var stops []string
	for _, c := range app.colors {
		stops = append(stops, c.Hex())
	}
	return fmt.Sprintf("linear-gradient(to right, %s)", strings.Join(stops, ", "))
}

func clampUint8(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func (app *PaletteerApp) View() tui.View {
	// Header
	header := tui.HeaderBar(fmt.Sprintf("🎨 Paletteer  [%d colors]", len(app.colors))).
		Bg(tui.ColorMagenta).
		Fg(tui.ColorWhite)

	var content tui.View
	switch app.mode {
	case ViewPalette:
		content = app.viewPalette()
	case ViewGradient:
		content = app.viewGradient()
	case ViewExport:
		content = app.viewExport()
	case ViewPreview:
		content = app.viewPreview()
	}

	return tui.Stack(
		header,
		content,
		tui.StatusBar(app.statusMsg),
	)
}

func (app *PaletteerApp) viewPalette() tui.View {
	// Color swatches
	var swatches []tui.View
	for i, c := range app.colors {
		swatches = append(swatches, app.swatchView(c, i == app.selected))
	}

	// Color bar
	colorBar := tui.Group(swatches...)

	// RGB sliders for current color
	var sliders []tui.View

	// R slider
	rSelected := app.editingComponent == 0
	rBar := app.renderSlider(app.editingR, color.NewRGB(255, 0, 0), rSelected)
	rLabel := "R"
	if rSelected {
		rLabel = "[R]"
	}
	sliders = append(sliders, tui.Group(
		tui.Text(" %s ", rLabel).Fg(tui.ColorRed).Bold(),
		rBar,
		tui.Text(" %3d", app.editingR).Fg(tui.ColorWhite),
	))

	// G slider
	gSelected := app.editingComponent == 1
	gBar := app.renderSlider(app.editingG, color.NewRGB(0, 255, 0), gSelected)
	gLabel := "G"
	if gSelected {
		gLabel = "[G]"
	}
	sliders = append(sliders, tui.Group(
		tui.Text(" %s ", gLabel).Fg(tui.ColorGreen).Bold(),
		gBar,
		tui.Text(" %3d", app.editingG).Fg(tui.ColorWhite),
	))

	// B slider
	bSelected := app.editingComponent == 2
	bBar := app.renderSlider(app.editingB, color.NewRGB(0, 0, 255), bSelected)
	bLabel := "B"
	if bSelected {
		bLabel = "[B]"
	}
	sliders = append(sliders, tui.Group(
		tui.Text(" %s ", bLabel).Fg(tui.ColorBlue).Bold(),
		bBar,
		tui.Text(" %3d", app.editingB).Fg(tui.ColorWhite),
	))

	// Current color preview
	currentRGB := color.NewRGB(app.editingR, app.editingG, app.editingB)
	preview := tui.Group(
		tui.Text(" Current: "),
		tui.Text("████████").FgRGB(currentRGB.R, currentRGB.G, currentRGB.B),
		tui.Text(" %s", fmt.Sprintf("#%02X%02X%02X", app.editingR, app.editingG, app.editingB)).Fg(tui.ColorBrightBlack),
	)

	// Gradient preview
	gradientBar := app.blocksView(app.gradientColors, "█")

	// Presets
	presetHelp := tui.Text(" Presets: 1=rainbow 2=warm 3=cool 4=mono 5=neon").Fg(tui.ColorBrightBlack)

	return tui.Stack(
		tui.Spacer().MinHeight(1),
		tui.Text(" Palette Colors:").Bold(),
		colorBar,
		tui.Spacer().MinHeight(1),
		tui.Divider(),
		tui.Spacer().MinHeight(1),
		tui.Text(" Edit Color:").Bold(),
		tui.Stack(sliders...),
		tui.Spacer().MinHeight(1),
		preview,
		tui.Spacer().MinHeight(1),
		tui.Divider(),
		tui.Spacer().MinHeight(1),
		tui.Text(" Gradient:").Bold(),
		tui.Group(tui.Text(" "), gradientBar),
		tui.Spacer(),
		presetHelp,
	).Padding(1)
}

func (app *PaletteerApp) renderSlider(value uint8, c color.RGB, selected bool) tui.View {
	// 30 character bar
	width := 30
	filled := int(value) * width / 255

	parts := make([]tui.View, 0, width+2)
	if selected {
		parts = append(parts, tui.Text("["))
	} else {
		parts = append(parts, tui.Text(" "))
	}

	for i := 0; i < width; i++ {
		if i < filled {
			parts = append(parts, tui.Text("█").FgRGB(c.R, c.G, c.B))
		} else {
			parts = append(parts, tui.Text("░"))
		}
	}

	if selected {
		parts = append(parts, tui.Text("]"))
	} else {
		parts = append(parts, tui.Text(" "))
	}

	return tui.Group(parts...)
}

func (app *PaletteerApp) viewGradient() tui.View {
	// Large gradient display
	gradientBar := app.blocksView(app.gradientColors, "██")

	// Steps display
	stepsBar := fmt.Sprintf("Steps per segment: %d (total: %d colors)",
		app.gradientSteps, len(app.gradientColors))

	return tui.Stack(
		tui.Spacer().MinHeight(1),
		tui.Text(" Gradient Preview").Bold(),
		tui.Spacer().MinHeight(1),
		tui.Group(tui.Text(" "), gradientBar),
		tui.Spacer().MinHeight(2),
		tui.Text(" %s", stepsBar).Fg(tui.ColorBrightBlack),
		tui.Spacer().MinHeight(1),
		tui.Text(" Use ←→ to adjust gradient smoothness").Fg(tui.ColorBrightBlack),
		tui.Spacer(),
		tui.Text(" c = copy CSS gradient").Fg(tui.ColorCyan),
	).Padding(1)
}

func (app *PaletteerApp) viewExport() tui.View {
	formats := []string{"Hex codes", "RGB values", "CSS variables", "ANSI escape codes"}

	var formatViews []tui.View
	for i, f := range formats {
		selected := i == app.exportFormat
		var bg color.Color
		if selected {
			bg = tui.ColorCyan
		}
		formatViews = append(formatViews, tui.Text(" %s ", f).Bg(bg))
	}

	// Preview of selected format
	var preview []tui.View
	preview = append(preview, tui.Text(" Output Preview:").Bold())
	preview = append(preview, tui.Spacer().MinHeight(1))

	switch app.exportFormat {
	case 0: // Hex
		for _, c := range app.colors {
			rgb := c.ToRGB()
			preview = append(preview, tui.Group(
				tui.Text(" "),
				tui.Text("██").FgRGB(rgb.R, rgb.G, rgb.B),
				tui.Text(" %s", c.Hex()),
			))
		}
	case 1: // RGB
		for _, c := range app.colors {
			rgb := c.ToRGB()
			preview = append(preview, tui.Group(
				tui.Text(" "),
				tui.Text("██").FgRGB(rgb.R, rgb.G, rgb.B),
				tui.Text(" rgb(%d, %d, %d)", c.R, c.G, c.B),
			))
		}
	case 2: // CSS
		preview = append(preview, tui.Text(" :root {").Fg(tui.ColorBrightBlack))
		for i, c := range app.colors {
			preview = append(preview, tui.Text("   --color-%d: %s;", i+1, c.Hex()).Fg(tui.ColorCyan))
		}
		preview = append(preview, tui.Text(" }").Fg(tui.ColorBrightBlack))
	case 3: // ANSI
		for _, c := range app.colors {
			preview = append(preview, tui.Text(" \\033[38;2;%d;%d;%dm", c.R, c.G, c.B).Fg(tui.ColorYellow))
		}
	}

	return tui.Stack(
		tui.Spacer().MinHeight(1),
		tui.Text(" Export Format:").Bold(),
		tui.Spacer().MinHeight(1),
		tui.Stack(formatViews...),
		tui.Spacer().MinHeight(2),
		tui.Divider(),
		tui.Spacer().MinHeight(1),
		tui.Stack(preview...),
		tui.Spacer(),
		tui.Text(" c = copy to clipboard").Fg(tui.ColorCyan),
	).Padding(1)
}

func (app *PaletteerApp) viewPreview() tui.View {
	// Show color combinations
	var previews []tui.View

	previews = append(previews, tui.Text(" Color Combinations:").Bold())
	previews = append(previews, tui.Spacer().MinHeight(1))

	// Text on each background
	for _, bg := range app.colors {
		bgRGB := bg.ToRGB()
		segments := []tui.View{
			tui.Text(" ").BgRGB(bgRGB.R, bgRGB.G, bgRGB.B),
		}
		for _, fg := range app.colors {
			fgRGB := fg.ToRGB()
			segments = append(segments, tui.Text(" Aa ").FgRGB(fgRGB.R, fgRGB.G, fgRGB.B).BgRGB(bgRGB.R, bgRGB.G, bgRGB.B))
		}
		previews = append(previews, tui.Group(segments...))
	}

	previews = append(previews, tui.Spacer().MinHeight(2))

	// Sample text with gradient
	previews = append(previews, tui.Text(" Sample Text:").Bold())
	previews = append(previews, tui.Spacer().MinHeight(1))

	sampleText := "The quick brown fox jumps over the lazy dog"
	gradientSegments := []tui.View{tui.Text(" ")}
	if len(app.gradientColors) == 0 {
		gradientSegments = append(gradientSegments, tui.Text("%s", sampleText))
	} else {
		for i, ch := range sampleText {
			c := app.gradientColors[i%len(app.gradientColors)]
			gradientSegments = append(gradientSegments, tui.Text("%c", ch).FgRGB(c.R, c.G, c.B))
		}
	}
	previews = append(previews, tui.Group(gradientSegments...))

	return tui.Stack(previews...).Padding(1)
}

func (app *PaletteerApp) swatchView(c PaletteColor, selected bool) tui.View {
	rgb := c.ToRGB()
	block := tui.Text("████").FgRGB(rgb.R, rgb.G, rgb.B)
	if selected {
		return tui.Group(tui.Text("["), block, tui.Text("]"))
	}
	return tui.Group(tui.Text(" "), block, tui.Text(" "))
}

func (app *PaletteerApp) blocksView(colors []color.RGB, block string) tui.View {
	if len(colors) == 0 {
		return tui.Text("")
	}
	parts := make([]tui.View, 0, len(colors))
	for _, c := range colors {
		parts = append(parts, tui.Text("%s", block).FgRGB(c.R, c.G, c.B))
	}
	return tui.Group(parts...)
}
