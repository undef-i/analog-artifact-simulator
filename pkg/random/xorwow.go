package random

import (
	"math"
	"time"
)

type XorWowRandom struct {
	x, y, z, w, v, d uint32
}

func NewXorWowRandom(seed uint32) *XorWowRandom {
	if seed == 0 {
		seed = uint32(time.Now().UnixNano())
	}
	return &XorWowRandom{
		x: seed,
		y: 362436069,
		z: 521288629,
		w: 88675123,
		v: 5783321,
		d: 6615241,
	}
}

func (r *XorWowRandom) Next() uint32 {
	t := r.x ^ (r.x >> 2)
	r.x = r.y
	r.y = r.z
	r.z = r.w
	r.w = r.v
	r.v = (r.v ^ (r.v << 4)) ^ (t ^ (t << 1))
	r.d += 362437
	return r.d + r.v
}

func (r *XorWowRandom) Float64() float64 {
	return float64(r.Next()) / float64(uint32(0xFFFFFFFF))
}

func (r *XorWowRandom) Normal(mean, stddev float64) float64 {
	u1 := r.Float64()
	u2 := r.Float64()
	z0 := math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
	return z0*stddev + mean
}

func (r *XorWowRandom) Uniform(min, max float64) float64 {
	return min + (max-min)*r.Float64()
}

func (r *XorWowRandom) NextInt() int32 {
	return int32(r.Next())
}
