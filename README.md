# FFCompress CLI

Cross-platform CLI for compressing multimedia files (image, video, audio) using FFmpeg, ported from the FFShare Android app.

## Requirements

- `ffmpeg` and `ffprobe` must be installed and accessible in PATH
- Go 1.23+ (for building from source)

## Build

```bash
CGO_ENABLED=0 go build -o ./build/ffcompress .
```

The result is a single static binary (`ffcompress`) in `./build` folder with no external dependencies.

## Usage

Enter the build folder

```bash
cd build
```

### Single file

```bash
ffcompress input.mp4
```

Output: `./compressed_input_YYYYMMDD_HHMMSS.mp4` (same directory as input)

### Multiple files

```bash
ffcompress input1.mp4 input2.mp4
```

Output: both files compressed in the same directory as the first input file

### Directory

```bash
ffcompress -o ./compressed-out ./my-media/
```

> **Note:** Flags must come before positional arguments (standard Go `flag` behavior).

Output: all supported media files in `./compressed-out`

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-o`, `--output-dir` | Output directory for compressed files | Same directory as input files |

> **Note:** Flags must be specified before input files/directories. For example: `ffcompress -o /output /input`.

## Compression Strategy

The CLI mirrors the FFShare Android app's FFmpeg parameter generation:

1. **CRF (Constant Rate Factor)**: `-crf 23` (default) – quality-preserving rate control
2. **Preset**: `-preset medium` (default) – speed/quality trade-off
3. **Video codec**: `-c:v libx264` (default) – H.264 encoder
4. **Audio codec**: `-c:a copy` (default) – lossless stream copy
5. **Resolution scaling**: Only if source exceeds `maxVideoResolution` (1080) or `maxImageResolution` (2160)
6. **Max file size / bitrate**: When `videoMaxFileSize` (0 = disabled) is set, calculates max bitrate and limits video bitrate with VBV
7. **Pixel format**: `-vf format=yuv420p` for universal compatibility
8. **JPEG quality**: `-qscale:v 10` (default, 0-100 scale) for JPEG outputs

## License

Ported from FFShare (GPLv3). See the [ffshare repo](https://github.com/caydey/ffshare) for the original source.
