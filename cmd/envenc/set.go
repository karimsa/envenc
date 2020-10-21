package main

import (
	"io/ioutil"
	"os"

	"github.com/karimsa/envenc"
	"github.com/urfave/cli"
)

var cmdSet = cli.Command{
	Name:  "set",
	Usage: "Set a value within a config file.",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "in",
			Usage:    "Path to the input file",
			Required: true,
		},
		&cli.StringFlag{
			Name:  "format",
			Usage: "Format of the input and output files (json, yaml, .env)",
			Value: "",
		},
		&cli.StringFlag{
			Name:  "strategy",
			Usage: "Encryption/decryption type (symmetric, asymmetric, or keyring)",
			Value: "symmetric",
		},
		&cli.StringFlag{
			Name:  "unsafe-passphrase",
			Usage: "Unsafely pass the passphrase for symmetric encryption",
			Value: "",
		},
		&cli.StringFlag{
			Name:     "key",
			Usage:    "Target key path to find secure value",
			Required: true,
		},
		&cli.StringFlag{
			Name: "value",
			Usage: "Target value to set at key path",
			Required: true,
		},
		flagLogLevel,
	},
	Action: func(ctx *cli.Context) error {
		format := ctx.String("format")
		inPath := ctx.String("in")
		key := ctx.String("key")
		value := ctx.String("value")

		if format == "" {
			format = getFormatFromPath(inPath)
		}

		data, err := ioutil.ReadFile(inPath)
		if err != nil {
			return err
		}

		cipher, err := getCipher(ctx)
		if err != nil {
			return err
		}

		securePaths := make(map[string]bool, 1)
		securePaths[key] = true

		envFile, err := envenc.Open(envenc.OpenEnvOptions{
			Format:      format,
			Data:        data,
			Cipher:      cipher,
			SecurePaths: securePaths,
		})
		if err != nil {
			return err
		}

		err = envFile.Set(key, value)
		if err != nil {
			return err
		}

		return envFile.ExportFile(format, inPath, os.O_RDWR)
	},
}
