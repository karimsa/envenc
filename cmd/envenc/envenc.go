package main

import (
	"os"
	"fmt"
	"strings"
	"io/ioutil"

	"github.com/urfave/cli"
	"github.com/howeyc/gopass"
	"github.com/karimsa/envenc"
	"github.com/karimsa/envenc/internal/encrypt"
)

const (
	CURSOR_LEFT = "\u001b[G"
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
						Usage: "Format of the input and output files (json, yaml, .env)",
						Value: "",
					},
					&cli.StringFlag{
						Name: "strategy",
						Usage: "Encryption/decryption type (symmetric, asymmetric, or keyring)",
						Value: "symmetric",
					},
					&cli.StringFlag{
						Name: "unsafe-passphrase",
						Usage: "Unsafely pass the passphrase for symmetric encryption",
						Value: "",
					},
					&cli.StringSliceFlag{
						Name: "key",
						Usage: "Target key path to find secure value",
						Required: true,
					},
				},
				Action: func(ctx *cli.Context) error {
					format := ctx.String("format")
					if format == "" {
						format = getFormatFromPath(ctx.String("in"))
					}

					data, err := ioutil.ReadFile(ctx.String("in"))
					if err != nil {
						return err
					}

					var cipher envenc.SimpleCipher
					strategy := ctx.String("strategy")

					switch strategy {
					case "symmetric":
						passphrase := []byte(ctx.String("unsafe-passphrase"))
						if len(passphrase) == 0 {
							passphrase = []byte(os.Getenv("ENVENC_PASSPHRASE"))
						}
						if len(passphrase) == 0 {
							fmt.Fprintf(os.Stderr, "Passphrase: ")
							passphrase, err = gopass.GetPasswdMasked()
							if err != nil {
								return err
							}
						}
						cipher, err = encrypt.NewSymmetricCipher(passphrase)
						if err != nil {
							return err
						}

					default:
						return fmt.Errorf("Unsupported encryption strategy given: %s", strategy)
					}

					envFile, err := envenc.New(envenc.NewEnvOptions{
						Format: format,
						Data: data,
						Cipher: cipher,
					})
					if err != nil {
						return err
					}

					for _, path := range ctx.StringSlice("key") {
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

					outFile, err = os.OpenFile(outPath, os.O_EXCL, 0755)
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
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}
