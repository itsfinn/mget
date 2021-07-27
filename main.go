package main

import (
	"log"
	"os"
	"runtime"

	"github.com/urfave/cli/v2"
)

func main() {
	concurrencyN := runtime.NumCPU()
	app := &cli.App{
		Name:  "mget",
		Usage: "File concurrency downloader",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "url",
				Aliases:  []string{"u"},
				Usage:    "`url` to download",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "output `filename`",
			},
			&cli.UintFlag{
				Name:    "concurrency",
				Aliases: []string{"n"},
				Usage:   "concurrency `number`",
				Value:   uint(concurrencyN),
			},
		},
		Action: func(c *cli.Context) error {
			strUrl := c.String("url")
			filename := c.String("output")
			concurrency := c.Uint("concurrency")
			return NewDownloader(concurrency).Download(strUrl, filename)
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
