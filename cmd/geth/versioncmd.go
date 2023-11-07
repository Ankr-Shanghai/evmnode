package main

import (
	"fmt"
	"runtime"

	"github.com/ethereum/go-ethereum/internal/version"
	"github.com/ethereum/go-ethereum/params"
	"github.com/urfave/cli/v2"
)

const (
	clientIdentifier = "ankr_node" // Client identifier to advertise over the network
)

var versionCommand = &cli.Command{
	Action:    printVersion,
	Name:      "version",
	Usage:     "Print version numbers",
	ArgsUsage: " ",
	Description: `
The output of this command is supposed to be machine-readable.
`,
}

func clientVersion() string {
	return fmt.Sprintf("%s/v%s/%s/%s/%s",
		clientIdentifier,
		params.VersionWithMeta,
		runtime.GOOS,
		runtime.GOARCH,
		runtime.Version(),
	)
}

func printVersion(ctx *cli.Context) error {
	git, _ := version.VCS()

	fmt.Println(clientIdentifier)
	fmt.Println("Version:", params.VersionWithMeta)
	if git.Commit != "" {
		fmt.Println("Git Commit:", git.Commit)
	}
	if git.Date != "" {
		fmt.Println("Git Commit Date:", git.Date)
	}
	fmt.Println("Architecture:", runtime.GOARCH)
	fmt.Println("Go Version:", runtime.Version())
	fmt.Println("Operating System:", runtime.GOOS)
	return nil
}
