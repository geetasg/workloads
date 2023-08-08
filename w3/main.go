package main

import (
	"context"
	//"flag"
	"fmt"
	"math/rand"

	//"path/filepath"
	"sync"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	//"k8s.io/client-go/tools/clientcmd"
	//"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/rest"
)

const (
	numGoroutines      = 200
	eventsPerGoroutine = 200
)

var usedEventNames = make(map[string]bool)
var mutex = &sync.Mutex{}

func randomEventName() string {
	mutex.Lock()
	defer mutex.Unlock()

	for {
		eventName := fmt.Sprintf("Event-%d", rand.Intn(1000000))
		if !usedEventNames[eventName] {
			usedEventNames[eventName] = true
			return eventName
		}
	}
}

func deleteEvents(clientset *kubernetes.Clientset) {
	events, err := clientset.CoreV1().Events("default").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Println("Error listing events:", err)
		return
	}

	for _, event := range events.Items {
		if usedEventNames[event.Name] {
			err := clientset.CoreV1().Events("default").Delete(context.Background(), event.Name, metav1.DeleteOptions{})
			if err != nil {
				fmt.Printf("Error deleting event %s: %v\n", event.Name, err)
			} else {
				fmt.Printf("Deleted event: %s\n", event.Name)
			}
		}
	}
}

func publishEvent(clientset *kubernetes.Clientset, wg *sync.WaitGroup) {
	defer wg.Done()

	for i := 0; i < eventsPerGoroutine; i++ {
		eventName := randomEventName()

		// Create a Kubernetes event object
		event := &corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      eventName,
				Namespace: "default", // Replace with the desired namespace
			},
			Reason: "CustomReason",
			Type:   "Normal",
			Source: corev1.EventSource{
				Component: "CustomComponent",
			},
			Message: "Custom message for the event",
		}

		// Publish the event to Kubernetes
		_, err := clientset.CoreV1().Events(event.Namespace).Create(context.Background(), event, metav1.CreateOptions{})
		if err != nil {
			fmt.Println("Error publishing event:", err)
		} else {
			fmt.Println("published event:", eventName)
		}

	}
}

func main() {
	/*
		var kubeconfig *string
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		} else {
			kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}
		flag.Parse()

		// Use the current context in kubeconfig
		//config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	*/
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	// Set increased QPS and Burst values in the config
	config.QPS = 1000   // Increase the QPS to 1000
	config.Burst = 2000 // Increase the Burst to 2000

	// Create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// Delete any previously created events with the same names
	deleteEvents(clientset)

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go publishEvent(clientset, &wg)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	fmt.Println("All events published.")
}
