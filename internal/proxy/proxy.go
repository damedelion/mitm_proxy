package proxy

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"time"

	"github.com/damedelion/mitm_proxy/internal/db"
	"github.com/damedelion/mitm_proxy/internal/parser"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func HandleConnection(conn net.Conn, client *mongo.Client) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	request, err := http.ReadRequest(reader)
	if err != nil {
		println("failed to read request:", err.Error())
	}

	//out.Request(request)

	if request.Method == "CONNECT" {
		handleHTTPS(conn, request, client)
		return
	}
	handleHTTP(conn, request, client)
}

func handleHTTP(conn net.Conn, request *http.Request, client *mongo.Client) {
	targetConn, err := net.Dial("tcp", net.JoinHostPort(request.Host, "80"))
	if err != nil {
		println(err.Error())
		return
	}
	defer targetConn.Close()

	bodyBytes, err := io.ReadAll(request.Body)
	if err != nil {
		println("Failed to read request body:", err.Error())
		return
	}
	request.Body.Close()
	request.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	alter(request)

	parsedRequest, err := parser.ParseRequest(request, bodyBytes)
	if err != nil {
		println(err.Error())
		return
	}
	id, err := db.WriteRequest(parsedRequest, client)
	if err != nil {
		println(err.Error())
		return
	} else {
		println("successful db write")
	}

	if err = request.Write(targetConn); err != nil {
		println(err.Error())
		return
	}

	targetReader := bufio.NewReader(targetConn)
	response, err := http.ReadResponse(targetReader, request)
	if err != nil {
		println(err.Error())
		return
	}
	bodyBytes, err = io.ReadAll(response.Body)
	if err != nil {
		println("Failed to read response body:", err.Error())
		return
	}
	response.Body.Close()
	response.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	parsedResponse, err := parser.ParseResponse(response, bodyBytes, id)
	if err != nil {
		println(err.Error())
		return
	}
	err = db.WriteResponse(parsedResponse, client)
	if err != nil {
		println(err.Error())
		return
	} else {
		println("successful db write")
	}

	if err = response.Write(conn); err != nil {
		println(err.Error())
		return
	}
}

func handleHTTPS(conn net.Conn, request *http.Request, client *mongo.Client) {
	conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	host, _, err := net.SplitHostPort(request.Host)
	if err != nil {
		host = request.Host // port is not set in old clients
	}

	cert, err := generateCert(host)
	if err != nil {
		println(err.Error())
		return
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	tlsConn := tls.Server(conn, tlsConfig)
	defer tlsConn.Close()

	if err = tlsConn.Handshake(); err != nil {
		println("handshake error", err.Error())
	}

	state := tlsConn.ConnectionState()
	fmt.Println("SSL ServerName:", state.ServerName)
	fmt.Println("SSL Handshake:", state.HandshakeComplete)

	reader := bufio.NewReader(tlsConn)
	request, err = http.ReadRequest(reader)
	if err != nil {
		println(err.Error())
		return
	}
	bodyBytes, err := io.ReadAll(request.Body)
	if err != nil {
		println("Failed to read request body:", err.Error())
		return
	}
	request.Body.Close()
	request.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	parsedRequest, err := parser.ParseRequest(request, bodyBytes)
	if err != nil {
		println(err.Error())
		return
	}
	id, err := db.WriteRequest(parsedRequest, client)
	if err != nil {
		println(err.Error())
		return
	} else {
		println("successful db write")
	}

	targetConn, err := tls.Dial("tcp", net.JoinHostPort(request.Host, "443"), &tls.Config{})
	if err != nil {
		println(err.Error())
		return
	}
	defer targetConn.Close()

	if err := request.Write(targetConn); err != nil {
		println(err.Error())
		return
	}

	resp, err := http.ReadResponse(bufio.NewReader(targetConn), request)
	if err != nil {
		println(err.Error())
		return
	}
	bodyBytes, err = io.ReadAll(resp.Body)
	if err != nil {
		println("Failed to read response body:", err.Error())
		return
	}
	resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	parsedResponse, err := parser.ParseResponse(resp, bodyBytes, id)
	if err != nil {
		println(err.Error())
		return
	}
	err = db.WriteResponse(parsedResponse, client)
	if err != nil {
		println(err.Error())
		return
	} else {
		println("successful db write")
	}

	if err := resp.Write(tlsConn); err != nil {
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

func generateCert(host string) (tls.Certificate, error) {
	serial := time.Now().UnixNano()

	gen := exec.Command("sh", "gen_cert.sh", host, fmt.Sprintf("%d", serial))
	certPEM, err := gen.Output()
	if err != nil {
		println("err:", err.Error())
		return tls.Certificate{}, err
	}

	keyPEM, err := os.ReadFile("cert.key")
	if err != nil {
		return tls.Certificate{}, err
	}

	return tls.X509KeyPair(certPEM, keyPEM)
}
