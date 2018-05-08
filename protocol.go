package imgserver

import (
	"github.com/swanky2009/imgserver/utils"
)

type FileInfo struct {
	FileName string `json:"filename"`
	FileSize int64  `json:"filesize"`
}

//解包
func Unpack(buffer []byte, readerChan chan []byte) []byte {
	length := len(buffer)
	if length <= 8 {
		return buffer
	}
	packlen := utils.BytesToInt(buffer[4:9])

	if packlen == 0 {
		return buffer
	}

	if length < packlen {
		return buffer
	}

	if length == packlen {
		readerChan <- buffer
		return make([]byte, 0)
	}
	readerChan <- buffer[:packlen]
	return buffer[packlen:]
}
