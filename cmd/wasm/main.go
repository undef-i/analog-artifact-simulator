//go:build wasm

package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	ntscImage "analog-artifact-simulator/pkg/image"
	"analog-artifact-simulator/pkg/ntsc"
	"strings"
	"syscall/js"
)

type ProcessRequest struct {
	ImageData string           `json:"imageData"`
	Config    *ntsc.NtscConfig `json:"config"`
	MaxWidth  int              `json:"maxWidth,omitempty"`
	MaxHeight int              `json:"maxHeight,omitempty"`
}

type ProcessResponse struct {
	ImageData string `json:"imageData"`
	Error     string `json:"error,omitempty"`
}

func main() {
	c := make(chan struct{}, 0)

	js.Global().Set("processNTSC", js.FuncOf(processNTSC))
	js.Global().Set("getPreset", js.FuncOf(getPreset))

	fmt.Println("NTSC WebAssembly module loaded")
	<-c
}

func processNTSC(this js.Value, args []js.Value) interface{} {
	if len(args) != 1 {
		return map[string]interface{}{
			"error": "Invalid number of arguments",
		}
	}

	reqJSON := args[0].String()
	var req ProcessRequest
	if err := json.Unmarshal([]byte(reqJSON), &req); err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("Failed to parse request: %v", err),
		}
	}

	if req.Config == nil {
		req.Config = ntsc.DefaultNtscConfig()
	}

	imageData, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(req.ImageData, "data:image/png;base64,"))
	if err != nil {
		imageData, err = base64.StdEncoding.DecodeString(strings.TrimPrefix(req.ImageData, "data:image/jpeg;base64,"))
		if err != nil {
			return map[string]interface{}{
				"error": fmt.Sprintf("Failed to decode image data: %v", err),
			}
		}
	}

	var img image.Image
	if strings.Contains(req.ImageData, "data:image/png") {
		img, err = png.Decode(bytes.NewReader(imageData))
	} else {
		img, err = jpeg.Decode(bytes.NewReader(imageData))
	}
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("Failed to decode image: %v", err),
		}
	}

	ntscImg := ntscImage.FromGoImage(img)

	maxWidth := req.MaxWidth
	maxHeight := req.MaxHeight

	if maxWidth > 0 || maxHeight > 0 {
		ntscImg = ntscImg.Resize(maxWidth, maxHeight)
	}

	processor := ntsc.NewNtscProcessor(req.Config)
	processedImg := processor.ProcessImage(ntscImg)
	resultImg := processedImg.ToGoImage()

	var buf bytes.Buffer
	if err := png.Encode(&buf, resultImg); err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("Failed to encode result image: %v", err),
		}
	}

	resultData := base64.StdEncoding.EncodeToString(buf.Bytes())
	return map[string]interface{}{
		"imageData": "data:image/png;base64," + resultData,
	}
}

func getPreset(this js.Value, args []js.Value) interface{} {
	if len(args) != 1 {
		return map[string]interface{}{
			"error": "Invalid number of arguments",
		}
	}

	presetName := args[0].String()
	var config *ntsc.NtscConfig

	switch presetName {
	case "default":
		config = ntsc.DefaultNtscConfig()
	case "composite":
		config = ntsc.DefaultNtscConfig()
		config.CompositePreemphasis = 0.8
		config.ColorBleedHoriz = int(0.6 * 10)
		config.Ringing = 0.4
	case "vhs":
		config = ntsc.DefaultNtscConfig()
		config.EmulatingVHS = true
		config.VHSChromaVertBlend = true
		config.VHSOutSharpen = 0.4
		config.VHSEdgeWave = int(0.2 * 100)
		config.VideoChromaLoss = int(0.3 * 100)
	case "broadcast":
		config = ntsc.DefaultNtscConfig()
		config.VideoNoise = int(0.1 * 100)
		config.VideoChromaNoise = int(0.05 * 100)
		config.VideoChromaPhaseNoise = int(0.02 * 100)
	default:
		config = ntsc.DefaultNtscConfig()
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("Failed to marshal config: %v", err),
		}
	}

	return map[string]interface{}{
		"config": string(configJSON),
	}
}
