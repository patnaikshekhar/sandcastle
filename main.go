package main

import (
	"log"
	"os"
	"sandcastle/internal/actions"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "sc",
		Usage: "Sandcastle is a code interpreter for running untrusted code",
		Commands: []*cli.Command{
			{
				Name:    "run",
				Aliases: []string{"r"},
				Usage:   "Run code",
				Action:  actions.Run,
			},
			{
				Name:    "start",
				Aliases: []string{"s"},
				Usage:   "Start the sandbox",
				Action:  actions.Start,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
