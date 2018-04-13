package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
)

type FileInfo struct {
	FileName string
	FileSize int64
}

func main() {
	filename := flag.String("img", "", "file path")

	flag.Parse()

	//filename := "D:\\web\\images\\test3.jpg"

	_, err := os.Stat(*filename)

	CheckError(err)

	fi, err := os.Open(*filename)

	CheckError(err)

	defer fi.Close()

	conn, err := net.Dial("tcp", "localhost:2300")

	CheckError(err)

	defer conn.Close()

	fileinfo, err := fi.Stat()

	//send fileinfo
	fileobj := FileInfo{
		FileName: fileinfo.Name(),
		FileSize: fileinfo.Size(),
	}
	filejson, err := json.Marshal(fileobj)
	if err != nil {
		fmt.Println("error:", err)
	}

	Log("send fileinfo:", string(filejson))

	_, err = conn.Write(Packet("info", filejson))
	if err != nil {
		fmt.Println("conn.Write", err.Error())
	}

	Log("send file start ...")

	t1 := time.Now().UnixNano()

	buff := make([]byte, 1024*1024)
	for {
		n, err := fi.Read(buff)
		if err != nil && err != io.EOF {
			panic(err)
		}
		if n == 0 {
			conn.Write(Packet("flag", []byte("filerecvend")))
			Log("send file end")
			break
		}
		_, err = conn.Write(Packet("data", buff))
		if err != nil {
			fmt.Println(err.Error())
		}
	}
	t2 := time.Now().UnixNano()
	Log("send time:", t2-t1)
}

//封包
func Packet(datatype string, data []byte) []byte {
	l := len(data) + 8
	return append(append([]byte(datatype), IntToBytes(l)...), data...)
}

//整形转换成字节
func IntToBytes(n int) []byte {
	x := int32(n)
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, x)
	return bytesBuffer.Bytes()
}

func CheckError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}

func Log(v ...interface{}) {
	log.Println(v...)
}
