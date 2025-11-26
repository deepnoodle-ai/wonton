package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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

func (app *HTTPApp) View() gooey.View {
	return gooey.VStack(
		// Title
		gooey.Text("GitHub User Lookup").Bold().Fg(gooey.ColorCyan),
		gooey.Spacer().MinHeight(1),

		// Keys
		gooey.HStack(
			gooey.Text("[1]").Fg(gooey.ColorYellow),
			gooey.Text("golang").Fg(gooey.ColorWhite),
			gooey.Spacer().MinWidth(2),
			gooey.Text("[2]").Fg(gooey.ColorYellow),
			gooey.Text("torvalds").Fg(gooey.ColorWhite),
			gooey.Spacer().MinWidth(2),
			gooey.Text("[3]").Fg(gooey.ColorYellow),
			gooey.Text("antirez").Fg(gooey.ColorWhite),
			gooey.Spacer().MinWidth(2),
			gooey.Text("[c]").Fg(gooey.ColorYellow),
			gooey.Text("clear").Fg(gooey.ColorWhite),
			gooey.Spacer().MinWidth(2),
			gooey.Text("[q]").Fg(gooey.ColorYellow),
			gooey.Text("quit").Fg(gooey.ColorWhite),
		).Gap(1),
		gooey.Spacer().MinHeight(1),

		// Content area
		app.contentView(),

		gooey.Spacer(),

		// Footer
		gooey.Text("Async HTTP demo - requests don't block the UI").Fg(gooey.ColorBrightBlack),
	).Padding(2)
}

func (app *HTTPApp) contentView() gooey.View {
	if app.loading {
		return gooey.Text("Loading...").Fg(gooey.ColorBrightBlack)
	}

	if app.error != nil {
		return gooey.Text("Error: %v", app.error).Fg(gooey.ColorRed)
	}

	if app.data != nil {
		user := app.data

		// Build user details
		var details []gooey.View

		// Header with login and name
		details = append(details, gooey.HStack(
			gooey.Text(user.Login).Bold().Fg(gooey.ColorCyan),
			gooey.If(user.Name != "", gooey.Text(user.Name).Fg(gooey.ColorBrightBlack)),
		).Gap(1))

		details = append(details, gooey.Spacer().MinHeight(1))

		// Bio
		if user.Bio != "" {
			details = append(details, gooey.Text(user.Bio).Fg(gooey.ColorWhite).MaxWidth(76))
			details = append(details, gooey.Spacer().MinHeight(1))
		}

		// Location and Company
		if user.Location != "" {
			details = append(details, gooey.HStack(
				gooey.Text("Location:").Fg(gooey.ColorBrightBlack),
				gooey.Text(user.Location).Fg(gooey.ColorGreen),
			).Gap(1))
		}
		if user.Company != "" {
			details = append(details, gooey.HStack(
				gooey.Text("Company:").Fg(gooey.ColorBrightBlack),
				gooey.Text(user.Company).Fg(gooey.ColorGreen),
			).Gap(1))
		}

		details = append(details, gooey.Spacer().MinHeight(1))

		// Stats
		details = append(details, gooey.HStack(
			gooey.Text("Repos: %d", user.PublicRepos).Fg(gooey.ColorGreen),
			gooey.Text("Followers: %d", user.Followers).Fg(gooey.ColorGreen),
			gooey.Text("Following: %d", user.Following).Fg(gooey.ColorGreen),
		).Gap(2))

		return gooey.VStack(details...)
	}

	return gooey.Text("Press 1, 2, or 3 to fetch a user").Fg(gooey.ColorBrightBlack)
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
	if err := gooey.Run(&HTTPApp{}); err != nil {
		log.Fatal(err)
	}
}
