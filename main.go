package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type ycbcr struct {
	y  uint8
	cb uint8
	cr uint8
}

func ycbcr2rgb(pixel ycbcr) (uint8, uint8, uint8) {
	y_ := float64(pixel.y)
	cb_ := float64(pixel.cb) - 128.0
	cr_ := float64(pixel.cr) - 128.0

	// JPEG conversion (https://en.wikipedia.org/wiki/YCbCr#JPEG_conversion)
	r := uint8(math.Min(math.Max(0, math.Round(y_+1.402*cr_)), 255))
	g := uint8(math.Min(math.Max(0, math.Round(y_-0.344136*cb_-0.714136*cr_)), 255))
	b := uint8(math.Min(math.Max(0, math.Round(y_+1.772*cb_)), 255))

	// // ITU-R BT.709 conversion (https://en.wikipedia.org/wiki/YCbCr#ITU-R_BT.709_conversion)
	// r := uint8(math.Min(math.Max(0, math.Round(y_+1.5748*cr_)), 255))
	// g := uint8(math.Min(math.Max(0, math.Round(y_-0.1873*cb_-0.4681*cr_)), 255))
	// b := uint8(math.Min(math.Max(0, math.Round(y_+1.8556*cb_)), 255))

	// // YUV conversion instead (https://en.wikipedia.org/wiki/YUV#HDTV_with_BT.709)
	// r := uint8(math.Min(math.Max(0, math.Round(y_+1.28033*cr_)), 255))
	// g := uint8(math.Min(math.Max(0, math.Round(y_-0.21482*cb_-0.38059*cr_)), 255))
	// b := uint8(math.Min(math.Max(0, math.Round(y_+2.12798*cb_)), 255))

	return r, g, b
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func saveImageToPng(path string, img image.Image) {
	outFile, err := os.Create(path)
	check(err)
	defer outFile.Close()

	err = png.Encode(outFile, img)
	check(err)
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: ithmbrdr id filename")
		os.Exit(1)
	}

	thumbsPath, err := filepath.Abs(os.Args[2])
	check(err)
	fmt.Printf("Reading from %s\n", thumbsPath)
	thumbsName := strings.Split(filepath.Base(thumbsPath), ".")[0]
	fmt.Printf("File is %s\n", thumbsName)

	// Create a directory of this name if it does not exist
	if _, err = os.Stat(thumbsName); errors.Is(err, os.ErrNotExist) {
		err = os.Mkdir(thumbsName, os.ModePerm)
		check(err)
	}

	f, err := os.Open(thumbsPath)
	check(err)
	defer f.Close()

	thumbIdMin := 0
	thumbIdMax := int((^uint(0)) >> 1)

	if os.Args[1] != "*" {
		thumbId, err := strconv.Atoi(os.Args[1])
		if err != nil {
			// Input might be a range
			thumbIds := strings.Split(os.Args[1], "-")
			if len(thumbIds) != 2 {
				fmt.Println("Ranges need to be of the form d1-d2, e.g. 2-10")
				os.Exit(1)
			} else {
				thumbIdMin, err = strconv.Atoi(thumbIds[0])
				check(err)
				thumbIdMax, err = strconv.Atoi(thumbIds[1])
				check(err)
				thumbIdMax += 1
			}
		} else {
			thumbIdMin = thumbId
			thumbIdMax = thumbId + 1
		}
	}

	thumbWidth := 720
	thumbHeight := 480
	thumbPixels := thumbWidth * thumbHeight
	thumbBytes := thumbPixels * 2
	thumbReadBuf := make([]byte, thumbBytes)

	// Start at minimum requested image ID
	f.Seek(int64(thumbBytes*thumbIdMin), 0)

	for i := thumbIdMin; i < thumbIdMax; i += 1 {
		_, err = io.ReadAtLeast(f, thumbReadBuf, thumbBytes)
		if err == io.EOF {
			fmt.Printf("Reached EOF at thumb %d\n", i-1)
			os.Exit(0)
		} else if err != nil {
			panic(err)
		}

		// // Extract all available image information as we know it
		// outImageAll := image.NewGray(image.Rect(0, 0, thumbWidth*2, thumbHeight))
		// for y := 0; y < thumbHeight; y = y + 1 {
		// 	for x := 0; x < thumbWidth*2; x = x + 1 {
		// 		luminance := thumbReadBuf[y*thumbWidth*2+x]
		// 		outImageAll.Set(x, y, color.Gray{luminance})
		// 	}
		// }

		// saveImageToPng("thumb_all.png", outImageAll)

		// // Extract luminance only and produce grayscale bitmap
		// outImageLuminance := image.NewGray(image.Rect(0, 0, thumbWidth, thumbHeight))
		// maxLuminance := byte(0)
		// for y := 0; y < thumbHeight; y = y + 1 {
		// 	for x := 0; x < thumbWidth; x = x + 1 {
		// 		luminance := thumbReadBuf[y*thumbWidth+x]
		// 		if maxLuminance < luminance {
		// 			maxLuminance = luminance
		// 		}
		// 		outImageLuminance.Set(x, y, color.Gray{luminance})
		// 	}
		// }
		// fmt.Printf("Maximum observed luminance: %d\n", maxLuminance)

		// saveImageToPng("thumb_luminance.png", outImageLuminance)

		// // Extract chroma only and produce grayscale bitmap
		// outImageChromaBlue := image.NewGray(image.Rect(0, 0, thumbWidth/2, thumbHeight/2))
		// outImageChromaRed := image.NewGray(image.Rect(0, 0, thumbWidth/2, thumbHeight/2))
		// maxChromaBlue := byte(0)
		// maxChromaRed := byte(0)
		// for y := 0; y < thumbHeight/2; y = y + 1 {
		// 	for x := 0; x < thumbWidth/2; x = x + 1 {
		// 		chromaBlue := thumbReadBuf[thumbPixels+y*thumbWidth/2+x]
		// 		if maxChromaBlue < chromaBlue {
		// 			maxChromaBlue = chromaBlue
		// 		}
		// 		outImageChromaBlue.Set(x, y, color.Gray{chromaBlue})

		// 		chromaRed := thumbReadBuf[thumbPixels*5/4+y*thumbWidth/2+x]
		// 		if maxChromaRed < chromaRed {
		// 			maxChromaRed = chromaRed
		// 		}
		// 		outImageChromaRed.Set(x, y, color.Gray{chromaRed})
		// 	}
		// }
		// fmt.Printf("Maximum observed chroma blue: %d\n", maxChromaBlue)
		// fmt.Printf("Maximum observed chroma red: %d\n", maxChromaRed)

		// saveImageToPng("thumb_chroma_blue.png", outImageChromaBlue)
		// saveImageToPng("thumb_chroma_red.png", outImageChromaRed)

		// Create array of y, cb, cr components
		outImage := image.NewRGBA(image.Rect(0, 0, thumbWidth, thumbHeight))
		for y := 0; y < thumbHeight; y = y + 1 {
			for x := 0; x < thumbWidth; x = x + 1 {
				luminance := thumbReadBuf[y*thumbWidth+x]
				chromaBlue := thumbReadBuf[thumbPixels+(y/2)*thumbWidth/2+(x/2)]
				chromaRed := thumbReadBuf[thumbPixels*5/4+(y/2)*thumbWidth/2+(x/2)]

				r, g, b := ycbcr2rgb(ycbcr{luminance, chromaBlue, chromaRed})
				outImage.Set(x, y, color.RGBA{r, g, b, 255})
			}
		}

		saveImageToPng(
			thumbsName+"/"+thumbsName+"_"+strconv.FormatInt(int64(i), 10)+".png",
			outImage,
		)

		// // Create histogram
		// var histRed [256]uint8
		// var histGreen [256]uint8
		// var histBlue [256]uint8
		// for y := 0; y < thumbHeight; y = y + 1 {
		// 	for x := 0; x < thumbWidth; x = x + 1 {
		// 		col := outImage.RGBAAt(x, y)
		// 		histRed[col.R] += 1
		// 		histGreen[col.G] += 1
		// 		histBlue[col.B] += 1
		// 	}
		// }

		// // Determine maximum values
		// maxRed := uint8(0)
		// maxGreen := uint8(0)
		// maxBlue := uint8(0)
		// for j := 0; j < 256; j += 1 {
		// 	if histRed[j] > maxRed {
		// 		maxRed = histRed[j]
		// 	}
		// 	if histGreen[j] > maxGreen {
		// 		maxGreen = histGreen[j]
		// 	}
		// 	if histBlue[j] > maxBlue {
		// 		maxBlue = histBlue[j]
		// 	}
		// }
		// maxCol := maxRed
		// if maxGreen > maxRed {
		// 	if maxBlue > maxGreen {
		// 		maxCol = maxBlue
		// 	} else {
		// 		maxCol = maxGreen
		// 	}
		// }

		// xPad := 10
		// yPad := 10
		// histHeight := 50

		// // Normalize to histogram height
		// scaleRed := float64(histHeight) / float64(maxCol)
		// scaleGreen := float64(histHeight) / float64(maxCol)
		// scaleBlue := float64(histHeight) / float64(maxCol)

		// outImageHistogram := image.NewRGBA(image.Rect(0, 0, 256+2*xPad, 3*histHeight+4*yPad))
		// for j := 0; j < 256+2*xPad; j += 1 {
		// 	for l := 0; l < 3*histHeight+4*yPad; l += 1 {
		// 		outImageHistogram.Set(j, l, color.RGBA{45, 45, 45, 255})
		// 	}
		// }
		// for j := 0; j < 256; j += 1 {
		// 	for l := 0; l < int(float64(histRed[j])*scaleRed); l += 1 {
		// 		outImageHistogram.Set(xPad+j, histHeight+yPad-l, color.RGBA{255, 0, 0, 255})
		// 	}

		// 	for l := 0; l < int(float64(histGreen[j])*scaleGreen); l += 1 {
		// 		outImageHistogram.Set(xPad+j, 2*histHeight+2*yPad-l, color.RGBA{0, 255, 0, 255})
		// 	}

		// 	for l := 0; l < int(float64(histBlue[j])*scaleBlue); l += 1 {
		// 		outImageHistogram.Set(xPad+j, 3*histHeight+3*yPad-l, color.RGBA{0, 0, 255, 255})
		// 	}
		// }

		// saveImageToPng(
		// 	thumbsName + "/" + thumbsName + "_" + strconv.FormatInt(int64(i), 10) + "_histogram.png",
		// 	outImageHistogram,
		// )
	}

	os.Exit(0)
}
