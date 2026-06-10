// Package main demonstrates async commands: HTTP requests run in tui.Cmd
// functions and deliver results back to HandleEvent as custom events, so
// the UI never blocks.
//
// Run with: go run ./examples/tui/tui_command
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/deepnoodle-ai/wonton/tui"
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
	Time time.Time
	User *GitHubUser
}

func (d DataResponse) Timestamp() time.Time { return d.Time }

// HTTPApp demonstrates async HTTP requests that don't block the UI
type HTTPApp struct {
	loading bool
	data    *GitHubUser
	err     error
}

func (app *HTTPApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		if e.Key == tui.KeyEscape || e.Key == tui.KeyCtrlC {
			return []tui.Cmd{tui.Quit()}
		}
		switch e.Rune {
		case '1':
			app.loading = true
			app.err = nil
			return []tui.Cmd{FetchGitHubUser("golang")}
		case '2':
			app.loading = true
			app.err = nil
			return []tui.Cmd{FetchGitHubUser("torvalds")}
		case '3':
			app.loading = true
			app.err = nil
			return []tui.Cmd{FetchGitHubUser("antirez")}
		case 'c', 'C':
			app.data = nil
			app.err = nil
		case 'q', 'Q':
			return []tui.Cmd{tui.Quit()}
		}

	case DataResponse:
		app.loading = false
		app.data = e.User

	case tui.ErrorEvent:
		app.loading = false
		app.err = e.Err
	}

	return nil
}

func (app *HTTPApp) View() tui.View {
	return tui.Stack(
		// Title
		tui.Text("GitHub User Lookup").Bold().Fg(tui.ColorCyan),
		tui.Spacer().MinHeight(1),

		// Keys
		tui.Group(
			tui.Text("[1]").Fg(tui.ColorYellow),
			tui.Text("golang").Fg(tui.ColorWhite),
			tui.Spacer().MinWidth(2),
			tui.Text("[2]").Fg(tui.ColorYellow),
			tui.Text("torvalds").Fg(tui.ColorWhite),
			tui.Spacer().MinWidth(2),
			tui.Text("[3]").Fg(tui.ColorYellow),
			tui.Text("antirez").Fg(tui.ColorWhite),
			tui.Spacer().MinWidth(2),
			tui.Text("[c]").Fg(tui.ColorYellow),
			tui.Text("clear").Fg(tui.ColorWhite),
			tui.Spacer().MinWidth(2),
			tui.Text("[q]").Fg(tui.ColorYellow),
			tui.Text("quit").Fg(tui.ColorWhite),
		).Gap(1),
		tui.Spacer().MinHeight(1),

		// Content area
		app.contentView(),

		tui.Spacer(),

		// Footer
		tui.Text("Async HTTP demo - requests don't block the UI").Fg(tui.ColorBrightBlack),
	).Padding(2)
}

func (app *HTTPApp) contentView() tui.View {
	if app.loading {
		return tui.Text("Loading...").Fg(tui.ColorBrightBlack)
	}

	if app.err != nil {
		return tui.Text("Error: %v", app.err).Fg(tui.ColorRed)
	}

	if app.data != nil {
		user := app.data

		// Build user details
		var details []tui.View

		// Header with login and name
		details = append(details, tui.Group(
			tui.Text("%s", user.Login).Bold().Fg(tui.ColorCyan),
			tui.If(user.Name != "", tui.Text("%s", user.Name).Fg(tui.ColorBrightBlack)),
		).Gap(1))

		details = append(details, tui.Spacer().MinHeight(1))

		// Bio
		if user.Bio != "" {
			details = append(details, tui.Text("%s", user.Bio).Fg(tui.ColorWhite).MaxWidth(76))
			details = append(details, tui.Spacer().MinHeight(1))
		}

		// Location and Company
		if user.Location != "" {
			details = append(details, tui.Group(
				tui.Text("Location:").Fg(tui.ColorBrightBlack),
				tui.Text("%s", user.Location).Fg(tui.ColorGreen),
			).Gap(1))
		}
		if user.Company != "" {
			details = append(details, tui.Group(
				tui.Text("Company:").Fg(tui.ColorBrightBlack),
				tui.Text("%s", user.Company).Fg(tui.ColorGreen),
			).Gap(1))
		}

		details = append(details, tui.Spacer().MinHeight(1))

		// Stats
		details = append(details, tui.Group(
			tui.Text("Repos: %d", user.PublicRepos).Fg(tui.ColorGreen),
			tui.Text("Followers: %d", user.Followers).Fg(tui.ColorGreen),
			tui.Text("Following: %d", user.Following).Fg(tui.ColorGreen),
		).Gap(2))

		return tui.Stack(details...)
	}

	return tui.Text("Press 1, 2, or 3 to fetch a user").Fg(tui.ColorBrightBlack)
}

// httpClient is used for all requests; the timeout ensures a stalled request
// eventually produces an ErrorEvent instead of loading forever.
var httpClient = &http.Client{Timeout: 10 * time.Second}

// FetchGitHubUser fetches a GitHub user profile asynchronously
func FetchGitHubUser(username string) tui.Cmd {
	return func() tui.Event {
		url := fmt.Sprintf("https://api.github.com/users/%s", username)
		resp, err := httpClient.Get(url)
		if err != nil {
			return tui.ErrorEvent{Time: time.Now(), Err: err}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return tui.ErrorEvent{
				Time: time.Now(),
				Err:  fmt.Errorf("HTTP %d", resp.StatusCode),
			}
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return tui.ErrorEvent{Time: time.Now(), Err: err}
		}

		var user GitHubUser
		if err := json.Unmarshal(body, &user); err != nil {
			return tui.ErrorEvent{Time: time.Now(), Err: err}
		}

		return DataResponse{Time: time.Now(), User: &user}
	}
}

func main() {
	if err := tui.Run(&HTTPApp{}); err != nil {
		log.Fatal(err)
	}
}
