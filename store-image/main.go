package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"net/http"
	"os"
)

type RequestData struct {
	Image      string `json:"image"`
	ImageName  string `json:"imageName"`
	BucketName string `json:"bucketName"`
}

var (
	configError = "Error retrieving AWS credentials"
)

func initializeAWS() aws.Config {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(os.Getenv("AWS_REGION")),
		config.WithSharedConfigProfile(os.Getenv("AWS_PROFILE")))

	if err != nil {
		panic(fmt.Sprintf("%s: %s\n", configError, err.Error()))
	}

	return cfg
}

func decodeBase64(base64img string) []byte {
	img, err := base64.StdEncoding.DecodeString(base64img)

	if err != nil {
		panic(fmt.Sprintf("Could not decode: %s\n%s", base64img, err.Error()))
	}

	return img
}

func UploadToS3(client *s3.Client, data RequestData) string {
	content := decodeBase64(data.Image)
	output, err := client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(data.BucketName),
		Key:         aws.String(fmt.Sprintf("%s.png", data.ImageName)),
		Body:        bytes.NewReader(content),
		ContentType: aws.String(http.DetectContentType(content)),
	})

	if err != nil {
		panic(fmt.Sprintf("Error during upload: %s", err.Error()))
	}

	jsonOutput, err := json.Marshal(output)

	if err != nil {
		panic(fmt.Sprintf("Error while converting metadata to json: %s", err.Error()))
	}

	return string(jsonOutput)
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	cfg := initializeAWS()
	s3Client := s3.NewFromConfig(cfg)
	data := RequestData{}
	json.Unmarshal([]byte(request.Body), &data)
	requestOutput := UploadToS3(s3Client, data)

	return events.APIGatewayProxyResponse{
		Body:       requestOutput,
		StatusCode: 201,
	}, nil
}

func main() {
	lambda.Start(handler)
}
