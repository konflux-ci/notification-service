package notifier

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/go-logr/logr"
)

type Notifier interface {
	Notify(context.Context, string) error
}

type Publisher interface {
	Publish(
		context.Context,
		*sns.PublishInput,
		...func(*sns.Options)) (*sns.PublishOutput, error)
}

type SNSNotifier struct {
	Pub Publisher
}

var logger logr.Logger

/*
To create the client, make sure you set the
AWS credentials as environment variables:
AWS_ACCESS_KEY_ID,
AWS_SECRET_ACCESS_KEY,
AWS_SESSION_TOKEN (optional)
In addition, make sure you also set the
NOTIFICATION_REGION for the region,
NOTIFICATION_TOPIC_ARN for the SNS topic ARN
*/

func NewSNSClient() (*sns.Client, error) {
	logger := logger.WithName("Notifier")
	region := os.Getenv("NOTIFICATION_REGION")
	sdkConfig, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		logger.Error(err, "Couldn't load default configuration. Have you set up your AWS account?")
		return &sns.Client{}, err
	}
	return sns.NewFromConfig(sdkConfig), nil
}

func (s *SNSNotifier) Notify(ctx context.Context, message string) error {

	topicArn := os.Getenv("NOTIFICATION_TOPIC_ARN")
	publishInput := sns.PublishInput{TopicArn: aws.String(topicArn), Message: aws.String(message)}
	_, err := s.Pub.Publish(ctx, &publishInput)
	if err != nil {
		logger.Error(err, "Couldn't notify to SNS")
		return err
	}
	logger.Info("message was sent successfully", message)
	return nil
}
