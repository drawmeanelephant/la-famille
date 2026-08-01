package main

import (
	"encoding/json"
	"fmt"
	"io"
	"runtime"
)

// These variables are deliberately simple strings so release builds can set
// them with documented -ldflags. Development builds remain useful and
// identify themselves as dev instead of pretending to be a tagged release.
var (
	buildVersion = "dev"
	buildCommit  = "unknown"
	buildDate    = "unknown"
)

type buildInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	Target    string `json:"target"`
	GoVersion string `json:"go_version"`
}

func currentBuildInfo() buildInfo {
	return buildInfo{
		Version:   buildVersion,
		Commit:    buildCommit,
		BuildDate: buildDate,
		Target:    runtime.GOOS + "/" + runtime.GOARCH,
		GoVersion: runtime.Version(),
	}
}

func writeBuildInfo(w io.Writer, asJSON bool) error {
	info := currentBuildInfo()
	if asJSON {
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(info)
	}

	_, err := fmt.Fprintf(w, "La Famille %s\ncommit: %s\nbuilt: %s\ntarget: %s\ngo: %s\n",
		info.Version, info.Commit, info.BuildDate, info.Target, info.GoVersion)
	return err
}

func argsRequestVersion(args []string) bool {
	for _, arg := range args {
		if arg == "--version" {
			return true
		}
		if arg == "--" {
			return false
		}
	}
	return false
}
