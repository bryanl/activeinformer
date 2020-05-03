package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/bryanl/activeinformer/pkg/activeinformer"
)

func main() {
	if err := run(); err != nil {
		log.Printf("%v", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	kubeconfig, err := activeinformer.FindKubeconfig()
	if err != nil {
		return fmt.Errorf("find kubeconfig: %w", err)
	}

	client, err := activeinformer.NewOutOfClusterClient(kubeconfig)
	if err != nil {
		return fmt.Errorf("create out of cluster client: %w", err)
	}

	resources, err := getResources(client)
	if err != nil {
		return fmt.Errorf("get resources: %w", err)
	}

	informer := activeinformer.NewInformer(client)
	if err := informer.Start(ctx); err != nil {
		return fmt.Errorf("start informer: %w", err)
	}

	if err := getPods(ctx, informer, resources); err != nil {
		return fmt.Errorf("get pods: %w", err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	fmt.Println("Press ctrl-c to exit")
	<-c

	log.Printf("stopping informer")
	if err := informer.Stop(); err != nil {
		return err
	}

	return nil
}

func getResources(client activeinformer.Client) (activeinformer.Resources, error) {
	var resources activeinformer.Resources

	if err := timeIt(func() error {
		list, err := client.Resources()
		if err != nil {
			return err
		}
		resources = list
		return nil
	}); err != nil {
		return nil, err
	}

	return resources, nil
}

func getPods(ctx context.Context, informer activeinformer.Informer, resources activeinformer.Resources) error {

	podResource, ok := resources.GroupVersionKind(schema.GroupVersionKind{Version: "v1", Kind: "Pod"})
	if !ok {
		return fmt.Errorf("there was no pod resource")
	}

	podList, err := informer.List(ctx, podResource.GroupVersionResource(), activeinformer.ListOption{})
	if err != nil {
		return fmt.Errorf("list pods: %w", err)
	}

	for _, pod := range podList.Items {
		fmt.Printf("%s, %s\n", pod.GetNamespace(), pod.GetName())
	}

	return nil
}

func timeIt(fn func() error) error {
	now := time.Now()
	if err := fn(); err != nil {
		return err
	}
	log.Printf("elapsed: %s", time.Since(now))
	return nil
}
