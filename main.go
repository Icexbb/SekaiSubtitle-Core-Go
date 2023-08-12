package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"SekaiSubtitle-Core/process"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"gocv.io/x/gocv"
)

var AppVersion = "v2.0.230812"
var TaskList = make(map[string]*process.Task)
var TaskListMux = new(sync.RWMutex)
var upgrader = websocket.Upgrader{}

type AliveMsg struct {
	Type string `json:"type"`
}

var AliveMsgObj = AliveMsg{Type: "alive"}
var AliveMsgString, _ = json.Marshal(AliveMsgObj)

type Msg struct {
	Type string `json:"type"`
	Data string `json:"data"`
}
type DataNewTask struct {
	Config process.TaskConfig `json:"config"`
	Rac    bool
}

type DataLog struct {
	Id      string `json:"id"`
	Message string `json:"message"`
}

type WsConn struct {
	*websocket.Conn
	Mux sync.RWMutex
}

func (c *WsConn) WriteMessage(msgType int, msgByte []byte) (err error) {
	c.Mux.Lock() // 加锁
	err = c.Conn.WriteMessage(msgType, msgByte)
	c.Mux.Unlock() // 解锁
	return err
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	conn, err := upgrader.Upgrade(w, r, nil)
	wsConn := WsConn{Conn: conn, Mux: sync.RWMutex{}}
	if err != nil {
		log.Print("Error during connection upgrading:", err)
		return
	}
	var stopGo = make(chan int)
	go func() {
		for {
			select {
			case <-stopGo:
				return
			default:
			}
			TaskListMux.Lock()
			for s := range TaskList {
				if TaskList[s] != nil {
					select {
					case logMsg := <-TaskList[s].LogChan:
						l, _ := json.Marshal(logMsg)
						m, _ := json.Marshal(DataLog{Id: s, Message: string(l)})
						b, _ := json.Marshal(Msg{Type: "log", Data: string(m)})
						err = wsConn.WriteMessage(websocket.TextMessage, b)
						if logMsg.Type == "string" {
							log.Printf("Task %s: %s\n", TaskList[s].Id, logMsg.Data)
						}
						if err != nil {
							log.Print("Error during Log Transfer:", err)
						}
					default:
						// PASS
					}
				}

			}
			TaskListMux.Unlock()
		}
	}() // Send Log
	go func() {
		var taskStatusLast string
		for {
			select {
			case <-stopGo:
				return
			default:
			}
			var taskStatus = make(map[string]string)
			TaskListMux.Lock()
			for s := range TaskList {
				if TaskList[s] != nil {
					if TaskList[s].Processing {
						taskStatus[TaskList[s].Id] = "processing"
					} else {
						taskStatus[TaskList[s].Id] = "idle"
					}
				}
			}
			TaskListMux.Unlock()
			taskStatusString, _ := json.Marshal(taskStatus)
			if string(taskStatusString) != taskStatusLast {
				m, _ := json.Marshal(Msg{Type: "tasks", Data: string(taskStatusString)})
				taskStatusLast = string(taskStatusString)
				err = wsConn.WriteMessage(websocket.TextMessage, m)
				log.Println("Task Status Changed", string(taskStatusString))
				if err != nil {
					log.Println("Error during message writing:", err)
				}
			}
		}
	}() // Send Task Info
	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error during message reading:", err)
			log.Println("Error Messages:", string(msgBytes))
			break
		}
		var msg Msg
		err = json.Unmarshal(msgBytes, &msg)
		if err != nil {
			log.Println("Error during message unmarshal:", err)
			log.Println("Error Messages:", string(msgBytes))
			break
		}
		go func() {
			switch msg.Type {
			case "new":
				log.Println("Received New Task Request")
				var msgData = DataNewTask{}
				err = json.Unmarshal([]byte(msg.Data), &msgData)
				if err != nil {
					break
				}
				task := process.NewTask(msgData.Config)
				TaskListMux.Lock()
				TaskList[task.Id] = &task
				TaskListMux.Unlock()
				if msgData.Rac {
					go task.Run()
				}
				log.Printf("New Task %s Created\n", task.Id)
			case "start":
				taskId := msg.Data
				log.Println("Received Task Start Request for " + taskId)
				TaskListMux.Lock()
				for s := range TaskList {
					if TaskList[s] != nil {
						if TaskList[s].Id == taskId {
							if !TaskList[s].Processing {
								go TaskList[s].Run()
							}
							break
						}
					}
				}
				TaskListMux.Unlock()
			case "stop":
				taskId := msg.Data
				log.Println("Received Task Stop Request for " + taskId)
				TaskListMux.Lock()
				for s := range TaskList {
					if TaskList[s] != nil {
						if TaskList[s].Id == taskId && TaskList[s].Processing {
							go TaskList[s].Stop()
							break
						}
					}
				}
				TaskListMux.Unlock()
			case "delete":
				taskId := msg.Data
				log.Println("Received Task Delete Request for " + taskId)
				exist := false
				TaskListMux.Lock()
				for s := range TaskList {
					if TaskList[s] != nil {
						if TaskList[s].Id == taskId {
							exist = true
							if TaskList[s].Processing {
								TaskList[s].Stop()
							}
							break
						}
					}
				}
				TaskListMux.Unlock()
				if exist {
					TaskListMux.Lock()
					TaskList[taskId] = nil
					TaskListMux.Unlock()
				}

			case "reload":
				var config process.TaskConfig
				taskId := msg.Data
				exist := false
				log.Println("Received Task Reload Request for " + taskId)
				TaskListMux.Lock()
				for s := range TaskList {
					if TaskList[s] != nil {
						if TaskList[s].Id == taskId {
							exist = true
							break
						}
					}
				}
				TaskListMux.Unlock()
				if exist {
					TaskListMux.Lock()
					if TaskList[taskId].Processing {
						TaskList[taskId].Stop()
					}
					config = TaskList[taskId].Config
					TaskList[taskId] = nil
					newTask := process.NewTask(config)
					TaskList[newTask.Id] = &newTask
					TaskListMux.Unlock()
				}
			default:
				err = wsConn.WriteMessage(websocket.TextMessage, AliveMsgString)
				if err != nil {
					log.Println("Error during sending alive:", err)
				}
			}
		}()
	}
	_ = conn.Close()
	close(stopGo)
	os.Exit(0)
}

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
		exists := process.FileExist(vf)
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

func taskConfigHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" && r.ParseForm() == nil {
		vf := r.FormValue("id")
		log.Printf("Get Task Info %s\n", vf)
		type resp struct {
			Success bool   `json:"success"`
			Data    string `json:"data"`
		}
		for s, task := range TaskList {
			if s == vf {
				c, _ := json.Marshal(task.Config)
				r, _ := json.Marshal(resp{Success: true, Data: string(c)})
				_, err := w.Write(r)
				if err != nil {
					log.Println("Error during message writing:", err)
				}
				return
			}
		}
		r, _ := json.Marshal(resp{Success: false, Data: "Task Does Not Exists"})
		w.WriteHeader(404)
		_, err := w.Write(r)
		if err != nil {
			log.Println("Error during message writing:", err)
			return
		}
	}
}

func serve(port int) {
	log.Printf("Sekai Subtitle Core %s Started", AppVersion)
	log.Printf("Serve on localhost:%d\n", port)
	router := mux.NewRouter()
	router.HandleFunc("/", wsHandler)
	router.HandleFunc("/video", videoInfoHandler)
	router.HandleFunc("/task", taskConfigHandler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", port), router))
}
func test() {
	fmt.Println("Nothing Here")
	// var testConfig = process.TaskConfig{
	// 	VideoFile:     "E:\\Project Sekai\\test\\Connect_live_mmj_01.mp4",
	// 	DataFile:      []string{"E:\\Project Sekai\\test\\Connect_live_mmj_01.pjs.txt"},
	// 	OutputPath:    "E:\\Project Sekai\\test\\Connect_live_mmj_01.ass",
	// 	Overwrite:     true,
	// 	Font:          "",
	// 	VideoOnly:     false,
	// 	Staff:         nil,
	// 	TyperInterval: [2]int{50, 80},
	// 	Duration:      [2]int{0, 0},
	// 	Debug:         true,
	// }
	// var task = process.NewTask(testConfig)
	// go task.Run()
	//
	// for {
	// 	select {
	// 	case logMsg := <-task.LogChan:
	// 		if logMsg.Type == "string" {
	// 			log.Printf("Task %s: %s\n", task.Id, logMsg.Data)
	// 			if logMsg.Data == "[Finish] Process Finished" {
	// 				os.Exit(0)
	// 			}
	// 		} else {
	// 			// log.Printf("Task %s: %s\n", task.Id, logMsg.Data)
	// 		}
	// 	default:
	// 		// PASS
	// 	}
	// }
}
func main() {
	var printVersion bool
	var testRun bool
	var port int
	flag.BoolVar(&printVersion, "v", false, "Print Core Version")
	flag.BoolVar(&testRun, "t", false, "run test()")
	flag.IntVar(&port, "p", 50000, "Select Core Port")
	flag.Parse()
	if printVersion {
		fmt.Println(AppVersion)
	} else if testRun {
		test()
	} else {
		serve(port)
	}
}
