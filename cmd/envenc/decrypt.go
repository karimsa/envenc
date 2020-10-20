package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/karimsa/envenc"
	"github.com/urfave/cli"
)

var cmdDecrypt = cli.Command{
	Name:  "decrypt",
	Usage: "Decrypt values from a config file",
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
		&cli.StringSliceFlag{
			Name:     "key",
			Usage:    "Target key path to find secure value",
			Required: true,
		},
		flagLogLevel,
	},
	Action: func(ctx *cli.Context) error {
		format := ctx.String("format")
		inPath := ctx.String("in")
		inputPaths := ctx.StringSlice("key")

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

		securePaths := make(map[string]bool, len(inputPaths))
		for _, path := range inputPaths {
			securePaths[path] = true
		}

		logLevel, err := getLogLevel(ctx)
		if err != nil {
			return err
		}

		envFile, err := envenc.Open(envenc.OpenEnvOptions{
			Format:      format,
			Data:        data,
			Cipher:      cipher,
			SecurePaths: securePaths,
			LogLevel: logLevel,
		})
		if err != nil {
			return err
		}

		buff, err := envFile.UnsafeRawExport(format)
		if err != nil {
			return err
		}

		// For decryption, always write to stderr
		fmt.Fprintf(os.Stderr, string(buff))

		return nil
	},
}
