package main

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

const (
	filePath    = "/mnt/mmap_example.txt"
	fileSize    = 1024 * 1024     // 1 MB
	concurrency = 200             // Number of concurrent fdatasync operations
	duration    = 5 * time.Minute // Duration to run the writes
)

func main() {
	// Create or open the file
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	fmt.Printf("Concurrency: %d\n", concurrency)

	// Expand the file to the desired size
	if err := file.Truncate(fileSize); err != nil {
		fmt.Println("Error truncating file:", err)
		return
	}

	// Memory map the file
	data, err := syscall.Mmap(int(file.Fd()), 0, fileSize, syscall.PROT_WRITE|syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		fmt.Println("Error mapping file to memory:", err)
		return
	}
	defer syscall.Munmap(data)

	// Create a channel to signal when fdatasync is done
	done := make(chan struct{})

	// Create a channel to collect latencies
	latencyChan := make(chan time.Duration, concurrency)

	// Perform concurrent fdatasync operations and latency measurement using goroutines
	for i := 0; i < concurrency; i++ {
		go func() {
			// Loop until the duration expires
			startTime := time.Now()
			for time.Since(startTime) < duration {
				startTimeOp := time.Now()
				// Write some data to the memory-mapped file (for demonstration)
				copy(data, []byte("Hello, world!\n"))

				// Synchronize data from memory to the file
				err := syscall.Fdatasync(int(file.Fd()))
				if err != nil {
					fmt.Println("Error performing fdatasync:", err)
					continue
				}

				latency := time.Since(startTimeOp)
				latencyChan <- latency
			}

			done <- struct{}{}
		}()
	}

	// Continuously print latencies
	go func() {
		for latency := range latencyChan {
			fmt.Printf("Latency: %s\n", latency)
		}
	}()

	// Wait for all fdatasync operations to complete
	for i := 0; i < concurrency; i++ {
		<-done
	}

	// Close the latency channel
	close(latencyChan)

	// Calculate average latency
	var totalLatency time.Duration
	count := 0
	for latency := range latencyChan {
		totalLatency += latency
		count++
	}

	if count > 0 {
		averageLatency := totalLatency / time.Duration(count)
		fmt.Printf("Average Latency: %s\n", averageLatency)
	} else {
		fmt.Println("No latencies measured.")
	}
}
