package process

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
)

type SnippetItem struct {
	Index            int     `json:"Index"`
	Action           int     `json:"Action"`
	ProgressBehavior int     `json:"ProgressBehavior"`
	ReferenceIndex   int     `json:"ReferenceIndex"`
	Delay            float64 `json:"Delay"`
}
type TalkDataItem struct {
	WindowDisplayName     string `json:"WindowDisplayName"`
	Body                  string `json:"Body"`
	WhenFinishCloseWindow int    `json:"WhenFinishCloseWindow"`
}
type SpecialEffectDataItem struct {
	EffectType   int     `json:"EffectType"`
	StringVal    string  `json:"StringVal"`
	StringValSub string  `json:"StringValSub"`
	Duration     float64 `json:"Duration"`
	IntVal       int     `json:"IntVal"`
}

type StoryData struct {
	TalkData          []TalkDataItem          `json:"TalkData"`
	Snippets          []SnippetItem           `json:"Snippets"`
	SpecialEffectData []SpecialEffectDataItem `json:"SpecialEffectData"`
}

func ReadJson(file string) StoryData {
	var err error
	dat, err := os.ReadFile(file)
	CheckErr(err)
	var result StoryData
	err = json.Unmarshal(dat, &result)
	CheckErr(err)
	return result
}

type DialogTranslate struct {
	Chara string
	Body  string
}

type BannerTranslate struct {
	Body string
}
type TagTranslate struct {
	Body string
}
type TranslateData struct {
	Dialogs []DialogTranslate
	Banners []BannerTranslate
	Tags    []TagTranslate
}

var DialogReg, _ = regexp.Compile("^([^：]+)：(.*)$")

func ReadText(file string) TranslateData {
	var err error
	dat, err := os.ReadFile(file)
	CheckErr(err)
	data := strings.Split(string(dat), "\n")
	data = ArrSplit(data)
	var Dialogs []DialogTranslate
	var Banners []BannerTranslate
	for _, v := range data {
		var res = DialogReg.FindStringSubmatch(v)
		if len(res) != 0 {
			r := DialogTranslate{Chara: res[1], Body: res[2]}
			Dialogs = append(Dialogs, r)
		} else {
			r := BannerTranslate{Body: v}
			Banners = append(Banners, r)
		}
	}
	var result = TranslateData{Dialogs: Dialogs, Banners: Banners}
	return result
}
