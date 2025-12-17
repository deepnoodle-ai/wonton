package git

import (
	"context"
	"strings"
)

// ConfigScope specifies the scope for git config operations.
//
// Git configuration is hierarchical with three levels: system (all users),
// global (current user), and local (current repository). More specific
// scopes override broader ones.
type ConfigScope string

const (
	// ConfigScopeLocal is repository-specific config (.git/config).
	ConfigScopeLocal ConfigScope = "local"
	// ConfigScopeGlobal is user-wide config (~/.gitconfig).
	ConfigScopeGlobal ConfigScope = "global"
	// ConfigScopeSystem is system-wide config (/etc/gitconfig).
	ConfigScopeSystem ConfigScope = "system"
)

// Config reads a git config value from any scope (local, global, or system).
//
// Returns an empty string if the key is not found. The search follows git's
// normal precedence: local overrides global, global overrides system.
//
// Example:
//
//	branch, err := repo.Config(ctx, "branch.main.remote")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if branch != "" {
//	    fmt.Printf("Remote for main branch: %s\n", branch)
//	}
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
//
// Use this to query config at a specific level without considering other scopes.
// Returns an empty string if the key is not found in the specified scope.
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

// ConfigList returns all config values as a map of key-value pairs.
//
// This includes configuration from all scopes (system, global, and local)
// with the effective values after applying git's precedence rules.
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

// UserName returns the configured user.name for commits.
// Returns an empty string if not configured.
func (r *Repository) UserName(ctx context.Context) (string, error) {
	return r.Config(ctx, "user.name")
}

// UserEmail returns the configured user.email for commits.
// Returns an empty string if not configured.
func (r *Repository) UserEmail(ctx context.Context) (string, error) {
	return r.Config(ctx, "user.email")
}

// User returns the configured user name and email as a Person.
// Returns nil if neither user.name nor user.email is configured.
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
