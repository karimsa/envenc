package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/karimsa/envenc"
	"github.com/urfave/cli"
)

var cmdEncrypt = cli.Command{
	Name:  "encrypt",
	Usage: "Encrypt values in a given file",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "in",
			Usage:    "Path to the input file",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "out",
			Usage:    "Path to the output file",
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

		logLevel, err := getLogLevel(ctx)
		if err != nil {
			return err
		}

		envFile, err := envenc.New(envenc.NewEnvOptions{
			Format: format,
			Data:   data,
			Cipher: cipher,
			LogLevel: logLevel,
		})
		if err != nil {
			return err
		}

		for _, path := range inputPaths {
			envFile.Touch(path)
		}

		buff, err := envFile.Export(format)
		if err != nil {
			return err
		}

		var outFile *os.File
		outPath := ctx.String("out")
		switch outPath {
		case "/dev/stdout":
			fmt.Printf(string(buff))
			return nil
		case "/dev/stderr":
			fmt.Fprintf(os.Stderr, string(buff))
			return nil
		}

		// For in-place edits, overwrite the file
		outFileMode := os.O_EXCL
		if outPath == inPath {
			outFileMode = os.O_WRONLY | os.O_TRUNC
		}

		outFile, err = os.OpenFile(outPath, outFileMode, 0755)
		if err != nil {
			return err
		}

		_, err = outFile.Write(buff)
		if err != nil {
			return err
		}

		err = outFile.Sync()
		if err != nil {
			return err
		}

		return nil
	},
}
