package main

import (
	"fmt"
	"os"

	"github.com/Bahjat/page-insight-tool/backend/internal/pageinsight"
	"github.com/Bahjat/page-insight-tool/backend/internal/platform/config"
	"github.com/Bahjat/page-insight-tool/backend/internal/platform/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel)
	log.Info("the tool started")

	fetcher := pageinsight.NewHTTPClient()
	checker := pageinsight.NewLinkChecker(cfg.LinkCheckConcurrency)
	_ = pageinsight.NewEngine(fetcher, checker)
}
