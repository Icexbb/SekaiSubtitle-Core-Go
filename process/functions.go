package process

import (
	"encoding/base64"
	"fmt"
	"gocv.io/x/gocv"
	"image"
	"math"
)

func scalingRatio(h, w int) (ratio float64) {
	var size float64
	if float64(w)/float64(h) > (16.0 / 9.0) {
		size = ((float64(h) / 1080) * 136) * (886 / 136)
	} else {
		size = (float64(w) / 1920) * 886
	}
	ratio = size / 886
	return
}
func b64ToCvMat(data string) gocv.Mat {
	var bytes []byte
	var mat gocv.Mat
	bytes, _ = base64.StdEncoding.DecodeString(data)
	mat, _ = gocv.IMDecode(bytes, gocv.IMReadAnyColor)
	gocv.CvtColor(mat, &mat, gocv.ColorBGRToGray)
	return mat
}
func getResizedB64Img(b64Img string, h, w int) gocv.Mat {
	var ratio float64
	var img gocv.Mat

	ratio = scalingRatio(h, w)
	img = b64ToCvMat(b64Img)
	gocv.Resize(img, &img, image.Point{X: 0, Y: 0}, ratio, ratio, gocv.InterpolationNearestNeighbor)

	return img
}

func getResizedDialogPointer(h, w int) gocv.Mat {
	return getResizedB64Img(B64Point, h, w)
}
func getResizedInterfaceMenu(h, w int) gocv.Mat {
	return getResizedB64Img(B64Menu, h, w)
}
func getResizedAreaTag(h, w int) gocv.Mat {
	return getResizedB64Img(B64Place, h, w)
}
func getResizedAreaEdge(h, w int) gocv.Mat {
	var resized gocv.Mat
	bannerMaskArea := getBannerArea(h, w)
	bannerMaskHeight := math.Abs(float64(bannerMaskArea[1] - bannerMaskArea[0]))
	bannerPatternSize := int(math.Abs(float64(bannerMaskArea[3])-float64(bannerMaskArea[2])+0.2*bannerMaskHeight) * 0.3)
	resized = b64ToCvMat(B64Banner)
	gocv.Resize(resized, &resized, image.Point{X: bannerPatternSize, Y: bannerPatternSize}, 0, 0, gocv.InterpolationNearestNeighbor)
	return resized
}

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
		MinInt(xls), MaxInt(xls), MinInt(yls), MaxInt(yls),
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
	var result bool
	var res = gocv.NewMat()
	menuHeight := menuSign.Rows()
	frameWidth := frame.Cols()
	cutDown := 3 * menuHeight
	cutLeft := frameWidth - int(float64(frameWidth)*0.3)
	cut := frame.Region(image.Rect(cutLeft, 0, frameWidth, cutDown))
	gocv.MatchTemplate(cut, menuSign, &res, gocv.TmCcoeffNormed, gocv.NewMat())
	_, maxVal, _, _ := gocv.MinMaxLoc(res)
	result = maxVal > 0.7
	return result
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
	var res = gocv.NewMat()
	gocv.MatchTemplate(cut, pointer, &res, gocv.TmCcoeffNormed, gocv.NewMat())
	_, maxVal, _, maxLoc := gocv.MinMaxLoc(res)

	if maxVal < 0.8 {
		return image.Point{X: 0, Y: 0}
	} else {
		x := cutLeft + maxLoc.X + int(float64(pointerSize)/2)
		y := cutUp + maxLoc.Y + int(float64(pointerSize)/2)
		return image.Point{X: x, Y: y}
	}
}
func checkFrameDialogStatus(frame, pointer gocv.Mat, pointCenter image.Point) uint8 {
	var result uint8
	if pointCenter.Eq(image.Point{}) {
		return 0
	}
	pointerSize := pointer.Cols()
	var color = 128
	var cut gocv.Mat

	left := pointCenter.X - pointerSize/2
	top := pointCenter.Y - pointerSize/2
	right := pointCenter.X + pointerSize/2
	bottom := pointCenter.Y + pointerSize/2

	top += int(1.9 * float64(pointerSize))
	bottom += int(1.9 * float64(pointerSize))

	cut = frame.Region(image.Rect(left, top, right, bottom))
	result += checkDark(cut, color)
	if result == 0 {
		return result
	} else {
		left += int(1.15 * float64(pointerSize))
		right += int(1.15 * float64(pointerSize))
		cut = frame.Region(image.Rect(left, top, right, bottom))
		result += checkDark(cut, color)
		return result
	}
}
func checkFrameAreaTagPosition(frame, tag gocv.Mat) image.Point {
	var res = gocv.NewMat()
	frameHeight := frame.Rows()
	frameWidth := frame.Cols()
	cut := frame.Region(image.Rect(0, 0, frameWidth/3, frameHeight/8))
	gocv.MatchTemplate(cut, tag, &res, gocv.TmCcoeffNormed, gocv.NewMat())
	_, maxVal, _, maxLoc := gocv.MinMaxLoc(res)
	if maxVal < 0.8 {
		return image.Point{}
	} else {
		return maxLoc
	}

}
func checkFrameAreaBannerEdge(frame, template gocv.Mat, area [4]int) bool {
	var result bool
	var cut gocv.Mat
	var c uint8 = 142
	height := int(math.Abs(float64(area[1] - area[0])))
	cut = frame.Region(image.Rect(
		int(float64(area[0])-0.1*float64(height)),
		int(float64(area[2])-0.1*float64(height)),
		int(float64(area[1])+0.1*float64(height)),
		int(float64(area[3])+0.1*float64(height))))
	sz := int(float64(cut.Cols()) * 0.3)
	gocv.Resize(cut, &cut, image.Point{X: sz, Y: sz}, 0, 0, gocv.InterpolationNearestNeighbor)
	x1, y1 := int(float64(sz)*0.3), int(float64(sz)*0.2)
	x2, y2 := int(float64(sz)*0.7), int(float64(sz)*0.8)

	for x := x1; x < x2; x++ {
		for y := y1; y < y2; y++ {
			cut.SetUCharAt(x, y, c)
		}
	}
	var tempCanny = gocv.NewMat()
	gocv.Canny(template, &tempCanny, 100, 200)
	for x := 0; x < tempCanny.Cols(); x++ {
		for y := 0; y < tempCanny.Rows(); y++ {
			tempCanny.SetUCharAt(x, y, uint8(255)-tempCanny.GetUCharAt(x, y))
		}
	}
	if checkFrameAreaBannerEdgeProcess(cut, tempCanny, [2]float32{50, 150}) {
		result = true
	}
	if !result {
		if checkFrameAreaBannerEdgeProcess(cut, tempCanny, [2]float32{25, 25}) {
			result = true
		}
	}
	if !result {
		if checkFrameAreaBannerEdgeProcess(cut, tempCanny, [2]float32{50, 50}) {
			result = true
		}
	}
	return result
}
func checkFrameAreaBannerEdgeProcess(cut, tempCanny gocv.Mat, thresholds [2]float32) bool {
	var result []float64
	var frameEdge = gocv.NewMat()
	gocv.Canny(cut, &frameEdge, thresholds[0], thresholds[1])
	result = append(result, math.Max(1-(frameEdge.Sum().Val1/tempCanny.Sum().Val1)*10.0, 0))
	var temp float64
	for x := 0; x < tempCanny.Cols(); x++ {
		for y := 0; y < tempCanny.Rows(); y++ {
			if tempCanny.GetUCharAt(x, y) == 255 {
				temp += float64(frameEdge.GetUCharAt(x, y))
			}
		}
	}
	result = append(result, temp/tempCanny.Sum().Val1/0.3)
	var tempM = gocv.NewMat()
	gocv.MatchTemplate(frameEdge, tempCanny, &tempM, gocv.TmCcoeffNormed, gocv.NewMat())
	result = append(result, tempM.GetDoubleAt(0, 0))
	return Prod(result) > 0.35
}

func getDialogMask(info patternSizeInfo, move [2]int) string {
	originMask := "m 232 785 " +
		"b 232 785 232 785 232 785 " +
		"b 137 836 137 964 232 1015 " +
		"b 244 1021 259 1025 272 1027 " +
		"l 1646 1027 " +
		"l 1687 1015 " +
		"b 1783 964 1783 836 1687 785 " +
		"l 1645 772 " +
		"l 889 772 " +
		"b 889 786 874 797 860 797 " +
		"l 254 798 " +
		"b 249 798 232 791 232 785"
	mask := AssDraw{proto: originMask}
	mask = mask.move(-154, -717).scale(info.ratio).move(info.area[0], info.area[1]).move(move[0], move[1])

	var result = "{\\p1\\pos(0,0)\\c&HFFFFFF&}"
	result += mask.proto
	return result
}
func getDialogCharacterMask(h, w int, pointCenter image.Point, pointSize int) string {
	var originMask = "m 385 1125 l 385 1220 l 1137 1220 b 1146 1220 1183 1211 1183 1170 b 1183 1134 1146 1125 1137 1125"
	var result = fmt.Sprintf("{\\p1\\c&H886667&\\pos(%d,%d)}", pointCenter.X, pointCenter.Y)
	mask := AssDraw{proto: originMask}
	mask = mask.move(-385, -1172).scale(getPatternSize(h, w).ratio / getPatternSize(1600, 2560).ratio)
	mask = mask.move(int(pointSize/2), 0)
	result += mask.proto
	return result
}
func getAreaBannerMask(info areaMaskInfo) string {
	var originMask = "m 566 472 l 517 608 l 1354 608 l 1403 472"
	var result = "{\\an7\\p1\\c&HB68B89&\\pos(0,0)\\fad(100,100)}"
	mask := AssDraw{proto: originMask}
	mask = mask.move(-517, -472).scale(info.ratio).move(info.area[0], info.area[1])
	result += mask.proto
	return result
}
func getAreaTagMask(h, w int, move image.Point) (string, [2]int) {
	var originMask = "m 88 64 b 70 65 54 85 54 102 b 54 119 70 138 88 138 l 749 138 b 770 138 787 118 787 102 " +
		"b 787 85 770 65 749 64"
	var result = "{\\an7\\p1\\c&H674445&\\pos(0,0)}"
	mask := AssDraw{proto: originMask}
	mask = mask.move(-420, -101).scale(1/scalingRatio(1600, 2560)).move(-275, 0)
	mask = mask.scale(scalingRatio(h, w)*0.99).move(move.X, move.Y)
	result += mask.proto
	size := [2]int{
		int(math.Abs(float64(Str2IntArr(mask.proto)[1] * 2))),
		int(math.Abs(float64(Str2IntArr(mask.proto)[6]))),
	}
	return result, size
}

func getDividerSubtitleEvent(msg string, slash int) (divider SubtitleEventItem) {
	var d = ""
	for i := 0; i < slash; i++ {
		d += "-"
	}
	divider.Type = "Comment"
	divider.Layer = 1
	divider.Start = "00:00:00.00"
	divider.End = "00:00:00.00"
	divider.Style = "screen"
	divider.Text = d + msg + d
	return
}

func GetSubtitleArraySurrounded(arr []SubtitleEventItem, msg string, slash int) (array []SubtitleEventItem) {
	a := getDividerSubtitleEvent(Strip(msg)+" Start", slash)
	b := getDividerSubtitleEvent(Strip(msg)+" End", slash)
	array = append([]SubtitleEventItem{a}, arr...)
	array = append(array, b)
	return
}
