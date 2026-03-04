package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Proviesec/PSFuzz/internal/config"
	"github.com/Proviesec/PSFuzz/internal/engine"
	"github.com/Proviesec/PSFuzz/internal/llm"
	"github.com/Proviesec/PSFuzz/internal/output"
)

// fatal prints err to stderr with a short context and exits with code 1. Used for unrecoverable setup/runtime errors.
func fatal(context string, err error) {
	if err == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "psfuzz: %s: %v\n", context, err)
	os.Exit(1)
}

func main() {
	for _, arg := range os.Args[1:] {
		switch arg {
		case "-h", "--help", "help":
			fmt.Println(config.Help())
			return
		case "-v", "--version", "version":
			fmt.Println("PSFuzz", config.Version)
			return
		}
	}

	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		fatal("config", err)
	}
	if !cfg.Quiet {
		fmt.Fprint(os.Stdout, config.Banner())
		fmt.Fprintf(os.Stdout, "Targets: %s | Concurrency: %d | Wordlist: %s\n", summarizeTargets(cfg), cfg.Concurrency, cfg.Wordlist)
	}
	if cfg.SaveConfigPath != "" {
		if err := config.Save(cfg, cfg.SaveConfigPath); err != nil {
			fatal("save config", err)
		}
		if cfg.URL == "" && len(cfg.URLs) == 0 {
			return
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.ExploreAI {
		probeURL := cfg.URL
		if probeURL == "" && len(cfg.URLs) > 0 {
			probeURL = cfg.URLs[0]
		}
		if !cfg.Quiet {
			fmt.Fprintf(os.Stdout, "Explore AI (%s): probing %s\n", string(llm.NormalizeProviderFromString(cfg.ExploreAIProvider)), probeURL)
		}
		result, err := engine.RunExploreAI(ctx, cfg)
		if err != nil {
			fatal("explore-ai", err)
		}
		if result != nil {
			wordlistPath, ok := engine.ResolveExploreWordlistPath(ctx, cfg, result)
			if !ok {
				if !cfg.Quiet {
					fmt.Fprintf(os.Stdout, "No matching wordlist (map, dir, or default) — run scan manually with suggested command above.\n")
				}
				return
			}
			cfg.Wordlist = wordlistPath
			cfg.Wordlists = nil
			if len(result.Extensions) > 0 {
				cfg.Extensions = result.Extensions
			}
			if len(result.StatusCodes) > 0 {
				cfg.FilterStatus = engine.StatusRangesFromCodes(result.StatusCodes)
			}
			cfg.ExploreAI = false
			if !cfg.Quiet {
				fmt.Fprintf(os.Stdout, "Using wordlist: %s — starting scan.\n", wordlistPath)
			}
		} else {
			return
		}
	}

	runner, err := engine.New(cfg)
	if err != nil {
		fatal("engine", err)
	}
	report, err := runner.Run(ctx)
	if err != nil {
		fatal("scan", err)
	}
	report.Commandline = strings.Join(os.Args, " ")

	if err := output.Write(cfg, report); err != nil {
		fatal("write output", err)
	}

	if cfg.ModuleConfig.ExtractedURLsFile != "" && len(report.ExtractedURLs) > 0 {
		body := strings.Join(report.ExtractedURLs, "\n") + "\n"
		if err := os.WriteFile(cfg.ModuleConfig.ExtractedURLsFile, []byte(body), 0644); err != nil {
			fatal("write extracted URLs file", err)
		}
		if !cfg.Quiet {
			fmt.Printf("Extracted URLs written to %s (%d)\n", cfg.ModuleConfig.ExtractedURLsFile, len(report.ExtractedURLs))
		}
	}

	if !cfg.Quiet {
		fmt.Printf("Scan complete: %d requests, %d findings, duration=%s\n", report.TotalRequests, len(report.Results), report.Duration)
	}
}

func summarizeTargets(cfg *config.Config) string {
	if cfg.URL != "" && len(cfg.URLs) == 0 {
		return cfg.URL
	}
	if len(cfg.URLs) == 1 {
		return cfg.URLs[0]
	}
	return fmt.Sprintf("%d targets", len(cfg.URLs))
}
