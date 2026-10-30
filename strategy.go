package main

import (
	"fmt"
	"math"
	"os/exec"
	"strings"
)

type MediaType string

const (
	MediaTypeVideo    MediaType = "video"
	MediaTypeImage    MediaType = "image"
	MediaTypeAudio    MediaType = "audio"
	MediaTypeUnknown  MediaType = "unknown"
)

type OutputMediaType string

const (
	OutputMediaTypeMP4  OutputMediaType = "mp4"
	OutputMediaTypeMKV  OutputMediaType = "mkv"
	OutputMediaTypeWEBM OutputMediaType = "webm"
	OutputMediaTypeAVI  OutputMediaType = "avi"
	OutputMediaTypeJPEG OutputMediaType = "jpeg"
	OutputMediaTypeJPG  OutputMediaType = "jpg"
	OutputMediaTypePNG  OutputMediaType = "png"
	OutputMediaTypeGIF  OutputMediaType = "gif"
	OutputMediaTypeWEBP OutputMediaType = "webp"
	OutputMediaTypeMP3  OutputMediaType = "mp3"
	OutputMediaTypeOGG  OutputMediaType = "ogg"
	OutputMediaTypeAAC  OutputMediaType = "aac"
	OutputMediaTypeWAV  OutputMediaType = "wav"
	OutputMediaTypeOPUS OutputMediaType = "opus"
)

type MediaInfo struct {
	Width    int
	Height   int
	Duration float64
	Size     int64
	Codec    string
}

type Settings struct {
	VideoCrf           int
	JpegQscale         int
	VideoMaxFileSize   int
	MaxVideoResolution int
	MaxImageResolution int
	CompressionPreset  string
	VideoCodec         string
	AudioCodec         string
	CopyExifTags       bool
}

type ParamMaker struct {
	settings Settings
}

func NewParamMaker() *ParamMaker {
	return &ParamMaker{
		settings: Settings{
			VideoCrf:           23,
			JpegQscale:         10,
			VideoMaxFileSize:   0,
			MaxVideoResolution: 1080,
			MaxImageResolution: 2160,
			CompressionPreset:  "medium",
			VideoCodec:         "libx264",
			AudioCodec:         "copy",
			CopyExifTags:       false,
		},
	}
}

func isVideoOutput(output OutputMediaType) bool {
	switch output {
	case OutputMediaTypeMP4, OutputMediaTypeMKV, OutputMediaTypeWEBM, OutputMediaTypeAVI:
		return true
	}
	return false
}

func isImageOutput(output OutputMediaType) bool {
	switch output {
	case OutputMediaTypeJPEG, OutputMediaTypeJPG, OutputMediaTypePNG, OutputMediaTypeGIF, OutputMediaTypeWEBP:
		return true
	}
	return false
}

func (pm *ParamMaker) Create(inputFile string, mediaType MediaType, mediaInfo *MediaInfo, outputMediaType OutputMediaType) string {
	var params []string
	var videoFormatParams []string

	// 2. Preset
	if outputMediaType != OutputMediaTypeWEBP {
		params = append(params, fmt.Sprintf("-preset %s", pm.settings.CompressionPreset))
	}

	// 3. Audio codec
	if mediaType == MediaTypeAudio || mediaType == MediaTypeVideo {
		if pm.settings.AudioCodec != "copy" {
			params = append(params, fmt.Sprintf("-c:a %s", pm.settings.AudioCodec))
		}
	}

	// 4. Resolution scaling
	videoScaleApplied := false
	if mediaType == MediaTypeVideo || mediaType == MediaTypeImage {
		tmpW, tmpH := pm.resolveMediaResolution(inputFile, mediaType)
		var maxRes int
		if isImageOutput(outputMediaType) {
			maxRes = pm.settings.MaxImageResolution
		} else {
			maxRes = pm.settings.MaxVideoResolution
		}
		if tmpW > maxRes || tmpH > maxRes {
			videoScaleApplied = true
			var pixelRounding string
			if isVideoOutput(outputMediaType) && (pm.settings.VideoCodec == "libx264" || pm.settings.VideoCodec == "libx265") {
				pixelRounding = "-2"
			} else {
				pixelRounding = "-1"
			}
			if tmpH > tmpW {
				videoFormatParams = append(videoFormatParams, fmt.Sprintf("scale=%d:%s,setsar=1", maxRes, pixelRounding))
			} else {
				videoFormatParams = append(videoFormatParams, fmt.Sprintf("scale=%s:%d,setsar=1", pixelRounding, maxRes))
			}
		}
	}

	// 5. Video encoding
	if isVideoOutput(outputMediaType) {
		params = append(params, fmt.Sprintf("-crf %d", pm.settings.VideoCrf))
		videoFormatParams = append(videoFormatParams, "format=yuv420p")
		if pm.settings.VideoCodec != "copy" {
			params = append(params, fmt.Sprintf("-c:v %s", pm.settings.VideoCodec))
		}
		if !videoScaleApplied {
			if mediaInfo != nil && (mediaInfo.Width%2 != 0 || mediaInfo.Height%2 != 0) {
				videoFormatParams = append(videoFormatParams, "crop=trunc(iw/2)*2:trunc(ih/2)*2")
			}
		}
	}

	// 6. Max file size (bitrate limiting)
	if pm.settings.VideoMaxFileSize != 0 {
		if mediaInfo != nil && mediaInfo.Duration > 0 {
			maxBitrate := int64(math.Ceil(float64(pm.settings.VideoMaxFileSize*8) / mediaInfo.Duration))
			audioSplit := maxBitrate / 3
			var audioBitrate int64
			switch {
			case audioSplit > 192:
				audioBitrate = 192
			case audioSplit > 128:
				audioBitrate = 128
			case audioSplit > 96:
				audioBitrate = 96
			case audioSplit > 64:
				audioBitrate = 64
			case audioSplit > 32:
				audioBitrate = 32
			default:
				audioBitrate = 24
			}
			params = append(params, fmt.Sprintf("-b:a %dk", audioBitrate))
			videoBitrate := maxBitrate - audioBitrate
			params = append(params, fmt.Sprintf("-maxrate %dk -bufsize %dk", videoBitrate, videoBitrate))
		}
	}

	// 7. JPEG quality
	if outputMediaType == OutputMediaTypeJPEG || outputMediaType == OutputMediaTypeJPG {
		params = append(params, fmt.Sprintf("-qscale:v %d", pm.settings.JpegQscale))
	}

	result := append([]string{}, params...)
	if len(videoFormatParams) > 0 {
		result = append(result, fmt.Sprintf("-vf %s", strings.Join(videoFormatParams, ",")))
	}
	return strings.Join(result, " ")
}

func (pm *ParamMaker) resolveMediaResolution(inputFile string, mediaType MediaType) (int, int) {
	if mediaType == MediaTypeImage {
		return getImageResolution(inputFile)
	}
	return getVideoResolution(inputFile)
}

func getImageResolution(inputFile string) (int, int) {
	cmd := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0",
		"-show_entries", "stream=width,height", "-of", "csv=p=0", inputFile)
	output, err := cmd.Output()
	if err != nil {
		return 0, 0
	}
	parts := strings.Split(strings.TrimSpace(string(output)), ",")
	if len(parts) < 2 {
		return 0, 0
	}
	var w, h int
	fmt.Sscanf(parts[0], "%d", &w)
	fmt.Sscanf(parts[1], "%d", &h)
	return w, h
}

func getVideoResolution(inputFile string) (int, int) {
	return getImageResolution(inputFile)
}
