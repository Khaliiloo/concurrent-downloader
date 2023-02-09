package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

var totlaSize int64 = 0
var downloadedSize int64 = 0

func downloadChunk(url string, start int64, end int64, wg *sync.WaitGroup) error {
	defer wg.Done()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Range", "bytes="+strconv.FormatInt(start, 10)+"-"+strconv.FormatInt(end, 10))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	filename := url[strings.LastIndex(url, "/")+1:]
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return err
	}

	if n, err := io.Copy(f, res.Body); err != nil {
		return err
	} else {
		downloadedSize += n
	}
	downloadingPercentage := int(float64(downloadedSize) / float64(totlaSize) * 100)
	fmt.Printf("Downloading:  %d%%\r", downloadingPercentage)
	return nil
}

func downloadFile(url string, chunkCount int) error {
	resp, err := http.Head(url)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status code: %d, couldn't download the file", resp.StatusCode)
	}
	defer resp.Body.Close()
	size, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	MB := float64(size) / 1024 / 1024

	totlaSize = size
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	chunkSize := (size + int64(chunkCount-1)) / int64(chunkCount)
	for i := 0; i < chunkCount; i++ {
		start := chunkSize * int64(i)
		end := start + chunkSize - 1
		if end >= size {
			end = size - 1
		}
		wg.Add(1)
		go downloadChunk(url, start, end, &wg)
	}
	fmt.Printf("File size: %.2fMB\n", MB)
	fmt.Println("Starting download:", url[strings.LastIndex(url, "/")+1:])
	wg.Wait()
	return nil
}

func main() {
	var url string
	flag.StringVar(&url, "url", "", "URL to the file")
	var chunks int
	flag.IntVar(&chunks, "chunks", 4, "the number of chuncks the file will be splitted to")
	flag.Parse()
	if url == "" {
		fmt.Println("URL is required. Please set it by -url ...")
		return
	}
	if err := downloadFile(url, chunks); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("File downloaded successfully")

}
