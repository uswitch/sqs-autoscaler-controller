package crd

import (
	"context"
	"time"

	log "github.com/Sirupsen/logrus"
	metrics "github.com/rcrowley/go-metrics"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type Cache struct {
	Store      cache.Indexer
	controller cache.Controller
}

func (c Cache) process(obj interface{}) error {
	deltas := obj.(cache.Deltas)

	for _, delta := range deltas {
		scaler := delta.Object.(*SqsAutoScaler)
		log.WithFields(log.Fields{"scaler": scaler}).Debug("Processing")

		switch delta.Type {
		case cache.Sync:
			c.Store.Add(delta.Object)
		case cache.Added:
			c.Store.Add(delta.Object)
		case cache.Updated:
			c.Store.Update(delta.Object)
		case cache.Deleted:
			c.Store.Delete(delta.Object)
		}
	}
	return nil
}

func (c Cache) Run(ctx context.Context) {
	c.controller.Run(ctx.Done())
}

func NewCache(client *rest.RESTClient, syncInterval time.Duration) *Cache {
	c := &Cache{}

	listWatch := cache.NewListWatchFromClient(client, "sqsautoscalers", "", fields.Everything())
	store := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, cache.Indexers{})
	c.Store = store
	config := &cache.Config{
		Queue:            cache.NewDeltaFIFO(cache.MetaNamespaceKeyFunc, nil, store),
		ListerWatcher:    listWatch,
		ObjectType:       &SqsAutoScaler{},
		FullResyncPeriod: syncInterval,
		RetryOnError:     false,
		Process:          c.process,
	}
	c.controller = cache.New(config)

	gauge := metrics.NewFunctionalGauge(func() int64 {
		return int64(len(store.List()))
	})
	metrics.Register("cache.scalers", gauge)

	return c
}
