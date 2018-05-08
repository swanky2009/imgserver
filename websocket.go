package imgserver

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"github.com/swanky2009/imgserver/utils"
	"html/template"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type websocketServer struct {
	ctx *context
}

const (
	// Time allowed to write the file to the client.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the client.
	pongWait = 60 * time.Second

	// Send pings to client with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Poll file for changes with this period.
	filePeriod = 10 * time.Second
)

var (
	homeTempl = template.Must(template.New("").Parse(homeHTML))

	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024 + 8,
		WriteBufferSize: 1024,
	}
)

func (p *websocketServer) Start() {

	var listener = p.ctx.imgserver.httpListener

	log.Infof("HTTP: listening on %s", listener.Addr())

	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", p.serveWs)
	http.HandleFunc("/websocket_example", serveExample)
	http.HandleFunc("/upload", p.serverUpload)
	http.HandleFunc("/http_example", servehttpExample)

	if err := http.Serve(listener, nil); err != nil {
		log.Warnf("httpserver (%s) start failed - %s", p.ctx.imgserver.getOpts().HTTPAddress, err)
		os.Exit(1)
	}
}

func (p *websocketServer) serveWs(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Warnf(err.Error())
		}
		return
	}

	defer ws.Close()

	log.Infof("%s - %s", r.RemoteAddr, "new websocket client connectioned")

	recivedChan := make(chan bool)

	go writer(ws, recivedChan)

	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error { ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	//声明一个管道用于接收解包的数据
	readerChan := make(chan []byte, 16)
	//声明一个临时缓冲区，用来存储被截断的数据
	tmpBuffer := make([]byte, 0)

	go p.reader(readerChan, recivedChan)

	for {
		_, buffer, err := ws.ReadMessage()
		if err != nil {
			log.Warnf("%s - read error : %s", r.RemoteAddr, err.Error())
			return
		}
		n := len(buffer)
		log.Debugf("%s - read length : %d", r.RemoteAddr, n)
		log.Debugf("%s - read type : %s ", r.RemoteAddr, string(buffer[0:4]))
		log.Debugf("%s - data length : %d", r.RemoteAddr, utils.BytesToInt(buffer[4:9]))
		if n == 0 {
			return
		}
		tmpBuffer = Unpack(append(tmpBuffer, buffer[:n]...), readerChan)
	}
}

func (p *websocketServer) reader(readerChan chan []byte, recivedChan chan bool) {
	data := make([]byte, 0)

	uploadPath := p.ctx.imgserver.getOpts().UploadPath

	var filename string
	var fo *os.File
	var err error

	for {
		select {
		case data = <-readerChan:
			datatype := string(data[0:4])
			buffer := data[8:]
			if datatype == "info" {
				//receive fileinfo
				var fileinfo FileInfo
				err = json.Unmarshal(buffer, &fileinfo)
				if err != nil {
					log.Warnf("json unmarshal error: %s", err.Error())
					return
				}
				log.Infof("filename : %s filesize: %d", fileinfo.FileName, fileinfo.FileSize)

				fileext := strings.ToLower(path.Ext(fileinfo.FileName))
				filename = uploadPath + utils.GetGuid() + fileext
				fo, err = os.Create(filename)
				if err != nil {
					log.Warnf("os.Create:%s", err.Error())
				}
			} else if datatype == "data" {
				//write to the file
				_, err = fo.Write(buffer)
				if err != nil {
					log.Warnf("write error:%s", err.Error())
				}
			} else if datatype == "flag" {
				//end the receiving file
				flag := string(buffer)
				if flag == "filerecvend" {
					log.Info("file received success")

					fo.Close()

					recivedChan <- true

					p.ctx.imgserver.watermarkChan <- filename
				}
			}
		}
	}
}

func writer(ws *websocket.Conn, recivedChan chan bool) {
	pingTicker := time.NewTicker(pingPeriod)
	defer func() {
		pingTicker.Stop()
		ws.Close()
	}()
	for {
		select {
		case <-recivedChan:
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ws.WriteMessage(websocket.TextMessage, []byte("file send finished!")); err != nil {
				return
			}
		case <-pingTicker.C:
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	var v = struct {
		Host string
	}{
		r.Host,
	}
	homeTempl.Execute(w, &v)
}

const homeHTML = `<!DOCTYPE html>
<html lang="en">
    <head>
        <title>WebSocket Service</title>
    </head>
    <body>
        <div>
        	<h2>It's a websocket image upload service</h2>
        	<div>
        		<p>the connection address is ws://{{.Host}}/ws</p>
				<p><a href="/websocket_example">websocket upload example</a></p>
				<p><a href="/http_example">http upload example</a></p>
        	</div>
        </div>        
    </body>
</html>
`

func serveExample(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/websocket_example" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	var v = struct {
		Host string
	}{
		r.Host,
	}
	currentDir := utils.GetCurrentDir()
	tempfile := utils.GetParentDir(currentDir) + "\\client\\websocket_send_example.html"
	exampleTempl, err := template.ParseFiles(tempfile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusExpectationFailed)
		return
	}
	exampleTempl.Execute(w, &v)
}

func (p *websocketServer) serverUpload(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/upload" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "multipart/form-data; charset=utf-8")

	if r.Method == "POST" {
		file, fileHeader, err := r.FormFile("imagefile")
		if err != nil {
			log.Warnf("receive file error : %s", err.Error())
			http.Error(w, err.Error(), 500)
			return
		}
		defer file.Close()

		file_name := fileHeader.Filename

		log.Info("receive filename" + file_name)

		uploadPath := p.ctx.imgserver.getOpts().UploadPath
		fileext := strings.ToLower(path.Ext(file_name))
		filename := uploadPath + utils.GetGuid() + fileext
		f, err := os.Create(filename)
		defer f.Close()
		io.Copy(f, file)

		log.Info("file received success")

		p.ctx.imgserver.watermarkChan <- filename

		return
	}
	return
}

func servehttpExample(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/http_example" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	var v = struct {
		Host string
	}{
		r.Host,
	}
	currentDir := utils.GetCurrentDir()
	tempfile := utils.GetParentDir(currentDir) + "\\client\\http_send_example.html"
	exampleTempl, err := template.ParseFiles(tempfile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusExpectationFailed)
		return
	}
	exampleTempl.Execute(w, &v)
}
