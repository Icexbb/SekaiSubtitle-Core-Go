package main

import (
	"SekaiSubtitle-Core/process"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"gocv.io/x/gocv"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

var Version = "v2.0.230601"
var TaskList = map[string]process.Task{}
var upgrader = websocket.Upgrader{}

func videoInfoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" && r.ParseForm() == nil {
		// 接收参数
		vf := r.FormValue("video_file")
		log.Printf("Get VideoInfo %s\n", vf)
		type vInfo struct {
			FrameHeight int     `json:"frameHeight"`
			FrameWidth  int     `json:"frameWidth"`
			FrameCount  int     `json:"frameCount"`
			VideoFps    float64 `json:"videoFps"`
		}
		type resp struct {
			Success bool   `json:"success"`
			Data    string `json:"data"`
		}
		exists, _ := process.PathExists(vf)
		if !exists {
			r, _ := json.Marshal(resp{
				Success: false,
				Data:    "Video Does Not Exists",
			})
			w.WriteHeader(404)
			_, err := w.Write(r)
			if err != nil {
				log.Println("Error during message writing:", err)
				return
			}
		}
		vc, _ := gocv.VideoCaptureFile(vf)
		v := vInfo{
			FrameHeight: int(vc.Get(gocv.VideoCaptureFrameHeight)),
			FrameWidth:  int(vc.Get(gocv.VideoCaptureFrameWidth)),
			FrameCount:  int(vc.Get(gocv.VideoCaptureFrameCount)),
			VideoFps:    vc.Get(gocv.VideoCaptureFPS),
		}
		vs, _ := json.Marshal(v)
		_, err := w.Write(vs)
		if err != nil {
			log.Println("Error during message writing:", err)
			return
		}
	}
}

func subtitleStatusHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("Error during connection upgradation:", err)
		return
	}
	vars := mux.Vars(r)
	tid, ok := vars["taskId"]
	if !ok {
		fmt.Println("tid is missing in parameters")
		return
	}
	var task process.Task
	for s := range TaskList {
		if s == tid {
			task = TaskList[s]
			break
		}
	}
	if len(task.Id) == 0 {
		_ = conn.WriteMessage(websocket.CloseMessage, nil)
	} else {
		type Msg struct {
			Type string `json:"type"`
			Data string `json:"data"`
		}
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("Error during message reading:", err)
				break
			}
			var msg = Msg{}
			err = json.Unmarshal(message, &msg)
			//log.Printf("Received: %s", message)
			res, err := strconv.Atoi(msg.Data)
			if err != nil {
				respData, _ := json.Marshal(task.Config)
				resp := Msg{
					Type: "config",
					Data: string(respData),
				}
				respText, _ := json.Marshal(resp)
				e := conn.WriteMessage(websocket.TextMessage, respText)
				if e != nil {
					log.Println("Error during message writing:", e)
					break
				}
			} else {
				logs := task.Logs[res:]
				var resp Msg
				if len(logs) > 0 {
					respData, _ := json.Marshal(logs)
					resp = Msg{
						Type: "config",
						Data: string(respData),
					}
				} else {
					resp = Msg{Type: "alive"}
				}
				respText, _ := json.Marshal(resp)
				e := conn.WriteMessage(websocket.TextMessage, respText)
				if e != nil {
					log.Println("Error during message writing:", e)
					break
				}
			}

		}
	}

	_ = conn.Close()
}

func subtitleTasksHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("Error during connection upgradation:", err)
		return
	}
	type taskOperateMsg struct {
		Type string `json:"type"`
		Data string `json:"data"`
		Rac  bool   `json:"runAfterCreate"`
	}
	var taskStatusLast map[string]string

	for {
		_, message, err := conn.ReadMessage()
		var msg taskOperateMsg
		err = json.Unmarshal(message, &msg)
		if err != nil {
			log.Println("Error during message reading:", err)
			break
		}
		switch msg.Type {
		case "alive":
			var taskStatus map[string]string
			for _, task := range TaskList {
				if task.Processing {
					taskStatus[task.Id] = "processing"
				} else {
					taskStatus[task.Id] = "idle"
				}
			}
			var sendData []byte
			if process.MapEquals(taskStatus, taskStatusLast) {
				sendData, _ = json.Marshal(taskOperateMsg{Type: "alive"})
			} else {
				taskStatusString, _ := json.Marshal(taskStatus)
				sendData, _ = json.Marshal(taskOperateMsg{Type: "tasks", Data: string(taskStatusString)})
			}
			err = conn.WriteMessage(websocket.TextMessage, sendData)
			if err != nil {
				log.Println("Error during message writing:", err)
				break
			}
			break
		case "new":
			var config = process.TaskConfig{}
			err := json.Unmarshal([]byte(msg.Data), &config)
			if err != nil {
				break
			}
			task := process.NewTask(config)
			if msg.Rac {
				go task.Run()
			}
			TaskList[task.Id] = task
			break
		case "start":
			taskId := msg.Data
			for s := range TaskList {
				if s == taskId {
					task := TaskList[s]
					if !task.Processing {
						go task.Run()
					}
					break
				}
			}
			break
		case "stop":
			taskId := msg.Data
			for s := range TaskList {
				if s == taskId {
					task := TaskList[s]
					if task.Processing {
						go task.Stop()
					}
					break
				}
			}
			break
		case "delete":
			taskId := msg.Data
			exist := false
			for s := range TaskList {
				if s == taskId {
					exist = true
					task := TaskList[s]
					if task.Processing {
						task.Stop()
					}
					break
				}
			}
			if exist {
				delete(TaskList, taskId)
			}
			break
		case "reload":
			taskId := msg.Data
			exist := false
			config := process.TaskConfig{}
			for s := range TaskList {
				if s == taskId {
					exist = true
					task := TaskList[s]
					if task.Processing {
						task.Stop()
					}
					config = task.Config
					break
				}
			}
			if exist {
				delete(TaskList, taskId)
				task := process.NewTask(config)
				if msg.Rac {
					go task.Run()
				}
				TaskList[task.Id] = task
			}
			break
		default:
			return
		}
		log.Println("Connection Alive End")
		err = conn.WriteMessage(websocket.CloseMessage, nil)
		if err != nil {
			log.Println("Error during message writing:", err)
			break
		}
	}
	_ = conn.Close()
}
func wsAliveHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("Error during connection upgrading:", err)
		return
	}

	type aliveMessage struct {
		Type string `json:"type"`
	}
	// The event loop
	var v = aliveMessage{Type: "alive"}
	vT, _ := json.Marshal(v)

	for {
		err = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		if err != nil {
			break
		}
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error during message reading:", err)
			break
		}
		var msgUnmarshal aliveMessage
		err = json.Unmarshal(msg, &msgUnmarshal)
		if err != nil {
			log.Println("Error during message reading:", err)
			break
		}
		//log.Printf("Received: %v", msgUnmarshal)

		if msgUnmarshal.Type != "alive" {
			log.Println("Connection Alive End")
			err = conn.WriteMessage(websocket.CloseMessage, msg)
			if err != nil {
				log.Println("Error during message writing:", err)
			}
			break
		} else {
			err = conn.WriteMessage(websocket.TextMessage, vT)
			if err != nil {
				log.Println("Error during message writing:", err)
				break
			}
		}
	}
	_ = conn.Close()
	os.Exit(0)
	return
}

func main() {
	var printVersion bool
	var port int
	flag.BoolVar(&printVersion, "v", false, "Print Core Version")
	flag.IntVar(&port, "p", 50000, "Select Core Port")
	flag.Parse()
	if printVersion {
		fmt.Println(Version)
	} else {
		log.Println("Sekai Subtitle Core Started")
		log.Printf("Serve on localhost:%d", port)
		http.HandleFunc("/alive", wsAliveHandler)
		http.HandleFunc("/subtitle/tasks", subtitleTasksHandler)
		http.HandleFunc("/subtitle/status/{taskId}", subtitleStatusHandler)
		http.HandleFunc("/subtitle/videoInfo", videoInfoHandler)
		log.Fatal(http.ListenAndServe(fmt.Sprintf("localhost:%d", port), nil))
	}

	//c := process.TaskConfig{
	//	VideoFile:     "/Users/xbb/Remote/2nd_countdown_04.mp4",
	//	JsonFile:      "/Users/xbb/Remote/2nd_countdown_04.json",
	//	TranslateFile: "",
	//	OutputPath:    "/Users/xbb/Remote/2nd_countdown_04.ass",
	//	Overwrite:     true,
	//	Font:          "",
	//	VideoOnly:     false,
	//	Staff:         nil,
	//	TyperInterval: [2]int{50, 80},
	//	Duration:      [2]int{},
	//	Debug:         false,
	//}
	//logChan := make(chan process.Log)
	//task := process.Task{
	//	Config:     c,
	//	LogChannel: logChan,
	//}
	//go task.Run()
	//for {
	//	select {
	//	case l := <-logChan:
	//		if l.Type == "string" {
	//			fmt.Printf("%v: %v\n", l.Type, l.Data)
	//		}
	//		if l.Data == "Process Finished" {
	//			return
	//		}
	//	default:
	//		//
	//	}
	//}
}
