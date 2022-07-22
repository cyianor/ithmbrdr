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

func printStatusAsync(queued, processed, written <-chan int, wg *sync.WaitGroup) {
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

func main() {
	var err error

	if !(len(os.Args) == 3 || len(os.Args) == 5) || (len(os.Args) == 5 && os.Args[1] != "-c") {
		fmt.Println("Usage: ithmbrdr [-c channelSize] id filename")
		os.Exit(1)
	}

	var idString string
	var pathString string

	channelSize := 100
	if len(os.Args) == 5 {
		channelSize, err = strconv.Atoi(os.Args[2])
		if err != nil {
			panic(err)
		}
		if channelSize <= 0 {
			fmt.Println("channel size needs to be at least 1")
			os.Exit(1)
		}
		idString = os.Args[3]
		pathString = os.Args[4]
	} else {
		idString = os.Args[1]
		pathString = os.Args[2]
	}
	fmt.Printf("Using channel size %d\n", channelSize)

	path, err := filepath.Abs(pathString)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Reading from %s\n", path)
	name := strings.Split(filepath.Base(path), ".")[0]
	fmt.Printf("File is %s\n", name)

	// Create a directory of this name if it does not exist
	if _, err = os.Stat(name); errors.Is(err, os.ErrNotExist) {
		if err = os.Mkdir(name, os.ModePerm); err != nil {
			panic(err)
		}
	}

	idMin := 0
	idMax := int((^uint(0)) >> 1)

	if idString != "*" {
		id, err := strconv.Atoi(idString)
		if err != nil {
			// Input might be a range
			ids := strings.Split(idString, "-")
			if len(ids) != 2 {
				fmt.Println("Ranges need to be of the form d1-d2, e.g. 2-10")
				os.Exit(1)
			} else {
				idMin, err = strconv.Atoi(ids[0])
				if err != nil {
					panic(err)
				}
				idMax, err = strconv.Atoi(ids[1])
				if err != nil {
					panic(err)
				}
				idMax += 1
			}
		} else {
			idMin = id
			idMax = id + 1
		}
	}

	width := 720
	height := 480
	bytes := width * height * 2

	// Setup channels for job queuing and async processing/writing
	jobs := make(chan indexedBuffer, channelSize)
	results := make(chan indexedImage, channelSize)

	queued := make(chan int, channelSize)
	processed := make(chan int, channelSize)
	written := make(chan int, channelSize)

	wgProcess := new(sync.WaitGroup)
	wgWrite := new(sync.WaitGroup)
	wgPrint := new(sync.WaitGroup)

	wgPrint.Add(1)
	go printStatusAsync(queued, processed, written, wgPrint)

	// Process workers
	numProcessWorkers := 5
	if channelSize < numProcessWorkers {
		numProcessWorkers = channelSize
	}

	for w := 1; w <= numProcessWorkers; w++ {
		wgProcess.Add(1)
		go convertYcbcr2RgbaAsync(jobs, results, processed, width, height, wgProcess)
	}

	go readBufferAsync(path, idMin, idMax, bytes, jobs, queued)

	go func() {
		wgProcess.Wait()
		close(results)
		close(processed)
	}()

	// Write workers
	numWriteWorkers := 5
	if channelSize < numWriteWorkers {
		numWriteWorkers = channelSize
	}

	for w := 1; w <= numWriteWorkers; w++ {
		wgWrite.Add(1)
		go saveImageToPngAsync(results, written, name, wgWrite)
	}

	go func() {
		wgWrite.Wait()
		close(written)
	}()

	wgPrint.Wait()

	os.Exit(0)
}
