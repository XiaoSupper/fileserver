package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Config struct {
	ExecFilePath       string
	SaveResultFilePath string
}
type JsonStruct struct {
}

var Configdata Config

var ticker = time.NewTicker(10 * time.Second)
var restartOK string = "no"

func init() {
	JsonParse := NewJsonStruct()
	//下面使用的是相对路径，config.json文件和main.go文件处于同一目录下
	JsonParse.Load("./config.json", &Configdata)
	fmt.Printf("ExecFilePath = %d", Configdata.ExecFilePath)
}
func main() {
	var wg sync.WaitGroup
	wg.Add(2)
	go timing(&wg)
	go fileSystem(&wg)
	wg.Wait()
}

func NewJsonStruct() *JsonStruct {
	return &JsonStruct{}
}

func (jst *JsonStruct) Load(filename string, v interface{}) {
	//ReadFile函数会读取文件的全部内容，并将结果以[]byte类型返回
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	//读取的数据为json格式，需要进行解码
	err = json.Unmarshal(data, v)
	if err != nil {
		return
	}
}
func rootHandle(w http.ResponseWriter, r *http.Request) {
	fmt.Println("404 Not Found")

}

func fileSystem(wg *sync.WaitGroup) {
	mux := http.NewServeMux()
	files := http.FileServer(http.Dir(Configdata.SaveResultFilePath))

	mux.Handle("/downfiles/", http.StripPrefix("/downfiles/", files))
	server := http.Server{
		Addr:    "localhost:8081",
		Handler: mux,
	}
	mux.HandleFunc("/", rootHandle)
	server.ListenAndServe()
	wg.Done()
	fmt.Println("fileSystem Done\n")
}
func timing(wg *sync.WaitGroup) {
	//定时器，10秒钟执行一次
	// ticker := time.NewTicker(10 * time.Second)
	tickerRestart := time.NewTicker(300 * time.Second)
	var startRreq int = 1
	for {
		client := &http.Client{
			Transport: &http.Transport{
				Dial: func(netw, addr string) (net.Conn, error) {
					conn, err := net.DialTimeout(netw, addr, time.Second*10) //设置建立连接超时
					if err != nil {
						return nil, err
					}
					conn.SetDeadline(time.Now().Add(time.Second * 2)) //设置发送接收数据超时
					return conn, nil
				},
				ResponseHeaderTimeout: time.Second * 2,
			},
		}

		rsp, err := client.Get("http://127.0.0.1:8001/isactive")
		if err != nil {
			fmt.Println("err info: ", err.Error())
			//查找 包含
			var isCon1 bool = strings.Contains(err.Error(), "timeout")
			var isCon2 bool = strings.Contains(err.Error(), "No connection could be made")
			var isCon3 bool = strings.Contains(err.Error(), "closed by the remote host")
			//var isCon4 bool = strings.Contains(err.Error(), "EOF")

			if (isCon1 || isCon2 || isCon3) && restartOK == "no" {
				if startRreq == 1 {
					command := "start " + Configdata.ExecFilePath
					cmdRestart := exec.Command("cmd", "/c", command)
					err := cmdRestart.Run()
					if err != nil {
						fmt.Printf("Execute Shell 1: failed with error:%s", err.Error())
						continue
					}
					startRreq = startRreq + 1
				} else {
					restartOK = "restarting"
					startRreq = startRreq + 1
					ticker = time.NewTicker(60 * time.Second)
					restartOK = restart()
				}

			}

		} else {
			restartOK = "no"
			defer rsp.Body.Close()
			body, err := ioutil.ReadAll(rsp.Body)

			if err != nil {
				fmt.Println("myHttpGet error is ", err)
				return
			}
			fmt.Println("response statuscode is ", rsp.StatusCode, "\nhead[name]=", rsp.Header["Name"], "\nbody is ", string(body))
		}

		time := <-ticker.C
		fmt.Println("定时器1====>", time.String())
		if restartOK == "restarting" {
			time := <-tickerRestart.C
			fmt.Println("定时器2====>", time.String())
		}
	}
	wg.Done()
	fmt.Println("timing() Done\n")
}

func restart() string {

	var wgRestart sync.WaitGroup
	wgRestart.Add(1)
	go killProcess(&wgRestart)
	wgRestart.Wait()

	fmt.Println("重启BCC\n")
	command := "start " + Configdata.ExecFilePath
	cmdRestart := exec.Command("cmd", "/c", command)
	err := cmdRestart.Run()
	if err != nil {
		fmt.Printf("重启失败: failed with error:%s", err.Error())
		return "no"
	}
	ticker = time.NewTicker(10 * time.Second)
	return "ok"
}
func killProcess(wgRestart *sync.WaitGroup) {
	fmt.Println("enter killProcess()\n")
	//"netstat -aon|findstr "8001" "
	command := "/killProcess.bat"
	cmd := exec.Command(command)
	var out bytes.Buffer
	cmd.Stdout = &out
	time.Sleep(10 * time.Microsecond)
	err := cmd.Run()
	fmt.Printf("Output : %q end here\n", out.String())
	if err != nil {
		fmt.Printf("Execute Shell findstr port: failed with error:%s", err.Error())
		return
	}

	cmdOutStr := out.String()

	sep1 := "\r\n"
	lines := strings.Split(cmdOutStr, sep1)
	for _, line := range lines {
		fmt.Printf("%s\n", line)
	}
	fmt.Printf("一共杀多少进程 :%d \n\n", len(lines))
	for _, line := range lines {
		fmt.Printf("一行:  %s\n", line)
		array := strings.Fields(line)

		if len(array)-2 < 0 || strings.HasPrefix(array[len(array)-2], "FIN_WAIT") || !strings.HasPrefix(array[0], "TCP") {
			fmt.Println("不处理\n")
		} else {
			pid, err := strconv.Atoi(array[len(array)-1])
			fmt.Printf("端口号:%d\n", pid)
			if err != nil {
				fmt.Printf("string to int failed:%s", err.Error())
				return
			}
			Kill(pid)
		}

	}
	wgRestart.Done()

}

func Kill(pids int) {
	pro, err := os.FindProcess(pids)
	if err != nil {
		return
	}
	pro.Kill()
}
