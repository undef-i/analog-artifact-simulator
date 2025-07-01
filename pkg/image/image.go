package image

import (
	"image"
	"image/draw"
	"math"
)

type RGBAPixel struct {
	R, G, B, A uint8
}

type Image struct {
	Width  int
	Height int
	Data   []uint8 // R, G, B, R, G, B, ...
}

type Pixel struct {
	R, G, B uint8
}

type YIQPixel struct {
	Y, I, Q float64
}

func NewImage(width, height int) *Image {
	data := make([]uint8, width*height*3)
	return &Image{
		Width:  width,
		Height: height,
		Data:   data,
	}
}

func (img *Image) GetPixel(x, y int) Pixel {
	if x < 0 || x >= img.Width || y < 0 || y >= img.Height {
		return Pixel{0, 0, 0}
	}
	idx := (y*img.Width + x) * 3
	return Pixel{R: img.Data[idx], G: img.Data[idx+1], B: img.Data[idx+2]}
}

func (img *Image) SetPixel(x, y int, pixel Pixel) {
	if x >= 0 && x < img.Width && y >= 0 && y < img.Height {
		idx := (y*img.Width + x) * 3
		img.Data[idx] = pixel.R
		img.Data[idx+1] = pixel.G
		img.Data[idx+2] = pixel.B
	}
}

func (img *Image) Clone() *Image {
	newImg := NewImage(img.Width, img.Height)
	copy(newImg.Data, img.Data)
	return newImg
}

// CloneWithPool creates a copy using object pool for better performance
func (img *Image) CloneWithPool(pool interface{}) *Image {
	if imagePool, ok := pool.(interface {
		Get(int, int) *Image
	}); ok {
		newImg := imagePool.Get(img.Width, img.Height)
		copy(newImg.Data, img.Data)
		return newImg
	}
	return img.Clone()
}

func BGRToYIQ(pixel Pixel) YIQPixel {
	r, g, b := float64(pixel.R)/255.0, float64(pixel.G)/255.0, float64(pixel.B)/255.0
	y := 0.299*r + 0.587*g + 0.114*b
	i := 0.5959*r - 0.2746*g - 0.3213*b
	q := 0.2115*r - 0.5227*g + 0.3112*b
	return YIQPixel{Y: y, I: i, Q: q}
}

func YIQToBGR(yiq YIQPixel) Pixel {
	r := yiq.Y + 0.956*yiq.I + 0.619*yiq.Q
	g := yiq.Y - 0.272*yiq.I - 0.647*yiq.Q
	b := yiq.Y - 1.106*yiq.I + 1.703*yiq.Q

	r = math.Max(0, math.Min(1, r))
	g = math.Max(0, math.Min(1, g))
	b = math.Max(0, math.Min(1, b))

	return Pixel{R: uint8(r * 255.0), G: uint8(g * 255.0), B: uint8(b * 255.0)}
}

func (img *Image) ToYIQ() [][]YIQPixel {
	yiq := make([][]YIQPixel, img.Height)
	for y := 0; y < img.Height; y++ {
		yiq[y] = make([]YIQPixel, img.Width)
		for x := 0; x < img.Width; x++ {
			yiq[y][x] = BGRToYIQ(img.GetPixel(x, y))
		}
	}
	return yiq
}

func YIQToImage(yiq [][]YIQPixel) *Image {
	height := len(yiq)
	width := len(yiq[0])
	img := NewImage(width, height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetPixel(x, y, YIQToBGR(yiq[y][x]))
		}
	}
	return img
}

func FromGoImage(src image.Image) *Image {
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	img := NewImage(width, height)

	srcRGBA, ok := src.(*image.RGBA)
	if !ok {
		srcRGBA = image.NewRGBA(bounds)
		draw.Draw(srcRGBA, bounds, src, bounds.Min, draw.Src)
	}

	// Optimized batch copy with stride calculation
	stride := srcRGBA.Stride
	for y := 0; y < height; y++ {
		srcRowStart := y * stride
		dstRowStart := y * width * 3
		for x := 0; x < width; x++ {
			srcIdx := srcRowStart + x*4
			dstIdx := dstRowStart + x*3
			img.Data[dstIdx] = srcRGBA.Pix[srcIdx]
			img.Data[dstIdx+1] = srcRGBA.Pix[srcIdx+1]
			img.Data[dstIdx+2] = srcRGBA.Pix[srcIdx+2]
		}
	}
	return img
}

func (img *Image) ToGoImage() image.Image {
	goImg := image.NewRGBA(image.Rect(0, 0, img.Width, img.Height))

	// Optimized batch copy with stride calculation
	stride := goImg.Stride
	for y := 0; y < img.Height; y++ {
		srcRowStart := y * img.Width * 3
		dstRowStart := y * stride
		for x := 0; x < img.Width; x++ {
			srcIdx := srcRowStart + x*3
			dstIdx := dstRowStart + x*4
			goImg.Pix[dstIdx] = img.Data[srcIdx]
			goImg.Pix[dstIdx+1] = img.Data[srcIdx+1]
			goImg.Pix[dstIdx+2] = img.Data[srcIdx+2]
			goImg.Pix[dstIdx+3] = 255
		}
	}
	return goImg
}

func (img *Image) Resize(maxWidth, maxHeight int) *Image {
	if maxWidth <= 0 && maxHeight <= 0 {
		return img.Clone()
	}

	originalWidth := float64(img.Width)
	originalHeight := float64(img.Height)

	var scaleX, scaleY float64 = 1.0, 1.0

	if maxWidth > 0 {
		scaleX = float64(maxWidth) / originalWidth
	}
	if maxHeight > 0 {
		scaleY = float64(maxHeight) / originalHeight
	}

	scale := math.Min(scaleX, scaleY)
	if scale >= 1.0 {
		return img.Clone()
	}

	newWidth := int(originalWidth * scale)
	newHeight := int(originalHeight * scale)

	resized := NewImage(newWidth, newHeight)

	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			srcX := float64(x) / scale
			srcY := float64(y) / scale

			x1 := int(math.Floor(srcX))
			y1 := int(math.Floor(srcY))
			x2 := int(math.Min(float64(x1+1), originalWidth-1))
			y2 := int(math.Min(float64(y1+1), originalHeight-1))

			dx := srcX - float64(x1)
			dy := srcY - float64(y1)

			p1 := img.GetPixel(x1, y1)
			p2 := img.GetPixel(x2, y1)
			p3 := img.GetPixel(x1, y2)
			p4 := img.GetPixel(x2, y2)

			r := float64(p1.R)*(1-dx)*(1-dy) + float64(p2.R)*dx*(1-dy) + float64(p3.R)*(1-dx)*dy + float64(p4.R)*dx*dy
			g := float64(p1.G)*(1-dx)*(1-dy) + float64(p2.G)*dx*(1-dy) + float64(p3.G)*(1-dx)*dy + float64(p4.G)*dx*dy
			b := float64(p1.B)*(1-dx)*(1-dy) + float64(p2.B)*dx*(1-dy) + float64(p3.B)*(1-dx)*dy + float64(p4.B)*dx*dy

			resized.SetPixel(x, y, Pixel{R: uint8(r), G: uint8(g), B: uint8(b)})
		}
	}

	return resized
}
