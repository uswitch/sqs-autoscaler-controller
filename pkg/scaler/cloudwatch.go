package scaler

import (
    "fmt"
    "time"
    "path"

    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/cloudwatch"
)

func NumberOfEmptyReceives(sess *session.Session, queueUrl string) (int64, error) {
    svc := cloudwatch.New(sess)

    period := int64(60)
    endTime := time.Now()
    duration, err := time.ParseDuration("-5m")
    if err != nil {
        return nil, err
    }
    startTime := endTime.Add(duration)

    query := &cloudwatch.MetricDataQuery{
		Id: aws.String("id1"),
		MetricStat: &cloudwatch.MetricStat{
			Metric: &cloudwatch.Metric{
				Namespace:  aws.String("AWS/SQS"),
				MetricName: aws.String("NumberOfEmptyReceives"),
				Dimensions: []*cloudwatch.Dimension{
					&cloudwatch.Dimension{
						Name:  aws.String("QueueName"),
						Value: aws.String(path.Base(queueUrl)),
					},
				},
			},
			Period: &period,
			Stat:   aws.String("Average"),
		},
	}

    resp, err := svc.GetMetricData(&cloudwatch.GetMetricDataInput{
		EndTime:           &endTime,
		StartTime:         &startTime,
		MetricDataQueries: []*cloudwatch.MetricDataQuery{query},
	})
    if err != nil {
        return nil, err
    }

    if len(resp.MetricDataResults) > 1 {
        return nil, fmt.Errorf("Expecting cloudwatch metric to return single data point")
    }

    return resp.MetricDataResults[0], nil
}
