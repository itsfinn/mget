package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/schollz/progressbar/v3"
)

type Downloader struct {
	concurrency uint
}

func NewDownloader(concurrency uint) *Downloader {
	return &Downloader{concurrency: concurrency}
}

func (d *Downloader) Download(strUrl, filename string) error {

	if filename == "" {
		filename = path.Base(strUrl)
	}

	resp, err := http.Head(strUrl)
	if err != nil {
		return err
	}

	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		"downloading",
	)
	if resp.StatusCode == http.StatusOK && resp.Header.Get("Accept-Ranges") == "bytes" {
		return d.md(strUrl, filename, int(resp.ContentLength), bar)
	}

	return d.sd(strUrl, filename, bar)
}

func (d *Downloader) md(strUrl, filename string, contentLen int, bar *progressbar.ProgressBar) error {
	partSize := contentLen / int(d.concurrency)

	partDir := d.getPartDir(filename)
	os.Mkdir(partDir, 0777)
	defer os.RemoveAll(partDir)

	var wg sync.WaitGroup
	wg.Add(int(d.concurrency))

	for i, rangeStart := 0, 0; i < int(d.concurrency); i, rangeStart = i+1, rangeStart+partSize+1 {
		go func(i, rangeStart int) {
			defer wg.Done()
			rangeEnd := rangeStart + partSize
			if i == int(d.concurrency)-1 {
				rangeEnd = contentLen
			}
			d.downloadPartial(strUrl, filename, rangeStart, rangeEnd, i, bar)
		}(i, rangeStart)
	}

	wg.Wait()

	d.mergePart(filename)
	return nil
}

func (d *Downloader) sd(strUrl, filename string, bar *progressbar.ProgressBar) error {
	req, err := http.NewRequest("GET", strUrl, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	flags := os.O_CREATE | os.O_WRONLY
	destFile, err := os.OpenFile(filename, flags, 0666)
	if err != nil {
		return err
	}
	defer destFile.Close()

	buf := make([]byte, 32*1024)
	_, err = io.CopyBuffer(io.MultiWriter(destFile, bar), resp.Body, buf)
	if err != nil && err != io.EOF {

		return err
	}
	return nil
}

func (d *Downloader) downloadPartial(strUrl, filename string, rangeStart, rangeEnd, i int, bar *progressbar.ProgressBar) {
	if rangeStart >= rangeEnd {
		return
	}
	req, err := http.NewRequest("GET", strUrl, nil)
	if err != nil {
		log.Fatal(err)
		return
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", rangeStart, rangeEnd))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer resp.Body.Close()

	flags := os.O_CREATE | os.O_WRONLY
	partFile, err := os.OpenFile(d.getPartFilename(filename, i), flags, 0666)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer partFile.Close()

	buf := make([]byte, 32*1024)
	_, err = io.CopyBuffer(io.MultiWriter(partFile, bar), resp.Body, buf)
	if err != nil && err != io.EOF {
		log.Fatal(err)
	}
}

func (d *Downloader) getPartDir(filename string) string {
	return strings.SplitN(filename, ".", 2)[0]
}

func (d *Downloader) getPartFilename(filename string, partNum int) string {
	partDir := d.getPartDir(filename)
	return fmt.Sprintf("%s/%s-%d", partDir, filename, partNum)
}

func (d *Downloader) mergePart(filename string) error {
	destFile, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer destFile.Close()

	for i := 0; i < int(d.concurrency); i++ {
		partFilename := d.getPartFilename(filename, i)
		partFile, err := os.Open(partFilename)
		if err != nil {
			return err
		}
		io.Copy(destFile, partFile)
		partFile.Close()
	}
	return nil
}
