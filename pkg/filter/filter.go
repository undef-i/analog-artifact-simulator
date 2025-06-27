package filter

import (
	"math"
	"math/cmplx"
)

type LowpassFilter struct {
	Cutoff float64
	Order  int
}

func NewLowpassFilter(cutoff float64, order int) *LowpassFilter {
	return &LowpassFilter{
		Cutoff: cutoff,
		Order:  order,
	}
}

func (f *LowpassFilter) Apply(data []float64, sampleRate float64) []float64 {
	n := len(data)
	result := make([]float64, n)
	copy(result, data)

	nyquist := sampleRate / 2.0
	normalizedCutoff := f.Cutoff / nyquist

	if normalizedCutoff >= 1.0 {
		return result
	}

	alpha := math.Exp(-2.0 * math.Pi * normalizedCutoff)
	for i := 1; i < n; i++ {
		result[i] = alpha*result[i-1] + (1-alpha)*result[i]
	}

	return result
}

func (f *LowpassFilter) ApplyHighpass(data []float64, sampleRate float64) []float64 {
	lowpass := f.Apply(data, sampleRate)
	result := make([]float64, len(data))
	for i := range data {
		result[i] = data[i] - lowpass[i]
	}
	return result
}

func DFT(data []float64) []complex128 {
	n := len(data)
	result := make([]complex128, n)

	for k := 0; k < n; k++ {
		sum := complex(0, 0)
		for j := 0; j < n; j++ {
			angle := -2 * math.Pi * float64(k) * float64(j) / float64(n)
			sum += complex(data[j], 0) * cmplx.Exp(complex(0, angle))
		}
		result[k] = sum
	}

	return result
}

func IDFT(data []complex128) []float64 {
	n := len(data)
	result := make([]float64, n)

	for j := 0; j < n; j++ {
		sum := complex(0, 0)
		for k := 0; k < n; k++ {
			angle := 2 * math.Pi * float64(k) * float64(j) / float64(n)
			sum += data[k] * cmplx.Exp(complex(0, angle))
		}
		result[j] = real(sum) / float64(n)
	}

	return result
}

func CompositePreemphasis(data []float64, emphasis float64) []float64 {
	if len(data) == 0 {
		return data
	}

	result := make([]float64, len(data))
	result[0] = data[0]

	for i := 1; i < len(data); i++ {
		result[i] = data[i] - emphasis*data[i-1]
	}

	return result
}

func CompositeLowpass(data []float64, cutoff, sampleRate float64) []float64 {
	filter := NewLowpassFilter(cutoff, 1)
	return filter.Apply(data, sampleRate)
}
