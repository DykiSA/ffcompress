# DECISIONS.md - FFCompress CLI Design Decisions

This document records key design decisions and their rationale for the FFCompress CLI project.

## 1. Language Choice: Go

**Decision:** Use Go (Go 1.23+) for the CLI implementation.

**Rationale:**
- **Single static binary:** Go compiles to a self-contained binary with no interpreter or runtime dependency required at runtime.
- **Cross-platform:** A single source codebase builds for Windows, macOS, and Linux without CGO.
- **Standard library:** `os/exec`, `path/filepath`, `flag` provide all needed functionality without external dependencies.
- **No packaging overhead:** Unlike Python (requires PyInstaller) or Node.js (requires pkg), Go requires no post-build packaging step.

**Trade-offs:**
- Slightly higher compile time compared to scripting languages.
- Larger binary size (~3MB vs ~50KB for a Node.js script), but acceptable for CLI distribution.

## 2. CLI Argument Design

**Decision:** Minimal flags: only `-o`/`--output-dir`.

**Rationale:**
- The Android app has many settings (CRF, codec, etc.) but the CLI requirement is to keep it simple.
- Advanced settings (CRF, codec, bitrate limits) are hardcoded with Android defaults and documented in `strategy.go`.
- Adding more flags would require a config file or environment variables for advanced users.
- Future versions can add flags for CRF, codec, max file size, resolution caps, etc.

**Chosen defaults:**
- CRF: `23` (matches FFShare Android default)
- Preset: `medium` (matches FFShare Android default)
- Video codec: `libx264` (matches FFShare Android default)
- Audio codec: `copy` (lossless stream copy, matches FFShare Android default)
- Video max file size: `0` (disabled)

## 3. Output File Naming

**Decision:** Single file: `compressed_<original>_<datetime>.<ext>`; multiple files: same directory as first input, original names preserved.

**Rationale:**
- The `compressed_` prefix distinguishes output from input.
- The datetime suffix prevents collisions when re-running on the same file.
- For multiple files, keeping original names avoids rename complexity.
- Using the input's directory as the default output location is intuitive (no path confusion).

**Collision handling:**
- If two files have the same base name in different directories, the datetime suffix prevents collision.
- If two files have the same name in the same directory, the datetime suffix still applies because the output is always in the same directory.

## 4. Directory Input Handling

**Decision:** A directory input is expanded to all non-directory files inside, then processed as multiple files.

**Rationale:**
- Users often want to compress all files in a folder.
- The output directory for a directory input defaults to the same directory as the input.
- Files are processed sequentially (not in parallel) for simplicity.

**Trade-offs:**
- No parallelism. Could be added later with `sync.WaitGroup` and goroutines.
- If a directory contains subdirectories, they are not recursed into. This is intentional for simplicity.

## 5. Resolution Scaling Logic

**Decision:** Scale the larger source dimension to `maxRes` while auto-calculating the other dimension to preserve aspect ratio, using FFmpeg's `-2` / `-1` pixel rounding values (faithful port of the Android `FFmpegParamMaker`).

**Rationale:**
- The Android app determines portrait vs. landscape via `resolutionHeight > resolutionWidth`.
- If **portrait** (`height > width`): scale filter is `scale=maxRes:pixelRounding` — width is fixed to `maxRes`, height is auto-calculated.
- If **landscape** (`width >= height`): scale filter is `scale=pixelRounding:maxRes` — height is fixed to `maxRes`, width is auto-calculated.
- The `pixelRounding` value is `-2` for H.264/H.265 codecs (auto-calculate to nearest even number, required by the codec specification) and `-1` for other codecs (auto-calculate with no divisibility constraint).
- This faithfully reproduces the Android app's logic, which correctly preserves aspect ratio rather than forcing a square output.
- The `-2` / `-1` values are valid FFmpeg `scale` filter syntax (not a bug) — they are the recommended approach for aspect-ratio-preserving scaling with even-dimension constraints.

**Example for a 1920×1080 (landscape) video with maxRes=1080:**
```
-vf scale=-2:1080,setsar=1,format=yuv420p
```
This sets height to 1080 and auto-calculates width (1920) maintaining the 16:9 aspect ratio.

**Example for a 1080×1920 (portrait) video with maxRes=1080:**
```
-vf scale=1080:-2,setsar=1,format=yuv420p
```
This sets width to 1080 and auto-calculates height (1920) maintaining the 9:16 aspect ratio.

## 6. Custom Params Override

**Decision:** Custom params (`customVideoParams`, `customImageParams`, `customAudioParams`) are NOT supported in the CLI.

**Rationale:**
- These settings are UI-driven in the Android app and have no UI equivalent in CLI.
- Adding a config file for custom params would add complexity and is not required for the current scope.
- Explicitly documented as a future enhancement in `README.md`.

**Trade-offs:**
- Advanced users cannot use custom FFmpeg filters via CLI.
- This is a known limitation; documented in `README.md` and `DECISIONS.md`.

## 7. FFprobe Invocation

**Decision:** Use `ffprobe` with JSON output for media info (resolution, duration, size, codec).

**Rationale:**
- FFprobe is the standard tool for inspecting media files.
- JSON output is easy to parse in Go.
- Combining multiple queries (`format=` and `stream=`) into one ffprobe call is efficient.

**Alternatives considered:**
- Parsing text output: fragile and locale-dependent.
- Running multiple ffprobe calls: slower and redundant.
- Using FFmpeg libraries directly (libavformat, libswscale): would require CGO and static linking of FFmpeg libraries, which contradicts the "simple single binary" requirement (the binary only wraps ffmpeg CLI, not FFmpeg itself).

## 8. Command Execution

**Decision:** Execute FFmpeg via `os/exec.Command()` with `cmd.Run()`.

**Rationale:**
- `os/exec` is the standard Go way to execute external commands.
- Output is passed to `os.Stdout` and `os.Stderr` so the user sees ffmpeg's progress.
- Error handling uses `c.Run()` which returns an error on non-zero exit code.

**Alternative:** Use `shell.Run()` with a shell string: rejected because shell quoting is error-prone and platform-dependent (Windows vs. Unix).

## 9. Dependency Management

**Decision:** No external Go modules. Uses only standard library (`os`, `os/exec`, `path/filepath`, `flag`, `fmt`, `math`, `strings`, `time`).

**Rationale:**
- Fewer dependencies = fewer security vulnerabilities and less maintenance.
- The project has no network requests or external packages.
- Ensures the build is reproducible without `go mod download`.

**Trade-offs:**
- Some features (like advanced JSON parsing with `encoding/json`) require importing `encoding/json` which is included in the standard library. The current implementation avoids `encoding/json` and uses manual parsing with `fmt.Sscanf` for simplicity.

## 10. Media Type Detection

**Decision:** Detect media type from file extension using a `switch` statement.

**Rationale:**
- Simple and fast.
- No dependency on external libraries.
- Extension-based detection is reliable for well-known file types.

**Limitations:**
- Cannot detect file type from magic bytes (e.g., a renamed MP4 file with .txt extension would be misidentified).
- This limitation is acceptable for the current scope.

## 11. MediaInfo Structure

**Decision:** `MediaInfo` struct holds `Width`, `Height`, `Duration`, `Size`, `Codec`.

**Rationale:**
- The struct is minimal and only includes fields needed for parameter generation.
- Width/Height used for resolution scaling.
- Duration used for bitrate calculation.
- Size (file size in bytes) available for future use.
- Codec used for potential codec-specific handling in the future.

## 12. Error Handling Philosophy

**Decision:** Fail fast with clear error messages to stderr and `os.Exit(1)` on fatal errors.

**Rationale:**
- CLI tools should not silently ignore errors.
- Clear error messages help users understand what went wrong.
- `os.Exit(1)` is appropriate for CLI tools (differs from library error handling).

**Scope:**
- Media type detection errors: skipped with a warning, not fatal.
- FFmpeg execution errors: reported to stderr.
- File I/O errors: fatal (exit with error).

## 13. FFmpeg Command Construction

**Decision:** Pass the filter string without shell escaping (no double quotes).

**Rationale:**
- The command is executed with `os/exec.Command()` which does not use a shell.
- Double quotes around `-vf` parameters are unnecessary and cause issues (the quotes are passed as literal characters).
- Removing quotes from the filter string avoids confusing FFmpeg errors.

**Example correct command:**
```
ffmpeg -y -i input.mp4 -preset medium -crf 23 -c:v libx264 -vf scale=-2:1080,setsar=1,format=yuv420p output.mp4
```

## 14. Testing Strategy

**Decision:** Manual testing with sample files (video + image) before committing.

**Rationale:**
- The project is a CLI wrapper around FFmpeg, not a library.
- Automated unit tests for the strategy logic are possible but not implemented yet.
- Manual testing verifies the full pipeline: input detection → ffprobe → parameter generation → FFmpeg execution → output verification.

**Future:**
- Unit tests for `ParamMaker.Create()` (strategy logic)
- Integration tests with real sample files
- CI pipeline for cross-platform builds

## 15. File Path Handling with Spaces

**Decision:** Build FFmpeg commands as argument arrays (`[]string`) via `exec.Command(name, args...)`, not as shell strings parsed with `strings.Fields()`.

**Rationale:**
- The original `executeCommand` used `strings.Fields(cmd)` to split a command string like `ffmpeg -y -i /path/with spaces/file.mp4 ...`. This broke file paths containing spaces into multiple arguments, causing "file not found" errors.
- `os/exec.Command` accepts an argument slice directly, preserving spaces in paths without any shell quoting.
- The `buildFFmpegArgs` function constructs the args array by starting with `["-y", "-i", input]`, appending `strings.Fields(params)` (safe because FFmpeg params never contain spaces within individual values), and appending the output path — all as separate slice elements.

**Example:**
- Input file: `/tmp/my video file.mp4`
- Args array: `["-y", "-i", "/tmp/my video file.mp4", "-preset", "medium", "-crf", "23", ...]`
- Each element (including the path with spaces) is passed as a single argument to FFmpeg.

## 16. Flag Ordering

**Decision:** Flags must be specified before positional (non-flag) arguments.

**Rationale:**
- Go's `flag.Parse()` stops parsing flags at the first non-flag argument.
- If flags appear after input files, they are treated as input file paths, not flag values.
- This is standard Go behavior and is documented in the README.

**Example:**
```bash
# Correct:
ffcompress -o /output /input/
# Incorrect (flag treated as file path):
ffcompress /input/ -o /output
```

## Summary

This CLI port faithfully reproduces the FFShare Android app's compression logic in a cross-platform, single-binary Go application. The design prioritizes simplicity (minimal flags, hardcoded defaults, standard library only) while preserving the key quality-preserving features (CRF, stream copy audio, conditional resolution scaling with aspect-ratio-preserving `-2`/`-1` pixel rounding). File path handling was fixed to support paths containing spaces by using argument arrays instead of shell-string parsing.
