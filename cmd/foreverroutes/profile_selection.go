package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Battlefeld/ForeverRoutes/route"
)

const automaticProfile = "auto"
const combatProfileHelp = "auto = class profile beside dataset; a profile file; empty = uncalibrated default estimate"

// resolveCombatProfile uses the same bundled class profile for free planning,
// guide comparisons and live replanning, regardless of the working directory.
// Missing profiles keep the explicitly labelled estimate; broken profiles fail
// instead of silently making one class appear faster than another.
func resolveCombatProfile(data, class, selected string) (string, error) {
	if selected != automaticProfile {
		return selected, nil
	}
	if route.ClassBits[class] == 0 {
		return "", fmt.Errorf("combat profile: unknown class %q", class)
	}
	path := filepath.Join(filepath.Dir(data), "profiles", strings.ToLower(class)+".json")
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("combat profile %s: %w", path, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("combat profile %s is a directory", path)
	}
	return path, nil
}
