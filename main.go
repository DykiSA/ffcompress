package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	var outputDir *string
	outputDir = flag.String("o", "", "output directory (default: same directory as input)")
	flag.Parse()
	var inputFiles []string
	inputFiles = flag.Args()

	if len(inputFiles) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no input files specified")
		fmt.Fprintln(os.Stderr, "Usage: ffcompress <input-file-or-directory> [input-file...]")
		fmt.Fprintln(os.Stderr, "       ffcompress -o <dir> <input1> <input2> ...")
		os.Exit(1)
	}

	// Collect all input paths (files and directories)
	var allInputs []string
	for _, arg := range inputFiles {
		abs, err := filepath.Abs(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving %s: %v\n", arg, err)
			os.Exit(1)
		}
		info, err := os.Stat(abs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error accessing %s: %v\n", arg, err)
			os.Exit(1)
		}
		if info.IsDir() {
			// Directory: add all supported media files inside
			files, err := filepath.Glob(filepath.Join(abs, "*"))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading directory %s: %v\n", abs, err)
				os.Exit(1)
			}
			for _, f := range files {
				if info, err := os.Stat(f); err == nil && !info.IsDir() {
					allInputs = append(allInputs, f)
				}
			}
		} else {
			allInputs = append(allInputs, abs)
		}
	}

	if len(allInputs) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no input files found")
		os.Exit(1)
	}

	// Determine output directory
	var outDir string
	if *outputDir != "" {
		outDir = *outputDir
	} else if len(allInputs) == 1 {
		// Single file: output next to input
		outDir = filepath.Dir(allInputs[0])
	} else {
		// Multiple: output next to first input
		outDir = filepath.Dir(allInputs[0])
	}

	// Create output directory if needed
	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory %s: %v\n", outDir, err)
		os.Exit(1)
	}

	// Compression settings (mirrors Android FFShare defaults)
	pm := NewParamMaker()

	// Process each file
	total := len(allInputs)
	for i, input := range allInputs {
		fmt.Printf("[%d/%d] Processing: %s\n", i+1, total, input)

		// Determine media type
		mediaType := detectMediaType(input)
		if mediaType == MediaTypeUnknown {
			fmt.Printf("  Skipping unsupported file type: %s\n", input)
			continue
		}

		// Get media info via ffprobe
		mediaInfo := getMediaInfo(input, mediaType)
		if mediaInfo == nil {
			fmt.Printf("  Error: could not read media info for %s\n", input)
			continue
		}

		// Determine output media type
		outputMediaType := determineOutputMediaType(mediaType, input)

		// Generate FFmpeg parameters
		params := pm.Create(input, mediaType, mediaInfo, outputMediaType)
		if params == "" {
			fmt.Printf("  Warning: no parameters generated for %s\n", input)
		}

		// Determine output file path
		ext := filepath.Ext(input)
		base := strings.TrimSuffix(filepath.Base(input), ext)
		datetime := getCurrentDateTime()
			outputFileName := fmt.Sprintf("compressed_%s_%s%s", base, datetime, ext)
		outputPath := filepath.Join(outDir, outputFileName)

		// Build and execute FFmpeg command
		args := buildFFmpegArgs(input, params, outputPath)
		fmt.Printf("  Command: ffmpeg %s\n", strings.Join(args, " "))

		err := executeCommand("ffmpeg", args)
		if err != nil {
			fmt.Printf("  Error processing %s: %v\n", input, err)
		} else {
			size := getOutputSize(outputPath)
			fmt.Printf("  Done: %s (%s)\n", outputPath, formatBytes(size))
		}
		fmt.Println()
	}

	fmt.Printf("Processed %d file(s). Output in: %s\n", total, outDir)
}
