package utils

import (
	"errors"
	"net"
	"net/http"
	"strings"
)

func GetIP(r *http.Request) (string, error) {
	ip, err := parseIp(r)
	if err != nil {
		return "", err
	}
	if ip == "::1" {
		return "127.0.0.1", nil
	}

	return ip, nil
}

// GetIP returns request real ip.
func parseIp(r *http.Request) (string, error) {
	ip := r.Header.Get("X-Real-IP")
	if net.ParseIP(ip) != nil {
		return ip, nil
	}

	ip = r.Header.Get("X-Forward-For")
	for _, i := range strings.Split(ip, ",") {
		if net.ParseIP(i) != nil {
			return i, nil
		}
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "", err
	}

	if net.ParseIP(ip) != nil {
		return ip, nil
	}

	return "", errors.New("no valid ip found")
}
