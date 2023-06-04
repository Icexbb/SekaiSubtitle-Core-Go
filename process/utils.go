package process

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"math"
	"os"
	"reflect"
	"strconv"
	"strings"
)

func Str2int(s string) int {
	res, err := strconv.Atoi(s)
	if err != nil {
		return 0
	} else {
		return res
	}
}
func Str2IntArr(s string) []int {
	var sArr = strings.Split(s, " ")
	var result []int
	for _, v := range sArr {
		res, err := strconv.Atoi(v)
		if err == nil {
			result = append(result, res)
		}
	}
	return result
}
func SplitArr[T any](s []T) ([]T, []T) {
	var odd []T
	var even []T
	for k, v := range s {
		if k%2 == 0 {
			even = append(even, v)
		} else {
			odd = append(odd, v)
		}
	}
	return odd, even
}
func MaxInt(arr []int) int {
	max := math.MinInt
	for _, v := range arr {
		if v > max {
			max = v
		}
	}
	return max
}
func MinInt(arr []int) int {
	min := math.MaxInt
	for _, v := range arr {
		if v < min {
			min = v
		}
	}
	return min
}
func Prod[T int64 | int32 | int | float64 | float32](arr []T) T {
	result := T(1)
	for _, v := range arr {
		result *= v
	}
	return result
}
func CheckErr(err error) {
	if err != nil {
		panic(err)
	}
}
func ArrSplit(arr []string) []string {
	ret := make([]string, 0, len(arr))
	var v string
	for _, v = range arr {
		v = Strip(v)
		if len(v) > 0 {
			ret = append(ret, v)
		}
	}
	return ret
}
func Strip(s string) string {
	v := s
	var con bool
	var wSpace = []string{"\n", " ", "\r", "\t"}
	for {
		con = false
		for _, s := range wSpace {
			v = strings.Trim(v, s)
		}
		for _, s := range wSpace {
			if strings.HasPrefix(v, s) || strings.HasSuffix(v, s) {
				con = true
				break
			}
		}
		if !con {
			break
		}
	}
	return v
}
func Md5(str string, len int) string {
	h := md5.New()
	h.Write([]byte(str))
	r := hex.EncodeToString(h.Sum(nil))
	return r[:len]
}
func Md5Len3(str string) string {
	return Md5(str, 3)
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
func MsToString(ms int) (res string) {
	res = fmt.Sprintf("%02d:%02d:%02d.%02d", (ms/1000)/60/60, (ms/1000)/60%60, ms/1000%60, ms%1000/10)
	return
}
func CheckMaxDistance(arr []image.Point) int {
	var xS []int
	var yS []int
	for _, point := range arr {
		xS = append(xS, point.X)
		yS = append(yS, point.Y)
	}
	return MaxInt([]int{MaxInt(xS) - MinInt(xS), MaxInt(yS) - MinInt(yS)})
}
func Index[T any](v T, array []T) int {
	if n := len(array); array != nil && n != 0 {
		i := 0
		for !reflect.DeepEqual(v, array[i]) {
			i++
		}
		if i != n {
			return i
		}
	}
	return -1
}
func WriteFile(filePath, content string) {
	ext, _ := PathExists(filePath)
	if ext {
		_ = os.Remove(filePath)
	}
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0666)
	CheckErr(err)
	write := bufio.NewWriter(file)
	_, err = write.WriteString(content)
	CheckErr(err)
	err = write.Flush()
	CheckErr(err)
	err = file.Close()
	CheckErr(err)
}
func MapEquals(data1, data2 map[string]string) bool {
	var keySlice []string
	var dataSlice1 []string
	var dataSlice2 []string
	for key, value := range data1 {
		keySlice = append(keySlice, key)
		dataSlice1 = append(dataSlice1, value)
	}
	for _, key := range keySlice {
		if data, ok := data2[key]; ok {
			dataSlice2 = append(dataSlice2, data)
		} else {
			return false
		}
	}
	dataStr1, _ := json.Marshal(dataSlice1)
	dataStr2, _ := json.Marshal(dataSlice2)

	return string(dataStr1) == string(dataStr2)
}
