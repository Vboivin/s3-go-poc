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

type requestData struct {
	Image      string `json:"image"`
	ImageName  string `json:"imageName"`
	BucketName string `json:"bucketName"`
}

type s3Output struct {
	RequestOutput *s3.PutObjectOutput `json:"requestOutput"`
	Link          string              `json:"link"`
}

func initializeAWS(ctx context.Context) (aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(ctx)

	if err != nil {
		return aws.Config{}, err
	}

	return cfg, err
}

func decodeBase64(base64img string) ([]byte, error) {
	img, err := base64.StdEncoding.DecodeString(base64img)

	if err != nil {
		return nil, err
	}

	return img, nil
}

func appendContentType(imgName string, content []byte) string {
	return fmt.Sprintf("%s.%s",
		imgName,
		strings.Split(http.DetectContentType(content), "/")[1])
}

func uploadToS3(client *s3.Client, data requestData, content []byte) (*s3.PutObjectOutput, error) {
	output, err := client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(data.BucketName),
		Key:         aws.String(appendContentType(data.ImageName, content)),
		Body:        bytes.NewReader(content),
		ContentType: aws.String(http.DetectContentType(content)),
		ACL:         types.ObjectCannedACLPublicRead,
	})

	if err != nil {
		return nil, err
	}

	return output, nil
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	cfg, _ := initializeAWS(context.TODO())

	s3Client := s3.NewFromConfig(cfg)

	data := requestData{}
	json.Unmarshal([]byte(request.Body), &data)
	content, _ := decodeBase64(data.Image)

	output, _ := uploadToS3(s3Client, data, content)

	s3Output, _ := json.Marshal(s3Output{
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
