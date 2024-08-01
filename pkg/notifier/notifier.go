package notifier

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/go-logr/logr"
)

type ClientRefresher func() (Publisher, error)

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
	Pub           Publisher
	RefreshClient ClientRefresher
	Log           logr.Logger
}

/*
To create the client, make sure you set a secret with AWS credentials
Set the key to be `credentials`
Set the value to be
`[default]
aws_access_key_id=<AWS_ACCESS_KEY>
aws_secret_access_key=<AWS_SECRET_ACCESS_KEY>`

Set the values to match your credentials
Mount the secret to `/.aws`

In addition, make sure you also set the
`NOTIFICATION_REGION` environment variable for the region,
and `NOTIFICATION_TOPIC_ARN` environment variable for the SNS topic ARN
*/

func RefreshClient() (Publisher, error) {
	region := os.Getenv("NOTIFICATION_REGION")
	sdkConfig, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return &sns.Client{}, err
	}
	return sns.NewFromConfig(sdkConfig), nil
}

func (s *SNSNotifier) Notify(ctx context.Context, message string) error {
	logger := s.Log.WithName("Notifier")
	cli, err := s.RefreshClient()
	if err != nil {
		logger.Error(err, "Could not create sns client")
		return err
	}
	s.Pub = cli
	topicArn := os.Getenv("NOTIFICATION_TOPIC_ARN")
	publishInput := sns.PublishInput{TopicArn: aws.String(topicArn), Message: aws.String(message)}
	_, err = s.Pub.Publish(ctx, &publishInput)
	if err != nil {
		logger.Error(err, "Could not send notification to SNS")
		return err
	}
	logger.Info("message was sent successfully")
	return nil
}
