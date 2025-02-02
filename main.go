package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"syscall"

	"tailscale.com/tsnet"
)

var (
	userConfigDir, _ = os.UserConfigDir()
	appDir           = filepath.Join(userConfigDir, "tailscale-reverse-proxy")
	authKey          = flag.String("auth-key", os.Getenv("TS_AUTHKEY"), "auth key to use in the tailnet")
	config           = flag.String("config", filepath.Join(appDir, "config.json"), "config file to use in the tailnet")
)

type Config struct {
	Hostname string `json:"hostname"`
	Url      string `json:"url"`
}

func main() {
	flag.Parse()

	data, err := os.ReadFile(*config)
	if err != nil {
		log.Fatal(err)
	}

	type config struct {
		AuthKey string   `json:"auth_key"`
		Servers []Config `json:"servers"`
	}
	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Fatal(err)
	}

	if len(cfg.Servers) == 0 {
		log.Fatal("no config")
	}

	if *authKey != "" {
		cfg.AuthKey = *authKey
	}

	if cfg.AuthKey == "" {
		log.Fatal("auth key is required")
	}

	type Server struct {
		tsnet *tsnet.Server
		ln    net.Listener
	}

	servers := make(map[string]*Server)

	defer func() {
		for _, s := range servers {
			s.ln.Close()
			s.tsnet.Close()
		}
	}()
	for _, c := range cfg.Servers {
		if c.Hostname == "" {
			log.Fatal("hostname is required")
		}

		url, err := url.Parse(c.Url)
		if err != nil {
			log.Fatal(err)
		}

		if _, ok := servers[c.Hostname]; ok {
			log.Fatalf("duplicate hostname %q", c.Hostname)
		}
		s := &tsnet.Server{
			Dir:          path.Join(appDir, c.Hostname),
			Hostname:     c.Hostname,
			AuthKey:      cfg.AuthKey,
			Ephemeral:    true,
			RunWebClient: true,
		}

		ln, err := s.ListenTLS("tcp", ":443")
		if err != nil {
			log.Fatal(err)
		}

		go func() {
			rs := httputil.NewSingleHostReverseProxy(url)
			rs.Transport = &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			}
			err := http.Serve(ln, rs)
			if err != nil {
				log.Fatal(err)
			}
		}()

		servers[c.Hostname] = &Server{
			tsnet: s,
			ln:    ln,
		}
	}

	signChannel := make(chan os.Signal, 1)
	signal.Notify(signChannel, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	<-signChannel
}
