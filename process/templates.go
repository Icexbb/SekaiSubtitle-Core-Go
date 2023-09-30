package process

import (
	"encoding/base64"
	"fmt"
	"image"
	"math"

	"gocv.io/x/gocv"
)

func getResizedB64Img(b64Img string, h, w int) gocv.Mat {
	ratio := scalingRatio(h, w)
	img := b64ToCvMat(b64Img)
	gocv.Resize(img, &img, image.Point{X: 0, Y: 0}, ratio, ratio, gocv.InterpolationNearestNeighbor)
	return img
}

func getResizedDialogPointer(h, w int) gocv.Mat {
	return getResizedB64Img(B64Point, h, w)
}
func getResizedInterfaceMenu(h, w int) gocv.Mat {
	return getResizedB64Img(B64Menu, h, w)
}
func getResizedAreaMarkerTemplate(h, w int) gocv.Mat {
	return getResizedB64Img(B64Place, h, w)
}
func getResizedAreaEdge(h, w int) gocv.Mat {
	bannerMaskArea := getBannerArea(h, w)
	bannerMaskHeight := math.Abs(float64(bannerMaskArea[1] - bannerMaskArea[0]))
	bannerPatternSize := int(math.Abs(float64(bannerMaskArea[3])-float64(bannerMaskArea[2])+0.2*bannerMaskHeight) * 0.3)
	resized := b64ToCvMat(B64Banner)
	gocv.Resize(resized, &resized, image.Point{X: bannerPatternSize, Y: bannerPatternSize}, 0, 0, gocv.InterpolationNearestNeighbor)
	return resized
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
	mask = mask.move(pointSize/2, 0)
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
func getAreaMarkerMask(h, w int) (string, [2]int) {
	var originMask = "m 88 64 b 70 65 54 85 54 102 b 54 119 70 138 88 138 l 749 138 b 770 138 787 118 787 102 " +
		"b 787 85 770 65 749 64"
	var result = "{\\an7\\p1\\c&H674445&}"
	mask := AssDraw{proto: originMask}
	mask = mask.move(-420, -101).scale(1/scalingRatio(1600, 2560)).move(-275, 0)
	mask = mask.scale(scalingRatio(h, w) * 0.99) // .move(move.X, move.Y)
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
