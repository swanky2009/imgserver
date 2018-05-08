package imgserver

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"github.com/swanky2009/imgserver/utils"
	"io"
	"net"
	"os"
	"path"
	"strings"
	"time"
)

type tcpServer struct {
	ctx *context
}

func (p *tcpServer) Start() {

	var listener = p.ctx.imgserver.tcpListener

	log.Infof("TCP: listening on %s", listener.Addr())

	timeout := p.ctx.imgserver.getOpts().ReceiveTimeout

	for {
		conn, err := listener.Accept()
		conn.SetDeadline(time.Now().Add(time.Duration(timeout) * time.Second))
		if err != nil {
			continue
		}
		log.Infof("%s - %s", conn.RemoteAddr().String(), "new client connectioned")

		go p.handleConnection(conn)
	}

	log.Infof("TCP: closing %s", listener.Addr())
}

//长连接入口
func (p *tcpServer) handleConnection(conn net.Conn) {
	buffer := make([]byte, 1024*1024+8)
	//声明一个管道用于接收解包的数据
	readerChan := make(chan []byte, 16)
	//声明一个临时缓冲区，用来存储被截断的数据
	tmpBuffer := make([]byte, 0)

	go p.reader(readerChan)

	for {
		n, err := conn.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Warnf("%s - read error : %s", conn.RemoteAddr().String(), err.Error())
				return
			}
		}
		if n == 0 {
			conn.Write([]byte("file recv finished!\r\n"))
			conn.Close()
			log.Debugf("%s - %s", conn.RemoteAddr().String(), "read end")
			return
		}
		tmpBuffer = Unpack(append(tmpBuffer, buffer[:n]...), readerChan)
	}

	log.Infof("%s - %s", conn.RemoteAddr().String(), "one client leave")
}

func (p *tcpServer) reader(readerChan chan []byte) {
	data := make([]byte, 0)
	//receive fileinfo
	data = <-readerChan
	datatype := string(data[0:4])
	buffer := data[8:]
	if datatype != "info" {
		log.Debugf("read datatype: %s", datatype)
		log.Warn("read error:is not received file info")
		return
	}

	var fileinfo FileInfo
	err := json.Unmarshal(buffer, &fileinfo)
	if err != nil {
		log.Warnf("json unmarshal error: %s", err.Error())
		return
	}

	fileext := strings.ToLower(path.Ext(fileinfo.FileName))

	uploadPath := p.ctx.imgserver.getOpts().UploadPath

	filename := uploadPath + utils.GetGuid() + fileext

	fo, err := os.Create(filename)
	if err != nil {
		log.Warnf("os.Create:%s", err.Error())
		return
	}

	defer fo.Close()

	log.Infof("filename : %s filesize: %d", fileinfo.FileName, fileinfo.FileSize)

	for {
		select {
		case data = <-readerChan:
			datatype := string(data[0:4])
			buffer := data[8:]
			if datatype == "data" {
				//write to the file
				_, err = fo.Write(buffer)
				if err != nil {
					log.Warnf("write error:%s", err.Error())
				}
			} else if datatype == "flag" {
				flag := string(buffer)
				if flag == "filerecvend" {
					log.Info("file receive success")

					p.ctx.imgserver.watermarkChan <- filename

					return
				}
			}
		}
	}
}
