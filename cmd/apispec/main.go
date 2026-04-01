// Command apispec generates OpenAPI specs from Go source at build time.
//
// Usage:
//
//	apispec gen [-c config.yaml] [-o openapi.yaml]
//
// Defaults: -c apispec.yaml, -o openapi.yaml
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/clarktrimble/apispec/static"
)

var (
	version string
	release string
)

func main() {

	genCmd := flag.NewFlagSet("gen", flag.ExitOnError)
	cfgPath := genCmd.String("c", "apispec.yaml", "config file path")
	outPath := genCmd.String("o", "openapi.yaml", "output file path")

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: apispec <gen|version> [flags]")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "gen":
		genCmd.Parse(os.Args[2:])

		err := static.Generate(*cfgPath, *outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("wrote %s\n", *outPath)
	case "version":
		fmt.Printf("apispec %s (release: %s)\n", version, release)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
