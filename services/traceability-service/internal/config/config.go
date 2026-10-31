// Package config validates runtime boundaries before any listener is opened.
package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Mode              string
	ListenAddress     string
	DatabaseURL       string
	PublicWebURL      string
	IsolatedContainer bool
	ShutdownTimeout   time.Duration
	VideoHosts        []string
}

// Load takes an injectable environment reader; it never includes secret values in errors.
func Load(getenv func(string) string) (Config, error) {
	c := Config{
		Mode:            getenv("APP_MODE"),
		ListenAddress:   getenv("HTTP_ADDR"),
		DatabaseURL:     getenv("DATABASE_URL"),
		PublicWebURL:    getenv("PUBLIC_WEB_BASE_URL"),
		ShutdownTimeout: 10 * time.Second,
	}
	if c.Mode == "" {
		return Config{}, errors.New("APP_MODE is required; choose sandbox or integration explicitly")
	}
	if c.Mode != "sandbox" && c.Mode != "integration" {
		return Config{}, errors.New("APP_MODE must be sandbox or integration; production is not enabled")
	}
	if c.ListenAddress == "" {
		c.ListenAddress = "127.0.0.1:8003"
	}
	host, port, err := net.SplitHostPort(c.ListenAddress)
	if err != nil {
		return Config{}, errors.New("HTTP_ADDR must be a numeric IP address and port")
	}
	ip := net.ParseIP(host)
	p, err := strconv.Atoi(port)
	if ip == nil || err != nil || p < 1 || p > 65535 {
		return Config{}, errors.New("HTTP_ADDR must be a numeric IP and port between 1 and 65535")
	}
	containerFlag := getenv("SANDBOX_ISOLATED_CONTAINER")
	if containerFlag != "" {
		c.IsolatedContainer, err = strconv.ParseBool(containerFlag)
		if err != nil {
			return Config{}, errors.New("SANDBOX_ISOLATED_CONTAINER must be a boolean")
		}
	}
	if c.IsolatedContainer && c.Mode != "sandbox" {
		return Config{}, errors.New("SANDBOX_ISOLATED_CONTAINER is only allowed in sandbox")
	}
	if c.Mode == "sandbox" && !ip.IsLoopback() && !c.IsolatedContainer {
		return Config{}, errors.New("sandbox must bind loopback unless isolated container networking is explicitly configured")
	}
	db, err := url.Parse(c.DatabaseURL)
	if err != nil || (db.Scheme != "postgres" && db.Scheme != "postgresql") || db.Hostname() == "" || strings.Trim(db.Path, "/") == "" {
		return Config{}, errors.New("DATABASE_URL must be a PostgreSQL URI with a host and database name")
	}
	web, err := url.Parse(c.PublicWebURL)
	if err != nil || web.Hostname() == "" || web.User != nil || web.RawQuery != "" || web.Fragment != "" {
		return Config{}, errors.New("PUBLIC_WEB_BASE_URL must be an absolute HTTP(S) URL without credentials, query or fragment")
	}
	webIP := net.ParseIP(web.Hostname())
	localWeb := web.Hostname() == "localhost" || (webIP != nil && webIP.IsLoopback())
	if web.Scheme != "https" && !(c.Mode == "sandbox" && web.Scheme == "http" && localWeb) {
		return Config{}, errors.New("PUBLIC_WEB_BASE_URL must use HTTPS (sandbox permits loopback HTTP)")
	}
	if timeout := getenv("SHUTDOWN_TIMEOUT"); timeout != "" {
		c.ShutdownTimeout, err = time.ParseDuration(timeout)
		if err != nil || c.ShutdownTimeout < time.Second || c.ShutdownTimeout > time.Minute {
			return Config{}, fmt.Errorf("SHUTDOWN_TIMEOUT must be between 1s and 1m")
		}
	}
	c.PublicWebURL = strings.TrimRight(c.PublicWebURL, "/")
	if raw := getenv("VIDEO_ALLOWED_HOSTS"); raw != "" {
		seen := map[string]bool{}
		for _, host := range strings.Split(raw, ",") {
			host = strings.ToLower(strings.TrimSpace(host))
			u, err := url.Parse("https://" + host)
			if err != nil || host == "" || len(host) > 253 || u.Host != host || u.Hostname() != host || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil || u.Port() != "" || seen[host] {
				return Config{}, errors.New("VIDEO_ALLOWED_HOSTS must contain distinct hostnames without scheme, credentials, path or port")
			}
			seen[host] = true
			c.VideoHosts = append(c.VideoHosts, host)
		}
		if len(c.VideoHosts) > 20 {
			return Config{}, errors.New("VIDEO_ALLOWED_HOSTS supports at most 20 hosts")
		}
	}
	return c, nil
}
