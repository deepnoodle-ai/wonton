package git

import (
	"context"
	"strings"
)

// ConfigScope specifies the scope for config operations.
type ConfigScope string

const (
	// ConfigScopeLocal is repository-specific config.
	ConfigScopeLocal ConfigScope = "local"
	// ConfigScopeGlobal is user-wide config.
	ConfigScopeGlobal ConfigScope = "global"
	// ConfigScopeSystem is system-wide config.
	ConfigScopeSystem ConfigScope = "system"
)

// Config reads a git config value.
func (r *Repository) Config(ctx context.Context, key string) (string, error) {
	out, err := r.run(ctx, "config", "--get", key)
	if err != nil {
		// Key not found is not an error for our purposes
		if strings.Contains(err.Error(), "exit status 1") {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// ConfigWithScope reads a git config value from a specific scope.
func (r *Repository) ConfigWithScope(ctx context.Context, key string, scope ConfigScope) (string, error) {
	out, err := r.run(ctx, "config", "--"+string(scope), "--get", key)
	if err != nil {
		if strings.Contains(err.Error(), "exit status 1") {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// ConfigList returns all config values as a map.
func (r *Repository) ConfigList(ctx context.Context) (map[string]string, error) {
	out, err := r.run(ctx, "config", "--list")
	if err != nil {
		return nil, err
	}

	config := make(map[string]string)
	for _, line := range strings.Split(string(out), "\n") {
		if idx := strings.Index(line, "="); idx > 0 {
			key := line[:idx]
			value := line[idx+1:]
			config[key] = value
		}
	}

	return config, nil
}

// UserName returns the configured user.name.
func (r *Repository) UserName(ctx context.Context) (string, error) {
	return r.Config(ctx, "user.name")
}

// UserEmail returns the configured user.email.
func (r *Repository) UserEmail(ctx context.Context) (string, error) {
	return r.Config(ctx, "user.email")
}

// User returns the configured user name and email.
func (r *Repository) User(ctx context.Context) (*Person, error) {
	name, err := r.UserName(ctx)
	if err != nil {
		return nil, err
	}
	email, err := r.UserEmail(ctx)
	if err != nil {
		return nil, err
	}
	if name == "" && email == "" {
		return nil, nil
	}
	return &Person{Name: name, Email: email}, nil
}
