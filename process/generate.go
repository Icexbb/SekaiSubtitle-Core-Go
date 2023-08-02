package process

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"math"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"gocv.io/x/gocv"
)

type StaffItem struct {
	Recorder       string  `json:"recorder"`
	Translator     string  `json:"translator"`
	TranslateProof string  `json:"translate_proof"`
	SubtitleMaker  string  `json:"subtitle_maker"`
	SubtitleProof  string  `json:"subtitle_proof"`
	Compositor     string  `json:"compositor"`
	Duration       int     `json:"duration"`
	Position       int     `json:"position"`
	Suffix         string  `json:"suffix"`
	Prefix         string  `json:"prefix"`
	Fade           [2]int  `json:"fade"`
	FontSize       float64 `json:"fontsize"`
	FontSizeType   string  `json:"fontsize_type"`
	MarginLR       int     `json:"margin_lr"`
	MarginV        int     `json:"margin_v"`
}

func makeStaffBody(item StaffItem) string {
	type StaffSort struct {
		Id   string
		Jobs []string
	}
	var result = ""
	if len(item.Prefix) > 0 {
		result += item.Prefix + "\n"
	}
	var staffs []StaffSort
	if len(item.Recorder) > 0 {
		index := 0
		for i, v := range staffs {
			if v.Id == item.Recorder {
				index = i
			}
		}
		if index == 0 {
			staffs[index] = StaffSort{Id: item.Recorder, Jobs: []string{"录制"}}
		} else {
			staffs[index].Jobs = append(staffs[index].Jobs, "录制")
		}
	}
	if len(item.Translator) > 0 {
		index := 0
		for i, v := range staffs {
			if v.Id == item.Translator {
				index = i
			}
		}
		if index == 0 {
			staffs[index] = StaffSort{Id: item.Translator, Jobs: []string{"翻译"}}
		} else {
			staffs[index].Jobs = append(staffs[index].Jobs, "翻译")
		}
	}
	if len(item.TranslateProof) > 0 {
		index := 0
		for i, v := range staffs {
			if v.Id == item.TranslateProof {
				index = i
			}
		}
		if index == 0 {
			staffs[index] = StaffSort{Id: item.TranslateProof, Jobs: []string{"校对"}}
		} else {
			staffs[index].Jobs = append(staffs[index].Jobs, "校对")
		}
	}
	if len(item.SubtitleMaker) > 0 {
		index := 0
		for i, v := range staffs {
			if v.Id == item.SubtitleMaker {
				index = i
			}
		}
		if index == 0 {
			staffs[index] = StaffSort{Id: item.SubtitleMaker, Jobs: []string{"时轴"}}
		} else {
			staffs[index].Jobs = append(staffs[index].Jobs, "时轴")
		}
	}
	if len(item.SubtitleProof) > 0 {
		index := 0
		for i, v := range staffs {
			if v.Id == item.SubtitleProof {
				index = i
			}
		}
		if index == 0 {
			staffs[index] = StaffSort{Id: item.SubtitleProof, Jobs: []string{"轴校"}}
		} else {
			staffs[index].Jobs = append(staffs[index].Jobs, "轴校")
		}
	}
	if len(item.Compositor) > 0 {
		index := 0
		for i, v := range staffs {
			if v.Id == item.Compositor {
				index = i
			}
		}
		if index == 0 {
			staffs[index] = StaffSort{Id: item.Compositor, Jobs: []string{"压制"}}
		} else {
			staffs[index].Jobs = append(staffs[index].Jobs, "压制")
		}
	}

	var StaffStrings []string
	for _, staff := range staffs {
		StaffStrings = append(StaffStrings, staff.Id+"："+strings.Join(staff.Jobs, "&"))
	}

	var StaffString = Strip(strings.Join(StaffStrings, "\n"))
	if len(StaffString) > 0 {
		result += StaffString + "\n"
	}
	if len(item.Suffix) > 0 {
		result += item.Suffix + "\n"
	}
	result = Strip(result)
	result = strings.ReplaceAll(result, "\n", "\\N")

	fadeString := fmt.Sprintf("{\\fad(%d,%d)}", item.Fade[0], item.Fade[1])
	result = fadeString + result

	return result
}
func makeStaffEvent(item StaffItem, dialogFontSize int, fontName string) (SubtitleEventItem, SubtitleStyleItem) {
	var body = makeStaffBody(item)
	var styleName = fmt.Sprintf("Staff-%s", Md5Len3(body))
	var e = SubtitleEventItem{
		Type:    "Dialogue",
		Layer:   1,
		Start:   "0:00:00.00",
		End:     fmt.Sprintf("0:00:%02d:%02d", item.Duration/60, item.Duration%60),
		Style:   styleName,
		Name:    "staff",
		MarginL: 0,
		MarginR: 0,
		MarginV: 0,
		Effect:  "",
		Text:    body,
	}
	var fs float64
	if item.FontSizeType == "ratio" {
		fs = float64(dialogFontSize) * item.FontSize
	} else {
		fs = item.FontSize
	}
	s := StaffStyleFormat
	s.Name = styleName
	s.Fontsize = int(fs)
	s.FontName = fontName
	s.Alignment = item.Position
	s.MarginL = item.MarginLR
	s.MarginR = item.MarginLR
	s.MarginV = item.MarginV
	return e, s
}

// MATCH
type frameDialogProcessResult struct {
	status      uint8
	pointCenter image.Point
}

func matchFrameDialog(frame, pointer gocv.Mat, lastPointPosition image.Point) frameDialogProcessResult {
	center := checkFrameDialogPointerPosition(frame, pointer, lastPointPosition)
	status := checkFrameDialogStatus(frame, pointer, center)
	result := frameDialogProcessResult{
		status:      status,
		pointCenter: center,
	}
	return result
}
func matchFrameBanner(frame, bannerCanny, bannerReverse gocv.Mat, bannerMaskArea [4]int) bool {
	return checkFrameAreaBannerEdge(frame, bannerCanny, bannerReverse, bannerMaskArea)
}
func matchFrameMarker(frame, marker gocv.Mat) image.Point {
	return checkFrameAreaMarkerPosition(frame, marker)
}
func matchCheckStart(frame, menuSign gocv.Mat) bool {
	return checkFrameContentStart(frame, menuSign)
}

// DIALOG
type dialogFrame struct {
	FrameId     int
	PointCenter image.Point
}

func dialogBodyTyper(body string, charInterval [2]int) string {
	returnChar := []string{"\n", "\\n", "\\N"}
	bodyCopy := body
	for _, s := range returnChar {
		bodyCopy = strings.ReplaceAll(bodyCopy, s, "\n")
	}
	res := ""
	nextStart := 0
	fadeTime := charInterval[0]
	charTime := charInterval[1]
	for _, v := range bodyCopy {
		c := string(v)
		r := ""
		var start int
		if fadeTime > 0 && charTime > 0 {
			start = nextStart
			end := start + fadeTime
			if c == "\n" {
				start += 300
			}
			r += fmt.Sprintf("{\\alphaFF\\t(%d,%d,1,\\alpha0)}", start, end)
		}
		if c == "\n" {
			r += "\\N"
		} else {
			r += c
		}
		res += r
		nextStart = start + charTime
	}
	return res
}
func dialogBodyTyperCalculator(body string, frameCount int, frameTimeMs float64, charInterval [2]int) string {
	returnChar := []string{"\n", "\\n", "\\N"}
	bodyCopy := body
	for _, s := range returnChar {
		bodyCopy = strings.ReplaceAll(bodyCopy, s, "\n")
	}
	nowTime := int(frameTimeMs * float64(frameCount) * 1000.0)
	transAlphaString := "{\\alpha&HFF&}"
	isTransNow := false
	charTimeNow := 0
	fadeTime := charInterval[0]
	charTime := charInterval[1]
	res := ""
	for _, v := range bodyCopy {
		c := string(v)
		addTrans := ""
		charTimeNow += charTime
		if c == "\n" {
			charTimeNow += 300
		}
		if charTimeNow < nowTime && nowTime < charTimeNow+fadeTime {
			la := (nowTime - charTimeNow) / fadeTime * 255
			addTrans = fmt.Sprintf("{\\alpha%d}", la)
		} else if charTimeNow > nowTime {
			if !isTransNow {
				addTrans = transAlphaString
				isTransNow = true
			}
		}
		if c == "\n" {
			c = "\\N"
		}
		if fadeTime > 0 && charTime > 0 {
			res += addTrans + c
		} else {
			res += c
		}

	}
	return res
}
func dialogMakeStyle(config TaskConfig, pointCenter image.Point, pointSize int) []SubtitleStyleItem {
	styles := GetDialogStyle()
	for i, style := range styles {
		style.Fontsize = int(float64(pointSize) * (83.0 / 56.0))
		if len(config.Font) > 0 {
			style.FontName = config.Font
		}
		if !strings.HasPrefix(style.Name, "staff") && !strings.HasPrefix(style.Name, "screen") {
			if style.MarginL == 325 || style.MarginL == 385 {
				style.MarginL = pointCenter.X - pointSize/2
				style.MarginV = pointCenter.Y + int(float64(pointSize)*1.25)
			}
		}
		styles[i] = style
	}
	return styles
}
func dialogMakeEvent(
	dialogInfo StoryEvent, pointSize, h, w int, frameTime float64, lastDialogLastFrame dialogFrame, dialogFrames []dialogFrame,
	lastDialogLastEvent SubtitleEventItem, dialogIsMaskStart bool, config TaskConfig,
) ([]SubtitleEventItem, []SubtitleEventItem, []SubtitleEventItem, []SubtitleEventItem) {
	startFrame := dialogFrames[0]
	endFrame := dialogFrames[len(dialogFrames)-1]

	var displayName = dialogInfo.Content().Character
	var dialogBody = dialogInfo.Content().Body
	var styleName = "関連人物"
	if len(dialogBody) > 0 {
		s := CIDStyle[dialogInfo.CharacterId]
		if s != "" {
			styleName = s
		}
	}

	var framePoints []image.Point
	for _, frame := range dialogFrames {
		framePoints = append(framePoints, frame.PointCenter)
	}

	jitter := CheckMaxDistance(framePoints) > 3
	if !jitter {
		pointCenterConst := dialogFrames[0].PointCenter
		startTime := MsToString(int(frameTime * float64(startFrame.FrameId)))
		endTime := MsToString(int(frameTime * float64(endFrame.FrameId)))
		if (!dialogIsMaskStart) && (lastDialogLastFrame.FrameId != 0) {
			startTime = lastDialogLastEvent.End
		}
		bodyEvent := SubtitleEventItem{
			Type: "Dialogue", Layer: 2, Start: startTime, End: endTime, Style: styleName, Name: displayName,
			MarginL: 0, MarginR: 0, MarginV: 0, Effect: "", Text: dialogBodyTyper(dialogBody, config.TyperInterval),
		}
		maskEvent := bodyEvent
		_, patternInfo := getFrameData(h, w, pointCenterConst)
		maskString := getDialogMask(patternInfo, [2]int{})
		maskEvent.Text = maskString
		maskEvent.Style = "screen"
		maskEvent.Layer = 1
		if dialogIsMaskStart {
			maskEvent.Text = "{\\fad(100,0)}" + maskEvent.Text
			maskEvent.Start = MsToString(int(frameTime * float64(MaxInt([]int{0, startFrame.FrameId - 6}))))
		}
		// Character
		charaMaskString := getDialogCharacterMask(h, w, pointCenterConst, pointSize)
		charaMaskEvent := bodyEvent
		charaMaskEvent.Text = charaMaskString
		charaMaskEvent.Style = "screen"
		charaMaskEvent.Layer = 1
		if config.VideoOnly {
			charaMaskEvent.Type = "Comment"
		}

		charaBodyEvent := charaMaskEvent
		_, cmo := SplitArr(Str2IntArr(charaMaskString))
		charaOffset := int(float64(MinInt(cmo)) * 1.1)
		charaBodyEvent.Text = fmt.Sprintf("{\\pos(%d,%d)\\an4}",
			pointCenterConst.X+charaOffset, pointCenterConst.Y) + displayName
		charaBodyEvent.Style = "character"
		charaBodyEvent.Layer = 2
		return []SubtitleEventItem{charaMaskEvent}, []SubtitleEventItem{charaBodyEvent}, []SubtitleEventItem{maskEvent}, []SubtitleEventItem{bodyEvent}
	} else {
		var bodyEvents, maskEvents, charaBodyEvents, charaMaskEvents []SubtitleEventItem
		_, patternInfo := getFrameData(h, w, dialogFrames[0].PointCenter)
		for i, frame := range dialogFrames {
			move := fmt.Sprintf("{\\an7\\pos(%d,%d)}",
				frame.PointCenter.X-pointSize/2, int(float64(frame.PointCenter.Y)+1.25*float64(pointSize)))
			body := dialogBodyTyperCalculator(dialogBody, i, frameTime, config.TyperInterval)
			frameBody := move + body
			bodyEvent := SubtitleEventItem{
				Type: "Dialogue", Layer: 1, MarginL: 0, MarginR: 0, MarginV: 0, Effect: "",
				Start: MsToString(int(frameTime * float64(frame.FrameId))),
				End:   MsToString(int(frameTime * float64(frame.FrameId+1))),
				Style: styleName, Name: displayName, Text: frameBody,
			}
			if len(bodyEvents) > 0 {
				bodyEvent.Start = bodyEvents[len(bodyEvents)-1].End
			}
			// Mask
			maskMove := [2]int{frame.PointCenter.X - startFrame.PointCenter.X, frame.PointCenter.Y - startFrame.PointCenter.Y}
			mask := getDialogMask(patternInfo, maskMove)
			maskEvent := bodyEvent
			maskEvent.Layer = 0
			maskEvent.Style = "screen"
			maskEvent.Text = mask

			// Character
			charaMaskString := getDialogCharacterMask(h, w, frame.PointCenter, pointSize)

			charaBodyEvent := bodyEvent
			charaBodyEvent.Style = "character"
			charaBodyEvent.Layer = 0
			_, cmo := SplitArr(Str2IntArr(charaMaskString))
			charaOffset := int(float64(MinInt(cmo)) * 1.1)
			charaBodyEvent.Text = fmt.Sprintf("{\\pos(%d,%d)\\an4}",
				frame.PointCenter.X+charaOffset, frame.PointCenter.Y) + displayName
			if config.VideoOnly {
				charaBodyEvent.Type = "Comment"
			}
			// Character Mask
			charaMaskEvent := charaBodyEvent
			charaMaskEvent.Text = getDialogCharacterMask(h, w, frame.PointCenter, pointSize)

			if len(bodyEvents) > 0 && bodyEvents[len(bodyEvents)-1].Text == bodyEvent.Text {
				bodyEvents[len(bodyEvents)-1].End = bodyEvent.End
				maskEvents[len(bodyEvents)-1].End = bodyEvent.End
				charaBodyEvents[len(bodyEvents)-1].End = bodyEvent.End
				charaMaskEvents[len(bodyEvents)-1].End = bodyEvent.End
			} else {
				bodyEvents = append(bodyEvents, bodyEvent)
				maskEvents = append(maskEvents, maskEvent)
				charaBodyEvents = append(charaBodyEvents, charaBodyEvent)
				charaMaskEvents = append(charaMaskEvents, charaMaskEvent)
			}
		}

		b := bodyEvents[len(bodyEvents)-1]
		b.Type = "Comment"
		b.Start = MsToString(int(frameTime * float64(startFrame.FrameId)))
		b.Text = dialogBody
		bodyEvents = append(bodyEvents, b)

		m := maskEvents[len(maskEvents)-1]
		m.Type = "Comment"
		m.Start = MsToString(int(frameTime * float64(startFrame.FrameId)))
		m.Text = getDialogMask(patternInfo, [2]int{})
		maskEvents = append(maskEvents, m)

		cb := charaBodyEvents[len(charaBodyEvents)-1]
		cb.Type = "Comment"
		cb.Start = MsToString(int(frameTime * float64(startFrame.FrameId)))
		cb.Text = displayName
		charaBodyEvents = append(charaBodyEvents, cb)

		cm := charaMaskEvents[len(charaMaskEvents)-1]
		cm.Type = "Comment"
		cm.Start = MsToString(int(frameTime * float64(startFrame.FrameId)))
		cb.Text = getDialogCharacterMask(h, w, startFrame.PointCenter, pointSize)
		return charaMaskEvents, charaBodyEvents, maskEvents, bodyEvents
	}
}

// BANNER
type bannerFrame struct {
	FrameId int
}

func bannerMakeEvent(bannerInfo StoryEvent, areaMask string, frameTime float64, frames []bannerFrame) []SubtitleEventItem {
	var fadeFrame = 100.0 / frameTime
	var mask = SubtitleEventItem{
		Type: "Dialogue", Style: "address", Layer: 1, Name: "", MarginL: 0, MarginR: 0, MarginV: 0, Effect: "",
		Start: MsToString(int(frameTime * float64(MaxInt([]int{int(float64(frames[0].FrameId) - fadeFrame), 0})))),
		End:   MsToString(int(frameTime * (float64(frames[len(frames)-1].FrameId) + fadeFrame))),
		Text:  "{\\fad(100,100)}" + areaMask}
	body := mask
	body.Text = "{\\fad(100,100)}" + bannerInfo.Content().Body
	body.Layer = 2
	var events = []SubtitleEventItem{mask, body}
	return events
}

// MARKER
type markerFrame struct {
	Position image.Point
	FrameId  int
}

func markerMakeEvent(markerInfo StoryEvent, h, w int, frameTime float64, frames []markerFrame) []SubtitleEventItem {
	var maskEvents []SubtitleEventItem
	var bodyEvents []SubtitleEventItem
	markerBody := markerInfo.Content().Body
	maskString, maskSize := getAreaMarkerMask(h, w)

	for _, frame := range frames {
		rightPosition := image.Point{X: frame.Position.X, Y: int(float64(frame.Position.Y) * 7 / 6)}
		ms := fmt.Sprintf("{\\pos(%d,%d)}%s", rightPosition.X+maskSize[1]/15, rightPosition.Y+maskSize[0]/2, maskString)

		bodyPosition := fmt.Sprintf("{\\an7\\fs%d\\pos(%d,%d)}",
			maskSize[0], rightPosition.X-(maskSize[1]*9/10), rightPosition.Y)
		bodyEvent := SubtitleEventItem{
			Type: "Dialogue", Style: "address", Layer: 2, Name: "",
			Start:   MsToString(int(float64(frame.FrameId) * frameTime)),
			End:     MsToString(int(float64(frame.FrameId+1) * frameTime)),
			MarginL: 0, MarginR: 0, MarginV: 0, Effect: "",
			Text: bodyPosition + markerBody}
		if len(bodyEvents) > 0 && bodyEvents[len(bodyEvents)-1].Text == bodyEvent.Text {
			bodyEvents[len(bodyEvents)-1].End = bodyEvent.End
			maskEvents[len(bodyEvents)-1].End = bodyEvent.End
		} else {
			bodyEvents = append(bodyEvents, bodyEvent)
			maskEvent := bodyEvent
			maskEvent.Text = ms
			maskEvent.Layer = 1
			maskEvents = append(maskEvents, maskEvent)
		}
	}
	if len(bodyEvents) > 0 {
		bodyEvent := SubtitleEventItem{
			Type: "Comment", Layer: 2, Style: "address", Name: "", MarginL: 0, MarginR: 0, MarginV: 0, Effect: "",
			Start: bodyEvents[0].Start, End: bodyEvents[len(bodyEvents)-1].End, Text: markerBody,
		}
		maskEvent := bodyEvent
		maskEvent.Text = maskString
		maskEvent.Layer = 1
		bodyEvents = append(bodyEvents, bodyEvent)
		maskEvents = append(maskEvents, maskEvent)
	}
	return append(maskEvents, bodyEvents...)
}

// TASK

type TaskConfig struct {
	VideoFile     string      `json:"video_file"`
	DataFile      []string    `json:"data_file"`
	OutputPath    string      `json:"output_path"`
	Overwrite     bool        `json:"overwrite"`
	Font          string      `json:"font"`
	VideoOnly     bool        `json:"video_only"`
	Staff         []StaffItem `json:"staff"`
	TyperInterval [2]int      `json:"typer_interval"`
	Duration      [2]int      `json:"duration"`
	Debug         bool        `json:"debug"`
}

type Task struct {
	Config     TaskConfig
	Processing bool
	Stopped    bool
	Logs       []Log
	LogChan    chan Log
	Id         string
}

type Log struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

type LogProgress struct {
	Frame    int     `json:"frame"`
	Time     int     `json:"time"`
	Remains  int     `json:"remains"`
	Progress float64 `json:"progress"`
	Speed    float64 `json:"speed"`
	Fps      float64 `json:"fps"`
}

func (t *Task) load() PJSTranslationData {
	var result = PJSTranslationData{}
	if len(t.Config.DataFile) > 1 {
		result = MakePJSData(t.Config.DataFile[0], t.Config.DataFile[1])
		go t.Log(Log{Type: "string", Data: "[Initial] Loaded Legacy Json File and Text File."})
	} else if len(t.Config.DataFile) == 1 {
		if strings.HasSuffix(t.Config.DataFile[0], "pjs.txt") {
			result = ReadPJSFile(t.Config.DataFile[0])
			go t.Log(Log{Type: "string", Data: "[Initial] Loaded PJS Story File."})
		} else {
			result = MakePJSData(t.Config.DataFile[0], "")
			go t.Log(Log{Type: "string", Data: "[Initial] Loaded Legacy Json File."})
		}
	} else {
		go t.Log(Log{Type: "string", Data: "[Initial] Using Empty Story Data"})
	}
	if result.Data.Count() > 0 {
		go t.Log(Log{Type: "string",
			Data: fmt.Sprintf("[Initial] Loaded %d Dialogs, %d Banners, %d Markers",
				result.Dialogs().Count(), result.Banners().Count(), result.Markers().Count())})
	}
	return result

}

func (t *Task) match() (
	dialogTalkDataEvents, dialogCharacterEvents, bannerEvents, markerEvents []SubtitleEventItem, dialogStyles []SubtitleStyleItem, err error) {

	timeStart := time.Now().UnixMilli()
	var vc *gocv.VideoCapture

	if FileExist(t.Config.VideoFile) {
		vc, _ = gocv.VideoCaptureFile(t.Config.VideoFile)
	} else {
		return nil, nil, nil, nil, nil, errors.New("video File Not Exist")
	}
	var StoryData = t.load()

	var videoHeight = int(vc.Get(gocv.VideoCaptureFrameHeight))
	var videoWidth = int(vc.Get(gocv.VideoCaptureFrameWidth))
	var videoFps = vc.Get(gocv.VideoCaptureFPS)
	var videoFrameCount = int(vc.Get(gocv.VideoCaptureFrameCount))

	// Templates
	var templateDialogPointer = getResizedDialogPointer(videoHeight, videoWidth)
	var templateMenuSign = getResizedInterfaceMenu(videoHeight, videoWidth)
	var templateMarker = getResizedAreaMarkerTemplate(videoHeight, videoWidth)
	var templateBannerReverse = gocv.NewMat()
	var templateBannerCanny = gocv.NewMat()
	var bannerArea = getBannerArea(videoHeight, videoWidth)
	{
		var bannerEdge = getResizedAreaEdge(videoHeight, videoWidth)
		s := int(math.Abs(float64(bannerArea[1] - bannerArea[0])))
		gocv.Resize(bannerEdge, &bannerEdge, image.Point{X: s, Y: s}, 0, 0, gocv.InterpolationLanczos4)
		gocv.Canny(bannerEdge, &templateBannerCanny, 50, 150)
		gocv.Threshold(bannerEdge, &templateBannerReverse, 128.0, 255.0, gocv.ThresholdBinaryInv)
		_ = bannerEdge.Close()
	}

	var contentStart = false
	var markerProcessRunning = true
	var bannerProcessRunning = true
	var dialogProcessRunning = true
	if !t.Config.VideoOnly {
		markerProcessRunning = StoryData.Markers().Count() > 0
		bannerProcessRunning = StoryData.Banners().Count() > 0
		dialogProcessRunning = StoryData.Dialogs().Count() > 0
	}

	var totalFrameCount int
	var videoCut = false
	var nowFrameCount = 0
	var setStopped = false
	var fpsTimeCounter = []LogProgress{{Time: int(timeStart)}}
	if t.Config.Duration == [2]int{} || t.Config.Duration == [2]int{0, videoFrameCount} {
		totalFrameCount = videoFrameCount
	} else {
		videoCut = true
		totalFrameCount = t.Config.Duration[1] - t.Config.Duration[0]
		vc.Set(gocv.VideoCapturePosFrames, float64(t.Config.Duration[0]))
		nowFrameCount = t.Config.Duration[0]
		contentStart = true
	}

	var dialogFrameSet [][]dialogFrame
	var bannerFrameSet [][]bannerFrame
	var markerFrameSet [][]markerFrame
	var dialogConstPointCenter image.Point
	func() {
		var bannerProcessingFrames []bannerFrame
		var bannerProcessedCount = 0
		var bannerLastResult = false

		var markerProcessingFrames []markerFrame
		var markerProcessedCount = 0
		var markerLastResult image.Point

		var dialogLastStatus = uint8(0)
		var dialogProcessedCount = 0
		var dialogProcessingFrames []dialogFrame
		var dialogLastPointCenter = image.Point{}

		for {
			if t.Stopped {
				setStopped = true
				break
			}
			var frame = gocv.NewMat()
			vc.Read(&frame)
			if frame.Empty() {
				break
			}

			gocv.CvtColor(frame, &frame, gocv.ColorBGRToGray)
			if !contentStart {
				contentStart = matchCheckStart(frame, templateMenuSign)
			}
			if contentStart {
				var running = true
				if !t.Config.VideoOnly {
					if StoryData.Dialogs().Count() == dialogProcessedCount &&
						StoryData.Banners().Count() == bannerProcessedCount &&
						StoryData.Markers().Count() == markerProcessedCount {
						running = false
					}
				}
				if running {
					var dialogProcessFrame gocv.Mat
					if dialogProcessRunning {
						dialogProcessFrame = frame.Clone()
					}

					var bannerProcessNow = false
					var bannerProcessFrame gocv.Mat
					if bannerProcessRunning {
						if videoCut || t.Config.VideoOnly {
							bannerProcessNow = true
						} else if bannerProcessedCount < StoryData.Banners().Count() {
							if StoryData.Data.IndexType("Banner", bannerProcessedCount) < StoryData.Data.IndexType("Dialog", dialogProcessedCount) {
								bannerProcessNow = true
							}
						}
						if bannerProcessNow {
							bannerProcessFrame = frame.Clone()
						}
					}

					var markerProcessNow = false
					var markerProcessFrame gocv.Mat
					if markerProcessRunning {
						if videoCut || t.Config.VideoOnly {
							markerProcessNow = true
						} else if markerProcessedCount < StoryData.Markers().Count() {
							if StoryData.Data.IndexType("Marker", bannerProcessedCount) < StoryData.Data.IndexType("Dialog", dialogProcessedCount) {
								markerProcessNow = true
							}
						}
						if markerProcessNow {
							markerProcessFrame = frame.Clone()
						}
					}

					var group = sync.WaitGroup{}
					group.Add(3)
					go func() {
						if dialogProcessRunning {
							dialogProcessResult := matchFrameDialog(dialogProcessFrame, templateDialogPointer, dialogLastPointCenter)
							if dialogConstPointCenter.Eq(image.Point{}) {
								if dialogProcessResult.status == 2 {
									dialogConstPointCenter = dialogProcessResult.pointCenter
								}
							}
							if dialogProcessResult.status != 2 && dialogLastStatus == 2 {
								dialogFrameSet = append(dialogFrameSet, dialogProcessingFrames)
								dialogProcessedCount = len(dialogFrameSet)

								go t.Log(Log{
									Type: "string",
									Data: fmt.Sprintf("[Processing] Locate %d Frames for Dialog No.%d", len(dialogProcessingFrames), dialogProcessedCount),
								})

								dialogProcessingFrames = []dialogFrame{}
								if !t.Config.VideoOnly && dialogProcessedCount == StoryData.Dialogs().Count() {
									dialogProcessRunning = false
								}
							}
							if dialogProcessResult.status != 0 {
								dialogProcessingFrames = append(dialogProcessingFrames, dialogFrame{
									FrameId: nowFrameCount, PointCenter: dialogProcessResult.pointCenter})
							}
							dialogLastStatus = dialogProcessResult.status
							dialogLastPointCenter = dialogProcessResult.pointCenter
							_ = dialogProcessFrame.Close()
						}
						group.Done()
					}()
					go func() {
						if bannerProcessNow {
							bannerProcessResult := matchFrameBanner(bannerProcessFrame, templateBannerCanny, templateBannerReverse, bannerArea)
							if bannerProcessResult {
								bannerProcessingFrames = append(bannerProcessingFrames, bannerFrame{FrameId: nowFrameCount})
							}
							if bannerLastResult && !bannerProcessResult {
								bannerFrameSet = append(bannerFrameSet, bannerProcessingFrames)
								bannerProcessedCount = len(bannerFrameSet)

								go t.Log(Log{
									Type: "string",
									Data: fmt.Sprintf("[Processing] Locate %d Frames for Banner No.%d", len(bannerProcessingFrames), bannerProcessedCount),
								})

								bannerProcessingFrames = []bannerFrame{}
								if !t.Config.VideoOnly && bannerProcessedCount == StoryData.Banners().Count() {
									bannerProcessRunning = false
								}
							}
							bannerLastResult = bannerProcessResult
							_ = bannerProcessFrame.Close()
						}
						group.Done()
					}()
					go func() {
						if markerProcessNow {
							markerProcessResult := matchFrameMarker(markerProcessFrame, templateMarker)
							if !markerProcessResult.Eq(image.Point{}) {
								markerProcessingFrames = append(markerProcessingFrames,
									markerFrame{Position: markerProcessResult, FrameId: nowFrameCount})
							}
							if !markerLastResult.Eq(image.Point{}) && markerProcessResult.Eq(image.Point{}) {
								markerFrameSet = append(markerFrameSet, markerProcessingFrames)
								markerProcessedCount = len(markerFrameSet)

								go t.Log(Log{
									Type: "string",
									Data: fmt.Sprintf("[Processing] Locate %d Frames for Marker No.%d", len(markerProcessingFrames), markerProcessedCount),
								})

								markerProcessingFrames = []markerFrame{}
								if !t.Config.VideoOnly && markerProcessedCount == StoryData.Markers().Count() {
									markerProcessRunning = false
								}
							}
							markerLastResult = markerProcessResult
							_ = markerProcessFrame.Close()
						}
						group.Done()
					}()
					group.Wait()
				}
			}
			_ = frame.Close()

			nowFrameCount += 1
			lp := LogProgress{
				Frame:    nowFrameCount,
				Time:     int(time.Now().UnixMilli() - timeStart),
				Remains:  totalFrameCount + t.Config.Duration[0] - nowFrameCount,
				Progress: float64(nowFrameCount-t.Config.Duration[0]) / float64(totalFrameCount),
				Speed:    float64(nowFrameCount) / (float64(time.Now().UnixMilli()-timeStart) / 1000.0),
			}
			lp.Fps = float64(lp.Frame-fpsTimeCounter[0].Frame) / float64(lp.Time-fpsTimeCounter[0].Time) * 1000.0
			if len(fpsTimeCounter) == int(videoFps/2.0) || fpsTimeCounter[0].Frame == 0 {
				fpsTimeCounter = append(fpsTimeCounter[1:], lp)
			} else {
				fpsTimeCounter = append(fpsTimeCounter, lp)
			}
			l, _ := json.Marshal(lp)
			go t.Log(Log{Type: "dict", Data: string(l)})
			if nowFrameCount-t.Config.Duration[0] > totalFrameCount {
				break
			}
		}
	}()
	var videoFrameTimeMs = 1000.0 / videoFps
	var bannerMask = getAreaBannerMask(getAreaMaskSize(videoHeight, videoWidth))

	if !setStopped {
		for i, frames := range dialogFrameSet {
			var dialogData StoryEvent
			if !t.Config.VideoOnly && i < StoryData.Dialogs().Count() {
				dialogData = StoryData.Dialogs()[i]
			}
			var dialogLastEndFrame dialogFrame
			var dialogLastEndEvent SubtitleEventItem
			if i > 0 {
				lfs := dialogFrameSet[i-1]
				if len(lfs) > 0 {
					dialogLastEndFrame = lfs[len(lfs)-1]
				}
				if len(dialogTalkDataEvents) > 0 {
					dialogLastEndEvent = dialogTalkDataEvents[len(dialogTalkDataEvents)-1]
				}
			}

			var dialogIsMaskStart bool
			if !t.Config.VideoOnly {
				index := Index(dialogData, StoryData.DPeriod())
				if index > 0 {
					dialogIsMaskStart = StoryData.DPeriod()[index-1].Type == "Period"
				} else if index == 0 {
					dialogIsMaskStart = true
				}
			} else {
				if i > 0 {
					lfs := dialogFrameSet[i-1]
					if len(lfs) > 0 && len(frames) > 0 && lfs[len(lfs)-1].FrameId == frames[0].FrameId-1 {
						dialogIsMaskStart = true
					}
				}
			}

			characterMasks, characterEvents, dialogMasks, dialogEvents := dialogMakeEvent(
				dialogData, templateDialogPointer.Cols(), videoHeight, videoWidth, videoFrameTimeMs, dialogLastEndFrame,
				frames, dialogLastEndEvent, dialogIsMaskStart, t.Config)

			dialogTalkDataEvents = append(dialogTalkDataEvents, dialogMasks...)
			dialogTalkDataEvents = append(dialogTalkDataEvents, dialogEvents...)
			dialogCharacterEvents = append(dialogCharacterEvents, characterMasks...)
			dialogCharacterEvents = append(dialogCharacterEvents, characterEvents...)

			go t.Log(Log{Type: "string",
				Data: fmt.Sprintf("[Processing] Generated %d Events for Dialog No.%d",
					len(characterMasks)+len(characterEvents)+len(dialogMasks)+len(dialogEvents), i+1)})
		}
		for i, frames := range bannerFrameSet {
			var bannerData StoryEvent
			if !t.Config.VideoOnly && i < StoryData.Banners().Count() {
				bannerData = StoryData.Banners()[i]
			}
			events := bannerMakeEvent(bannerData, bannerMask, videoFrameTimeMs, frames)
			bannerEvents = append(bannerEvents, events...)
			go t.Log(Log{Type: "string",
				Data: fmt.Sprintf("[Processing] Generated %d Events for Banner No.%d", len(events), i+1),
			})
		}
		for i, frames := range markerFrameSet {
			var markerData StoryEvent
			if !t.Config.VideoOnly && i < StoryData.Markers().Count() {
				markerData = StoryData.Markers()[i]
			}
			events := markerMakeEvent(markerData, videoHeight, videoWidth, videoFrameTimeMs, frames)
			markerEvents = append(markerEvents, events...)
			go t.Log(Log{
				Type: "string",
				Data: fmt.Sprintf("[Processing] Generated %d Events for Banner No.%d", len(events), i+1),
			})
		}
	}

	if !setStopped {
		if len(dialogTalkDataEvents)+len(dialogCharacterEvents)+len(bannerEvents)+len(markerEvents) == 0 {
			err = errors.New("no Event Matched")
		} else {
			dialogStyles = dialogMakeStyle(t.Config, dialogConstPointCenter, templateDialogPointer.Cols())
			if !t.Config.VideoOnly {
				var recheck []string
				if len(dialogFrameSet) != StoryData.Dialogs().Count() {
					recheck = append(recheck, "Dialog")
				}
				if len(bannerFrameSet) != StoryData.Banners().Count() {
					recheck = append(recheck, "Banner")
				}
				if len(markerFrameSet) != StoryData.Markers().Count() {
					recheck = append(recheck, "Marker")
				}
				if len(recheck) > 0 {
					go t.Log(Log{Type: "string",
						Data: fmt.Sprintf("[Warning] Unmatched Event Exists:%s", strings.Join(recheck, ","))})
				}
			}
			err = nil
		}
	} else {
		err = errors.New("process was Stopped")
	}

	_ = templateDialogPointer.Close()
	_ = templateBannerCanny.Close()
	_ = templateBannerReverse.Close()
	_ = templateMenuSign.Close()
	_ = templateMarker.Close()
	return
}
func (t *Task) Run() {
	t.Processing = true
	t.Stopped = false

	timeStart := time.Now().UnixMilli()
	go t.Log(Log{Type: "string", Data: "[Processing] Process Started"})
	dialogsEvents, charactersEvents, bannerEvents, markerEvents, dialogStyles, err := t.match()
	if err != nil {
		go t.Log(Log{Type: "string", Data: fmt.Sprintf("[Error] Process Failed: %s", err.Error())})
	} else {
		go t.Log(Log{Type: "string", Data: fmt.Sprintf("[Finish] Process Finished in %ds", (time.Now().UnixMilli()-timeStart)/1000)})
		var staffEvents []SubtitleEventItem
		var staffStyle []SubtitleStyleItem
		for _, staff := range t.Config.Staff {
			e, s := makeStaffEvent(staff, dialogStyles[0].Fontsize, dialogStyles[0].FontName)
			staffEvents = append(staffEvents, e)
			staffStyle = append(staffStyle, s)
		}
		filename := path.Base(t.Config.VideoFile)
		events := []SubtitleEventItem{getDividerSubtitleEvent(filename+" - Made by SekaiSubtitle", 5)}
		events = append(events, GetSubtitleArraySurrounded(staffEvents, "Staff", 15)...)
		events = append(events, GetSubtitleArraySurrounded(bannerEvents, "Banner", 15)...)
		events = append(events, GetSubtitleArraySurrounded(markerEvents, "Marker", 15)...)
		events = append(events, GetSubtitleArraySurrounded(charactersEvents, "Character", 15)...)
		events = append(events, GetSubtitleArraySurrounded(dialogsEvents, "Dialog", 15)...)

		vc, err := gocv.VideoCaptureFile(t.Config.VideoFile)
		CheckErr(err)
		res := Subtitle{
			ScriptInfo: SubtitleScriptInfo{
				Title: filename, ScriptType: "v4.00+",
				PlayRexX: int(vc.Get(gocv.VideoCaptureFrameWidth)),
				PlayRexY: int(vc.Get(gocv.VideoCaptureFrameHeight))},
			Garbage: SubtitleGarbage{AudioFile: filename, VideoFile: filename},
			Styles:  SubtitleStyles{Items: append(dialogStyles, staffStyle...)},
			Events:  SubtitleEvents{Items: events},
		}

		exists := FileExist(t.Config.OutputPath)
		con := false
		if exists {
			if t.Config.Overwrite {
				con = true
				go t.Log(Log{Type: "string", Data: "[Finish] Overwriting Existed File"})
			}
		} else {
			con = true
		}
		if con {
			WriteFileString(t.Config.OutputPath, res.string())
			go t.Log(Log{Type: "string", Data: "[Finish] Process Finished"})
		} else {
			go t.Log(Log{Type: "string", Data: "[Finish] Skipped Output Because of File Exists"})
		}
		err = vc.Close()
		CheckErr(err)
	}
	t.Processing = false
}
func (t *Task) Log(log Log) {
	t.LogChan <- log
}
func (t *Task) Stop() {
	t.Stopped = true
}

func NewTask(config TaskConfig) Task {
	// var c = config
	// var defaultTyperInterval = [2]int{50, 80}

	var task = Task{
		Config:     config,
		Processing: false,
		Stopped:    false,
		Logs:       []Log{},
		LogChan:    make(chan Log, 1e3),
		Id:         Md5(strconv.FormatInt(time.Now().UnixMilli(), 10)+config.VideoFile, 6),
	}
	return task
}
