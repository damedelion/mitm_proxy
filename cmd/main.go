package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/damedelion/mitm_proxy/internal/proxy"
)

func main() {
	port := 8080

	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", port))
	println("mitm proxy is listening on", addr.String())
	if err != nil {
		println(err.Error())
		return
	}

	listener, err := net.ListenTCP("tcp", addr)
	defer listener.Close()
	if err != nil {
		println(err.Error())
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			listener.SetDeadline(time.Now().Add(1 * time.Second))
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					listener.Close()
					println("Listener closed")
					return
				default:
					if netErr, ok := err.(net.Error); ok && netErr.Timeout() { // handle timeout
						continue
					}

					println(err.Error())
					return
				}
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				select {
				case <-ctx.Done():
					return
				default:
					proxy.HandleConnection(conn)
				}
			}()
		}
	}()

	wg.Wait()
	listener.Close()
	//sigChan.Close()
}
