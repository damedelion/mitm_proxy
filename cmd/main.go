package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/damedelion/mitm_proxy/internal/proxy"
	"github.com/damedelion/mitm_proxy/internal/server"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func main() {
	proxyPort := 8080
	apiPort := 8081

	client, err := mongo.Connect(options.Client().ApplyURI("mongodb://mongodb:27017"))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func() {
		if err := client.Disconnect(context.Background()); err != nil {
			log.Printf("Failed to disconnect MongoDB: %v", err)
		}
	}()

	if err := client.Ping(context.Background(), nil); err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	proxyListener, err := net.Listen("tcp", fmt.Sprintf(":%d", proxyPort))
	if err != nil {
		log.Fatalf("Failed to start proxy listener: %v", err)
	}
	log.Println("MITM proxy is listening on :", proxyPort)

	apiServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", apiPort),
		Handler: setupRouter(client),
	}
	log.Println("Starting API server on :", apiPort)

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		runProxyServer(ctx, proxyListener, client, wg)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := apiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("API server error: %v", err)
		}
	}()

	<-sigChan
	log.Println("Shutting down servers...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("API server shutdown error: %v", err)
	}

	cancel()
	proxyListener.Close()

	wg.Wait()
	log.Println("Servers stopped")
}

func setupRouter(client *mongo.Client) *mux.Router {
	srv := server.NewServer(client)
	r := mux.NewRouter()
	r.HandleFunc("/requests", srv.GetRequests).Methods("GET")
	r.HandleFunc("/requests/{id}", srv.GetRequest).Methods("GET")
	r.HandleFunc("/repeat/{id}", srv.RepeatRequest).Methods("GET")
	r.HandleFunc("/scan/{id}", srv.ScanRequest).Methods("GET")
	return r
}

func runProxyServer(ctx context.Context, listener net.Listener, client *mongo.Client, wg *sync.WaitGroup) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			listener.(*net.TCPListener).SetDeadline(time.Now().Add(1 * time.Second))
			conn, err := listener.Accept()
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				log.Printf("Proxy accept error: %v", err)
				return
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				proxy.HandleConnection(conn, client)
			}()
		}
	}
}
