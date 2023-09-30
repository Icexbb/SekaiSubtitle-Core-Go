package process

import (
	"image"
	"math"

	"gocv.io/x/gocv"
)

type areaMaskInfo struct {
	size  [2]int
	ratio float64
	area  [4]int
}
type patternSizeInfo struct {
	size  [2]int
	ratio float64
	area  [4]int
}

func getAreaMaskSize(h, w int) areaMaskInfo {
	var areaMaskSize = [...]int{0, 0}
	if (float64(w) / float64(h)) > (16.0 / 9.0) {
		areaMaskSize[1] = int(float64(h) / 1080.0 * 136.0)
		areaMaskSize[0] = int(float64(areaMaskSize[1]) * (886.0 / 136.0))
	} else {
		areaMaskSize[0] = int(float64(w) / 1920.0 * 886.0)
		areaMaskSize[1] = int(float64(areaMaskSize[0]) * (136.0 / 886.0))
	}

	var areaMaskArea = [...]int{
		(w - areaMaskSize[0]) / 2,
		(h - areaMaskSize[1]) / 2,
		(w + areaMaskSize[0]) / 2,
		(h + areaMaskSize[1]) / 2,
	}
	var ratio = float64(areaMaskSize[0]) / 886.0
	var result = areaMaskInfo{
		area:  areaMaskArea,
		ratio: ratio,
		size:  areaMaskSize,
	}
	return result
}
func getPatternSize(h, w int) patternSizeInfo {
	var patternSize = [2]int{0, 0}
	if (float64(w) / float64(h)) > (16.0 / 9.0) {
		patternSize[1] = int(float64(h) / 1080.0 * 317.0)
		patternSize[0] = int(float64(patternSize[1]) * (1612.0 / 317.0))
	} else {
		patternSize[0] = int(float64(w) / 1920.0 * 1612.0)
		patternSize[1] = int(float64(patternSize[0]) * (317.0 / 1612.0))
	}
	return patternSizeInfo{size: patternSize, ratio: float64(patternSize[0]) / 1612.0}
}
func getFrameData(h, w int, pointCenter image.Point) (areaMaskInfo, patternSizeInfo) {
	pSizeInfo := getPatternSize(h, w)
	aMaskInfo := getAreaMaskSize(h, w)

	startPoint := [2]int{
		int(float64(pointCenter.X) - 110.0*pSizeInfo.ratio),
		int(float64(pointCenter.Y) - 42.0*pSizeInfo.ratio),
	}
	patternArea := [4]int{
		startPoint[0], startPoint[1],
		startPoint[0] + pSizeInfo.size[0],
		startPoint[1] + pSizeInfo.size[1],
	}
	pSizeInfo.area = patternArea

	return aMaskInfo, pSizeInfo
}

func getBannerArea(h, w int) [4]int {
	mask := getAreaMaskSize(h, w)
	maskString := Str2IntArr(getAreaBannerMask(mask))
	var xls []int
	var yls []int

	for k, v := range maskString {
		if k%2 == 0 {
			xls = append(xls, v)
		} else {
			yls = append(yls, v)
		}
	}
	var res = [4]int{
		MinInt(yls), MaxInt(yls), MinInt(xls), MaxInt(xls),
	}
	return res
}

func checkDark(image gocv.Mat, color int) uint8 {
	minVal, _, _, _ := gocv.MinMaxLoc(image)
	if minVal < float32(color) {
		return 1
	} else {
		return 0
	}
}
func checkFrameContentStart(frame, menuSign gocv.Mat) bool {
	menuHeight := menuSign.Rows()
	frameWidth := frame.Cols()
	cutDown := 3 * menuHeight
	cutLeft := frameWidth - int(float64(frameWidth)*0.3)

	res := gocv.NewMat()
	empty := gocv.NewMat()
	cut := frame.Region(image.Rect(cutLeft, 0, frameWidth, cutDown))
	gocv.MatchTemplate(cut, menuSign, &res, gocv.TmCcoeffNormed, empty)
	_, maxVal, _, _ := gocv.MinMaxLoc(res)
	_ = res.Close()
	_ = cut.Close()
	_ = empty.Close()
	return maxVal > 0.7
}

func checkFrameDialogPointerPosition(frame gocv.Mat, pointer gocv.Mat, lastPointCenter image.Point) image.Point {
	h := frame.Rows()
	w := frame.Cols()
	pointerSize := pointer.Cols()
	var cutUp, cutDown, cutLeft, cutRight int

	if lastPointCenter.Eq(image.Point{}) {
		cutUp = int(float64(h) * 0.6)
		cutDown = int(float64(h) * 0.85)
		cutLeft = 0
		cutRight = int(float64(w) * 0.3)
	} else {
		left := float64(lastPointCenter.X)
		top := float64(lastPointCenter.Y)
		border := float64(pointerSize) * 0.9
		cutUp = int(top - border)
		cutDown = int(top + border)
		cutLeft = int(left - border)
		cutRight = int(left + border)
	}
	cut := frame.Region(image.Rect(cutLeft, cutUp, cutRight, cutDown))
	res := gocv.NewMat()
	empty := gocv.NewMat()
	gocv.MatchTemplate(cut, pointer, &res, gocv.TmCcoeffNormed, empty)
	_, maxVal, _, maxLoc := gocv.MinMaxLoc(res)

	_ = cut.Close()
	_ = res.Close()
	_ = empty.Close()

	if maxVal < 0.8 {
		return image.Point{X: 0, Y: 0}
	} else {
		return image.Point{
			X: cutLeft + maxLoc.X + int(float64(pointerSize)/2),
			Y: cutUp + maxLoc.Y + int(float64(pointerSize)/2),
		}
	}
}
func checkFrameDialogStatus(frame, pointer gocv.Mat, pointCenter image.Point) uint8 {
	var result uint8
	var color = 128
	if pointCenter.Eq(image.Point{}) {
		return 0
	}
	pointerSize := pointer.Cols()

	left := pointCenter.X - pointerSize/2
	top := pointCenter.Y - pointerSize/2
	right := pointCenter.X + pointerSize/2
	bottom := pointCenter.Y + pointerSize/2

	top += int(1.9 * float64(pointerSize))
	bottom += int(1.9 * float64(pointerSize))

	cut := frame.Region(image.Rect(left, top, right, bottom))
	result += checkDark(cut, 128)
	_ = cut.Close()
	if result != 0 {
		left += int(1.15 * float64(pointerSize))
		right += int(1.15 * float64(pointerSize))
		cut2 := frame.Region(image.Rect(left, top, right, bottom))
		result += checkDark(cut2, color)
		_ = cut2.Close()
	}
	return result
}
func checkFrameAreaMarkerPosition(frame, marker gocv.Mat) image.Point {
	frameHeight := frame.Rows()
	frameWidth := frame.Cols()

	res := gocv.NewMat()
	empty := gocv.NewMat()
	cut := frame.Region(image.Rect(0, 0, frameWidth/3, frameHeight/8))
	gocv.MatchTemplate(cut, marker, &res, gocv.TmCcoeffNormed, empty)
	_, maxVal, _, maxLoc := gocv.MinMaxLoc(res)

	_ = res.Close()
	_ = cut.Close()
	_ = empty.Close()

	if maxVal < 0.8 {
		return image.Point{}
	} else {
		return maxLoc
	}
}
func checkFrameAreaBannerEdge(frame, templateCanny, templateReverse gocv.Mat, area [4]int) bool {
	height := int(math.Abs(float64(area[1] - area[0])))
	var cutArea = image.Rect(
		int(float64(area[2])-0.1*float64(height)), int(float64(area[0])-0.1*float64(height)),
		int(float64(area[3])+0.1*float64(height)), int(float64(area[1])+0.1*float64(height)),
	)
	mat := frame.Region(cutArea)
	gocv.Resize(mat, &mat, image.Point{X: height, Y: height}, 0, 0, gocv.InterpolationLanczos4)
	sp, ep := int(float64(height)*0.2), int(float64(height)*0.8)
	for x := sp; x < ep; x++ {
		for y := sp; y < ep; y++ {
			mat.SetUCharAt(x, y, 150)
		}
	}

	canny := gocv.NewMat()
	result := gocv.NewMat()
	gocv.Canny(mat, &canny, 50, 150)
	gocv.MatchTemplate(canny, templateCanny, &result, gocv.TmCcoeffNormed, templateReverse)
	res := result.GetFloatAt(0, 0) > 0.4

	_ = mat.Close()
	_ = canny.Close()
	_ = result.Close()

	return res
}
