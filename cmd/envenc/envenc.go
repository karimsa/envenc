package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/howeyc/gopass"
	"github.com/karimsa/envenc"
	"github.com/karimsa/envenc/internal/encrypt"
	"github.com/urfave/cli"
)

const (
	CURSOR_LEFT = "\u001b[G"
)

func getFormatFromPath(path string) string {
	ext := path[strings.LastIndexByte(path, '.'):]
	return ext
}

func getCipher(ctx *cli.Context) (envenc.SimpleCipher, error) {
	strategy := ctx.String("strategy")

	if strategy == "symmetric" {
		// 1) Read from flags
		if pass := ctx.String("unsafe-passphrase"); len(pass) != 0 {
			return encrypt.NewSymmetricCipher([]byte(pass)), nil
		}

		// 2) Read from env
		if pass := os.Getenv("ENVENC_PASSPHRASE"); len(pass) != 0 {
			return encrypt.NewSymmetricCipher([]byte(pass)), nil
		}

		// 3) Read from stdin
		fmt.Fprintf(os.Stderr, "Passphrase: ")
		pass, err := gopass.GetPasswdMasked()
		if err != nil {
			return nil, err
		}
		return encrypt.NewSymmetricCipher(pass), nil
	}

	return nil, fmt.Errorf("Unsupported strategy: %s", strategy)
}

func main() {
	app := &cli.App{
		Name:  "envenc",
		Usage: "Manage secrets in config files.",
		Commands: []cli.Command{
			{
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

					envFile, err := envenc.New(envenc.NewEnvOptions{
						Format: format,
						Data:   data,
						Cipher: cipher,
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
			},
			{
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

					envFile, err := envenc.Open(envenc.OpenEnvOptions{
						Format:      format,
						Data:        data,
						Cipher:      cipher,
						SecurePaths: securePaths,
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
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}
