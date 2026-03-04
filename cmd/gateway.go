package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/pltanton/lingti-bot/internal/agent"
	"github.com/pltanton/lingti-bot/internal/config"
	"github.com/pltanton/lingti-bot/internal/gateway"
	"github.com/pltanton/lingti-bot/internal/logger"
	"github.com/pltanton/lingti-bot/internal/router"
	"github.com/spf13/cobra"
)

var (
	gatewayAddr       string
	gatewayAuthToken  string
	gatewayAuthTokens []string
)

var gatewayCmd = &cobra.Command{
	Use:   "gateway",
	Short: "Start the WebSocket gateway",
	Long: `Start the WebSocket gateway for real-time AI interaction.

The gateway provides a WebSocket interface on localhost:18789 (configurable)
for clients to connect and interact with the AI assistant in real-time.

This enables:
  - CLI clients
  - Web UIs
  - Mobile companion apps
  - Custom integrations

Environment variables:
  - GATEWAY_ADDR: Address to listen on (default: :18789)
  - GATEWAY_AUTH_TOKEN: Single authentication token
  - GATEWAY_AUTH_TOKENS: Comma-separated list of authentication tokens (multiple admins)
  - AI_API_KEY: API Key for the AI provider`,
	Run: runGateway,
}

func init() {
	rootCmd.AddCommand(gatewayCmd)

	gatewayCmd.Flags().StringVar(&gatewayAddr, "addr", "", "Gateway address (or GATEWAY_ADDR env, default: :18789)")
	gatewayCmd.Flags().StringVar(&gatewayAuthToken, "auth-token", "", "Single authentication token (or GATEWAY_AUTH_TOKEN env)")
	gatewayCmd.Flags().StringSliceVar(&gatewayAuthTokens, "auth-tokens", nil, "Multiple authentication tokens, comma-separated (or GATEWAY_AUTH_TOKENS env)")
	gatewayCmd.Flags().StringVar(&aiProvider, "provider", "", "AI provider: claude, deepseek, kimi, qwen (or AI_PROVIDER env)")
	gatewayCmd.Flags().StringVar(&aiAPIKey, "api-key", "", "AI API Key (or AI_API_KEY env)")
	gatewayCmd.Flags().StringVar(&aiBaseURL, "base-url", "", "AI API base URL (or AI_BASE_URL env)")
	gatewayCmd.Flags().StringVar(&aiModel, "model", "", "Model name (or AI_MODEL env)")
}

func runGateway(cmd *cobra.Command, args []string) {
	// Get config from flags or environment
	if gatewayAddr == "" {
		gatewayAddr = os.Getenv("GATEWAY_ADDR")
		if gatewayAddr == "" {
			gatewayAddr = ":18789"
		}
	}
	if gatewayAuthToken == "" {
		gatewayAuthToken = os.Getenv("GATEWAY_AUTH_TOKEN")
	}
	if len(gatewayAuthTokens) == 0 {
		if v := os.Getenv("GATEWAY_AUTH_TOKENS"); v != "" {
			for _, t := range strings.Split(v, ",") {
				if t = strings.TrimSpace(t); t != "" {
					gatewayAuthTokens = append(gatewayAuthTokens, t)
				}
			}
		}
	}
	if aiProvider == "" {
		aiProvider = os.Getenv("AI_PROVIDER")
	}
	if aiAPIKey == "" {
		aiAPIKey = os.Getenv("AI_API_KEY")
		if aiAPIKey == "" {
			aiAPIKey = os.Getenv("ANTHROPIC_API_KEY")
		}
	}
	if aiModel == "" {
		aiModel = os.Getenv("AI_MODEL")
		if aiModel == "" {
			aiModel = os.Getenv("ANTHROPIC_MODEL")
		}
	}
	if aiBaseURL == "" {
		aiBaseURL = os.Getenv("AI_BASE_URL")
		if aiBaseURL == "" {
			aiBaseURL = os.Getenv("ANTHROPIC_BASE_URL")
		}
	}

	// Load ~/.lingti.yaml and resolve named provider
	if savedCfg, cfgErr := config.Load(); cfgErr == nil {
		providerRef := aiProvider
		if resolved, found := savedCfg.ResolveProvider(providerRef); found {
			if aiProvider == "" {
				aiProvider = resolved.Provider
			}
			if aiAPIKey == "" {
				aiAPIKey = resolved.APIKey
			}
			if aiBaseURL == "" {
				aiBaseURL = resolved.BaseURL
			}
			if aiModel == "" {
				aiModel = resolved.Model
			}
		}
	}

	if aiAPIKey == "" {
		fmt.Fprintln(os.Stderr, "Error: AI_API_KEY is required")
		os.Exit(1)
	}

	// Create the AI agent
	aiAgent, err := agent.New(agent.Config{
		Provider:     aiProvider,
		APIKey:       aiAPIKey,
		BaseURL:      aiBaseURL,
		Model:        aiModel,
		AllowedPaths:     loadAllowedPaths(),
		DisableFileTools: loadDisableFileTools(),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating agent: %v\n", err)
		os.Exit(1)
	}

	// Create the gateway
	gw := gateway.New(gateway.Config{
		Addr:       gatewayAddr,
		AuthToken:  gatewayAuthToken,  // single token (backward-compat)
		AuthTokens: gatewayAuthTokens, // multiple tokens
	})

	// Set up message handler that wraps agent responses for streaming
	gw.SetMessageHandler(func(ctx context.Context, clientID, sessionID, text string) (<-chan gateway.ResponsePayload, error) {
		respChan := make(chan gateway.ResponsePayload, 1)

		go func() {
			defer close(respChan)

			// Create a router.Message for the agent
			msg := router.Message{
				ID:        sessionID,
				Platform:  "gateway",
				ChannelID: clientID,
				UserID:    clientID,
				Username:  "gateway-user",
				Text:      text,
				Metadata:  map[string]string{"session_id": sessionID},
			}

			response, err := aiAgent.HandleMessage(ctx, msg)

			if err != nil {
				respChan <- gateway.ResponsePayload{
					Text:      fmt.Sprintf("Error: %v", err),
					SessionID: sessionID,
					Done:      true,
				}
				return
			}

			respChan <- gateway.ResponsePayload{
				Text:      response.Text,
				SessionID: sessionID,
				Done:      true,
			}
		}()

		return respChan, nil
	})

	// Start the gateway
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := gw.Start(ctx); err != nil {
			logger.Error("Gateway error: %v", err)
		}
	}()

	logger.Info("Gateway started on %s", gatewayAddr)
	total := len(gatewayAuthTokens)
	if gatewayAuthToken != "" {
		total++
	}
	if total > 0 {
		logger.Info("Authentication enabled (%d token(s))", total)
	}
	logger.Info("Press Ctrl+C to stop.")

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("Shutting down...")
	gw.Stop()
}
