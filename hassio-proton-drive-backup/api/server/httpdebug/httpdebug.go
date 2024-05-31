package httpdebug

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	// Log request
	dump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Println(err)
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		fmt.Println(err)
	}
	netIP := net.ParseIP(ip)
	if netIP != nil {
		ip := netIP.String()
		if ip == "::1" {
			ip = "127.0.0.1"
		}
	}

	fmt.Println(string(dump))
	fmt.Println(ip)
	w.Write([]byte(dump))
	w.Write([]byte(ip))
}
