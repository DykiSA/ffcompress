package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func detectMediaType(path string) MediaType {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".mp4":
		return MediaTypeVideo
	case ".mkv":
		return MediaTypeVideo
	case ".webm":
		return MediaTypeVideo
	case ".avi":
		return MediaTypeVideo
	case ".jpg", ".jpeg":
		return MediaTypeImage
	case ".png":
		return MediaTypeImage
	case ".gif":
		return MediaTypeImage
	case ".webp":
		return MediaTypeImage
	case ".mp3":
		return MediaTypeAudio
	case ".ogg":
		return MediaTypeAudio
	case ".aac":
		return MediaTypeAudio
	case ".wav":
		return MediaTypeAudio
	case ".opus":
		return MediaTypeAudio
	}
	return MediaTypeUnknown
}

func getMediaInfo(inputFile string, mediaType MediaType) *MediaInfo {
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration,size", "-show_entries", "stream=width,height,codec_name", "-of", "json", inputFile)
	output, err := cmd.Output()
	if err != nil {
		return nil
	}
	var info struct {
		Format struct {
			Duration string `json:"duration"`
			Size     string `json:"size"`
		} `json:"format"`
		Streams []struct {
			Width      string `json:"width"`
			Height     string `json:"height"`
			Codec_name string `json:"codec_name"`
		} `json:"streams"`
	}
	if err := parseJSON(output, &info); err != nil {
		return nil
	}
	mi := &MediaInfo{
		Duration: parseFloatOrZero(info.Format.Duration),
		Size:     parseIntOrZero(info.Format.Size),
	}
	if len(info.Streams) > 0 {
		mi.Width = int(parseIntOrZero(info.Streams[0].Width))
		mi.Height = int(parseIntOrZero(info.Streams[0].Height))
		mi.Codec = info.Streams[0].Codec_name
	}
	return mi
}

func determineOutputMediaType(mediaType MediaType, input string) OutputMediaType {
	ext := strings.ToLower(filepath.Ext(input))
	switch ext {
	case ".mp4":
		return OutputMediaTypeMP4
	case ".mkv":
		return OutputMediaTypeMKV
	case ".webm":
		return OutputMediaTypeWEBM
	case ".avi":
		return OutputMediaTypeAVI
	case ".jpg", ".jpeg":
		return OutputMediaTypeJPEG
	case ".png":
		return OutputMediaTypePNG
	case ".gif":
		return OutputMediaTypeGIF
	case ".webp":
		return OutputMediaTypeWEBP
	case ".mp3":
		return OutputMediaTypeMP3
	case ".ogg":
		return OutputMediaTypeOGG
	case ".aac":
		return OutputMediaTypeAAC
	case ".wav":
		return OutputMediaTypeWAV
	case ".opus":
		return OutputMediaTypeOPUS
	}
	return OutputMediaTypeMP4
}

func buildFFmpegArgs(input, params, output string) []string {
	args := []string{"-y", "-i", input}
	args = append(args, strings.Fields(params)...)
	args = append(args, output)
	return args
}

func executeCommand(name string, args []string) error {
	c := exec.Command(name, args...)
	c.Stdin = nil
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func getOutputSize(path string) int64 {
	data, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return data.Size()
}

func formatBytes(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%d KB", bytes/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
}

func getCurrentDateTime() string {
	now := time.Now()
	return now.Format("20060102_150405")
}

func parseJSON(data []byte, v interface{}) error {
	return nil
}

func parseIntOrZero(s string) int64 {
	if s == "" {
		return 0
	}
	var n int64
	_, err := fmt.Sscanf(s, "%d", &n)
	if err != nil {
		return 0
	}
	return n
}

func parseFloatOrZero(s string) float64 {
	if s == "" {
		return 0
	}
	var n float64
	_, err := fmt.Sscanf(s, "%f", &n)
	if err != nil {
		return 0
	}
	return n
}
