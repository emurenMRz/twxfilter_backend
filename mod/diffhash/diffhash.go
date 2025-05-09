package diffhash

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"os"
)

func openImage(inputNane string) (img image.Image, err error) {
	file, err := os.Open(inputNane)
	if err != nil {
		return
	}
	defer file.Close()

	return decodeImageFromReader(file)
}

func decodeImage(inputIamge []byte) (img image.Image, err error) {
	return decodeImageFromReader(bytes.NewReader(inputIamge))
}

func decodeImageFromReader(reader io.Reader) (img image.Image, err error) {
	_, format, err := image.DecodeConfig(reader)
	if err != nil {
		return
	}

	if seeker, ok := reader.(io.Seeker); ok {
		seeker.Seek(0, 0)
	}

	if format == "jpeg" {
		img, err = jpeg.Decode(reader)
	} else if format == "png" {
		img, err = png.Decode(reader)
	} else {
		err = fmt.Errorf("unsupported image format: %s", format)
	}

	return
}

func toGray(img image.Image) *image.Gray {
	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y
	gray := image.NewGray(bounds)
	grayPix := gray.Pix

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			grayPix[y*width+x] = uint8(0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b))
		}
	}

	return gray
}

func resizeImage(img image.Image, newWidth, newHeight int) image.Image {
	bounds := img.Bounds()
	srcWidth, srcHeight := bounds.Max.X, bounds.Max.Y
	if srcWidth <= newWidth || srcHeight <= newHeight {
		return img
	}

	rgbaImg := image.NewRGBA(img.Bounds())
	draw.Draw(rgbaImg, img.Bounds(), img, image.Point{}, draw.Src)
	sPix := rgbaImg.Pix
	sPitch := rgbaImg.Stride

	resizedImg := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	dPix := resizedImg.Pix
	dPitch := resizedImg.Stride

	blockWidth := float64(srcWidth) / float64(newWidth)
	blockHeight := float64(srcHeight) / float64(newHeight)
	sIndex := 0.0
	dIndex := 0
	for y := 0; y < newHeight; y++ {
		sOffset := 0.0
		dOffset := 0
		for x := 0; x < newWidth; x++ {
			r := 0
			g := 0
			b := 0
			count := 0

			index := int(sIndex) * sPitch
			for by := 0; by < int(blockHeight); by++ {
				offset := int(sOffset) << 2
				for bx := 0; bx < int(blockWidth); bx++ {
					r += int(sPix[index+offset+0])
					g += int(sPix[index+offset+1])
					b += int(sPix[index+offset+2])
					offset += 4
					count++
				}
				index += sPitch
			}

			if count > 0 {
				dPix[dIndex+dOffset+0] = uint8(r / count)
				dPix[dIndex+dOffset+1] = uint8(g / count)
				dPix[dIndex+dOffset+2] = uint8(b / count)
				dPix[dIndex+dOffset+3] = 0xff
			}

			sOffset += blockWidth
			dOffset += 4
		}
		sIndex += blockHeight
		dIndex += dPitch
	}

	return resizedImg
}

func isMonochrome(grayImage *image.Gray) bool {
	width, height := grayImage.Bounds().Size().X, grayImage.Bounds().Size().Y
	border := int(float64(width) * float64(height) * (2.0 / 3.0))

	numbers := map[uint8]int{}
	for i := 0; i < width*height; i++ {
		c := grayImage.Pix[i]
		if c < 8 {
			c = 0
		} else if c > 247 {
			c = 255
		}
		numbers[c]++
	}

	for _, count := range numbers {
		if count > border {
			return true
		}
	}

	return false
}

func calcDiffHash(grayImage *image.Gray) (hash uint64) {
	size := grayImage.Bounds().Size()
	width, height := size.X, size.Y
	pix := grayImage.Pix
	index := 0
	for y := 0; y < height; y++ {
		for x := 0; x < width-1; x++ {
			left := pix[index+x]
			right := pix[index+x+1]
			if left > right {
				hash |= 1
			}
			hash <<= 1
		}
		index += width
	}

	return
}

func computeImageDiffHash(img image.Image) uint64 {
	grayImage := toGray(resizeImage(img, 9, 8))
	if isMonochrome(grayImage) {
		return 0
	}

	return calcDiffHash(grayImage)
}

func CalcDiffHashFromFile(inputName string) uint64 {
	img, err := openImage(inputName)
	if err != nil {
		return 0
	}

	return computeImageDiffHash(img)
}

func CalcDiffHashFromImage(inputImage []byte) uint64 {
	img, err := decodeImage(inputImage)
	if err != nil {
		return 0
	}

	return computeImageDiffHash(img)
}

func CompDiffHash(hash1 uint64, hash2 uint64) int {
	bits := hash1 ^ hash2
	bits = bits - ((bits >> 1) & 0x5555555555555555)
	bits = (bits & 0x3333333333333333) + ((bits >> 2) & 0x3333333333333333)
	bits = ((bits >> 4) + bits) & 0x0f0f0f0f0f0f0f0f
	bits = (bits >> 8) + bits
	bits = (bits >> 16) + bits

	return int(((bits >> 32) + bits) & 0x7f)
}
