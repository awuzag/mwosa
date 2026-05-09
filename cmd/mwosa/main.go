package main

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/ev3rlit/mwosa/cli"
)

var (
	version = ""
	commit  = ""
	date    = ""
)

func main() {
	if err := cli.NewRootCommand(currentBuildInfo()).ExecuteContext(context.Background()); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func currentBuildInfo() cli.BuildInfo {
	info, ok := debug.ReadBuildInfo()
	return buildInfoFrom(version, commit, date, info, ok)
}

func buildInfoFrom(version, commit, date string, info *debug.BuildInfo, ok bool) cli.BuildInfo {
	build := cli.BuildInfo{
		Version: version,
		Commit:  commit,
		Date:    date,
	}
	if !ok || info == nil {
		return build
	}
	if build.Version == "" && info.Main.Version == "(devel)" {
		build.Development = true
	}
	if build.Version == "" && info.Main.Version != "" && info.Main.Version != "(devel)" {
		build.Version = info.Main.Version
	}
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			if build.Commit == "" {
				build.Commit = setting.Value
			}
		case "vcs.time":
			if build.Date == "" {
				build.Date = setting.Value
			}
		}
	}
	return build
}
