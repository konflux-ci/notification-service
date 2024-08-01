package notifier

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/go-logr/logr"
)

const (
	refreshErrorMessage = "Refresh Error"
	publishErrorMessage = "Publish Error"
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
		return out, errors.New(publishErrorMessage)
	}
}

func MockSucceessfulRefreshSuccessPublishClient() (Publisher, error) {
	mp := &MockSNSPublisher{fail: false}
	return mp, nil
}

func MockSucceessfulRefreshFailPublishClient() (Publisher, error) {
	mp := &MockSNSPublisher{fail: true}
	return mp, nil
}

func MockFailedRefreshClient() (Publisher, error) {
	return nil, errors.New(refreshErrorMessage)
}

var logger logr.Logger

func TestRefreshClient(t *testing.T) {
	region := "myRegion"
	err := os.Setenv("NOTIFICATION_REGION", region)
	if err != nil {
		t.Errorf("Could not set the region environment variable, failing the test")
	}
	_, err = RefreshClient()
	if err != nil {
		t.Errorf("got %q, wanted nil", err)
	}

}

func TestSuccessNotify(t *testing.T) {
	ctx := context.TODO()
	mp := &MockSNSPublisher{fail: false}
	ntf := &SNSNotifier{
		Pub:           mp,
		RefreshClient: MockSucceessfulRefreshSuccessPublishClient,
		Log:           logger,
	}
	err := ntf.Notify(ctx, "testMessage")
	if err != nil {
		t.Errorf("got %q, wanted nil", err)
	}
}

func TestFailedRefreshNotify(t *testing.T) {
	ctx := context.TODO()
	mp := &MockSNSPublisher{fail: false}
	ntf := &SNSNotifier{
		Pub:           mp,
		RefreshClient: MockFailedRefreshClient,
		Log:           logger,
	}
	err := ntf.Notify(ctx, "testMessage")
	if err == nil {
		t.Errorf("got %q, wanted nil", err)
	}
	if err.Error() != refreshErrorMessage {
		t.Errorf("got %q, wanted %q", err.Error(), refreshErrorMessage)
	}
}

func TestFailedPublishNotify(t *testing.T) {
	ctx := context.TODO()
	mp := &MockSNSPublisher{fail: true}
	ntf := &SNSNotifier{
		Pub:           mp,
		RefreshClient: MockSucceessfulRefreshFailPublishClient,
		Log:           logger,
	}

	err := ntf.Notify(ctx, "testMessage")
	if err == nil {
		t.Errorf("got %q, wanted nil", err)
	}
	if err.Error() != publishErrorMessage {
		t.Errorf("got %q, wanted %q", err.Error(), publishErrorMessage)
	}
}
