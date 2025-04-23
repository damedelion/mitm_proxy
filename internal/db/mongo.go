package db

import (
	"context"

	"github.com/damedelion/mitm_proxy/internal/parser"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func WriteRequest(req *parser.HTTPRequest, client *mongo.Client) (bson.ObjectID, error) {
	collection := client.Database("http_data").Collection("requests")

	res, err := collection.InsertOne(context.Background(), req)
	id := res.InsertedID
	if err != nil {
		return bson.ObjectID{}, err
	}

	return id.(bson.ObjectID), nil
}

func WriteResponse(resp *parser.HTTPResponse, client *mongo.Client) error {
	collection := client.Database("http_data").Collection("responses")

	_, err := collection.InsertOne(context.Background(), resp)
	if err != nil {
		return err
	}

	return nil
}
