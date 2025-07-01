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
	ntscImage "ntsc-wasm/pkg/image"
	"ntsc-wasm/pkg/ntsc"
	"strings"
	"syscall/js"
	"time"
)

var debugMode = false

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
	js.Global().Set("setDebugMode", js.FuncOf(setDebugMode))
	js.Global().Set("getDebugMode", js.FuncOf(getDebugMode))

	fmt.Println("NTSC WebAssembly module loaded")
	<-c
}

func processNTSC(this js.Value, args []js.Value) interface{} {
	startTotal := time.Now()

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

	// Decode image data
	start := time.Now()
	imageData, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(req.ImageData, "data:image/png;base64,"))
	if err != nil {
		imageData, err = base64.StdEncoding.DecodeString(strings.TrimPrefix(req.ImageData, "data:image/jpeg;base64,"))
		if err != nil {
			return map[string]interface{}{
				"error": fmt.Sprintf("Failed to decode image data: %v", err),
			}
		}
	}
	if debugMode {
		fmt.Printf("DEBUG: Base64 decode took %v\n", time.Since(start))
	}

	// Decode image
	start = time.Now()
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
	if debugMode {
		fmt.Printf("DEBUG: Image decode took %v\n", time.Since(start))
	}

	// Convert to ntscImage
	start = time.Now()
	ntscImg := ntscImage.FromGoImage(img)
	if debugMode {
		fmt.Printf("DEBUG: FromGoImage took %v\n", time.Since(start))
	}

	maxWidth := req.MaxWidth
	maxHeight := req.MaxHeight

	// Resize image
	if maxWidth > 0 || maxHeight > 0 {
		start = time.Now()
		ntscImg = ntscImg.Resize(maxWidth, maxHeight)
		if debugMode {
			fmt.Printf("DEBUG: Resize took %v\n", time.Since(start))
		}
	}

	processor := ntsc.NewNtscProcessor(req.Config)

	// Process image
	start = time.Now()
	processedImg := processor.ProcessImage(ntscImg)
	if debugMode {
		fmt.Printf("DEBUG: ProcessImage took %v\n", time.Since(start))
	}

	// Convert back to Go image
	start = time.Now()
	resultImg := processedImg.ToGoImage()
	if debugMode {
		fmt.Printf("DEBUG: ToGoImage took %v\n", time.Since(start))
	}

	// Encode result image
	start = time.Now()
	var buf bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.NoCompression}
	if err := encoder.Encode(&buf, resultImg); err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("Failed to encode result image: %v", err),
		}
	}
	if debugMode {
		fmt.Printf("DEBUG: Image encode took %v\n", time.Since(start))
	}

	// Encode to base64
	start = time.Now()
	resultData := base64.StdEncoding.EncodeToString(buf.Bytes())
	if debugMode {
		fmt.Printf("DEBUG: Base64 encode took %v\n", time.Since(start))
	}

	if debugMode {
		fmt.Printf("DEBUG: Total processNTSC took %v\n", time.Since(startTotal))
	}
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

func setDebugMode(this js.Value, args []js.Value) interface{} {
	if len(args) != 1 {
		return map[string]interface{}{
			"error": "Invalid number of arguments",
		}
	}

	debugMode = args[0].Bool()
	return map[string]interface{}{
		"debugMode": debugMode,
	}
}

func getDebugMode(this js.Value, args []js.Value) interface{} {
	return map[string]interface{}{
		"debugMode": debugMode,
	}
}
