package main

import (
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/judwhite/go-svc/svc"
	"github.com/mreiferson/go-options"
	"imgserver"
	"imgserver/g"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

func nsqdFlagSet(opts *imgserver.Options) *flag.FlagSet {
	flagSet := flag.NewFlagSet("imgserver", flag.ExitOnError)

	// basic options
	flagSet.Bool("version", false, "print version string")
	flagSet.String("config", "", "path to config file")

	flagSet.String("log-level", "info", "set log verbosity: debug, info, warn, error, or fatal")
	flagSet.Int64("node-id", opts.ID, "unique part for message IDs, (int) in range [0,1024) (default is hash of hostname)")

	flagSet.String("http-address", opts.HTTPAddress, "<addr>:<port> to listen on for HTTP clients")
	flagSet.String("tcp-address", opts.TCPAddress, "<addr>:<port> to listen on for TCP clients")
	flagSet.String("broadcast-address", opts.BroadcastAddress, "address that will be registered with lookupd (defaults to the OS hostname)")

	flagSet.String("upload-path", opts.UploadPath, "<addr>:<port> to listen on for TCP clients")
	flagSet.String("watermark-path", opts.WatermarkPath, "<addr>:<port> to listen on for TCP clients")

	flagSet.Int("receive-timeout", opts.ReceiveTimeout, "duration of time to wait receive request")

	return flagSet
}

type config map[string]interface{}

// Validate settings in the config file, and fatal on errors
func (cfg config) Validate() {
	// special validation/translation
}

type program struct {
	imgserver *imgserver.IMGSERVER
}

func main() {
	prg := &program{}
	if err := svc.Run(prg, syscall.SIGINT, syscall.SIGTERM); err != nil {
		log.Fatal(err)
	}
}

func (p *program) Init(env svc.Environment) error {
	runtime.GOMAXPROCS(runtime.NumCPU())

	if env.IsWindowsService() {
		dir := filepath.Dir(os.Args[0])
		return os.Chdir(dir)
	}
	return nil
}

func (p *program) Start() error {
	opts := imgserver.NewOptions()

	flagSet := nsqdFlagSet(opts)
	flagSet.Parse(os.Args[1:])

	rand.Seed(time.Now().UTC().UnixNano())

	if flagSet.Lookup("version").Value.(flag.Getter).Get().(bool) {
		fmt.Println(g.String("imgserver"))
		os.Exit(0)
	}

	var cfg config
	configFile := flagSet.Lookup("config").Value.String()
	if configFile != "" {
		_, err := toml.DecodeFile(configFile, &cfg)
		if err != nil {
			log.Fatalf("ERROR: failed to load config file %s - %s", configFile, err.Error())
		}
	}
	cfg.Validate()

	options.Resolve(opts, flagSet, cfg)

	imgserver := imgserver.New(opts)

	imgserver.Main()

	p.imgserver = imgserver

	return nil
}

func (p *program) Stop() error {
	if p.imgserver != nil {
		p.imgserver.Exit()
	}
	return nil
}
