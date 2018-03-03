package main

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v2"
)

var (
	config = flag.String("config", "config.yaml", "Config file")
	addr   = flag.String("addr", ":8080", "Address to listen")
)

func main() {
	flag.Parse()

	configs, err := loadConfig(*config)
	if err != nil {
		log.Fatal("can not load config; ", err)
	}

	h := &redirectHandler{}
	h.Set(configs)

	// watch config file
	configWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer configWatcher.Close()

	go func() {
		for {
			select {
			case event := <-configWatcher.Events:
				time.Sleep(time.Second)
				configWatcher.Add(*config)
				if event.Op&fsnotify.Write == fsnotify.Write ||
					event.Op&fsnotify.Rename == fsnotify.Rename {

					configs, err := loadConfig(*config)
					if err != nil {
						log.Println("config reload error; ", err)
						continue
					}
					h.Set(configs)
					log.Println("config reloaded")
				}
			case err := <-configWatcher.Errors:
				log.Println("watcher error; ", err)
			}
		}
	}()

	err = configWatcher.Add(*config)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthz)
	mux.Handle("/", h)

	srv := http.Server{
		Addr:    *addr,
		Handler: mux,
	}

	go func() {
		log.Printf("start server on %s\n", *addr)
		err = srv.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Fatal("can not start server; ", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	err = srv.Shutdown(ctx)
	if err != nil {
		log.Println("can not shutdown server; ", err)
	}
}

type redirectConfig struct {
	From   string `yaml:"from"`
	To     string `yaml:"to"`
	Status int    `yaml:"status"`
}

func loadConfig(filename string) ([]*redirectConfig, error) {
	var configs []*redirectConfig
	bs, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(bs, &configs)
	if err != nil {
		return nil, err
	}
	return configs, nil
}

type redirectHandler struct {
	m       sync.RWMutex
	configs []*redirectConfig
}

func (h *redirectHandler) Set(cfgs []*redirectConfig) {
	h.m.Lock()
	h.configs = cfgs
	h.m.Unlock()
}

func (h *redirectHandler) Get(host string) *redirectConfig {
	h.m.RLock()
	defer h.m.RUnlock()

	for _, cfg := range h.configs {
		if cfg.From == host {
			return cfg
		}
	}
	return nil
}

func (h *redirectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := getHost(r)

	cfg := h.Get(host)
	if cfg != nil {
		redirectTo := cfg.To + r.RequestURI
		http.Redirect(w, r, redirectTo, cfg.Status)
		return
	}

	http.NotFound(w, r)
}

// getHost gets real host from request
func getHost(r *http.Request) string {
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	return host
}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
