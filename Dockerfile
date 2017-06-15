FROM scratch

ADD bin/sqs-autoscaler-controller /sqs-autoscaler-controller

ENTRYPOINT ["/sqs-autoscaler-controller"]