package main

import (
	"os"
	"fmt"
	"strings"

	"github.com/urfave/cli"
)

func getFormatFromPath(path string) string {
	ext := path[strings.LastIndexByte(path, '.'):]
	return ext
}

func main() {
	app := &cli.App{
		Name: "envenc",
		Usage: "Manage secrets in config files.",
		Commands: []cli.Command{
			{
				Name: "encrypt",
				Usage: "Encrypt values in a given file",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name: "in",
						Usage: "Path to the input file",
						Required: true,
					},
					&cli.StringFlag{
						Name: "out",
						Usage: "Path to the output file",
						Required: true,
					},
					&cli.StringFlag{
						Name: "format",
						Usage: "Specify the format of the input and output files",
						Value: "",
					},
				},
				Action: func(ctx *cli.Context) error {
					format := ctx.String("format")
					if format == "" {
						format = getFormatFromPath(ctx.String("in"))
					}

					fmt.Printf("format: %s\n", format)

					return nil
				},
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}
