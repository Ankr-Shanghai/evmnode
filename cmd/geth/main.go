package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/ethereum/go-ethereum/cmd/geth/utils"
	"github.com/urfave/cli/v2"
)

var app = &cli.App{}

func init() {
	app.Name = "geth"
	app.Usage = "EVM node command line interface"
	app.Copyright = "Copyright 2013-2023 The go-ethereum/BSC/Ankr Authors"
	app.Commands = []*cli.Command{
		initCmd,
		startCmd,
		importCmd,
		rpccmd,
		versionCommand,
	}
	app.Flags = []cli.Flag{
		utils.Backend,
		utils.DbHost,
		utils.DbPort,
	}

	sort.Sort(cli.CommandsByName(app.Commands))

	setHelpTemplate(app)
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}
}

func setHelpTemplate(app *cli.App) {
	app.CustomAppHelpTemplate = `NAME:
  {{.Name}} - {{.Usage}}
USAGE:
  {{.HelpName}} {{if .VisibleFlags}}[global options]{{end}}{{if .Commands}} command [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}
  {{if len .Authors}}
AUTHOR:
  {{range .Authors}}{{ . }}{{end}}
  {{end}}{{if .Commands}}
COMMANDS:
 {{range .Commands}}{{if not .HideHelp}}   {{join .Names ","}}{{ "\t"}}{{.Usage}}{{ "\n" }}{{end}}{{end}}{{end}}{{if .VisibleFlags}}
GLOBAL OPTIONS:
  {{range .VisibleFlags}}{{.}}
  {{end}}{{end}}{{if .Copyright }}
COPYRIGHT:
  {{.Copyright}}
  {{end}}{{if .Version}}
VERSION:
  {{.Version}}
{{end}}
 `
}
