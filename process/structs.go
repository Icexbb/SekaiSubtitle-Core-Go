package process

import (
	"fmt"
	"strings"
)

type Point2D struct {
	x int
	y int
}

func (m *Point2D) move(x, y int) {
	m.x = m.x + x
	m.y = m.y + y
}
func (m *Point2D) scale(ratio float64) {
	m.x = int(float64(m.x) * ratio)
	m.y = int(float64(m.y) * ratio)
}
func (m *Point2D) string() string {
	return fmt.Sprintf("%d %d ", m.x, m.y)
}

type Move struct {
	point Point2D
}

func (m *Move) move(x, y int) {
	m.point.move(x, y)
}
func (m *Move) scale(ratio float64) {
	m.point.scale(ratio)
}
func (m *Move) string() string {
	return fmt.Sprintf("m %s", m.point.string())
}

type Line struct {
	point Point2D
}

func (m *Line) move(x, y int) {
	m.point.move(x, y)
}
func (m *Line) scale(ratio float64) {
	m.point.scale(ratio)
}
func (m *Line) string() string {
	return fmt.Sprintf("l %s", m.point.string())
}

type Bezier struct {
	points [3]Point2D
}

func (b *Bezier) move(x, y int) {
	for i, point := range b.points {
		point.move(x, y)
		b.points[i] = point
	}
}
func (b *Bezier) scale(ratio float64) {
	for i, point := range b.points {
		point.scale(ratio)
		b.points[i] = point
	}
}
func (b *Bezier) string() string {
	return fmt.Sprintf("b %s%s%s", b.points[0].string(), b.points[1].string(), b.points[2].string())
}

type AssDraw struct {
	proto string
}

func (a AssDraw) move(x, y int) AssDraw {
	var newProto = ""
	var sArr = strings.Split(a.proto, " ")
	for i := 0; i < len(sArr); i++ {
		switch sArr[i] {
		case "m":
			var step = Move{point: Point2D{x: Str2int(sArr[i+1]), y: Str2int(sArr[i+2])}}
			step.move(x, y)
			newProto += step.string()
			i += 2
		case "l":
			var step = Line{point: Point2D{x: Str2int(sArr[i+1]), y: Str2int(sArr[i+2])}}
			step.move(x, y)
			newProto += step.string()
			i += 2
		case "b":
			var step = Bezier{points: [3]Point2D{
				{x: Str2int(sArr[i+1]), y: Str2int(sArr[i+2])},
				{x: Str2int(sArr[i+3]), y: Str2int(sArr[i+4])},
				{x: Str2int(sArr[i+5]), y: Str2int(sArr[i+6])},
			}}
			step.move(x, y)
			newProto += step.string()
			i += 6
		}

	}
	return AssDraw{proto: newProto}
}
func (a AssDraw) scale(ratio float64) AssDraw {
	var newProto = ""
	var sArr = strings.Split(a.proto, " ")
	for i := 0; i < len(sArr); i++ {
		switch sArr[i] {
		case "m":
			var step = Move{point: Point2D{x: Str2int(sArr[i+1]), y: Str2int(sArr[i+2])}}
			step.scale(ratio)
			newProto += step.string()
			i += 2
		case "l":
			var step = Line{point: Point2D{x: Str2int(sArr[i+1]), y: Str2int(sArr[i+2])}}
			step.scale(ratio)
			newProto += step.string()
			i += 2
		case "b":
			var step = Bezier{points: [3]Point2D{
				{x: Str2int(sArr[i+1]), y: Str2int(sArr[i+2])},
				{x: Str2int(sArr[i+3]), y: Str2int(sArr[i+4])},
				{x: Str2int(sArr[i+5]), y: Str2int(sArr[i+6])},
			}}
			step.scale(ratio)
			newProto += step.string()
			i += 6
		}
	}
	return AssDraw{proto: newProto}
}

type SubtitleScriptInfo struct {
	Title      string
	ScriptType string
	PlayRexX   int
	PlayRexY   int
}

func (s SubtitleScriptInfo) string() string {
	return fmt.Sprintf(
		"[Script Info]\nTitle: %s\nScriptType: %s\nPlayRexX: %d\nPlayRexY: %d\n\n",
		s.Title, s.ScriptType, s.PlayRexX, s.PlayRexY,
	)
}

type SubtitleGarbage struct {
	AudioFile string
	VideoFile string
}

func (s SubtitleGarbage) string() string {
	return fmt.Sprintf(
		"[Aegisub Project Garbage]\nAudio File: %s\nVideo File: %s\n\n",
		s.AudioFile, s.VideoFile,
	)
}

type SubtitleStyleItem struct {
	Name            string  `json:"Name"`
	FontName        string  `json:"Fontname"`
	Fontsize        int     `json:"Fontsize"`
	PrimaryColour   string  `json:"PrimaryColour"`
	SecondaryColour string  `json:"SecondaryColour"`
	OutlineColour   string  `json:"OutlineColour"`
	BackColour      string  `json:"BackColour"`
	Bold            int     `json:"Bold"`
	Italic          int     `json:"Italic"`
	Underline       int     `json:"Underline"`
	StrikeOut       int     `json:"StrikeOut"`
	ScaleX          int     `json:"ScaleX"`
	ScaleY          int     `json:"ScaleY"`
	Spacing         float64 `json:"Spacing"`
	Angle           int     `json:"Angle"`
	BorderStyle     int     `json:"BorderStyle"`
	Outline         float64 `json:"Outline"`
	Shadow          float64 `json:"Shadow"`
	Alignment       int     `json:"Alignment"`
	MarginL         int     `json:"MarginL"`
	MarginR         int     `json:"MarginR"`
	MarginV         int     `json:"MarginV"`
	Encoding        int     `json:"Encoding"`
}

func (s SubtitleStyleItem) string() string {
	return fmt.Sprintf("Style:%s,%s,%d,%s,%s,%s,%s,%d,%d,%d,%d,%d,%d,%.1f,%d,%d,%.1f,%.1f,%d,%d,%d,%d,%d\n",
		s.Name, s.FontName, int(s.Fontsize), s.PrimaryColour, s.SecondaryColour, s.OutlineColour, s.BackColour,
		s.Bold, s.Italic, s.Underline, s.StrikeOut, s.ScaleX, s.ScaleY, s.Spacing, s.Angle, s.BorderStyle,
		s.Outline, s.Shadow, s.Alignment, s.MarginL, s.MarginR, s.MarginV, s.Encoding,
	)
}

type SubtitleStyles struct {
	Items []SubtitleStyleItem
}

func (s SubtitleStyles) string() string {
	h := "[V4+ Styles]\nFormat: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, " +
		"Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, " +
		"MarginL, MarginR, MarginV, Encoding\n"
	for _, i := range s.Items {
		h += i.string()
	}
	return h + "\n"
}

type SubtitleEventItem struct {
	Type    string
	Layer   int
	Start   string
	End     string
	Style   string
	Name    string
	MarginL int
	MarginR int
	MarginV int
	Effect  string
	Text    string
}

func (e SubtitleEventItem) string() string {
	return fmt.Sprintf("%s: %d,%s,%s,%s,%s,%d,%d,%d,%s,%s\n",
		e.Type, e.Layer, e.Start, e.End, e.Style, e.Name, e.MarginL, e.MarginR, e.MarginV, e.Effect, e.Text,
	)
}

type SubtitleEvents struct {
	Items []SubtitleEventItem
}

func (e SubtitleEvents) string() string {
	h := "[Events]\nFormat: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n"
	for _, i := range e.Items {
		h += i.string()
	}
	return h + "\n"
}

type Subtitle struct {
	ScriptInfo SubtitleScriptInfo
	Garbage    SubtitleGarbage
	Styles     SubtitleStyles
	Events     SubtitleEvents
}

func (s Subtitle) string() string {
	return s.ScriptInfo.string() + s.Garbage.string() + s.Styles.string() + s.Events.string()
}
