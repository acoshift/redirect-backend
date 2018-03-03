package main

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthz)
	mux.Handle("/", &redirectHandler{configs})

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
	configs []*redirectConfig
}

func (h *redirectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := getHost(r)

	for _, cfg := range h.configs {
		if cfg.From == host {
			redirectTo := cfg.To + r.RequestURI
			http.Redirect(w, r, redirectTo, cfg.Status)
			return
		}
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
