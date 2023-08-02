package process

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type VoiceData struct {
	Character2DId int     `json:"Character2dId"`
	VoiceId       string  `json:"VoiceId"`
	Volume        float64 `json:"Volume"`
}

func (v VoiceData) CharacterId() int {
	var l2dCid = C2Did_to_Cid[v.Character2DId]
	if l2dCid >= 1 && l2dCid <= 26 {
		return l2dCid
	}
	s := strings.Split(v.VoiceId, "_")
	var ns []int
	for i := range s {
		n := Str2int(s[i])
		if n != 0 {
			ns = append(ns, n)
		}
	}
	if len(ns) > 0 {
		return ns[len(ns)-1]
	}
	return 0
}

type SnippetItem struct {
	Action int `json:"Action"`
	// Index            int     `json:"Index"`
	// ProgressBehavior int     `json:"ProgressBehavior"`
	// ReferenceIndex   int     `json:"ReferenceIndex"`
	// Delay            float64 `json:"Delay"`
}
type TalkDataItem struct {
	WindowDisplayName     string      `json:"WindowDisplayName"`
	Body                  string      `json:"Body"`
	WhenFinishCloseWindow int         `json:"WhenFinishCloseWindow"`
	Voices                []VoiceData `json:"Voices"`
}

func (t TalkDataItem) CharacterId() int {
	if len(t.Voices) == 1 {
		if t.Voices[0].CharacterId() >= 0 || t.Voices[0].CharacterId() <= 26 {
			return t.Voices[0].CharacterId()
		}
	}
	return 0
}

type SpecialEffectDataItem struct {
	EffectType int    `json:"EffectType"`
	StringVal  string `json:"StringVal"`
	// StringValSub string  `json:"StringValSub"`
	// Duration     float64 `json:"Duration"`
	// IntVal       int     `json:"IntVal"`
}
type GameStoryData struct {
	TalkData          []TalkDataItem          `json:"TalkData"`
	Snippets          []SnippetItem           `json:"Snippets"`
	SpecialEffectData []SpecialEffectDataItem `json:"SpecialEffectData"`
}

func (s *GameStoryData) Clean() {
	var sn []SnippetItem
	seCount := 0
	for _, snippet := range s.Snippets {
		if snippet.Action == 1 {
			sn = append(sn, snippet)
		} else if snippet.Action == 6 {
			seData := s.SpecialEffectData[seCount]
			if seData.EffectType == 8 || seData.EffectType == 18 {
				sn = append(sn, snippet)
			}
			seCount += 1
		}
	}
	s.Snippets = sn
	var se []SpecialEffectDataItem
	for _, datum := range s.SpecialEffectData {
		if datum.EffectType == 8 || datum.EffectType == 18 {
			se = append(se, datum)
		}
	}
	s.SpecialEffectData = se
}

func (s *GameStoryData) Empty() bool {
	return len(s.Snippets)+len(s.SpecialEffectData)+len(s.TalkData) == 0
}

func ReadJson(file string) GameStoryData {
	var result GameStoryData
	if FileExist(file) {
		dat, _ := os.ReadFile(file)
		_ = json.Unmarshal(dat, &result)
		result.Clean()
	}
	return result
}

// Translations From Original Text File

type DialogTranslate struct {
	Chara string
	Body  string
}
type EffectTranslate struct {
	Body string
}

//	type BannerTranslate struct {
//		Body string
//	}
//
//	type MarkerTranslate struct {
//		Body string
//	}
type TranslateData struct {
	Dialogs []DialogTranslate
	Effects []EffectTranslate
	// Banners []BannerTranslate
	// Markers []MarkerTranslate
}

func (t TranslateData) Empty() bool {
	return len(t.Dialogs)+len(t.Effects) == 0
}

var DialogReg, _ = regexp.Compile("^([^：]+)：(.*)$")

func ReadText(file string) TranslateData {
	var result = TranslateData{}
	if FileExist(file) {
		var Dialogs []DialogTranslate
		var Effects []EffectTranslate

		dat, _ := os.ReadFile(file)
		data := strings.Split(string(dat), "\n")
		data = ArrSplit(data)
		for _, v := range data {
			var res = DialogReg.FindStringSubmatch(v)
			if len(res) != 0 {
				r := DialogTranslate{Chara: res[1], Body: res[2]}
				Dialogs = append(Dialogs, r)
			} else {
				r := EffectTranslate{Body: v}
				Effects = append(Effects, r)
			}
		}
		result.Dialogs = Dialogs
		result.Effects = Effects
	}

	return result
}

// New Yaml File PJS ContentT

type CharacterTranslation struct {
	Origin     string
	Translated string
}

func (c CharacterTranslation) Content() string {
	if len(c.Translated) > 0 {
		return c.Translated
	} else {
		return c.Origin
	}
}

//	type TranslationSet struct {
//		Translated  string `yaml:"初翻"`
//		Proofread   string `yaml:"校对"`
//		Appropriate string `yaml:"合意"`
//	}
//
//	func (t TranslationSet) Latest() string {
//		if len(t.Appropriate) > 0 {
//			return t.Appropriate
//		} else if len(t.Proofread) > 0 {
//			return t.Proofread
//		} else {
//			return t.Translated
//		}
//	}

type StoryEvent struct {
	Type        string
	CharacterId int
	CharacterO  string
	CharacterT  string
	ContentO    string
	ContentT    string
}
type EventContent struct {
	Body      string
	Character string
}

func (s StoryEvent) Content() EventContent {
	var chara, body string
	if len(s.ContentT) > 0 {
		body = s.ContentT
	} else {
		body = s.ContentO
	}
	if len(s.CharacterT) > 0 {
		chara = s.CharacterT
	} else {
		chara = s.CharacterO
	}
	return EventContent{body, chara}
}
func (s StoryEvent) String() string {
	return fmt.Sprintf("%s,%02d,%s,%s,%s,%s",
		s.Type,
		s.CharacterId,
		s.CharacterO,
		s.CharacterT,
		s.ContentO,
		s.ContentT)
}
func EventFromString(s string) StoryEvent {
	sArr := strings.Split(s, ",")

	var result = StoryEvent{
		Type:        sArr[0],
		CharacterId: Str2int(sArr[1]),
		CharacterO:  sArr[2],
		CharacterT:  sArr[3],
		ContentO:    sArr[4],
		ContentT:    sArr[5],
	}
	return result
}

type StoryEventSet []StoryEvent

func (s StoryEventSet) Count() int {
	return len(s)
}
func (s StoryEventSet) IndexType(t string, i int) int {
	similarCount := 0
	for i2, event := range s {
		if event.Type == t {
			if similarCount == i {
				return i2
			}
			similarCount += 1
		}
	}
	return -1
}
func (s StoryEventSet) IndexTypes(ts []string, i int) int {
	similarCount := 0
	for i2, event := range s {
		for _, t := range ts {
			if event.Type == t {
				if similarCount == i {
					return i2
				}
				similarCount += 1
			}
		}

	}
	return -1
}

type PJSTranslationData struct {
	Data StoryEventSet `yaml:"内容"`
}

func (y PJSTranslationData) get(t []string) StoryEventSet {
	var result = StoryEventSet{}
	for _, datum := range y.Data {
		for _, s := range t {
			if datum.Type == s {
				result = append(result, datum)
				continue
			}
		}
	}
	return result
}
func (y PJSTranslationData) Dialogs() StoryEventSet { return y.get([]string{"Dialog"}) }
func (y PJSTranslationData) Banners() StoryEventSet { return y.get([]string{"Banner"}) }
func (y PJSTranslationData) Markers() StoryEventSet { return y.get([]string{"Marker"}) }
func (y PJSTranslationData) Effects() StoryEventSet { return y.get([]string{"Banner", "Marker"}) }
func (y PJSTranslationData) DPeriod() StoryEventSet { return y.get([]string{"Dialog", "Period"}) }
func (y PJSTranslationData) String() string {
	var s []string
	for _, datum := range y.Data {

		if datum.Type != "Dialog" {
			s = append(s, datum.String()+"\n")
		} else {
			s = append(s, datum.String())
		}
	}
	var result = strings.Join(s, "\n")
	return result
}
func ReadPJSFile(filename string) PJSTranslationData {
	var result = PJSTranslationData{}
	var SE = StoryEventSet{}
	if FileExist(filename) {
		dat, _ := os.ReadFile(filename)
		sArr := strings.Split(string(dat), "\n")
		for _, s := range sArr {
			if len(s) > 0 {
				e := EventFromString(s)
				SE = append(SE, e)
			}
		}
	}
	result.Data = SE
	return result
}
func MakePJSData(jsonFile, textFile string) PJSTranslationData {
	jsonData := ReadJson(jsonFile)
	textData := ReadText(textFile)
	result := PJSTranslationData{}
	if !jsonData.Empty() {
		dialogCount := 0
		effectCount := 0
		for _, snippet := range jsonData.Snippets {
			if snippet.Action == 1 {
				dialogData := jsonData.TalkData[dialogCount]
				if dialogCount < len(jsonData.TalkData) {
					s := StoryEvent{
						Type:        "Dialog",
						CharacterId: dialogData.CharacterId(),
						CharacterO:  dialogData.WindowDisplayName,
						ContentO:    strings.ReplaceAll(dialogData.Body, "\n", "\\N"),
					}
					result.Data = append(result.Data, s)
					if dialogData.WhenFinishCloseWindow == 1 {
						result.Data = append(result.Data, StoryEvent{Type: "Period"})
					}
				}
				dialogCount += 1
			}
			if snippet.Action == 6 {
				effectData := jsonData.SpecialEffectData[effectCount]
				var t string
				if effectData.EffectType == 8 {
					t = "Banner"
				}
				if effectData.EffectType == 18 {
					t = "Marker"
				}
				s := StoryEvent{
					Type:     t,
					ContentO: effectData.StringVal,
				}
				result.Data = append(result.Data, s)
				effectCount += 1
			}
		}
	}
	if !textData.Empty() {
		for i, dialog := range textData.Dialogs {
			if i < result.Dialogs().Count() {
				iT := result.Data.IndexType("Dialog", i)
				if iT >= 0 {
					result.Data[iT].ContentT = dialog.Body
					result.Data[iT].CharacterT = dialog.Chara
				}
			}
		}
		for i, effect := range textData.Effects {
			if i < result.Effects().Count() {
				iT := result.Data.IndexTypes([]string{"Banner", "Marker"}, i)
				if iT >= 0 {
					result.Data[iT].ContentT = effect.Body
				}
			}
		}
	}
	return result
}
