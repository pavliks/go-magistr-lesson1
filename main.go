package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func main() {
	url := "http://srv.msk01.gigacorp.local/_stats"
	client := &http.Client{Timeout: 3 * time.Second}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	numOfErrResp := 0

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-sig:
			return
		case <-ticker.C:
			ok := poll(client, url)
			if !ok {
				numOfErrResp++
				if numOfErrResp >= 3 {
					fmt.Println("Unable to fetch server statistic")
					numOfErrResp = 0
				}
			} else {
				numOfErrResp = 0
			}
		}
	}
}

func poll(client *http.Client, url string) bool {
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return false
	}

	response, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}
	line := strings.TrimSpace(string(response))
	parts := strings.Split(line, ",")
	if len(parts) != 7 {
		return false
	}

	vals := make([]int, 0, 7)
	for _, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return false
		}
		vals = append(vals, n)
	}

	la := vals[0]
	memoryAvailable, memoryUsed := vals[1], vals[2]
	diskAvailable, diskUsed := vals[3], vals[4]
	networkAvailable, networkUsed := vals[5], vals[6]
	
	if la >= 30 {
		fmt.Printf("Load Average is too high: %d\n", la)
	}
	
	if memoryAvailable > 0 {
		memoryPercent := int(float32(memoryUsed) / float32(memoryAvailable) * 100)
		if memoryPercent > 80 {
			fmt.Printf("Memory usage too high: %d%%\n", memoryPercent)
		}
	}
	
	if diskAvailable > 0 {
		diskPercentUsed := int(float32(diskUsed) / float32(diskAvailable) * 100)
		if diskPercentUsed > 90 {
			diskFreeMB := (diskAvailable - diskUsed) / 1024 / 1024
			fmt.Printf("Free disk space is too low: %d Mb left\n", diskFreeMB)
		}
	}
	
	if networkAvailable > 0 {
		netPercentUsed := int(float32(networkUsed) / float32(networkAvailable) * 100)
		if netPercentUsed > 90 {
			freeNetworkMbps := (networkAvailable - networkUsed) / 1000 / 1000
			fmt.Printf("Network bandwidth usage high: %d Mbit/s available\n", freeNetworkMbps)
		}
	}

	return true
}
