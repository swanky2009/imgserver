package imgserver

import (
	"crypto/md5"
	log "github.com/Sirupsen/logrus"
	"hash/crc32"
	"io"
	"os"
)

type Options struct {
	// basic options
	ID               int64  `flag:"node-id" cfg:"id"`
	BroadcastAddress string `flag:"broadcast-address"`

	LogLevel string `flag:"log-level"` //[info, debug, warn]

	TCPAddress  string `flag:"tcp-address"`
	HTTPAddress string `flag:"http-address"`

	UploadPath    string `flag:"upload-path"`
	WatermarkPath string `flag:"watermark-path"`

	ReceiveTimeout int `flag:"receive-timeout"`
}

func NewOptions() *Options {
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err.Error())
	}

	h := md5.New()
	io.WriteString(h, hostname)
	defaultID := int64(crc32.ChecksumIEEE(h.Sum(nil)) % 1024)

	return &Options{
		ID:               defaultID,
		BroadcastAddress: hostname,
		LogLevel:         "debug",
		TCPAddress:       "0.0.0.0:2300",
		HTTPAddress:      "0.0.0.0:2301",
		UploadPath:       "d:\\web\\upload\\",
		WatermarkPath:    "watermark.png",
		ReceiveTimeout:   120,
	}
}
