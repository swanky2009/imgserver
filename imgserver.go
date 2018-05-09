package imgserver

import (
	log "github.com/Sirupsen/logrus"
	"github.com/swanky2009/imgserver/utils"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

type IMGSERVER struct {
	sync.RWMutex

	opts atomic.Value

	tcpListener  net.Listener
	httpListener net.Listener

	waitGroup utils.WaitGroupWrapper

	watermarkChan chan string
}

func New(opts *Options) *IMGSERVER {

	utils.InitLog(opts.LogLevel)

	n := &IMGSERVER{
		watermarkChan: make(chan string, 10),
	}
	n.swapOpts(opts)
	return n
}

func (n *IMGSERVER) getOpts() *Options {
	return n.opts.Load().(*Options)
}

func (n *IMGSERVER) swapOpts(opts *Options) {
	n.opts.Store(opts)
}

func (n *IMGSERVER) RealTCPAddr() *net.TCPAddr {
	n.RLock()
	defer n.RUnlock()
	return n.tcpListener.Addr().(*net.TCPAddr)
}

func (n *IMGSERVER) RealHTTPAddr() *net.TCPAddr {
	n.RLock()
	defer n.RUnlock()
	return n.httpListener.Addr().(*net.TCPAddr)
}

func (n *IMGSERVER) Main() {

	ctx := &context{n}

	tcpListener, err := net.Listen("tcp", n.getOpts().TCPAddress)
	if err != nil {
		log.Warnf("listen (%s) failed - %s", n.getOpts().TCPAddress, err)
		os.Exit(1)
	}
	n.Lock()
	n.tcpListener = tcpListener
	n.Unlock()
	tcpServer := &tcpServer{ctx: ctx}
	n.waitGroup.Wrap(func() {
		tcpServer.Start()
	})

	var httpListener net.Listener
	httpListener, err = net.Listen("tcp", n.getOpts().HTTPAddress)
	if err != nil {
		log.Warnf("listen (%s) failed - %s", n.getOpts().HTTPAddress, err)
		os.Exit(1)
	}
	n.Lock()
	n.httpListener = httpListener
	n.Unlock()
	websocketServer := &websocketServer{ctx: ctx}
	n.waitGroup.Wrap(func() {
		websocketServer.Start()
	})
	n.waitGroup.Wrap(func() { n.Watermark() })
}

func (n *IMGSERVER) Exit() {
	if n.tcpListener != nil {
		n.tcpListener.Close()
	}

	if n.httpListener != nil {
		n.httpListener.Close()
	}
	close(n.watermarkChan)
	n.waitGroup.Wait()
}

func (n *IMGSERVER) Watermark() {
	watermarkpath := filepath.Join(utils.GetCurrentDir(), n.getOpts().WatermarkPath)
	for {
		select {
		case filename := <-n.watermarkChan:
			err := Watermark(filename, watermarkpath)
			if err != nil {
				log.Warnf("watermark error: %s", err.Error())
			}
		}
	}
}
