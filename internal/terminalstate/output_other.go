//go:build !darwin && !linux

package terminalstate

import "os"

func drain(*os.File) error { return nil }
