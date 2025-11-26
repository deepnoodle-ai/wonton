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
	Name        string `json:"name"`
	Company     string `json:"company"`
	Location    string `json:"location"`
	Bio         string `json:"bio"`
	PublicRepos int    `json:"public_repos"`
	Followers   int    `json:"followers"`
	Following   int    `json:"following"`
}

// DataResponse is a custom event that carries HTTP response data
type DataResponse struct {
	User     *GitHubUser
	Username string
}

func (d DataResponse) Timestamp() time.Time { return time.Now() }

// HTTPApp demonstrates async HTTP requests that don't block the UI
type HTTPApp struct {
	loading bool
	data    *GitHubUser
	error   error
}

func (app *HTTPApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		if e.Key == gooey.KeyEscape || e.Key == gooey.KeyCtrlC {
			return []gooey.Cmd{gooey.Quit()}
		}
		switch e.Rune {
		case '1':
			app.loading = true
			app.error = nil
			return []gooey.Cmd{FetchGitHubUser("golang")}
		case '2':
			app.loading = true
			app.error = nil
			return []gooey.Cmd{FetchGitHubUser("torvalds")}
		case '3':
			app.loading = true
			app.error = nil
			return []gooey.Cmd{FetchGitHubUser("antirez")}
		case 'c', 'C':
			app.data = nil
			app.error = nil
		case 'q', 'Q':
			return []gooey.Cmd{gooey.Quit()}
		}

	case DataResponse:
		app.loading = false
		app.data = e.User

	case gooey.ErrorEvent:
		app.loading = false
		app.error = e.Err
	}

	return nil
}

func (app *HTTPApp) Render(frame gooey.RenderFrame) {
	width, height := frame.Size()

	// Styles
	title := gooey.NewStyle().WithBold().WithForeground(gooey.ColorCyan)
	dim := gooey.NewStyle().WithForeground(gooey.ColorBrightBlack)
	key := gooey.NewStyle().WithForeground(gooey.ColorYellow)
	label := gooey.NewStyle().WithForeground(gooey.ColorWhite)
	value := gooey.NewStyle().WithForeground(gooey.ColorGreen)
	errStyle := gooey.NewStyle().WithForeground(gooey.ColorRed)

	// Clear
	frame.FillStyled(0, 0, width, height, ' ', gooey.NewStyle())

	// Title
	frame.PrintStyled(2, 1, "GitHub User Lookup", title)

	// Keys
	frame.PrintStyled(2, 3, "[1]", key)
	frame.PrintStyled(6, 3, "golang", label)
	frame.PrintStyled(15, 3, "[2]", key)
	frame.PrintStyled(19, 3, "torvalds", label)
	frame.PrintStyled(30, 3, "[3]", key)
	frame.PrintStyled(34, 3, "antirez", label)
	frame.PrintStyled(44, 3, "[c]", key)
	frame.PrintStyled(48, 3, "clear", label)
	frame.PrintStyled(56, 3, "[q]", key)
	frame.PrintStyled(60, 3, "quit", label)

	// Content area
	y := 5
	if app.loading {
		frame.PrintStyled(2, y, "Loading...", dim)
	} else if app.error != nil {
		frame.PrintStyled(2, y, fmt.Sprintf("Error: %v", app.error), errStyle)
	} else if app.data != nil {
		user := app.data

		frame.PrintStyled(2, y, user.Login, title)
		if user.Name != "" {
			frame.PrintStyled(2+len(user.Login)+1, y, user.Name, dim)
		}
		y += 2

		if user.Bio != "" {
			bio := user.Bio
			if len(bio) > width-4 {
				bio = bio[:width-7] + "..."
			}
			frame.PrintStyled(2, y, bio, label)
			y += 2
		}

		if user.Location != "" {
			frame.PrintStyled(2, y, "Location:", dim)
			frame.PrintStyled(12, y, user.Location, value)
			y++
		}
		if user.Company != "" {
			frame.PrintStyled(2, y, "Company:", dim)
			frame.PrintStyled(12, y, user.Company, value)
			y++
		}
		y++

		frame.PrintStyled(2, y, fmt.Sprintf("Repos: %d", user.PublicRepos), value)
		frame.PrintStyled(16, y, fmt.Sprintf("Followers: %d", user.Followers), value)
		frame.PrintStyled(34, y, fmt.Sprintf("Following: %d", user.Following), value)
	} else {
		frame.PrintStyled(2, y, "Press 1, 2, or 3 to fetch a user", dim)
	}

	// Footer
	frame.PrintStyled(2, height-1, "Async HTTP demo - requests don't block the UI", dim)
}

// FetchGitHubUser fetches a GitHub user profile asynchronously
func FetchGitHubUser(username string) gooey.Cmd {
	return func() gooey.Event {
		url := fmt.Sprintf("https://api.github.com/users/%s", username)
		resp, err := http.Get(url)
		if err != nil {
			return gooey.ErrorEvent{Time: time.Now(), Err: err}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return gooey.ErrorEvent{
				Time: time.Now(),
				Err:  fmt.Errorf("HTTP %d", resp.StatusCode),
			}
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return gooey.ErrorEvent{Time: time.Now(), Err: err}
		}

		var user GitHubUser
		if err := json.Unmarshal(body, &user); err != nil {
			return gooey.ErrorEvent{Time: time.Now(), Err: err}
		}

		return DataResponse{User: &user, Username: username}
	}
}

func main() {
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create terminal: %v\n", err)
		return
	}
	defer terminal.Close()

	app := &HTTPApp{}
	runtime := gooey.NewRuntime(terminal, app, 30)
	if err := runtime.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
	}
}
