package proxy

import (
	"bufio"
	"bytes"
	"net"
	"net/http"
	"net/url"
)

func HandleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	request, err := http.ReadRequest(reader)
	if err != nil {
		println("failed to read request:", err.Error())
	}

	alter(request)

	buf := new(bytes.Buffer)
	request.Write(buf)

	port := request.URL.Port()
	if port == "" {
		if request.URL.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}

	targetConn, err := net.Dial("tcp", net.JoinHostPort(request.Host, port))
	if err != nil {
		println(err.Error())
		return
	}
	defer targetConn.Close()

	if err = request.Write(targetConn); err != nil {
		println(err.Error())
		return
	}
	//targetConn.Write([]byte(buf.String())) //alternative

	reader = bufio.NewReader(targetConn)
	response, err := http.ReadResponse(reader, request)
	if err != nil {
		println(err.Error())
		return
	}
	defer response.Body.Close()

	if err = response.Write(conn); err != nil {
		println(err.Error())
		return
	}
}

func alter(r *http.Request) {
	host := r.URL.Host
	path := r.URL.Path

	r.URL, _ = url.Parse(path)
	r.Header.Set("Host", host)
	r.Header.Del("Proxy-Connection")
}
