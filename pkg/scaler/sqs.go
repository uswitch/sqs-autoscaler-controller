package scaler

import (
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

func CurrentQueueSize(sess *session.Session, queueUrl string) (int64, error) {
	svc := sqs.New(sess)
	output, err := svc.GetQueueAttributes(&sqs.GetQueueAttributesInput{
		QueueUrl:       &queueUrl,
		AttributeNames: []*string{aws.String("ApproximateNumberOfMessages")},
	})
	if err != nil {
		return 0, err
	}
	s, ok := output.Attributes["ApproximateNumberOfMessages"]
	if !ok {
		return 0, fmt.Errorf("ApproximateNumberOfMessages did not exist: %+v", output.Attributes)
	}
	return strconv.ParseInt(*s, 10, 64)
}
