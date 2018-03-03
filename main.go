package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"

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

	log.Printf("start server on %s\n", *addr)
	err = http.ListenAndServe(*addr, &redirectHandler{configs})
	if err != nil {
		log.Fatal("can not start server; ", err)
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
