// for i := idMin; i < idMax; i += 1 {
// // Extract all available image information as we know it
// outImageAll := image.NewGray(image.Rect(0, 0, width*2, height))
// for y := 0; y < height; y = y + 1 {
// 	for x := 0; x < width*2; x = x + 1 {
// 		luminance := thumbReadBuf[y*width*2+x]
// 		outImageAll.Set(x, y, color.Gray{luminance})
// 	}
// }

// err = saveImageToPng("thumb_all.png", outImageAll)
// if err != nil {
// 	panic(err)
// }

// // Extract luminance only and produce grayscale bitmap
// outImageLuminance := image.NewGray(image.Rect(0, 0, width, height))
// maxLuminance := byte(0)
// for y := 0; y < height; y = y + 1 {
// 	for x := 0; x < width; x = x + 1 {
// 		luminance := thumbReadBuf[y*width+x]
// 		if maxLuminance < luminance {
// 			maxLuminance = luminance
// 		}
// 		outImageLuminance.Set(x, y, color.Gray{luminance})
// 	}
// }
// fmt.Printf("Maximum observed luminance: %d\n", maxLuminance)

// err = saveImageToPng("thumb_luminance.png", outImageLuminance)
// if err != nil {
//	panic(err)
// }

// // Extract chroma only and produce grayscale bitmap
// outImageChromaBlue := image.NewGray(image.Rect(0, 0, width/2, height/2))
// outImageChromaRed := image.NewGray(image.Rect(0, 0, width/2, height/2))
// maxChromaBlue := byte(0)
// maxChromaRed := byte(0)
// for y := 0; y < height/2; y = y + 1 {
// 	for x := 0; x < width/2; x = x + 1 {
// 		chromaBlue := thumbReadBuf[thumbPixels+y*width/2+x]
// 		if maxChromaBlue < chromaBlue {
// 			maxChromaBlue = chromaBlue
// 		}
// 		outImageChromaBlue.Set(x, y, color.Gray{chromaBlue})

// 		chromaRed := thumbReadBuf[thumbPixels*5/4+y*width/2+x]
// 		if maxChromaRed < chromaRed {
// 			maxChromaRed = chromaRed
// 		}
// 		outImageChromaRed.Set(x, y, color.Gray{chromaRed})
// 	}
// }
// fmt.Printf("Maximum observed chroma blue: %d\n", maxChromaBlue)
// fmt.Printf("Maximum observed chroma red: %d\n", maxChromaRed)

// err = saveImageToPng("thumb_chroma_blue.png", outImageChromaBlue)
// if err != nil {
//	panic(err)
// }
// err = saveImageToPng("thumb_chroma_red.png", outImageChromaRed)
// if err != nil {
//	panic(err)
// }

// // Extract luminance of rest and produce grayscale bitmap
// outImageRestLuminance := image.NewGray(image.Rect(0, 0, width, height/4))
// for y := 360; y < height; y = y + 1 {
// 	for x := 0; x < width; x = x + 1 {
// 		luminance := thumbReadBuf[y*width*2+2*x+1]
// 		outImageRestLuminance.Set(x, y-360, color.Gray{luminance})
// 	}
// }

// err = saveImageToPng("thumb_rest_luminance.png", outImageRestLuminance)
// if err != nil {
//	panic(err)
// }

// // Extract only the last fourth of image data and produce grayscale bitmap
// outImageRestChromaBlue := image.NewGray(image.Rect(0, 0, width/2, height/4))
// outImageRestChromaRed := image.NewGray(image.Rect(0, 0, width/2, height/4))
// for y := 360; y < height; y = y + 1 {
// 	for x := 0; x < width/2; x = x + 1 {
// 		chromaBlue := thumbReadBuf[y*width*2+4*x]
// 		chromaRed := thumbReadBuf[y*width*2+4*x+2]
// 		outImageRestChromaBlue.Set(x, y-360, color.Gray{chromaBlue})
// 		outImageRestChromaRed.Set(x, y-360, color.Gray{chromaRed})
// 	}
// }

// err = saveImageToPng("thumb_rest_chroma_blue.png", outImageRestChromaBlue)
// if err != nil {
//	panic(err)
// }
// err = saveImageToPng("thumb_rest_chroma_red.png", outImageRestChromaRed)
// if err != nil {
//	panic(err)
// }

// // Re-create image from rest
// outImageRest := image.NewRGBA(image.Rect(0, 0, width, height/4))
// for y := 360; y < height; y = y + 1 {
// 	for x := 0; x < width; x = x + 1 {
// 		luminance, chromaBlue, chromaRed := uint8(0), uint8(0), uint8(0)
// 		if x%2 == 0 {
// 			xhalf := x / 2
// 			luminance = thumbReadBuf[y*width*2+4*xhalf+1]
// 			chromaBlue = thumbReadBuf[y*width*2+4*xhalf]
// 			chromaRed = thumbReadBuf[y*width*2+4*xhalf+2]
// 		} else {
// 			xm1half := (x - 1) / 2
// 			luminance = thumbReadBuf[y*width*2+4*xm1half+3]
// 			chromaBlue = thumbReadBuf[y*width*2+4*xm1half]
// 			chromaRed = thumbReadBuf[y*width*2+4*xm1half+2]
// 		}

// 		r, g, b := ycbcr2rgb(ycbcr{luminance, chromaBlue, chromaRed})
// 		outImageRest.Set(x, y-360, color.RGBA{r, g, b, 255})
// 	}
// }

// err = saveImageToPng("thumb_rest.png", outImageRest)
// if err != nil {
//	panic(err)
// }

// // Create array of y, cb, cr components
// outImage := image.NewRGBA(image.Rect(0, 0, width, height))
// for y := 0; y < height; y = y + 1 {
// 	for x := 0; x < width; x = x + 1 {
// 		luminance := thumbReadBuf[y*width+x]
// 		chromaBlue := thumbReadBuf[thumbPixels+(y/2)*width/2+(x/2)]
// 		chromaRed := thumbReadBuf[thumbPixels*5/4+(y/2)*width/2+(x/2)]

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
// for y := 0; y < height; y = y + 1 {
// 	for x := 0; x < width; x = x + 1 {
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
// if err != nil {
//	panic(err)
// }