package ntsc

import (
	"encoding/json"
	"fmt"
	"math"
	"ntsc-wasm/pkg/image"
	"ntsc-wasm/pkg/pool"
	"ntsc-wasm/pkg/random"
	"sync"
	"time"
)

const (
	NTSC_RATE     = 315000000.0 / 88 * 4
	M_PI          = math.Pi
	Int_MIN_VALUE = -2147483648
	Int_MAX_VALUE = 2147483647
	debugMode     = false
)

type VHSSpeed struct {
	LumaCut     float64
	ChromaCut   float64
	ChromaDelay int
}

var (
	VHS_SP = VHSSpeed{2400000.0, 320000.0, 9}
	VHS_LP = VHSSpeed{1900000.0, 300000.0, 12}
	VHS_EP = VHSSpeed{1400000.0, 280000.0, 14}
)

func (v *VHSSpeed) UnmarshalJSON(data []byte) error {
	var speedIndex int
	if err := json.Unmarshal(data, &speedIndex); err != nil {
		return err
	}

	switch speedIndex {
	case 0:
		*v = VHS_SP
	case 1:
		*v = VHS_LP
	case 2:
		*v = VHS_EP
	default:
		*v = VHS_SP
	}
	return nil
}

func (v VHSSpeed) MarshalJSON() ([]byte, error) {
	switch v {
	case VHS_SP:
		return json.Marshal(0)
	case VHS_LP:
		return json.Marshal(1)
	case VHS_EP:
		return json.Marshal(2)
	default:
		return json.Marshal(0)
	}
}

type LowpassFilter struct {
	timeInterval float64
	tau          float64
	alpha        float64
	prev         float64
}

func NewLowpassFilter(rate, hz, value float64) *LowpassFilter {
	timeInterval := 1.0 / rate
	tau := 1.0 / (hz * 2.0 * M_PI)
	alpha := timeInterval / (tau + timeInterval)
	return &LowpassFilter{
		timeInterval: timeInterval,
		tau:          tau,
		alpha:        alpha,
		prev:         value,
	}
}

func (lp *LowpassFilter) Lowpass(sample float64) float64 {
	stage1 := sample * lp.alpha
	stage2 := lp.prev - lp.prev*lp.alpha
	lp.prev = stage1 + stage2
	return lp.prev
}

func (lp *LowpassFilter) Highpass(sample float64) float64 {
	stage1 := sample * lp.alpha
	stage2 := lp.prev - lp.prev*lp.alpha
	lp.prev = stage1 + stage2
	return sample - lp.prev
}

func (lp *LowpassFilter) LowpassArray(samples []float64) []float64 {
	result := make([]float64, len(samples))
	prev := lp.prev // Use a local variable for prev
	for i, sample := range samples {
		stage1 := sample * lp.alpha
		stage2 := prev - prev*lp.alpha
		prev = stage1 + stage2
		result[i] = prev
	}
	lp.prev = prev // Update lp.prev after the loop
	return result
}

func (lp *LowpassFilter) HighpassArray(samples []float64) []float64 {
	result := make([]float64, len(samples))
	prev := lp.prev // Use a local variable for prev
	for i, sample := range samples {
		stage1 := sample * lp.alpha
		stage2 := prev - prev*lp.alpha
		prev = stage1 + stage2
		result[i] = sample - prev
	}
	lp.prev = prev // Update lp.prev after the loop
	return result
}

type NtscConfig struct {
	CompositePreemphasis          float64
	CompositePreemphasisCut       float64
	VHSOutSharpen                 float64
	VHSEdgeWave                   int
	VHSHeadSwitching              bool
	VHSHeadSwitchingPoint         float64
	VHSHeadSwitchingPhase         float64
	VHSHeadSwitchingPhaseNoise    float64
	HeadSwitchingSpeed            int
	ColorBleedBefore              bool
	ColorBleedHoriz               int
	ColorBleedVert                int
	Ringing                       float64
	EnableRinging2                bool
	RingingPower                  int
	RingingShift                  int
	FreqNoiseSize                 float64
	FreqNoiseAmplitude            float64
	CompositeInChromaLowpass      bool
	CompositeOutChromaLowpass     bool
	CompositeOutChromaLowpassLite bool
	VideoChromaNoise              int
	VideoChromaPhaseNoise         int
	VideoChromaLoss               int
	VideoNoise                    int
	SubcarrierAmplitude           int
	SubcarrierAmplitudeBack       int
	EmulatingVHS                  bool
	NoColorSubcarrier             bool
	VHSChromaVertBlend            bool
	VHSSVideoOut                  bool
	OutputNTSC                    bool
	VideoScanlinePhaseShift       int
	VideoScanlinePhaseShiftOffset int
	OutputVHSTapeSpeed            VHSSpeed
	BlackLineCut                  bool
	Precise                       bool

	RandomSeed  uint32
	RandomSeed2 uint32
}

func DefaultNtscConfig() *NtscConfig {
	return &NtscConfig{
		CompositePreemphasis:          0.0,
		CompositePreemphasisCut:       1000000.0,
		VHSOutSharpen:                 1.5,
		VHSEdgeWave:                   0,
		VHSHeadSwitching:              false,
		VHSHeadSwitchingPoint:         1.0 - (4.5+0.01)/262.5,
		VHSHeadSwitchingPhase:         (1.0 - 0.01) / 262.5,
		VHSHeadSwitchingPhaseNoise:    1.0 / 500 / 262.5,
		HeadSwitchingSpeed:            0,
		ColorBleedBefore:              true,
		ColorBleedHoriz:               0,
		ColorBleedVert:                0,
		Ringing:                       1.0,
		EnableRinging2:                false,
		RingingPower:                  2,
		RingingShift:                  0,
		FreqNoiseSize:                 0,
		FreqNoiseAmplitude:            2,
		CompositeInChromaLowpass:      true,
		CompositeOutChromaLowpass:     true,
		CompositeOutChromaLowpassLite: true,
		VideoChromaNoise:              0,
		VideoChromaPhaseNoise:         0,
		VideoChromaLoss:               0,
		VideoNoise:                    2,
		SubcarrierAmplitude:           50,
		SubcarrierAmplitudeBack:       50,
		EmulatingVHS:                  false,
		NoColorSubcarrier:             false,
		VHSChromaVertBlend:            true,
		VHSSVideoOut:                  false,
		OutputNTSC:                    true,
		VideoScanlinePhaseShift:       180,
		VideoScanlinePhaseShiftOffset: 0,
		OutputVHSTapeSpeed:            VHS_SP,
		BlackLineCut:                  false,
		Precise:                       false,

		RandomSeed:  12345,
		RandomSeed2: 67890,
	}
}

type YIQImage = pool.YIQImage

type NtscProcessor struct {
	Config        *NtscConfig
	Random        *random.XorWowRandom
	Precise       bool
	Umult         []int32
	Vmult         []int32
	chromaBuffers *ChromaBuffers
	samplesBuffer []float64
	int32Buffer   []int32
	float64Buffer []float64
}

func NewNtscProcessor(config *NtscConfig) *NtscProcessor {
	p := &NtscProcessor{
		Config:  config,
		Random:  random.NewXorWowRandom(31374242),
		Precise: false,
		Umult:   []int32{1, 0, -1, 0},
		Vmult:   []int32{0, 1, 0, -1},
	}
	return p
}

func (p *NtscProcessor) ProcessImage(img *image.Image) *image.Image {
	dst := pool.DefaultImagePool.Get(img.Width, img.Height)
	copy(dst.Data, img.Data)
	yiq := p.bgr2yiq(img)
	defer pool.DefaultYIQImagePool.Put(yiq)

	var wg sync.WaitGroup
	wg.Add(2) // Two fields to process

	// Process field 0
	go func() {
		defer wg.Done()
		p.compositeLayer(dst, img, yiq, 0, 0)
	}()

	// Process field 1
	go func() {
		defer wg.Done()
		p.compositeLayer(dst, img, yiq, 1, 1)
	}()

	wg.Wait() // Wait for both fields to complete

	return dst
}

func (p *NtscProcessor) bgr2yiq(img *image.Image) *YIQImage {
	height := img.Height
	width := img.Width

	yiq := pool.DefaultYIQImagePool.Get(width, height)
	yiqData := yiq.Data
	imgData := img.Data

	// Batch process multiple pixels at once for better cache locality
	batchSize := 8
	for y := 0; y < height; y++ {
		rowStart := y * width
		imgRowStart := rowStart * 3
		yRowStart := rowStart
		iRowStart := height*width + rowStart
		qRowStart := 2*height*width + rowStart

		// Process in batches for better performance
		for x := 0; x < width; x += batchSize {
			end := x + batchSize
			if end > width {
				end = width
			}

			for i := x; i < end; i++ {
				imgIdx := imgRowStart + i*3
				r := int32(imgData[imgIdx])
				g := int32(imgData[imgIdx+1])
				b := int32(imgData[imgIdx+2])

				// Use integer arithmetic where possible
				dY := (77*r + 151*g + 28*b) >> 8 // 0.30*256, 0.59*256, 0.11*256

				yiqData[yRowStart+i] = dY
				yiqData[iRowStart+i] = (189*(r-dY) - 69*(b-dY)) >> 8
				yiqData[qRowStart+i] = (123*(r-dY) + 105*(b-dY)) >> 8
			}
		}
	}

	return yiq
}

func (p *NtscProcessor) yiq2bgr(yiq *YIQImage, dst *image.Image, field int) {
	height := yiq.Height
	width := yiq.Width
	dstData := dst.Data

	// Batch processing for better cache locality
	batchSize := 8
	for y := field; y < height; y += 2 {
		rowStart := y * width
		dstRowStart := rowStart * 3
		yRowStart := rowStart
		iRowStart := height*width + rowStart
		qRowStart := 2*height*width + rowStart

		for x := 0; x < width; x += batchSize {
			end := x + batchSize
			if end > width {
				end = width
			}

			for i := x; i < end; i++ {
				Y := yiq.Data[yRowStart+i]
				I := yiq.Data[iRowStart+i]
				Q := yiq.Data[qRowStart+i]

				// Use integer arithmetic for better performance
				r := Y + (245*I+159*Q)>>8
				g := Y - (70*I+166*Q)>>8
				b := Y + (-283*I+436*Q)>>8

				// Clamp values using branchless operations where possible
				if r < 0 {
					r = 0
				} else if r > 255 {
					r = 255
				}
				if g < 0 {
					g = 0
				} else if g > 255 {
					g = 255
				}
				if b < 0 {
					b = 0
				} else if b > 255 {
					b = 255
				}

				dstIdx := dstRowStart + i*3
				dstData[dstIdx] = uint8(r)
				dstData[dstIdx+1] = uint8(g)
				dstData[dstIdx+2] = uint8(b)
			}
		}
	}
}

func (p *NtscProcessor) compositeLayer(dst *image.Image, src *image.Image, yiq *YIQImage, field int, fieldno int) {
	start := time.Now()
	if p.Config.BlackLineCut {
		p.cutBlackLineBorder(src)
		if debugMode {
			fmt.Printf("DEBUG: cutBlackLineBorder took %v\n", time.Since(start))
		}
	}

	start = time.Now()
	if p.Config.ColorBleedBefore && (p.Config.ColorBleedVert != 0 || p.Config.ColorBleedHoriz != 0) {
		p.colorBleed(yiq, field)
		if debugMode {
			fmt.Printf("DEBUG: colorBleed took %v\n", time.Since(start))
		}
	}

	start = time.Now()
	if p.Config.CompositeInChromaLowpass {
		p.compositeLowpass(yiq, field, fieldno)
		if debugMode {
			fmt.Printf("DEBUG: compositeLowpass took %v\n", time.Since(start))
		}
	}

	start = time.Now()
	if p.Config.Ringing != 1.0 {
		p.ringing(yiq, field)
		if debugMode {
			fmt.Printf("DEBUG: ringing took %v\n", time.Since(start))
		}
	}

	start = time.Now()
	p.chromaIntoLuma(yiq, field, fieldno, p.Config.SubcarrierAmplitude)
	if debugMode {
		fmt.Printf("DEBUG: chromaIntoLuma took %v\n", time.Since(start))
	}

	start = time.Now()
	if p.Config.CompositePreemphasis != 0.0 && p.Config.CompositePreemphasisCut > 0 {
		p.compositePreemphasis(yiq, field, p.Config.CompositePreemphasis, p.Config.CompositePreemphasisCut)
		if debugMode {
			fmt.Printf("DEBUG: compositePreemphasis took %v\n", time.Since(start))
		}
	}

	start = time.Now()
	if p.Config.VideoNoise != 0 {
		p.videoNoise(yiq, field, p.Config.VideoNoise)
		if debugMode {
			fmt.Printf("DEBUG: videoNoise took %v\n", time.Since(start))
		}
	}

	start = time.Now()
	if p.Config.VHSHeadSwitching {
		p.vhsHeadSwitching(yiq, field)
		if debugMode {
			fmt.Printf("DEBUG: vhsHeadSwitching took %v\n", time.Since(start))
		}
	}

	start = time.Now()
	if !p.Config.NoColorSubcarrier {
		p.chromaFromLuma(yiq, field, fieldno, p.Config.SubcarrierAmplitudeBack)
		if debugMode {
			fmt.Printf("DEBUG: chromaFromLuma took %v\n", time.Since(start))
		}
	}

	start = time.Now()
	if p.Config.VideoChromaNoise != 0 {
		p.videoChromaNoise(yiq, field, p.Config.VideoChromaNoise)
		if debugMode {
			fmt.Printf("DEBUG: videoChromaNoise took %v\n", time.Since(start))
		}
	}

	start = time.Now()
	if p.Config.VideoChromaPhaseNoise != 0 {
		p.videoChromaPhaseNoise(yiq, field, p.Config.VideoChromaPhaseNoise)
		if debugMode {
			fmt.Printf("DEBUG: videoChromaPhaseNoise took %v\n", time.Since(start))
		}
	}

	start = time.Now()
	if p.Config.EmulatingVHS {
		p.emulateVHS(yiq, field, fieldno)
		if debugMode {
			fmt.Printf("DEBUG: emulateVHS took %v\n", time.Since(start))
		}
	}

	start = time.Now()
	if p.Config.VideoChromaLoss != 0 {
		p.vhsChromaLoss(yiq, field, p.Config.VideoChromaLoss)
		if debugMode {
			fmt.Printf("DEBUG: vhsChromaLoss took %v\n", time.Since(start))
		}
	}

	start = time.Now()
	if p.Config.CompositeOutChromaLowpass {
		if p.Config.CompositeOutChromaLowpassLite {
			p.compositeLowpassTV(yiq, field, fieldno)
		} else {
			p.compositeLowpass(yiq, field, fieldno)
		}
		if debugMode {
			fmt.Printf("DEBUG: compositeOutChromaLowpass took %v\n", time.Since(start))
		}
	}

	start = time.Now()
	if !p.Config.ColorBleedBefore && (p.Config.ColorBleedVert != 0 || p.Config.ColorBleedHoriz != 0) {
		p.colorBleed(yiq, field)
		if debugMode {
			fmt.Printf("DEBUG: colorBleed (after) took %v\n", time.Since(start))
		}
	}

	start = time.Now()
	p.blurChroma(yiq, field)
	if debugMode {
		fmt.Printf("DEBUG: blurChroma took %v\n", time.Since(start))
	}

	start = time.Now()
	p.yiq2bgr(yiq, dst, field)
	if debugMode {
		fmt.Printf("DEBUG: yiq2bgr took %v\n", time.Since(start))
	}
}

func (p *NtscProcessor) chromaLumaXi(fieldno, y int) int {
	if p.Config.VideoScanlinePhaseShift == 90 {
		return (fieldno + p.Config.VideoScanlinePhaseShiftOffset + (y >> 1)) & 3
	} else if p.Config.VideoScanlinePhaseShift == 180 {
		return ((((fieldno + y) & 2) + p.Config.VideoScanlinePhaseShiftOffset) & 3)
	} else if p.Config.VideoScanlinePhaseShift == 270 {
		return ((fieldno + p.Config.VideoScanlinePhaseShiftOffset) & 3)
	} else {
		return (p.Config.VideoScanlinePhaseShiftOffset & 3)
	}
}

func (p *NtscProcessor) chromaIntoLuma(yiq *YIQImage, field, fieldno, subcarrierAmplitude int) {
	height := yiq.Height
	width := yiq.Width

	for y := field; y < height; y += 2 {
		xi := p.chromaLumaXi(fieldno, y)

		for x := 0; x < width; x++ {
			umultIdx := (xi + x) % 4
			vmultIdx := (xi + x) % 4

			// Calculate flattened indices
			idxY := y*width + x
			idxI := height*width + y*width + x
			idxQ := 2*height*width + y*width + x

			chroma := yiq.Data[idxI]*int32(subcarrierAmplitude)*p.Umult[umultIdx] +
				yiq.Data[idxQ]*int32(subcarrierAmplitude)*p.Vmult[vmultIdx]

			yiq.Data[idxY] += chroma / 50
			yiq.Data[idxI] = 0
			yiq.Data[idxQ] = 0
		}
	}
}

// Pre-allocated buffers for chromaFromLuma to avoid repeated allocations
type ChromaBuffers struct {
	chroma []int32
	y2     []int32
	yd4    []int32
	sums   []int32
	sums0  []int32
	acc    []int32
	acc4   []int32
	cxi    []int32
	cxi1   []int32
}

func newChromaBuffers(width int) *ChromaBuffers {
	return &ChromaBuffers{
		chroma: make([]int32, width),
		y2:     make([]int32, width),
		yd4:    make([]int32, width),
		sums:   make([]int32, width),
		sums0:  make([]int32, width+1),
		acc:    make([]int32, width),
		acc4:   make([]int32, width),
		cxi:    make([]int32, width/2+1),
		cxi1:   make([]int32, width/2+1),
	}
}

func (p *NtscProcessor) chromaFromLuma(yiq *YIQImage, field, fieldno, subcarrierAmplitude int) {
	height := yiq.Height
	width := yiq.Width

	// Use pre-allocated buffers
	if p.chromaBuffers == nil || len(p.chromaBuffers.chroma) < width {
		p.chromaBuffers = newChromaBuffers(width)
	}
	buf := p.chromaBuffers

	for y := field; y < height; y += 2 {
		Y_row_start := y * width
		I_row_start := height*width + y*width
		Q_row_start := 2*height*width + y*width

		sum := yiq.Data[Y_row_start] + yiq.Data[Y_row_start+1]

		for i := 0; i < width-2; i++ {
			buf.y2[i] = yiq.Data[Y_row_start+i+2]
		}

		for i := 2; i < width; i++ {
			buf.yd4[i] = yiq.Data[Y_row_start+i-2]
		}

		for i := 0; i < width; i++ {
			buf.sums[i] = buf.y2[i] - buf.yd4[i]
		}

		buf.sums0[0] = sum
		for i := 0; i < width; i++ {
			buf.sums0[i+1] = buf.sums[i]
		}

		accumulator := buf.sums0[0]
		for i := 0; i < width; i++ {
			accumulator += buf.sums0[i+1]
			buf.acc[i] = accumulator
		}

		for i := 0; i < width; i++ {
			buf.acc4[i] = buf.acc[i] / 4
		}

		for i := 0; i < width; i++ {
			buf.chroma[i] = buf.y2[i] - buf.acc4[i]
		}

		for i := 0; i < width; i++ {
			yiq.Data[Y_row_start+i] = buf.acc4[i]
		}

		xi := p.chromaLumaXi(fieldno, y)

		x := (4 - xi) & 3

		for i := x + 2; i < width; i += 4 {
			buf.chroma[i] = -buf.chroma[i]
		}
		for i := x + 3; i < width; i += 4 {
			buf.chroma[i] = -buf.chroma[i]
		}

		for i := 0; i < width; i++ {
			if subcarrierAmplitude != 0 {
				buf.chroma[i] = buf.chroma[i] * 50 / int32(subcarrierAmplitude)
			} else {
				buf.chroma[i] = 0
			}
		}

		cxiCount := 0
		for i := xi; i < width; i += 2 {
			buf.cxi[cxiCount] = -buf.chroma[i]
			cxiCount++
		}

		cxi1Count := 0
		for i := xi + 1; i < width; i += 2 {
			buf.cxi1[cxi1Count] = -buf.chroma[i]
			cxi1Count++
		}

		for i := 0; i < width; i++ {
			yiq.Data[I_row_start+i] = 0
			yiq.Data[Q_row_start+i] = 0
		}

		for i := 0; i < cxiCount && i*2 < width; i++ {
			yiq.Data[I_row_start+i*2] = buf.cxi[i]
		}

		for i := 0; i < cxi1Count && i*2 < width; i++ {
			yiq.Data[Q_row_start+i*2] = buf.cxi1[i]
		}

		for x := 1; x < width-2; x += 2 {
			yiq.Data[I_row_start+x] = (yiq.Data[I_row_start+x-1] + yiq.Data[I_row_start+x+1]) >> 1
		}

		for x := 1; x < width-2; x += 2 {
			yiq.Data[Q_row_start+x] = (yiq.Data[Q_row_start+x-1] + yiq.Data[Q_row_start+x+1]) >> 1
		}

		for x := width - 2; x < width; x++ {
			yiq.Data[I_row_start+x] = 0
			yiq.Data[Q_row_start+x] = 0
		}
	}
}

func (p *NtscProcessor) compositeLowpass(yiq *YIQImage, field, fieldno int) {
	height := yiq.Height
	width := yiq.Width

	// Pre-allocate samples buffer
	if p.samplesBuffer == nil || len(p.samplesBuffer) < width {
		p.samplesBuffer = make([]float64, width)
	}
	samples := p.samplesBuffer[:width]

	for comp := 1; comp < 3; comp++ {
		cutoff := 1300000.0
		delay := 2
		if comp == 2 {
			cutoff = 600000.0
			delay = 4
		}

		lp := LowpassFilters(cutoff, 0.0, NTSC_RATE)
		for y := field; y < height; y += 2 {
			rowStart := comp*height*width + y*width
			for x := 0; x < width; x++ {
				samples[x] = float64(yiq.Data[rowStart+x])
			}

			f := lp[0].LowpassArray(samples)
			f = lp[1].LowpassArray(f)
			f = lp[2].LowpassArray(f)

			for x := 0; x < width-delay; x++ {
				yiq.Data[rowStart+x] = int32(f[x+delay])
			}
		}
	}
}

func (p *NtscProcessor) compositeLowpassTV(yiq *YIQImage, field, fieldno int) {
	height := yiq.Height
	width := yiq.Width

	if p.samplesBuffer == nil || len(p.samplesBuffer) < width {
		p.samplesBuffer = make([]float64, width)
	}
	samples := p.samplesBuffer[:width]

	for comp := 1; comp < 3; comp++ {
		delay := 1
		lp := LowpassFilters(2600000.0, 0.0, NTSC_RATE)

		for y := field; y < height; y += 2 {
			rowStart := comp*height*width + y*width
			for x := 0; x < width; x++ {
				samples[x] = float64(yiq.Data[rowStart+x])
			}

			f := lp[0].LowpassArray(samples)
			f = lp[1].LowpassArray(f)
			f = lp[2].LowpassArray(f)

			for x := 0; x < width-delay; x++ {
				yiq.Data[rowStart+x] = int32(f[x+delay])
			}
		}
	}
}

func (p *NtscProcessor) compositePreemphasis(yiq *YIQImage, field int, compositePreemphasis, compositePreemphasisCut float64) {
	height := yiq.Height
	width := yiq.Width

	if p.samplesBuffer == nil || len(p.samplesBuffer) < width {
		p.samplesBuffer = make([]float64, width)
	}
	samples := p.samplesBuffer[:width]

	for y := field; y < height; y += 2 {
		pre := NewLowpassFilter(NTSC_RATE, compositePreemphasisCut, 16.0)
		rowStart := y * width

		for x := 0; x < width; x++ {
			samples[x] = float64(yiq.Data[rowStart+x])
		}

		highpass := pre.HighpassArray(samples)
		for x := 0; x < width; x++ {
			filtered := samples[x] + highpass[x]*compositePreemphasis
			yiq.Data[rowStart+x] = int32(filtered)
		}
	}
}

func (p *NtscProcessor) videoNoise(yiq *YIQImage, field, videoNoise int) {
	height := yiq.Height
	width := yiq.Width

	noiseMod := videoNoise*2 + 1
	fieldHeight := (height + 1) / 2

	if !p.Precise {
		lp := NewLowpassFilter(1, 1, 0)
		lp.alpha = 0.5

		rnds := make([]float64, width*fieldHeight)
		for i := 0; i < len(rnds); i++ {
			rnds[i] = float64(p.Random.NextInt()%int32(noiseMod) - int32(videoNoise))
		}

		noises := lp.LowpassArray(rnds)
		noisesInt := make([]int32, len(noises))
		for i, v := range noises {
			noisesInt[i] = int32(v)
		}
		shiftedNoises := shiftArray(noisesInt, 1)

		idx := 0
		for y := field; y < height; y += 2 {
			for x := 0; x < width; x++ {
				yiq.Data[y*width+x] += shiftedNoises[idx]
				idx++
			}
		}
	} else {
		for y := field; y < height; y += 2 {
			rnds := make([]int32, width)
			for x := 0; x < width; x++ {
				rnds[x] = p.Random.NextInt()%int32(noiseMod) - int32(videoNoise)
			}
			noise := int32(0)
			for x := 0; x < width; x++ {
				yiq.Data[y*width+x] += noise
				noise += rnds[x]
				noise = noise / 2
			}
		}
	}
}

func (p *NtscProcessor) videoChromaNoise(yiq *YIQImage, field, videoChromaNoise int) {
	height := yiq.Height
	width := yiq.Width

	noiseMod := videoChromaNoise*2 + 1

	if !p.Precise {
		// Simplified noise generation and application for potential vectorization
		for y := field; y < height; y += 2 {
			for x := 0; x < width; x++ {
				rndU := p.Random.NextInt()%int32(noiseMod) - int32(videoChromaNoise)
				rndV := p.Random.NextInt()%int32(noiseMod) - int32(videoChromaNoise)
				yiq.Data[height*width+y*width+x] += rndU   // I component
				yiq.Data[2*height*width+y*width+x] += rndV // Q component
			}
		}
	} else {
		noiseU := int32(0)
		noiseV := int32(0)
		for y := field; y < height; y += 2 {
			for x := 0; x < width; x++ {
				yiq.Data[height*width+y*width+x] += noiseU
				noiseU += p.Random.NextInt()%int32(noiseMod) - int32(videoChromaNoise)
				noiseU = noiseU / 2

				yiq.Data[2*height*width+y*width+x] += noiseV
				noiseV += p.Random.NextInt()%int32(noiseMod) - int32(videoChromaNoise)
				noiseV = noiseV / 2
			}
		}
	}
}

func (p *NtscProcessor) videoChromaPhaseNoise(yiq *YIQImage, field, videoChromaPhaseNoise int) {
	height := yiq.Height
	width := yiq.Width

	noiseMod := videoChromaPhaseNoise*2 + 1
	noise := int32(0)

	for y := field; y < height; y += 2 {
		noise += p.Random.NextInt()%int32(noiseMod) - int32(videoChromaPhaseNoise)
		noise = noise / 2
		pi := float64(noise) * M_PI / 100
		sinpi := math.Sin(pi)
		cospi := math.Cos(pi)

		for x := 0; x < width; x++ {
			idxI := height*width + y*width + x
			idxQ := 2*height*width + y*width + x

			u := float64(yiq.Data[idxI])
			v := float64(yiq.Data[idxQ])

			newU := u*cospi - v*sinpi
			newV := u*sinpi + v*cospi

			yiq.Data[idxI] = int32(newU)
			yiq.Data[idxQ] = int32(newV)
		}
	}
}

func (p *NtscProcessor) vhsHeadSwitching(yiq *YIQImage, field int) {
	height := yiq.Height
	width := yiq.Width

	twidth := width + width/10
	shy := 0
	noise := 0.0

	if p.Config.VHSHeadSwitchingPhaseNoise != 0.0 {
		x := p.Random.NextInt() * p.Random.NextInt() * p.Random.NextInt() * p.Random.NextInt()
		x %= 2000000000
		noise = float64(x)/1000000000.0 - 1.0
		noise *= p.Config.VHSHeadSwitchingPhaseNoise
	}

	t := float64(twidth) * 262.5
	if !p.Config.OutputNTSC {
		t = float64(twidth) * 312.5
	}

	dynamicSwitchingPoint := p.Config.VHSHeadSwitchingPoint
	if p.Config.HeadSwitchingSpeed != 0 {
		speedIncrement := float64(p.Config.HeadSwitchingSpeed) / 1000.0
		frameOffset := float64(p.Random.NextInt()%1000) / 1000.0
		dynamicSwitchingPoint += speedIncrement * frameOffset
	}

	switchPoint := int(math.Mod(dynamicSwitchingPoint+noise, 1.0) * t)
	y := int(float64(switchPoint)/float64(twidth)*2) + field
	phasePoint := int(math.Mod(p.Config.VHSHeadSwitchingPhase+noise, 1.0) * t)
	x := phasePoint % twidth

	if p.Config.OutputNTSC {
		y -= (262 - 240) * 2
	} else {
		y -= (312 - 288) * 2
	}

	tx := x
	ishif := x - twidth/2
	if x < twidth/2 {
		ishif = x
	}
	shif := 0

	for y < height {
		if y >= 0 {
			if shif != 0 {
				tmp := make([]int32, twidth)
				// Copy data for the current row (Y component)
				copy(tmp, yiq.Data[y*width:(y+1)*width])

				x2 := (tx + twidth + shif) % twidth

				for i := 0; i < width; i++ {
					yiq.Data[y*width+i] = tmp[x2]
					x2++
					if x2 == twidth {
						x2 = 0
					}
				}
			}
		}

		if shy == 0 {
			shif = ishif
		} else {
			shif = shif * 7 / 8
		}
		tx = 0
		y += 2
		shy++
	}
}

func (p *NtscProcessor) emulateVHS(yiq *YIQImage, field, fieldno int) {
	vhsSpeed := p.Config.OutputVHSTapeSpeed

	if p.Config.VHSEdgeWave != 0 {
		p.vhsEdgeWave(yiq, field)
	}

	p.vhsLumaLowpass(yiq, field, vhsSpeed.LumaCut)
	p.vhsChromaLowpass(yiq, field, vhsSpeed.ChromaCut, vhsSpeed.ChromaDelay)

	if p.Config.VHSChromaVertBlend && p.Config.OutputNTSC {
		p.vhsChromaVertBlend(yiq, field)
	}

	p.vhsSharpen(yiq, field, vhsSpeed.LumaCut)

	if !p.Config.VHSSVideoOut {
		p.chromaIntoLuma(yiq, field, fieldno, p.Config.SubcarrierAmplitude)
		p.chromaFromLuma(yiq, field, fieldno, p.Config.SubcarrierAmplitude)
	}
}

func (p *NtscProcessor) vhsLumaLowpass(yiq *YIQImage, field int, lumaCut float64) {
	height := yiq.Height
	width := yiq.Width

	lp := LowpassFilters(lumaCut, 16.0, NTSC_RATE)
	pre := NewLowpassFilter(NTSC_RATE, lumaCut, 16.0)

	for y := field; y < height; y += 2 {
		samples := make([]float64, width)
		for x := 0; x < width; x++ {
			samples[x] = float64(yiq.Data[y*width+x])
		}

		f0 := lp[0].LowpassArray(samples)
		f1 := lp[1].LowpassArray(f0)
		f2 := lp[2].LowpassArray(f1)
		highpass := pre.HighpassArray(f2)

		for x := 0; x < width; x++ {
			f3 := f2[x] + highpass[x]*1.6
			yiq.Data[y*width+x] = int32(f3)
		}
	}
}

func (p *NtscProcessor) vhsChromaLowpass(yiq *YIQImage, field int, chromaCut float64, chromaDelay int) {
	height := yiq.Height
	width := yiq.Width

	for comp := 1; comp < 3; comp++ {
		lp := LowpassFilters(chromaCut, 0.0, NTSC_RATE)
		for y := field; y < height; y += 2 {
			samples := make([]float64, width)
			for x := 0; x < width; x++ {
				samples[x] = float64(yiq.Data[comp*height*width+y*width+x])
			}

			f0 := lp[0].LowpassArray(samples)
			f1 := lp[1].LowpassArray(f0)
			f2 := lp[2].LowpassArray(f1)

			for x := 0; x < width-chromaDelay; x++ {
				yiq.Data[comp*height*width+y*width+x] = int32(f2[x+chromaDelay])
			}
		}
	}
}

func (p *NtscProcessor) vhsChromaVertBlend(yiq *YIQImage, field int) {
	height := yiq.Height
	width := yiq.Width

	for comp := 1; comp < 3; comp++ {
		for y := field + 2; y < height; y += 2 {
			for x := 0; x < width; x++ {
				delay := yiq.Data[comp*height*width+(y-2)*width+x]
				current := yiq.Data[comp*height*width+y*width+x]
				yiq.Data[comp*height*width+y*width+x] = (delay + current + 1) >> 1
			}
		}
	}
}

func (p *NtscProcessor) vhsSharpen(yiq *YIQImage, field int, lumaCut float64) {
	height := yiq.Height
	width := yiq.Width

	for y := field; y < height; y += 2 {
		lp1 := NewLowpassFilter(NTSC_RATE, lumaCut*4, 0.0)
		lp2 := NewLowpassFilter(NTSC_RATE, lumaCut*4, 0.0)
		lp3 := NewLowpassFilter(NTSC_RATE, lumaCut*4, 0.0)

		samples := make([]float64, width)
		for x := 0; x < width; x++ {
			samples[x] = float64(yiq.Data[y*width+x])
		}

		ts := lp1.LowpassArray(samples)
		ts = lp2.LowpassArray(ts)
		ts = lp3.LowpassArray(ts)

		for x := 0; x < width; x++ {
			sharpened := samples[x] + (samples[x]-ts[x])*p.Config.VHSOutSharpen*2.0
			yiq.Data[y*width+x] = int32(sharpened)
		}
	}
}

func (p *NtscProcessor) colorBleed(yiq *YIQImage, field int) {
	height := yiq.Height
	width := yiq.Width

	for comp := 1; comp < 3; comp++ {
		for y := field; y < height; y += 2 {
			for x := 0; x < width; x++ {
				srcY := y - p.Config.ColorBleedVert
				srcX := x - p.Config.ColorBleedHoriz

				if srcY >= 0 && srcY < height && srcX >= 0 && srcX < width {
					// Calculate flattened index
					idxDst := comp*height*width + y*width + x
					idxSrc := comp*height*width + srcY*width + srcX
					yiq.Data[idxDst] = yiq.Data[idxSrc]
				}
			}
		}
	}
}

func (p *NtscProcessor) vhsEdgeWave(yiq *YIQImage, field int) {
	height := yiq.Height
	width := yiq.Width

	rnds := make([]int32, height/2)
	for i := range rnds {
		rnds[i] = p.Random.NextInt() % int32(p.Config.VHSEdgeWave)
	}

	lp := NewLowpassFilter(NTSC_RATE, p.Config.OutputVHSTapeSpeed.LumaCut, 0)
	rndsFloat := make([]float64, len(rnds))
	for i, v := range rnds {
		rndsFloat[i] = float64(v)
	}
	rndsFloat = lp.LowpassArray(rndsFloat)

	for i, v := range rndsFloat {
		rnds[i] = int32(v)
	}

	for comp := 0; comp < 3; comp++ {
		for i, y := 0, field; y < height; i, y = i+1, y+2 {
			if rnds[i] != 0 {
				shift := int(rnds[i])
				// Create a temporary buffer for the row
				rowStart := comp*height*width + y*width
				originalRow := make([]int32, width)
				copy(originalRow, yiq.Data[rowStart:rowStart+width])

				// Apply shift with padding (similar to numpy.pad)
				for x := 0; x < width; x++ {
					srcX := x - shift
					if srcX >= 0 && srcX < width {
						yiq.Data[rowStart+x] = originalRow[srcX]
					} else if srcX < 0 {
						yiq.Data[rowStart+x] = originalRow[0]
					} else {
						yiq.Data[rowStart+x] = originalRow[width-1]
					}
				}
			}
		}
	}
}

func (p *NtscProcessor) vhsChromaLoss(yiq *YIQImage, field, videoChromaLoss int) {
	height := yiq.Height
	width := yiq.Width

	for y := field; y < height; y += 2 {
		if p.Random.NextInt()%100000 < int32(videoChromaLoss) {
			for x := 0; x < width; x++ {
				yiq.Data[height*width+y*width+x] = 0   // I component
				yiq.Data[2*height*width+y*width+x] = 0 // Q component
			}
		}
	}
}

func (p *NtscProcessor) ringing(yiq *YIQImage, field int) {
	height := yiq.Height
	width := yiq.Width

	if !p.Config.EnableRinging2 {

		original := pool.DefaultSlicePool.GetInt32(width)
		defer pool.DefaultSlicePool.PutInt32(original)

		for comp := 0; comp < 3; comp++ {
			for y := field; y < height; y += 2 {
				// Copy data for the current component and row
				copy(original, yiq.Data[comp*height*width+y*width:comp*height*width+(y+1)*width])

				for x := 1; x < width-1; x++ {
					diff := original[x+1] - original[x-1]
					ringingEffect := int32(float64(diff) * (p.Config.Ringing - 1.0) * 0.1)
					yiq.Data[comp*height*width+y*width+x] += ringingEffect
				}
			}
		}
	} else {

		for comp := 0; comp < 3; comp++ {
			for y := field; y < height; y += 2 {
				// Create a slice for the current component and row
				row := yiq.Data[comp*height*width+y*width : comp*height*width+(y+1)*width]
				p.ringing2(row, p.Config.RingingPower, float64(p.Config.RingingShift))
			}
		}
	}
}

func (p *NtscProcessor) ringingFreqDomain(img []int32, alpha, noiseSize, noiseValue float64) {
	width := len(img)
	if width == 0 {
		return
	}

	samples := pool.DefaultSlicePool.GetFloat64(width)
	defer pool.DefaultSlicePool.PutFloat64(samples)
	for i := 0; i < width; i++ {
		samples[i] = float64(img[i])
	}

	complexData := make([]complex128, width)
	for i := 0; i < width; i++ {
		complexData[i] = complex(samples[i], 0)
	}

	start := time.Now()
	complexData = fft(complexData)
	if debugMode {
		fmt.Printf("DEBUG: ringingFreqDomain FFT took %v\n", time.Since(start))
	}

	complexData = fftShift(complexData)

	mask := make([]complex128, width)
	center := width / 2
	maskH := int(math.Min(float64(center), 1+alpha*float64(center)))

	for i := 0; i < width; i++ {
		mask[i] = complex(0, 0)
	}

	for i := center - maskH; i < center+maskH; i++ {
		if i >= 0 && i < width {
			mask[i] = complex(1, 0)
		}
	}

	if noiseSize > 0 {
		start := int(float64(center) - (1-noiseSize)*float64(center))
		stop := int(float64(center) + (1-noiseSize)*float64(center))

		for i := 0; i < width; i++ {
			if i < start || i >= stop {

				noise := (p.Random.Float64() - 0.5) * noiseValue
				mask[i] = complex(real(mask[i])+noise, imag(mask[i]))
			}
		}
	}

	for i := 0; i < width; i++ {
		complexData[i] *= mask[i]
	}

	complexData = ifftShift(complexData)

	start = time.Now()
	complexData = ifft(complexData)
	if debugMode {
		fmt.Printf("DEBUG: ringingFreqDomain IFFT took %v\n", time.Since(start))
	}

	minVal := samples[0]
	maxVal := samples[0]
	for _, v := range samples {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	for i := 0; i < width; i++ {
		result := real(complexData[i])
		if result < minVal {
			result = minVal
		}
		if result > maxVal {
			result = maxVal
		}
		img[i] = int32(result)
	}
}

func (p *NtscProcessor) ringing2(img []int32, power int, shift float64) {
	width := len(img)

	// Early return if input is empty
	if width == 0 {
		return
	}

	samples := make([]float64, width)
	for i := 0; i < width; i++ {
		samples[i] = float64(img[i])
	}

	complexData := make([]complex128, width)
	for i := 0; i < width; i++ {
		complexData[i] = complex(samples[i], 0)
	}

	startFFT := time.Now()
	complexData = fft(complexData)
	if debugMode {
		fmt.Printf("DEBUG: ringing2 FFT took %v\n", time.Since(startFFT))
	}

	complexData = fftShift(complexData)

	scaleCols := int(float64(width) * (1.0 + shift))
	// Ensure scaleCols is at least 1 to avoid empty arrays
	if scaleCols <= 0 {
		scaleCols = 1
	}

	// Early return if RingPattern is empty
	if len(RingPattern) == 0 {
		return
	}

	mask := make([]float64, scaleCols)

	for i := 0; i < scaleCols; i++ {
		pos := float64(i) / float64(scaleCols) * float64(len(RingPattern))
		index := int(pos)
		if index >= len(RingPattern) {
			index = len(RingPattern) - 1
		}
		mask[i] = RingPattern[index]
	}

	centerMask := make([]float64, width)
	startLoop := (scaleCols / 2) - (width / 2)
	for i := 0; i < width; i++ {
		if startLoop+i >= 0 && startLoop+i < scaleCols {
			centerMask[i] = mask[startLoop+i]
		}
	}

	for i := 0; i < width; i++ {
		val := centerMask[i]
		for j := 1; j < power; j++ {
			centerMask[i] *= val
		}
	}

	for i := 0; i < width; i++ {
		complexData[i] *= complex(centerMask[i], 0)
	}

	complexData = ifftShift(complexData)

	startIFFT := time.Now()
	complexData = ifft(complexData)
	if debugMode {
		fmt.Printf("DEBUG: ringing2 IFFT took %v\n", time.Since(startIFFT))
	}

	for i := 0; i < width; i++ {
		img[i] = int32(real(complexData[i]))
	}
}

func (p *NtscProcessor) blurChroma(yiq *YIQImage, field int) {
	height := yiq.Height
	width := yiq.Width

	for comp := 1; comp < 3; comp++ {
		for y := field; y < height; y += 2 {
			original := make([]int32, width)
			copy(original, yiq.Data[comp*height*width+y*width:comp*height*width+(y+1)*width])

			for x := 1; x < width-1; x++ {

				avg := (original[x-1] + original[x]*2 + original[x+1]) / 4
				yiq.Data[comp*height*width+y*width+x] = avg
			}
		}
	}
}

func LowpassFilters(cutoff, reset, rate float64) []*LowpassFilter {
	filters := make([]*LowpassFilter, 3)
	for i := 0; i < 3; i++ {
		filters[i] = NewLowpassFilter(rate, cutoff, reset)
	}
	return filters
}

func RandomNtscConfig(seed uint32) *NtscConfig {
	rnd := random.NewXorWowRandom(seed)
	config := DefaultNtscConfig()

	config.CompositePreemphasis = rnd.Uniform(0, 8)
	config.VHSOutSharpen = triangular(rnd, 1, 5, 1.5)
	config.CompositeInChromaLowpass = rnd.Float64() < 0.8
	config.CompositeOutChromaLowpass = rnd.Float64() < 0.8
	config.CompositeOutChromaLowpassLite = rnd.Float64() < 0.8
	config.VideoChromaNoise = int(triangular(rnd, 0, 16384, 2))
	config.VideoChromaPhaseNoise = int(triangular(rnd, 0, 50, 2))
	config.VideoChromaLoss = int(triangular(rnd, 0, 50000, 10))
	config.VideoNoise = int(triangular(rnd, 0, 4200, 2))
	config.EmulatingVHS = rnd.Float64() < 0.2
	config.VHSEdgeWave = int(triangular(rnd, 0, 5, 0))

	phases := []int{0, 90, 180, 270}
	config.VideoScanlinePhaseShift = phases[int(rnd.NextInt())%len(phases)]
	config.VideoScanlinePhaseShiftOffset = int(rnd.NextInt()) % 4

	speeds := []VHSSpeed{VHS_SP, VHS_LP, VHS_EP}
	config.OutputVHSTapeSpeed = speeds[int(rnd.NextInt())%len(speeds)]

	if rnd.Float64() < 0.8 {
		config.Ringing = rnd.Uniform(0.3, 0.7)
		if rnd.Float64() < 0.8 {
			config.FreqNoiseSize = rnd.Uniform(0.5, 0.99)
			config.FreqNoiseAmplitude = rnd.Uniform(0.5, 2.0)
		}
		config.EnableRinging2 = rnd.Float64() < 0.5
		config.RingingPower = int(rnd.NextInt())%6 + 2
	}

	config.ColorBleedBefore = int(rnd.NextInt())%2 == 1
	config.ColorBleedHoriz = int(triangular(rnd, 0, 8, 0))
	config.ColorBleedVert = int(triangular(rnd, 0, 8, 0))

	config.HeadSwitchingSpeed = int(triangular(rnd, 0, 100, 0))
	config.BlackLineCut = rnd.Float64() < 0.1
	config.Precise = rnd.Float64() < 0.3

	return config
}

func triangular(rnd *random.XorWowRandom, low, high, mode float64) float64 {
	u := rnd.Float64()
	c := (mode - low) / (high - low)
	if u < c {
		return low + math.Sqrt(u*c*(high-low)*(mode-low))
	}
	return high - math.Sqrt((1-u)*(1-c)*(high-low)*(high-mode))
}

func fmod(x, y float64) float64 {
	return math.Mod(x, y)
}

func (p *NtscProcessor) cutBlackLineBorder(img *image.Image) {
	width := img.Width
	height := img.Height
	lineWidth := int(float64(width) * 0.017)

	for y := 0; y < height; y++ {
		rowStart := y * width * 3
		for x := width - lineWidth; x < width; x++ {
			pixelStart := rowStart + x*3
			img.Data[pixelStart] = 0
			img.Data[pixelStart+1] = 0
			img.Data[pixelStart+2] = 0
		}
	}
}

func shiftArray(arr []int32, shift int) []int32 {
	result := make([]int32, len(arr))
	if shift > 0 {
		copy(result[shift:], arr[:len(arr)-shift])
	} else if shift < 0 {
		copy(result[:len(arr)+shift], arr[-shift:])
	} else {
		copy(result, arr)
	}
	return result
}
