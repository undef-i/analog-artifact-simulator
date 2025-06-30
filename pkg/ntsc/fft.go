package ntsc

import (
	"math"
	"math/cmplx"
)

func bitReverseCopy(x []complex128) []complex128 {
	N := len(x)
	result := make([]complex128, N)
	for i := 0; i < N; i++ {
		rev := 0
		for j := 0; 1<<j < N; j++ {
			rev = (rev << 1) | ((i >> j) & 1)
		}
		result[rev] = x[i]
	}
	return result
}

func fft(x []complex128) []complex128 {
	N := len(x)
	if N == 0 || (N&(N-1)) != 0 { // Check if N is a power of 2
		// Handle non-power-of-2 sizes if necessary, or return error
		return nil // For simplicity, returning nil for now
	}

	data := bitReverseCopy(x)

	for s := 1; 1<<s <= N; s++ {
		m := 1 << s
		wm := cmplx.Exp(complex(0, -2*math.Pi/float64(m)))
		for k := 0; k < N; k += m {
			w := complex(1, 0)
			for j := 0; j < m/2; j++ {
				t := data[k+j+m/2] * w
				u := data[k+j]
				data[k+j] = u + t
				data[k+j+m/2] = u - t
				w *= wm
			}
		}
	}
	return data
}

func ifft(x []complex128) []complex128 {
	N := len(x)
	if N == 0 || (N&(N-1)) != 0 { // Check if N is a power of 2
		return nil // For simplicity, returning nil for now
	}

	data := bitReverseCopy(x)

	for s := 1; 1<<s <= N; s++ {
		m := 1 << s
		wm := cmplx.Exp(complex(0, 2*math.Pi/float64(m)))
		for k := 0; k < N; k += m {
			w := complex(1, 0)
			for j := 0; j < m/2; j++ {
				t := data[k+j+m/2] * w
				u := data[k+j]
				data[k+j] = u + t
				data[k+j+m/2] = u - t
				w *= wm
			}
		}
	}

	for i := 0; i < N; i++ {
		data[i] /= complex(float64(N), 0)
	}
	return data
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
