package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/deepnoodle-ai/gooey"
)

// GitHubUser represents a GitHub user profile response
type GitHubUser struct {
	Login       string `json:"login"`
	ID          int    `json:"id"`
	AvatarURL   string `json:"avatar_url"`
	Name        string `json:"name"`
	Company     string `json:"company"`
	Blog        string `json:"blog"`
	Location    string `json:"location"`
	Email       string `json:"email"`
	Bio         string `json:"bio"`
	PublicRepos int    `json:"public_repos"`
	Followers   int    `json:"followers"`
	Following   int    `json:"following"`
}

// DataResponse is a custom event that carries HTTP response data
type DataResponse struct {
	User *GitHubUser
	Raw  string // Raw response for display
}

// Implement Event interface
func (d DataResponse) Timestamp() time.Time {
	return time.Now()
}

// HTTPApp demonstrates async HTTP requests that don't block the UI
type HTTPApp struct {
	loading     bool
	data        *GitHubUser
	error       error
	lastRequest string
	history     []string // List of previous fetches
}

// HandleEvent processes events in the application
func (app *HTTPApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		switch e.Rune {
		case 'f', 'F':
			// Fetch data when 'f' is pressed
			app.loading = true
			app.error = nil
			return []gooey.Cmd{FetchGitHubUser("golang")}

		case 'g', 'G':
			// Fetch different user
			app.loading = true
			app.error = nil
			return []gooey.Cmd{FetchGitHubUser("torvalds")}

		case 'c', 'C':
			// Clear data
			app.data = nil
			app.error = nil
			app.lastRequest = ""
			return nil

		case 'q', 'Q':
			// Quit
			return []gooey.Cmd{gooey.Quit()}
		}

	case DataResponse:
		// Handle successful response
		app.loading = false
		app.data = e.User
		if e.Raw != "" {
			app.lastRequest = e.Raw
			app.history = append(app.history, e.Raw)
			if len(app.history) > 5 {
				app.history = app.history[len(app.history)-5:]
			}
		}

	case gooey.ErrorEvent:
		// Handle error from HTTP request
		app.loading = false
		app.error = e.Err

	case gooey.TickEvent:
		// Could add animation effects here
	}

	return nil
}

// Render draws the application UI
func (app *HTTPApp) Render(frame gooey.RenderFrame) {
	_, height := frame.Size()

	// Title
	titleStyle := gooey.NewStyle().WithBold().WithForeground(gooey.ColorCyan)
	frame.PrintStyled(2, 1, "GitHub User Fetcher", titleStyle)

	// Instructions
	instructionStyle := gooey.NewStyle().WithForeground(gooey.ColorYellow)
	frame.PrintStyled(2, 3, "Press 'f' to fetch golang user  |  Press 'g' to fetch Linus Torvalds", instructionStyle)
	frame.PrintStyled(2, 4, "Press 'c' to clear  |  Press 'q' to quit", instructionStyle)

	// Loading indicator
	if app.loading {
		spinnerStyle := gooey.NewStyle().WithForeground(gooey.ColorGreen).WithBold()
		spinner := []rune{'|', '/', '-', '\\'}
		frame.PrintStyled(2, 6, fmt.Sprintf("Loading... %c", spinner[0]), spinnerStyle)
	} else if app.error != nil {
		// Error display
		errorStyle := gooey.NewStyle().WithForeground(gooey.ColorRed).WithBold()
		frame.PrintStyled(2, 6, fmt.Sprintf("Error: %v", app.error), errorStyle)
	} else if app.data != nil {
		// Display user data
		app.renderUserData(frame)
	} else {
		// Empty state
		emptyStyle := gooey.NewStyle().WithForeground(gooey.ColorBrightBlack).WithDim()
		frame.PrintStyled(2, 6, "Press 'f' or 'g' to fetch a user", emptyStyle)
	}

	// Display history at the bottom
	if len(app.history) > 0 {
		historyStyle := gooey.NewStyle().WithForeground(gooey.ColorMagenta).WithDim()
		historyY := height - 1 - len(app.history)
		frame.PrintStyled(2, historyY, "Recent requests:", historyStyle)
		for i, h := range app.history {
			frame.PrintStyled(4, historyY+1+i, fmt.Sprintf("• %s", h), historyStyle)
		}
	}

	// Footer
	footerStyle := gooey.NewStyle().WithForeground(gooey.ColorBrightBlack).WithDim()
	frame.PrintStyled(2, height-1, "Message-Driven HTTP Example - async requests don't block the UI", footerStyle)
}

// renderUserData displays the fetched user information
func (app *HTTPApp) renderUserData(frame gooey.RenderFrame) {
	user := app.data
	if user == nil {
		return
	}

	y := 7
	dataStyle := gooey.NewStyle().WithForeground(gooey.ColorGreen)
	labelStyle := gooey.NewStyle().WithForeground(gooey.ColorCyan).WithBold()

	// Display user information
	frame.PrintStyled(2, y, "User Profile:", labelStyle)
	y++

	frame.PrintStyled(4, y, fmt.Sprintf("Login: %s", user.Login), dataStyle)
	y++

	if user.Name != "" {
		frame.PrintStyled(4, y, fmt.Sprintf("Name: %s", user.Name), dataStyle)
		y++
	}

	if user.Location != "" {
		frame.PrintStyled(4, y, fmt.Sprintf("Location: %s", user.Location), dataStyle)
		y++
	}

	if user.Company != "" {
		frame.PrintStyled(4, y, fmt.Sprintf("Company: %s", user.Company), dataStyle)
		y++
	}

	if user.Bio != "" {
		bio := user.Bio
		if len(bio) > 60 {
			bio = bio[:60] + "..."
		}
		frame.PrintStyled(4, y, fmt.Sprintf("Bio: %s", bio), dataStyle)
		y++
	}

	y++ // Blank line

	// Statistics
	frame.PrintStyled(4, y, fmt.Sprintf("Public Repos: %d", user.PublicRepos), dataStyle)
	y++
	frame.PrintStyled(4, y, fmt.Sprintf("Followers: %d", user.Followers), dataStyle)
	y++
	frame.PrintStyled(4, y, fmt.Sprintf("Following: %d", user.Following), dataStyle)
}

// FetchGitHubUser is a command that fetches a GitHub user profile
// This runs in a separate goroutine and doesn't block the UI
func FetchGitHubUser(username string) gooey.Cmd {
	return func() gooey.Event {
		// Perform the HTTP request (may take time)
		url := fmt.Sprintf("https://api.github.com/users/%s", username)
		resp, err := http.Get(url)
		if err != nil {
			return gooey.ErrorEvent{
				Time:  time.Now(),
				Err:   err,
				Cause: fmt.Sprintf("failed to fetch user '%s'", username),
			}
		}
		defer resp.Body.Close()

		// Check HTTP status
		if resp.StatusCode != http.StatusOK {
			return gooey.ErrorEvent{
				Time:  time.Now(),
				Err:   fmt.Errorf("HTTP %d: user not found", resp.StatusCode),
				Cause: fmt.Sprintf("failed to fetch user '%s'", username),
			}
		}

		// Read response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return gooey.ErrorEvent{
				Time:  time.Now(),
				Err:   err,
				Cause: "failed to read response body",
			}
		}

		// Parse JSON
		var user GitHubUser
		err = json.Unmarshal(body, &user)
		if err != nil {
			return gooey.ErrorEvent{
				Time:  time.Now(),
				Err:   err,
				Cause: "failed to parse JSON response",
			}
		}

		// Return successful response
		return DataResponse{
			User: &user,
			Raw:  fmt.Sprintf("Fetched %s at %s", username, time.Now().Format("15:04:05")),
		}
	}
}

func main() {
	// Initialize terminal
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create terminal: %v\n", err)
		return
	}
	defer terminal.Close()

	// Create application
	app := &HTTPApp{}

	// Create and run runtime
	runtime := gooey.NewRuntime(terminal, app, 30)
	if err := runtime.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
	}
}
