package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/maxkorbacher/vibed/internal/builder"
	"github.com/maxkorbacher/vibed/internal/config"
	"github.com/maxkorbacher/vibed/internal/deployer"
	"github.com/maxkorbacher/vibed/internal/environment"
	"github.com/maxkorbacher/vibed/internal/frontend"
	"github.com/maxkorbacher/vibed/internal/k8s"
	mcppkg "github.com/maxkorbacher/vibed/internal/mcp"
	"github.com/maxkorbacher/vibed/internal/orchestrator"
	"github.com/maxkorbacher/vibed/internal/storage"
	"github.com/maxkorbacher/vibed/internal/store"
	"github.com/maxkorbacher/vibed/pkg/api"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	knversioned "knative.dev/serving/pkg/client/clientset/versioned"
)

func main() {
	var (
		configPath string
		transport  string
	)
	flag.StringVar(&configPath, "config", "", "Path to vibed.yaml config file")
	flag.StringVar(&transport, "transport", "", "Override transport: stdio, http, or both")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load(configPath)
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	if transport != "" {
		cfg.Server.Transport = transport
	}

	logger.Info("starting vibeD",
		"transport", cfg.Server.Transport,
		"namespace", cfg.Deployment.Namespace,
		"storage", cfg.Storage.Backend,
	)

	// Initialize Kubernetes clients
	k8sClients, err := k8s.NewClients(cfg.Kubernetes)
	if err != nil {
		logger.Error("failed to create k8s clients", "error", err)
		os.Exit(1)
	}

	// Initialize subsystems
	detector := environment.NewDetector(k8sClients, logger)

	bldr := builder.NewPackBuilder(cfg.Builder, logger)

	// Initialize storage
	var stg storage.Storage
	switch cfg.Storage.Backend {
	case "local":
		stg, err = storage.NewLocalStorage(cfg.Storage.Local.BasePath)
		if err != nil {
			logger.Error("failed to create local storage", "error", err)
			os.Exit(1)
		}
	case "github":
		stg, err = storage.NewGitHubStorage(
			cfg.Storage.GitHub.Owner,
			cfg.Storage.GitHub.Repo,
			cfg.Storage.GitHub.Branch,
			"", // Token from GITHUB_TOKEN env var
			cfg.Storage.Local.BasePath, // Local cache dir
		)
		if err != nil {
			logger.Error("failed to create GitHub storage", "error", err)
			os.Exit(1)
		}
	default:
		logger.Error("unsupported storage backend", "backend", cfg.Storage.Backend)
		os.Exit(1)
	}

	// Initialize artifact store
	var st store.ArtifactStore
	switch cfg.Store.Backend {
	case "memory":
		st = store.NewMemoryStore()
	case "configmap":
		st = store.NewConfigMapStore(
			k8sClients.Clientset,
			cfg.Store.ConfigMap.Name,
			cfg.Store.ConfigMap.Namespace,
		)
	default:
		logger.Error("unsupported store backend", "backend", cfg.Store.Backend)
		os.Exit(1)
	}

	// Initialize deployers
	factory := deployer.NewFactory()

	// Register Knative deployer
	knClient, err := knversioned.NewForConfig(k8sClients.RestConfig)
	if err != nil {
		logger.Warn("failed to create Knative client (Knative may not be installed)", "error", err)
	} else {
		knDeployer := deployer.NewKnativeDeployer(knClient, k8sClients.Clientset, cfg.Deployment, cfg.Knative, logger)
		factory.Register(api.TargetKnative, knDeployer)
	}

	// Register Kubernetes deployer
	k8sDeployer := deployer.NewKubernetesDeployer(k8sClients.Clientset, cfg.Deployment, logger)
	factory.Register(api.TargetKubernetes, k8sDeployer)

	// Register wasmCloud deployer
	wasmDeployer := deployer.NewWasmCloudDeployer(k8sClients.DynamicClient, k8sClients.Clientset, cfg.Deployment, logger)
	factory.Register(api.TargetWasmCloud, wasmDeployer)

	// Create orchestrator
	orch := orchestrator.NewOrchestrator(cfg, detector, bldr, factory, stg, st, logger)

	// Create MCP server
	mcpServer := mcppkg.NewServer(orch)

	// Run based on transport mode
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	switch cfg.Server.Transport {
	case "stdio":
		logger.Info("starting MCP server on stdio")
		if err := mcpServer.Run(ctx, &mcp.StdioTransport{}); err != nil {
			logger.Error("stdio server error", "error", err)
			os.Exit(1)
		}

	case "http":
		runHTTPServer(ctx, cfg, mcpServer, orch, logger)

	case "both":
		go runHTTPServer(ctx, cfg, mcpServer, orch, logger)
		logger.Info("starting MCP server on stdio")
		if err := mcpServer.Run(ctx, &mcp.StdioTransport{}); err != nil {
			logger.Error("stdio server error", "error", err)
			os.Exit(1)
		}

	default:
		logger.Error("unknown transport", "transport", cfg.Server.Transport)
		os.Exit(1)
	}
}

func runHTTPServer(ctx context.Context, cfg *config.Config, mcpServer *mcp.Server, orch *orchestrator.Orchestrator, logger *slog.Logger) {
	mux := http.NewServeMux()

	// MCP HTTP endpoint
	mcpHandler := mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server { return mcpServer },
		nil,
	)
	mux.Handle("/mcp/", mcpHandler)

	// Frontend + API
	frontendHandler := frontend.NewHandler(orch)
	mux.Handle("/", frontendHandler)

	server := &http.Server{
		Addr:    cfg.Server.HTTPAddr,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		logger.Info("shutting down HTTP server")
		server.Close()
	}()

	logger.Info("starting HTTP server", "addr", cfg.Server.HTTPAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "HTTP server error: %v\n", err)
		os.Exit(1)
	}
}
