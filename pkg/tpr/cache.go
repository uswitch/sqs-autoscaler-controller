package tpr

import (
	"context"
	"time"

	log "github.com/Sirupsen/logrus"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type Cache struct {
	store      cache.Indexer
	controller cache.Controller
}

func (c Cache) process(obj interface{}) error {
	deltas := obj.(cache.Deltas)

	for _, delta := range deltas {
		scaler := delta.Object.(*SqsAutoScaler)
		log.WithFields(log.Fields{"scaler": scaler}).Debug("Processing")

		switch delta.Type {
		case cache.Sync:
			c.store.Add(delta.Object)
		case cache.Added:
			c.store.Add(delta.Object)
		case cache.Updated:
			c.store.Update(delta.Object)
		case cache.Deleted:
			c.store.Delete(delta.Object)
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
	c.store = store
	config := &cache.Config{
		Queue:            cache.NewDeltaFIFO(cache.MetaNamespaceKeyFunc, nil, store),
		ListerWatcher:    listWatch,
		ObjectType:       &SqsAutoScaler{},
		FullResyncPeriod: syncInterval,
		RetryOnError:     false,
		Process:          c.process,
	}
	c.controller = cache.New(config)

	return c
}
