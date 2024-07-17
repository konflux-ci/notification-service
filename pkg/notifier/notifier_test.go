package notifier

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sns"
)

type MockSNSPublisher struct {
	fail bool
}

func (msn *MockSNSPublisher) Publish(
	context.Context,
	*sns.PublishInput,
	...func(*sns.Options)) (*sns.PublishOutput, error) {
	out := &sns.PublishOutput{}
	if msn.fail == false {
		return out, nil
	} else {
		return out, errors.New("error")
	}

}

func TestNewSNSNotifier(t *testing.T) {
	region := "myRegion"
	err := os.Setenv("NOTIFICATION_REGION", region)
	if err != nil {
		t.Errorf("Could not set the region environment variable, failing the test")
	}
	snsClient, _ := NewSNSClient()
	if snsClient.Options().Region != region {
		t.Errorf("got %q, wanted %q", snsClient.Options().Region, region)
	}
}

func TestSuccessNotify(t *testing.T) {
	ctx := context.TODO()
	m := &MockSNSPublisher{fail: false}
	mocker := SNSNotifier{Pub: m}

	got := mocker.Notify(ctx, "testMessage")
	if got != nil {
		t.Errorf("got %q, wanted nil", got)
	}
}

func TestFailedNotify(t *testing.T) {
	ctx := context.TODO()
	m := &MockSNSPublisher{fail: true}
	mocker := SNSNotifier{Pub: m}

	got := mocker.Notify(ctx, "testMessage")
	if got == nil {
		t.Errorf("got %q, wanted nil", got)
	}
}
