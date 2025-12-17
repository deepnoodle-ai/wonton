package git

import (
	"context"
	"strings"
)

// Remotes returns the list of remotes.
func (r *Repository) Remotes(ctx context.Context) ([]Remote, error) {
	out, err := r.run(ctx, "remote", "-v")
	if err != nil {
		return nil, err
	}

	if len(out) == 0 {
		return nil, nil
	}

	// Parse remote -v output which shows each remote twice (fetch and push)
	remoteMap := make(map[string]*Remote)
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		name := parts[0]
		url := parts[1]
		kind := strings.Trim(parts[2], "()")

		remote, exists := remoteMap[name]
		if !exists {
			remote = &Remote{Name: name}
			remoteMap[name] = remote
		}

		if kind == "fetch" {
			remote.FetchURL = url
		} else if kind == "push" {
			remote.PushURL = url
		}
	}

	var remotes []Remote
	for _, remote := range remoteMap {
		remotes = append(remotes, *remote)
	}

	return remotes, nil
}

// Remote returns a specific remote by name.
func (r *Repository) Remote(ctx context.Context, name string) (*Remote, error) {
	remotes, err := r.Remotes(ctx)
	if err != nil {
		return nil, err
	}

	for _, remote := range remotes {
		if remote.Name == name {
			return &remote, nil
		}
	}

	return nil, nil
}

// RemoteURL returns the URL for a remote.
func (r *Repository) RemoteURL(ctx context.Context, name string) (string, error) {
	out, err := r.run(ctx, "remote", "get-url", name)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// OriginURL returns the URL for the origin remote.
func (r *Repository) OriginURL(ctx context.Context) (string, error) {
	return r.RemoteURL(ctx, "origin")
}
