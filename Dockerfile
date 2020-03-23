FROM golang:1.14.0-alpine3.11 as base
RUN apk add git
WORKDIR /sqs-autoscaler-controller
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o sqs-autoscaler-controller main.go 

FROM scratch
COPY --from=base /sqs-autoscaler-controller/sqs-autoscaler-controller /sqs-autoscaler-controller
ENTRYPOINT ["/sqs-autoscaler-controller"]
