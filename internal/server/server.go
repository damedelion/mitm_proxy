package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/damedelion/mitm_proxy/internal/parser"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Server struct {
	client *mongo.Client
}

func NewServer(client *mongo.Client) *Server {
	return &Server{client: client}
}

func (s *Server) GetRequests(w http.ResponseWriter, r *http.Request) {
	collection := s.client.Database("http_data").Collection("requests")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		log.Printf("Failed to find requests: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var requests []parser.HTTPRequest
	if err := cursor.All(ctx, &requests); err != nil {
		log.Printf("Failed to decode requests: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(requests); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

func (s *Server) GetRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	collection := s.client.Database("http_data").Collection("requests")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var request parser.HTTPRequest
	err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&request)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			http.Error(w, "Request not found", http.StatusNotFound)
			return
		}
		log.Printf("Failed to find request: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(request); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

func (s *Server) GetResponses(w http.ResponseWriter, r *http.Request) {
	collection := s.client.Database("http_data").Collection("responses")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		log.Printf("Failed to find responses: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var responses []parser.HTTPResponse
	if err := cursor.All(ctx, &responses); err != nil {
		log.Printf("Failed to decode responses: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(responses); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

func (s *Server) RepeatRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	collection := s.client.Database("http_data").Collection("requests")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var request parser.HTTPRequest
	err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&request)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			http.Error(w, "Request not found", http.StatusNotFound)
			return
		}
		log.Printf("Failed to find request: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	httpReq, err := BuildRequestFromDB(&request)
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		log.Printf("Failed to send request: %v", err)
		http.Error(w, "Failed to repeat request", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  resp.Status,
		"headers": resp.Header,
	})
}

func (s *Server) ScanRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	collection := s.client.Database("http_data").Collection("requests")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var request parser.HTTPRequest
	err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&request)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			http.Error(w, "Request not found", http.StatusNotFound)
			return
		}
		log.Printf("Failed to find request: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	httpReq, err := BuildRequestFromDB(&request)
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	client := &http.Client{}

	file, err := os.Open("params.txt")
	if err != nil {
		log.Fatal(err)
		return
	}
	defer file.Close()

	testStr := "shefuisehfuishe"
	unsecureParams := make([]string, 0)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		param := scanner.Text()
		httpReq.URL.Query().Add(param, testStr)
		resp, err := client.Do(httpReq)
		if err != nil {
			log.Printf("Failed to send request: %v", err)
			http.Error(w, "Failed to repeat request", http.StatusInternalServerError)
			return
		}
		resp.Body.Close()
		for name, values := range resp.Header {
			for _, value := range values {
				if strings.Contains(value, testStr) {
					unsecureParams = append(unsecureParams, name)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"unsecure_parameters": unsecureParams,
	})
}

func BuildRequestFromDB(request *parser.HTTPRequest) (*http.Request, error) {
	targetURL := url.URL{
		Scheme: "http",
		Host:   request.Headers["Host"],
		Path:   request.Path,
	}

	query := targetURL.Query()
	for k, v := range request.GetParams {
		query.Add(k, fmt.Sprintf("%v", v))
	}
	targetURL.RawQuery = query.Encode()

	httpReq, err := http.NewRequest(request.Method, targetURL.String(), nil)
	if err != nil {
		return &http.Request{}, err
	}

	for k, v := range request.Headers {
		httpReq.Header.Add(k, v)
	}

	for k, v := range request.Cookies {
		httpReq.AddCookie(&http.Cookie{
			Name:  k,
			Value: fmt.Sprintf("%v", v),
		})
	}
	return httpReq, nil
}
