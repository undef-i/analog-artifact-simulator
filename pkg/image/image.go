package image

import (
	"image"
	"image/color"
	"math"
)

type RGBAPixel struct {
	R, G, B, A uint8
}

type Image struct {
	Width  int
	Height int
	Data   [][]Pixel
}

type Pixel struct {
	R, G, B float64
}

type YIQPixel struct {
	Y, I, Q float64
}

func NewImage(width, height int) *Image {
	data := make([][]Pixel, height)
	for i := range data {
		data[i] = make([]Pixel, width)
	}
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
	return img.Data[y][x]
}

func (img *Image) SetPixel(x, y int, pixel Pixel) {
	if x >= 0 && x < img.Width && y >= 0 && y < img.Height {
		img.Data[y][x] = pixel
	}
}

func (img *Image) Clone() *Image {
	newImg := NewImage(img.Width, img.Height)
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			newImg.Data[y][x] = img.Data[y][x]
		}
	}
	return newImg
}

func BGRToYIQ(pixel Pixel) YIQPixel {
	r, g, b := pixel.R/255.0, pixel.G/255.0, pixel.B/255.0
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

	return Pixel{R: r * 255.0, G: g * 255.0, B: b * 255.0}
}

func (img *Image) ToYIQ() [][]YIQPixel {
	yiq := make([][]YIQPixel, img.Height)
	for y := 0; y < img.Height; y++ {
		yiq[y] = make([]YIQPixel, img.Width)
		for x := 0; x < img.Width; x++ {
			yiq[y][x] = BGRToYIQ(img.Data[y][x])
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
			img.Data[y][x] = YIQToBGR(yiq[y][x])
		}
	}
	return img
}

func FromGoImage(src image.Image) *Image {
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	img := NewImage(width, height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, _ := src.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			img.Data[y][x] = Pixel{
				R: float64(r >> 8),
				G: float64(g >> 8),
				B: float64(b >> 8),
			}
		}
	}
	return img
}

func (img *Image) ToGoImage() image.Image {
	goImg := image.NewRGBA(image.Rect(0, 0, img.Width, img.Height))

	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			pixel := img.Data[y][x]
			goImg.Set(x, y, color.RGBA{
				R: uint8(math.Max(0, math.Min(255, pixel.R))),
				G: uint8(math.Max(0, math.Min(255, pixel.G))),
				B: uint8(math.Max(0, math.Min(255, pixel.B))),
				A: 255,
			})
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

			r := p1.R*(1-dx)*(1-dy) + p2.R*dx*(1-dy) + p3.R*(1-dx)*dy + p4.R*dx*dy
			g := p1.G*(1-dx)*(1-dy) + p2.G*dx*(1-dy) + p3.G*(1-dx)*dy + p4.G*dx*dy
			b := p1.B*(1-dx)*(1-dy) + p2.B*dx*(1-dy) + p3.B*(1-dx)*dy + p4.B*dx*dy

			resized.SetPixel(x, y, Pixel{R: r, G: g, B: b})
		}
	}

	return resized
}
