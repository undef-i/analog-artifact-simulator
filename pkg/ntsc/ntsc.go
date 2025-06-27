package ntsc

import (
	"encoding/json"
	"math"
	"analog-artifact-simulator/pkg/image"
	"analog-artifact-simulator/pkg/random"
)

const (
	NTSC_RATE     = 315000000.0 / 88 * 4
	M_PI          = math.Pi
	Int_MIN_VALUE = -2147483648
	Int_MAX_VALUE = 2147483647
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
	for i, sample := range samples {
		result[i] = lp.Lowpass(sample)
	}
	return result
}

func (lp *LowpassFilter) HighpassArray(samples []float64) []float64 {
	result := make([]float64, len(samples))
	for i, sample := range samples {
		result[i] = lp.Highpass(sample)
	}
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
		RandomSeed:                    12345,
		RandomSeed2:                   67890,
	}
}

type NtscProcessor struct {
	Config  *NtscConfig
	Random  *random.XorWowRandom
	Precise bool
	Umult   []int32
	Vmult   []int32
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
	dst := img.Clone()
	yiq := p.bgr2yiq(img)

	for field := 0; field < 2; field++ {
		p.compositeLayer(dst, img, yiq, field, 0)
	}

	return dst
}

func (p *NtscProcessor) bgr2yiq(img *image.Image) [][][]int32 {
	height := img.Height
	width := img.Width
	yiq := make([][][]int32, 3)
	for i := range yiq {
		yiq[i] = make([][]int32, height)
		for j := range yiq[i] {
			yiq[i][j] = make([]int32, width)
		}
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pixel := img.GetPixel(x, y)
			r := float64(pixel.R)
			g := float64(pixel.G)
			b := float64(pixel.B)

			dY := 0.30*r + 0.59*g + 0.11*b

			yiq[0][y][x] = int32(dY * 256)
			yiq[1][y][x] = int32(256 * (-0.27*(b-dY) + 0.74*(r-dY)))
			yiq[2][y][x] = int32(256 * (0.41*(b-dY) + 0.48*(r-dY)))
		}
	}

	return yiq
}

func (p *NtscProcessor) yiq2bgr(yiq [][][]int32, dst *image.Image, field int) {
	height := len(yiq[0])
	width := len(yiq[0][0])

	for y := field; y < height; y += 2 {
		for x := 0; x < width; x++ {
			Y := float64(yiq[0][y][x])
			I := float64(yiq[1][y][x])
			Q := float64(yiq[2][y][x])

			r := int32((1.000*Y + 0.956*I + 0.621*Q) / 256)
			g := int32((1.000*Y + -0.272*I + -0.647*Q) / 256)
			b := int32((1.000*Y + -1.106*I + 1.703*Q) / 256)

			if r < 0 {
				r = 0
			}
			if r > 255 {
				r = 255
			}
			if g < 0 {
				g = 0
			}
			if g > 255 {
				g = 255
			}
			if b < 0 {
				b = 0
			}
			if b > 255 {
				b = 255
			}

			dst.SetPixel(x, y, image.Pixel{
				R: float64(r),
				G: float64(g),
				B: float64(b),
			})
		}
	}
}

func (p *NtscProcessor) compositeLayer(dst *image.Image, src *image.Image, yiq [][][]int32, field int, fieldno int) {
	if p.Config.ColorBleedBefore && (p.Config.ColorBleedVert != 0 || p.Config.ColorBleedHoriz != 0) {
		p.colorBleed(yiq, field)
	}

	if p.Config.CompositeInChromaLowpass {
		p.compositeLowpass(yiq, field, fieldno)
	}

	if p.Config.Ringing != 1.0 {
		p.ringing(yiq, field)
	}

	p.chromaIntoLuma(yiq, field, fieldno, p.Config.SubcarrierAmplitude)

	if p.Config.CompositePreemphasis != 0.0 && p.Config.CompositePreemphasisCut > 0 {
		p.compositePreemphasis(yiq, field, p.Config.CompositePreemphasis, p.Config.CompositePreemphasisCut)
	}

	if p.Config.VideoNoise != 0 {
		p.videoNoise(yiq, field, p.Config.VideoNoise)
	}

	if p.Config.VHSHeadSwitching {
		p.vhsHeadSwitching(yiq, field)
	}

	if !p.Config.NoColorSubcarrier {
		p.chromaFromLuma(yiq, field, fieldno, p.Config.SubcarrierAmplitudeBack)
	}

	if p.Config.VideoChromaNoise != 0 {
		p.videoChromaNoise(yiq, field, p.Config.VideoChromaNoise)
	}

	if p.Config.VideoChromaPhaseNoise != 0 {
		p.videoChromaPhaseNoise(yiq, field, p.Config.VideoChromaPhaseNoise)
	}

	if p.Config.EmulatingVHS {
		p.emulateVHS(yiq, field, fieldno)
	}

	if p.Config.VideoChromaLoss != 0 {
		p.vhsChromaLoss(yiq, field, p.Config.VideoChromaLoss)
	}

	if p.Config.CompositeOutChromaLowpass {
		if p.Config.CompositeOutChromaLowpassLite {
			p.compositeLowpassTV(yiq, field, fieldno)
		} else {
			p.compositeLowpass(yiq, field, fieldno)
		}
	}

	if !p.Config.ColorBleedBefore && (p.Config.ColorBleedVert != 0 || p.Config.ColorBleedHoriz != 0) {
		p.colorBleed(yiq, field)
	}

	p.blurChroma(yiq, field)

	p.yiq2bgr(yiq, dst, field)
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

func (p *NtscProcessor) chromaIntoLuma(yiq [][][]int32, field, fieldno, subcarrierAmplitude int) {
	height := len(yiq[0])
	width := len(yiq[0][0])

	for y := field; y < height; y += 2 {
		xi := p.chromaLumaXi(fieldno, y)

		for x := 0; x < width; x++ {
			umultIdx := (xi + x) % 4
			vmultIdx := (xi + x) % 4

			chroma := yiq[1][y][x]*int32(subcarrierAmplitude)*p.Umult[umultIdx] +
				yiq[2][y][x]*int32(subcarrierAmplitude)*p.Vmult[vmultIdx]

			yiq[0][y][x] += chroma / 50
			yiq[1][y][x] = 0
			yiq[2][y][x] = 0
		}
	}
}

func (p *NtscProcessor) chromaFromLuma(yiq [][][]int32, field, fieldno, subcarrierAmplitude int) {
	height := len(yiq[0])
	width := len(yiq[0][0])

	for y := field; y < height; y += 2 {
		Y := yiq[0][y]
		I := yiq[1][y]
		Q := yiq[2][y]

		chroma := make([]int32, width)

		sum := Y[0] + Y[1]

		y2 := make([]int32, width)
		yd4 := make([]int32, width)

		for i := 0; i < width-2; i++ {
			y2[i] = Y[i+2]
		}

		for i := 2; i < width; i++ {
			yd4[i] = Y[i-2]
		}

		sums := make([]int32, width)
		for i := 0; i < width; i++ {
			sums[i] = y2[i] - yd4[i]
		}

		sums0 := make([]int32, width+1)
		sums0[0] = sum
		for i := 0; i < width; i++ {
			sums0[i+1] = sums[i]
		}

		acc := make([]int32, width)
		accumulator := sums0[0]
		for i := 0; i < width; i++ {
			accumulator += sums0[i+1]
			acc[i] = accumulator
		}

		acc4 := make([]int32, width)
		for i := 0; i < width; i++ {
			acc4[i] = acc[i] / 4
		}

		for i := 0; i < width; i++ {
			chroma[i] = y2[i] - acc4[i]
		}

		for i := 0; i < width; i++ {
			Y[i] = acc4[i]
		}

		xi := p.chromaLumaXi(fieldno, y)

		x := (4 - xi) & 3

		for i := x + 2; i < width; i += 4 {
			chroma[i] = -chroma[i]
		}
		for i := x + 3; i < width; i += 4 {
			chroma[i] = -chroma[i]
		}

		for i := 0; i < width; i++ {
			chroma[i] = chroma[i] * 50 / int32(subcarrierAmplitude)
		}

		cxi := make([]int32, 0)
		for i := xi; i < width; i += 2 {
			cxi = append(cxi, -chroma[i])
		}

		cxi1 := make([]int32, 0)
		for i := xi + 1; i < width; i += 2 {
			cxi1 = append(cxi1, -chroma[i])
		}

		for i := 0; i < width; i++ {
			I[i] = 0
			Q[i] = 0
		}

		for i := 0; i < len(cxi) && i*2 < width; i++ {
			I[i*2] = cxi[i]
		}

		for i := 0; i < len(cxi1) && i*2 < width; i++ {
			Q[i*2] = cxi1[i]
		}

		for x := 1; x < width-2; x += 2 {
			I[x] = (I[x-1] + I[x+1]) >> 1
		}

		for x := 1; x < width-2; x += 2 {
			Q[x] = (Q[x-1] + Q[x+1]) >> 1
		}

		for x := width - 2; x < width; x++ {
			I[x] = 0
			Q[x] = 0
		}
	}
}

func (p *NtscProcessor) compositeLowpass(yiq [][][]int32, field, fieldno int) {
	height := len(yiq[0])
	width := len(yiq[0][0])

	for p := 1; p < 3; p++ {
		cutoff := 1300000.0
		delay := 2
		if p == 2 {
			cutoff = 600000.0
			delay = 4
		}

		lp := LowpassFilters(cutoff, 0.0, NTSC_RATE)
		for y := field; y < height; y += 2 {
			samples := make([]float64, width)
			for x := 0; x < width; x++ {
				samples[x] = float64(yiq[p][y][x])
			}

			f := lp[0].LowpassArray(samples)
			f = lp[1].LowpassArray(f)
			f = lp[2].LowpassArray(f)

			for x := 0; x < width-delay; x++ {
				yiq[p][y][x] = int32(f[x+delay])
			}
		}
	}
}

func (p *NtscProcessor) compositeLowpassTV(yiq [][][]int32, field, fieldno int) {
	height := len(yiq[0])
	width := len(yiq[0][0])

	for p := 1; p < 3; p++ {
		delay := 1
		lp := LowpassFilters(2600000.0, 0.0, NTSC_RATE)

		for y := field; y < height; y += 2 {
			samples := make([]float64, width)
			for x := 0; x < width; x++ {
				samples[x] = float64(yiq[p][y][x])
			}

			f := lp[0].LowpassArray(samples)
			f = lp[1].LowpassArray(f)
			f = lp[2].LowpassArray(f)

			for x := 0; x < width-delay; x++ {
				yiq[p][y][x] = int32(f[x+delay])
			}
		}
	}
}

func (p *NtscProcessor) compositePreemphasis(yiq [][][]int32, field int, compositePreemphasis, compositePreemphasisCut float64) {
	height := len(yiq[0])
	width := len(yiq[0][0])

	for y := field; y < height; y += 2 {
		pre := NewLowpassFilter(NTSC_RATE, compositePreemphasisCut, 16.0)

		samples := make([]float64, width)
		for x := 0; x < width; x++ {
			samples[x] = float64(yiq[0][y][x])
		}

		highpass := pre.HighpassArray(samples)
		for x := 0; x < width; x++ {
			filtered := samples[x] + highpass[x]*compositePreemphasis
			yiq[0][y][x] = int32(filtered)
		}
	}
}

func (p *NtscProcessor) videoNoise(yiq [][][]int32, field, videoNoise int) {
	height := len(yiq[0])
	width := len(yiq[0][0])

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
				yiq[0][y][x] += shiftedNoises[idx]
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
				yiq[0][y][x] += noise
				noise += rnds[x]
				noise = noise / 2
			}
		}
	}
}

func (p *NtscProcessor) videoChromaNoise(yiq [][][]int32, field, videoChromaNoise int) {
	height := len(yiq[0])
	width := len(yiq[0][0])

	noiseMod := videoChromaNoise*2 + 1
	fieldHeight := (height + 1) / 2

	if !p.Precise {
		lp := NewLowpassFilter(1, 1, 0)
		lp.alpha = 0.5

		rndsU := make([]float64, width*fieldHeight)
		rndsV := make([]float64, width*fieldHeight)
		for i := 0; i < len(rndsU); i++ {
			rndsU[i] = float64(p.Random.NextInt()%int32(noiseMod) - int32(videoChromaNoise))
			rndsV[i] = float64(p.Random.NextInt()%int32(noiseMod) - int32(videoChromaNoise))
		}

		noisesU := lp.LowpassArray(rndsU)
		noisesV := lp.LowpassArray(rndsV)
		noisesUInt := make([]int32, len(noisesU))
		noisesVInt := make([]int32, len(noisesV))
		for i, v := range noisesU {
			noisesUInt[i] = int32(v)
		}
		for i, v := range noisesV {
			noisesVInt[i] = int32(v)
		}
		shiftedNoisesU := shiftArray(noisesUInt, 1)
		shiftedNoisesV := shiftArray(noisesVInt, 1)

		idx := 0
		for y := field; y < height; y += 2 {
			for x := 0; x < width; x++ {
				yiq[1][y][x] += shiftedNoisesU[idx]
				yiq[2][y][x] += shiftedNoisesV[idx]
				idx++
			}
		}
	} else {
		noiseU := int32(0)
		noiseV := int32(0)
		for y := field; y < height; y += 2 {
			for x := 0; x < width; x++ {
				yiq[1][y][x] += noiseU
				noiseU += p.Random.NextInt()%int32(noiseMod) - int32(videoChromaNoise)
				noiseU = noiseU / 2

				yiq[2][y][x] += noiseV
				noiseV += p.Random.NextInt()%int32(noiseMod) - int32(videoChromaNoise)
				noiseV = noiseV / 2
			}
		}
	}
}

func (p *NtscProcessor) videoChromaPhaseNoise(yiq [][][]int32, field, videoChromaPhaseNoise int) {
	height := len(yiq[0])
	width := len(yiq[0][0])

	noiseMod := videoChromaPhaseNoise*2 + 1
	noise := int32(0)

	for y := field; y < height; y += 2 {
		noise += p.Random.NextInt()%int32(noiseMod) - int32(videoChromaPhaseNoise)
		noise = noise / 2
		pi := float64(noise) * M_PI / 100
		sinpi := math.Sin(pi)
		cospi := math.Cos(pi)

		for x := 0; x < width; x++ {
			u := float64(yiq[1][y][x])
			v := float64(yiq[2][y][x])

			newU := u*cospi - v*sinpi
			newV := u*sinpi + v*cospi

			yiq[1][y][x] = int32(newU)
			yiq[2][y][x] = int32(newV)
		}
	}
}

func (p *NtscProcessor) vhsHeadSwitching(yiq [][][]int32, field int) {
	height := len(yiq[0])
	width := len(yiq[0][0])

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

	switchPoint := int(math.Mod(p.Config.VHSHeadSwitchingPoint+noise, 1.0) * t)
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
				x2 := (tx + twidth + shif) % twidth

				for i := 0; i < width; i++ {
					tmp[i] = yiq[0][y][i]
				}

				x := tx
				for x < width {
					yiq[0][y][x] = tmp[x2]
					x2++
					if x2 == twidth {
						x2 = 0
					}
					x++
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

func (p *NtscProcessor) emulateVHS(yiq [][][]int32, field, fieldno int) {
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

func (p *NtscProcessor) vhsLumaLowpass(yiq [][][]int32, field int, lumaCut float64) {
	height := len(yiq[0])
	width := len(yiq[0][0])

	lp := LowpassFilters(lumaCut, 16.0, NTSC_RATE)
	pre := NewLowpassFilter(NTSC_RATE, lumaCut, 16.0)

	for y := field; y < height; y += 2 {
		samples := make([]float64, width)
		for x := 0; x < width; x++ {
			samples[x] = float64(yiq[0][y][x])
		}

		f0 := lp[0].LowpassArray(samples)
		f1 := lp[1].LowpassArray(f0)
		f2 := lp[2].LowpassArray(f1)
		highpass := pre.HighpassArray(f2)

		for x := 0; x < width; x++ {
			f3 := f2[x] + highpass[x]*1.6
			yiq[0][y][x] = int32(f3)
		}
	}
}

func (p *NtscProcessor) vhsChromaLowpass(yiq [][][]int32, field int, chromaCut float64, chromaDelay int) {
	height := len(yiq[0])
	width := len(yiq[0][0])

	for comp := 1; comp < 3; comp++ {
		lp := LowpassFilters(chromaCut, 0.0, NTSC_RATE)
		for y := field; y < height; y += 2 {
			samples := make([]float64, width)
			for x := 0; x < width; x++ {
				samples[x] = float64(yiq[comp][y][x])
			}

			f0 := lp[0].LowpassArray(samples)
			f1 := lp[1].LowpassArray(f0)
			f2 := lp[2].LowpassArray(f1)

			for x := 0; x < width-chromaDelay; x++ {
				yiq[comp][y][x] = int32(f2[x+chromaDelay])
			}
		}
	}
}

func (p *NtscProcessor) vhsChromaVertBlend(yiq [][][]int32, field int) {
	height := len(yiq[0])
	width := len(yiq[0][0])

	for comp := 1; comp < 3; comp++ {
		for y := field + 2; y < height; y += 2 {
			for x := 0; x < width; x++ {
				delay := yiq[comp][y-2][x]
				current := yiq[comp][y][x]
				yiq[comp][y][x] = (delay + current + 1) >> 1
			}
		}
	}
}

func (p *NtscProcessor) vhsSharpen(yiq [][][]int32, field int, lumaCut float64) {
	height := len(yiq[0])
	width := len(yiq[0][0])

	for y := field; y < height; y += 2 {
		lp1 := NewLowpassFilter(NTSC_RATE, lumaCut*4, 0.0)
		lp2 := NewLowpassFilter(NTSC_RATE, lumaCut*4, 0.0)
		lp3 := NewLowpassFilter(NTSC_RATE, lumaCut*4, 0.0)

		samples := make([]float64, width)
		for x := 0; x < width; x++ {
			samples[x] = float64(yiq[0][y][x])
		}

		ts := lp1.LowpassArray(samples)
		ts = lp2.LowpassArray(ts)
		ts = lp3.LowpassArray(ts)

		for x := 0; x < width; x++ {
			sharpened := samples[x] + (samples[x]-ts[x])*p.Config.VHSOutSharpen*2.0
			yiq[0][y][x] = int32(sharpened)
		}
	}
}

func (p *NtscProcessor) colorBleed(yiq [][][]int32, field int) {
	height := len(yiq[0])
	width := len(yiq[0][0])

	for comp := 1; comp < 3; comp++ {
		for y := field; y < height; y += 2 {
			for x := 0; x < width; x++ {
				srcY := y - p.Config.ColorBleedVert
				srcX := x - p.Config.ColorBleedHoriz

				if srcY >= 0 && srcY < height && srcX >= 0 && srcX < width {
					yiq[comp][y][x] = yiq[comp][srcY][srcX]
				}
			}
		}
	}
}

func (p *NtscProcessor) vhsEdgeWave(yiq [][][]int32, field int) {
	height := len(yiq[0])
	width := len(yiq[0][0])

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
				for x := 0; x < width; x++ {
					srcX := x - shift
					if srcX >= 0 && srcX < width {
						yiq[comp][y][x] = yiq[comp][y][srcX]
					} else {
						yiq[comp][y][x] = 0
					}
				}
			}
		}
	}
}

func (p *NtscProcessor) vhsChromaLoss(yiq [][][]int32, field, videoChromaLoss int) {
	height := len(yiq[0])

	for y := field; y < height; y += 2 {
		if p.Random.NextInt()%100000 < int32(videoChromaLoss) {
			for x := 0; x < len(yiq[1][y]); x++ {
				yiq[1][y][x] = 0
				yiq[2][y][x] = 0
			}
		}
	}
}

func (p *NtscProcessor) ringing(yiq [][][]int32, field int) {
	height := len(yiq[0])
	width := len(yiq[0][0])

	if !p.Config.EnableRinging2 {

		for comp := 0; comp < 3; comp++ {
			for y := field; y < height; y += 2 {

				original := make([]int32, width)
				copy(original, yiq[comp][y])

				for x := 1; x < width-1; x++ {
					diff := original[x+1] - original[x-1]
					ringingEffect := int32(float64(diff) * (p.Config.Ringing - 1.0) * 0.1)
					yiq[comp][y][x] += ringingEffect
				}
			}
		}
	} else {

		for comp := 0; comp < 3; comp++ {
			for y := field; y < height; y += 2 {
				p.ringing2(yiq[comp][y], p.Config.RingingPower, float64(p.Config.RingingShift))
			}
		}
	}
}

func (p *NtscProcessor) ringingFreqDomain(img []int32, alpha, noiseSize, noiseValue float64) {
	width := len(img)
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

	complexData = fft(complexData)

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

	complexData = ifft(complexData)

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

	samples := make([]float64, width)
	for i := 0; i < width; i++ {
		samples[i] = float64(img[i])
	}

	complexData := make([]complex128, width)
	for i := 0; i < width; i++ {
		complexData[i] = complex(samples[i], 0)
	}

	complexData = fft(complexData)

	complexData = fftShift(complexData)

	scaleCols := int(float64(width) * (1.0 + shift))
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
	start := (scaleCols / 2) - (width / 2)
	for i := 0; i < width; i++ {
		if start+i >= 0 && start+i < scaleCols {
			centerMask[i] = mask[start+i]
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

	complexData = ifft(complexData)

	for i := 0; i < width; i++ {
		img[i] = int32(real(complexData[i]))
	}
}

func (p *NtscProcessor) blurChroma(yiq [][][]int32, field int) {
	height := len(yiq[0])
	width := len(yiq[0][0])

	for comp := 1; comp < 3; comp++ {
		for y := field; y < height; y += 2 {
			original := make([]int32, width)
			copy(original, yiq[comp][y])

			for x := 1; x < width-1; x++ {

				avg := (original[x-1] + original[x]*2 + original[x+1]) / 4
				yiq[comp][y][x] = avg
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
