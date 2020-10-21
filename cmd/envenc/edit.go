package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"fmt"

	"github.com/karimsa/envenc"
	"github.com/urfave/cli"
)

var cmdEdit = cli.Command{
	Name:  "edit",
	Usage: "Edit a file with encrypted values",
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
		&cli.StringFlag{
			Name: "editor",
			Usage: "Text editor to open for temporary file",
			EnvVar: "EDITOR",
			Value: "vi",
		},
		flagLogLevel,
	},
	Action: func(ctx *cli.Context) error {
		format := ctx.String("format")
		inPath := ctx.String("in")
		inputPaths := ctx.StringSlice("key")
		editor := ctx.String("editor")

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

		// Create temporary version for user edits
		tmp, err := ioutil.TempFile("/tmp", "*")
		if err != nil {
			return fmt.Errorf("failed to create temporary file for editing: %s", err)
		}
		defer os.Remove(tmp.Name())

		buff, err := envFile.UnsafeRawExport(format)
		if err != nil {
			return err
		}

		_, err = tmp.Write(buff)
		if err != nil {
			return err
		}

		err = tmp.Sync()
		if err != nil {
			return err
		}

		cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("%s %s", editor, tmp.Name()))
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			return err
		}

		editBuffer, err := ioutil.ReadFile(tmp.Name())
		if err != nil {
			return err
		}

		err = envFile.UpdateFrom(format, editBuffer)
		if err != nil {
			return err
		}

		return envFile.ExportFile(format, inPath, os.O_RDWR)
	},
}
