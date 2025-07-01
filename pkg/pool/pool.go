package pool

import (
	"sync"
	"analog-artifact-simulator/pkg/image"
)

type ImagePool struct {
	pool sync.Pool
}

func NewImagePool() *ImagePool {
	return &ImagePool{
		pool: sync.Pool{
			New: func() interface{} {
				return &image.Image{}
			},
		},
	}
}

func (p *ImagePool) Get(width, height int) *image.Image {
	img := p.pool.Get().(*image.Image)
	if img.Width != width || img.Height != height || len(img.Data) != width*height*3 {
		img.Width = width
		img.Height = height
		if cap(img.Data) < width*height*3 {
			img.Data = make([]uint8, width*height*3)
		} else {
			img.Data = img.Data[:width*height*3]
		}
	}
	return img
}

func (p *ImagePool) Put(img *image.Image) {
	if img != nil {
		p.pool.Put(img)
	}
}

type YIQImagePool struct {
	pool sync.Pool
}

type YIQImage struct {
	Data   []int32
	Width  int
	Height int
}

func NewYIQImagePool() *YIQImagePool {
	return &YIQImagePool{
		pool: sync.Pool{
			New: func() interface{} {
				return &YIQImage{}
			},
		},
	}
}

func (p *YIQImagePool) Get(width, height int) *YIQImage {
	yiq := p.pool.Get().(*YIQImage)
	if yiq.Width != width || yiq.Height != height || len(yiq.Data) != width*height*3 {
		yiq.Width = width
		yiq.Height = height
		if cap(yiq.Data) < width*height*3 {
			yiq.Data = make([]int32, width*height*3)
		} else {
			yiq.Data = yiq.Data[:width*height*3]
		}
	}
	return yiq
}

func (p *YIQImagePool) Put(yiq *YIQImage) {
	if yiq != nil {
		p.pool.Put(yiq)
	}
}

type SlicePool struct {
	int32Pool   sync.Pool
	float64Pool sync.Pool
}

func NewSlicePool() *SlicePool {
	return &SlicePool{
		int32Pool: sync.Pool{
			New: func() interface{} {
				return make([]int32, 0, 1024)
			},
		},
		float64Pool: sync.Pool{
			New: func() interface{} {
				return make([]float64, 0, 1024)
			},
		},
	}
}

func (p *SlicePool) GetInt32(size int) []int32 {
	slice := p.int32Pool.Get().([]int32)
	if cap(slice) < size {
		slice = make([]int32, size)
	} else {
		slice = slice[:size]
	}
	return slice
}

func (p *SlicePool) PutInt32(slice []int32) {
	if slice != nil && cap(slice) >= 64 {
		p.int32Pool.Put(slice[:0])
	}
}

func (p *SlicePool) GetFloat64(size int) []float64 {
	slice := p.float64Pool.Get().([]float64)
	if cap(slice) < size {
		slice = make([]float64, size)
	} else {
		slice = slice[:size]
	}
	return slice
}

func (p *SlicePool) PutFloat64(slice []float64) {
	if slice != nil && cap(slice) >= 64 {
		p.float64Pool.Put(slice[:0])
	}
}

var (
	DefaultImagePool    = NewImagePool()
	DefaultYIQImagePool = NewYIQImagePool()
	DefaultSlicePool    = NewSlicePool()
)