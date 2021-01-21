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
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"net/http"
	"os"
	"strings"
)

type RequestData struct {
	Image      string `json:"image"`
	ImageName  string `json:"imageName"`
	BucketName string `json:"bucketName"`
}

type S3Output struct {
	RequestOutput *s3.PutObjectOutput `json:"requestOutput"`
	Link          string              `json:"link"`
}

const (
	ConfigError = "Error retrieving AWS credentials"
)

func initializeAWS() aws.Config {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(os.Getenv("AWS_REGION")),
		config.WithSharedConfigProfile(os.Getenv("AWS_PROFILE")))

	if err != nil {
		panic(fmt.Sprintf("%s: %s\n", ConfigError, err.Error()))
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

func appendContentType(imgName string, content []byte) string {
	return fmt.Sprintf("%s.%s",
		imgName,
		strings.Split(http.DetectContentType(content), "/")[1])
}

func UploadToS3(client *s3.Client, data RequestData, content []byte) *s3.PutObjectOutput {
	output, err := client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(data.BucketName),
		Key:         aws.String(appendContentType(data.ImageName, content)),
		Body:        bytes.NewReader(content),
		ContentType: aws.String(http.DetectContentType(content)),
		ACL:         types.ObjectCannedACLPublicRead,
	})

	if err != nil {
		panic(fmt.Sprintf("Error during upload: %s", err.Error()))
	}

	return output
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	cfg := initializeAWS()
	s3Client := s3.NewFromConfig(cfg)

	data := RequestData{}
	json.Unmarshal([]byte(request.Body), &data)
	content := decodeBase64(data.Image)

	output := UploadToS3(s3Client, data, content)
	s3Output, _ := json.Marshal(S3Output{
		output,
		fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s",
			data.BucketName,
			os.Getenv("AWS_REGION"),
			appendContentType(data.ImageName, content)),
	})

	return events.APIGatewayProxyResponse{
		Body:       string(s3Output),
		StatusCode: 201,
	}, nil
}

func main() {
	lambda.Start(handler)
}
