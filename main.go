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
	"sync"
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

type indexedBuffer struct {
	index int
	buf   []byte
}

type indexedImage struct {
	index int
	img   image.Image
}

func readBuffer(f *os.File, size int) ([]byte, error) {
	buf := make([]byte, size)
	_, err := io.ReadAtLeast(f, buf, size)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func readBufferAsync(path string, idMin int, idMax int, bytes int, jobs chan<- indexedBuffer, queued chan<- int) {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	defer f.Close()
	defer func() {
		close(jobs)
		close(queued)
	}()

	// Start at minimum requested image ID
	f.Seek(int64(bytes*idMin), 0)

	for i := idMin; i < idMax; i += 1 {
		buf, err := readBuffer(f, bytes)
		if errors.Is(err, io.EOF) {
			return
		} else if err != nil {
			fmt.Println(err)
			return
		}

		jobs <- indexedBuffer{i, buf}
		queued <- i
	}
}

func convertYcbcr2Rgba(buf []byte, width int, height int) image.Image {
	// Create array of y, cb, cr components
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y = y + 1 {
		for x := 0; x < width; x = x + 1 {
			luminance := buf[y*width+x]
			chromaBlue := buf[width*height+(y/2)*width/2+(x/2)]
			chromaRed := buf[width*height*5/4+(y/2)*width/2+(x/2)]

			r, g, b := ycbcr2rgb(ycbcr{luminance, chromaBlue, chromaRed})
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}

	return img
}

func convertYcbcr2RgbaAsync(jobs <-chan indexedBuffer, results chan<- indexedImage, processed chan<- int, width int, height int, wg *sync.WaitGroup) {
	defer wg.Done()

	for ibuf := range jobs {
		processed <- ibuf.index
		results <- indexedImage{ibuf.index, convertYcbcr2Rgba(ibuf.buf, width, height)}
	}
}

func printStatus(queued, processed, written <-chan int, wg *sync.WaitGroup) {
	defer wg.Done()

	total_queued := 0
	total_processed := 0
	total_written := 0
	for processed != nil || queued != nil || written != nil {
		fmt.Printf(
			"\rWritten/Processed/Queued: %d/%d/%d",
			total_written,
			total_processed,
			total_queued,
		)
		select {
		case _, ok := <-queued:
			if !ok {
				queued = nil
			} else {
				total_queued += 1
			}
		case _, ok := <-processed:
			if !ok {
				processed = nil
			} else {
				total_processed += 1
			}
		case _, ok := <-written:
			if !ok {
				written = nil
			} else {
				total_written += 1
			}
		}
	}

	fmt.Printf("\n")
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func saveImageToPng(path string, img image.Image) error {
	outFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer outFile.Close()

	return png.Encode(outFile, img)
}

func saveImageToPngAsync(results <-chan indexedImage, written chan<- int, name string, wg *sync.WaitGroup) {
	defer wg.Done()

	for iimg := range results {
		path := name + "/" + name + "_" + strconv.FormatInt(int64(iimg.index), 10) + ".png"
		if err := saveImageToPng(path, iimg.img); err != nil {
			fmt.Printf("Failed to save image %d\n", iimg.index)
			fmt.Println(err.Error())
		}
		written <- iimg.index
	}
}

func main() {
	var err error

	if !(len(os.Args) == 3 || len(os.Args) == 5) || (len(os.Args) == 5 && os.Args[1] != "-c") {
		fmt.Println("Usage: ithmbrdr [-c channelSize] id filename")
		os.Exit(1)
	}

	var idString string
	var pathString string

	size := 100
	if len(os.Args) == 5 {
		size, err = strconv.Atoi(os.Args[2])
		if err != nil {
			panic(err)
		}
		idString = os.Args[3]
		pathString = os.Args[4]
	} else {
		idString = os.Args[1]
		pathString = os.Args[2]
	}
	fmt.Printf("Using channel size %d\n", size)

	path, err := filepath.Abs(pathString)
	check(err)
	fmt.Printf("Reading from %s\n", path)
	name := strings.Split(filepath.Base(path), ".")[0]
	fmt.Printf("File is %s\n", name)

	// Create a directory of this name if it does not exist
	if _, err = os.Stat(name); errors.Is(err, os.ErrNotExist) {
		if err = os.Mkdir(name, os.ModePerm); err != nil {
			panic(err)
		}
	}

	thumbIdMin := 0
	thumbIdMax := int((^uint(0)) >> 1)

	if idString != "*" {
		thumbId, err := strconv.Atoi(idString)
		if err != nil {
			// Input might be a range
			thumbIds := strings.Split(idString, "-")
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
	thumbBytes := thumbWidth * thumbHeight * 2

	jobs := make(chan indexedBuffer, size)
	results := make(chan indexedImage, size)

	queued := make(chan int, size)
	processed := make(chan int, size)
	written := make(chan int, size)

	wgProcess := new(sync.WaitGroup)
	wgWrite := new(sync.WaitGroup)
	wgPrint := new(sync.WaitGroup)

	wgPrint.Add(1)
	go printStatus(queued, processed, written, wgPrint)

	// Start some workers
	for w := 1; w <= 5; w++ {
		wgProcess.Add(1)
		go convertYcbcr2RgbaAsync(jobs, results, processed, thumbWidth, thumbHeight, wgProcess)
	}

	go readBufferAsync(path, thumbIdMin, thumbIdMax, thumbBytes, jobs, queued)

	go func() {
		wgProcess.Wait()
		close(results)
		close(processed)
	}()

	for w := 1; w <= 20; w++ {
		wgWrite.Add(1)
		go saveImageToPngAsync(results, written, name, wgWrite)
	}

	go func() {
		wgWrite.Wait()
		close(written)
	}()

	wgPrint.Wait()

	// for i := thumbIdMin; i < thumbIdMax; i += 1 {
	// // Extract all available image information as we know it
	// outImageAll := image.NewGray(image.Rect(0, 0, thumbWidth*2, thumbHeight))
	// for y := 0; y < thumbHeight; y = y + 1 {
	// 	for x := 0; x < thumbWidth*2; x = x + 1 {
	// 		luminance := thumbReadBuf[y*thumbWidth*2+x]
	// 		outImageAll.Set(x, y, color.Gray{luminance})
	// 	}
	// }

	// err = saveImageToPng("thumb_all.png", outImageAll)
	// check(err)

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

	// err = saveImageToPng("thumb_luminance.png", outImageLuminance)
	// check(err)

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

	// err = saveImageToPng("thumb_chroma_blue.png", outImageChromaBlue)
	// check(err)
	// err = saveImageToPng("thumb_chroma_red.png", outImageChromaRed)
	// check(err)

	// // Extract luminance of rest and produce grayscale bitmap
	// outImageRestLuminance := image.NewGray(image.Rect(0, 0, thumbWidth, thumbHeight/4))
	// for y := 360; y < thumbHeight; y = y + 1 {
	// 	for x := 0; x < thumbWidth; x = x + 1 {
	// 		luminance := thumbReadBuf[y*thumbWidth*2+2*x+1]
	// 		outImageRestLuminance.Set(x, y-360, color.Gray{luminance})
	// 	}
	// }

	// err = saveImageToPng("thumb_rest_luminance.png", outImageRestLuminance)
	// check(err)

	// // Extract only the last fourth of image data and produce grayscale bitmap
	// outImageRestChromaBlue := image.NewGray(image.Rect(0, 0, thumbWidth/2, thumbHeight/4))
	// outImageRestChromaRed := image.NewGray(image.Rect(0, 0, thumbWidth/2, thumbHeight/4))
	// for y := 360; y < thumbHeight; y = y + 1 {
	// 	for x := 0; x < thumbWidth/2; x = x + 1 {
	// 		chromaBlue := thumbReadBuf[y*thumbWidth*2+4*x]
	// 		chromaRed := thumbReadBuf[y*thumbWidth*2+4*x+2]
	// 		outImageRestChromaBlue.Set(x, y-360, color.Gray{chromaBlue})
	// 		outImageRestChromaRed.Set(x, y-360, color.Gray{chromaRed})
	// 	}
	// }

	// err = saveImageToPng("thumb_rest_chroma_blue.png", outImageRestChromaBlue)
	// check(err)
	// err = saveImageToPng("thumb_rest_chroma_red.png", outImageRestChromaRed)
	// check(err)

	// // Re-create image from rest
	// outImageRest := image.NewRGBA(image.Rect(0, 0, thumbWidth, thumbHeight/4))
	// for y := 360; y < thumbHeight; y = y + 1 {
	// 	for x := 0; x < thumbWidth; x = x + 1 {
	// 		luminance, chromaBlue, chromaRed := uint8(0), uint8(0), uint8(0)
	// 		if x%2 == 0 {
	// 			xhalf := x / 2
	// 			luminance = thumbReadBuf[y*thumbWidth*2+4*xhalf+1]
	// 			chromaBlue = thumbReadBuf[y*thumbWidth*2+4*xhalf]
	// 			chromaRed = thumbReadBuf[y*thumbWidth*2+4*xhalf+2]
	// 		} else {
	// 			xm1half := (x - 1) / 2
	// 			luminance = thumbReadBuf[y*thumbWidth*2+4*xm1half+3]
	// 			chromaBlue = thumbReadBuf[y*thumbWidth*2+4*xm1half]
	// 			chromaRed = thumbReadBuf[y*thumbWidth*2+4*xm1half+2]
	// 		}

	// 		r, g, b := ycbcr2rgb(ycbcr{luminance, chromaBlue, chromaRed})
	// 		outImageRest.Set(x, y-360, color.RGBA{r, g, b, 255})
	// 	}
	// }

	// err = saveImageToPng("thumb_rest.png", outImageRest)
	// check(err)

	// // Create array of y, cb, cr components
	// outImage := image.NewRGBA(image.Rect(0, 0, thumbWidth, thumbHeight))
	// for y := 0; y < thumbHeight; y = y + 1 {
	// 	for x := 0; x < thumbWidth; x = x + 1 {
	// 		luminance := thumbReadBuf[y*thumbWidth+x]
	// 		chromaBlue := thumbReadBuf[thumbPixels+(y/2)*thumbWidth/2+(x/2)]
	// 		chromaRed := thumbReadBuf[thumbPixels*5/4+(y/2)*thumbWidth/2+(x/2)]

	// 		r, g, b := ycbcr2rgb(ycbcr{luminance, chromaBlue, chromaRed})
	// 		outImage.Set(x, y, color.RGBA{r, g, b, 255})
	// 	}
	// }

	// thumbPath := thumbName + "/" + thumbName + "_" + strconv.FormatInt(int64(i), 10) + ".png"
	// if err = saveImageToPng(thumbPath, outImage); err != nil {
	// 	fmt.Printf("Failed to save image %d\n", i)
	// 	fmt.Println(err.Error())
	// }

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

	// err = saveImageToPng(
	// 	thumbName + "/" + thumbName + "_" + strconv.FormatInt(int64(i), 10) + "_histogram.png",
	// 	outImageHistogram,
	// )
	// check(err)
	// }

	os.Exit(0)
}
