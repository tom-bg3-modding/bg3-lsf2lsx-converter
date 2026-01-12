package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	var inputFile = flag.String("i", "", "Input LSF file path")
	var outputFile = flag.String("o", "", "Output LSX file path (optional, defaults to stdout)")
	flag.Parse()

	// For git textconv, accept file path as positional argument
	args := flag.Args()
	if len(args) > 0 && *inputFile == "" {
		*inputFile = args[0]
	}

	if *inputFile == "" {
		fmt.Fprintf(os.Stderr, "Error: input file path is required\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <input-file>\n", os.Args[0])
		flag.Usage()
		os.Exit(1)
	}

	// Read LSF file
	resource, err := ReadLSF(*inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading LSF file: %v\n", err)
		os.Exit(1)
	}

	// Write LSX to stdout or file
	if *outputFile == "" {
		// Write to stdout (for git textconv)
		err = WriteLSXToWriter(os.Stdout, resource)
	} else {
		err = WriteLSX(*outputFile, resource)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing LSX: %v\n", err)
		os.Exit(1)
	}
}
