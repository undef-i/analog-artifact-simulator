package ntsc

import (
	"math"
	"math/cmplx"
)

func fft(x []complex128) []complex128 {
	N := len(x)
	if N <= 1 {
		return x
	}

	even := make([]complex128, N/2)
	odd := make([]complex128, N/2)
	for i := 0; i < N/2; i++ {
		even[i] = x[2*i]
		odd[i] = x[2*i+1]
	}

	even = fft(even)
	odd = fft(odd)

	result := make([]complex128, N)
	for k := 0; k < N/2; k++ {
		angle := -2 * math.Pi * float64(k) / float64(N)
		tw := cmplx.Rect(1, angle) * odd[k]
		result[k] = even[k] + tw
		result[k+N/2] = even[k] - tw
	}

	return result
}

func ifft(x []complex128) []complex128 {
	N := len(x)

	x_conj := make([]complex128, N)
	for i := 0; i < N; i++ {
		x_conj[i] = cmplx.Conj(x[i])
	}

	y := fft(x_conj)

	result := make([]complex128, N)
	for i := 0; i < N; i++ {
		result[i] = cmplx.Conj(y[i]) / complex(float64(N), 0)
	}

	return result
}

func fftShift(x []complex128) []complex128 {
	N := len(x)
	result := make([]complex128, N)

	midpoint := N / 2

	for i := 0; i < midpoint; i++ {
		result[i] = x[i+midpoint]
	}

	for i := 0; i < midpoint; i++ {
		result[i+midpoint] = x[i]
	}

	return result
}

func ifftShift(x []complex128) []complex128 {

	return fftShift(x)
}
