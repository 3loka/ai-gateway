package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/trilok/ai-gateway/internal/config"
	"github.com/trilok/ai-gateway/internal/gateway"
	"github.com/trilok/ai-gateway/internal/provider"
	"github.com/trilok/ai-gateway/internal/server"
)

func main() {
	configPath := flag.String("config", "gateway.yaml", "Path to gateway config file")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Build provider map from config.
	providers := buildProviders(cfg)
	if len(providers) == 0 {
		log.Fatal("No providers configured. Check gateway.yaml.")
	}

	gw := gateway.New(cfg, providers)
	srv := server.New(gw, cfg.Server)

	// Graceful shutdown on signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Shutting down...")
		os.Exit(0)
	}()

	log.Printf("ai-gateway starting (providers: %v)", providerNames(providers))
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func buildProviders(cfg *config.Config) map[string]provider.Provider {
	out := make(map[string]provider.Provider)

	for name, pc := range cfg.Providers {
		switch name {
		case "anthropic":
			if pc.APIKey != "" {
				out["anthropic"] = provider.NewAnthropic(pc.APIKey)
				log.Printf("Provider registered: anthropic")
			} else {
				log.Printf("Skipping anthropic: no api_key configured")
			}
		case "openai":
			if pc.APIKey != "" {
				out["openai"] = provider.NewOpenAI(pc.APIKey)
				log.Printf("Provider registered: openai")
			} else {
				log.Printf("Skipping openai: no api_key configured")
			}
		case "ollama":
			out["ollama"] = provider.NewOllama(pc.BaseURL)
			log.Printf("Provider registered: ollama (base_url: %s)", pc.BaseURL)
		default:
			log.Printf("Unknown provider %q in config — skipping", name)
		}
	}

	return out
}

func providerNames(providers map[string]provider.Provider) []string {
	names := make([]string, 0, len(providers))
	for n := range providers {
		names = append(names, n)
	}
	return names
}
