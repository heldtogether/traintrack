package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"github.com/heldtogether/traintrack/internal/auth"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

var verbose bool

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with the traintrack backend",
	Run: func(cmd *cobra.Command, args []string) {
		RunLogin()
	},
}

const (
	port    = 42069
	timeout = 2 * time.Minute
)

func init() {
	loginCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Output additional details after login")
	rootCmd.AddCommand(loginCmd)
}

type OAuthResult struct {
	Code  string
	State string
	Err   error
}

func RunLogin() {
	state := uuid.NewString()
	ctx := context.Background()

	loadedConfig, err := auth.LoadConfig(auth.DefaultConfigPath)
	if err != nil {
		log.Fatalf("couldn't load config: %s", err.Error())
	}

	p, err := oidc.NewProvider(ctx, loadedConfig.AuthURL)
	if err != nil {
		log.Fatalf("failed to create OIDC provider: %s", err.Error())
	}

	config := &oauth2.Config{
		ClientID:    loadedConfig.ClientID,
		RedirectURL: fmt.Sprintf("http://localhost:%d/auth/callback", port),
		Scopes:      []string{"openid", "profile", "email", "offline_access"},
		Endpoint:    p.Endpoint(),
	}

	authURL := config.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("audience", loadedConfig.Name),
		oauth2.SetAuthURLParam("prompt", "consent"),
	)

	resultChan := startCallbackServer(port, timeout)

	if err := openBrowser(authURL); err != nil {
		log.Fatalf("failed to open browser: %s", err.Error())
	}

	result := <-resultChan
	if result.Err != nil {
		log.Fatalf("login failed: %s", result.Err.Error())
	}

	if result.State != state {
		log.Fatalf("state mismatch: potential CSRF")
	}

	token, err := config.Exchange(ctx, result.Code)
	if err != nil {
		log.Fatalf("token exchange failed: %s", err.Error())
	}

	auth.SaveToken(auth.DefaultTokenPath, token)

	if verbose {
		fmt.Println("Access Token:", token.AccessToken)
	}

	idTokenRaw, ok := token.Extra("id_token").(string)
	if ok {
		verifier := p.Verifier(&oidc.Config{ClientID: loadedConfig.ClientID})
		idToken, err := verifier.Verify(ctx, idTokenRaw)
		if err != nil {
			log.Fatalf("failed to verify ID token: %s", err.Error())
		}

		var claims struct {
			Email   string `json:"email"`
			Name    string `json:"name"`
			Picture string `json:"picture"`
		}

		if err := idToken.Claims(&claims); err != nil {
			log.Fatalf("failed to decode claims: %s", err.Error())
		}

		fmt.Println("Logged in as:", claims.Name, "<"+claims.Email+">")
	}
}

func startCallbackServer(port int, timeout time.Duration) <-chan OAuthResult {
	resultChan := make(chan OAuthResult, 1)
	mux := http.NewServeMux()

	srv := &http.Server{
		Addr:    fmt.Sprintf("localhost:%d", port),
		Handler: mux,
	}

	mux.HandleFunc(
		"/auth/callback",
		func(w http.ResponseWriter, r *http.Request) {
			code := r.URL.Query().Get("code")
			state := r.URL.Query().Get("state")

			if code == "" || state == "" {
				http.Error(w, "Missing code or state", http.StatusBadRequest)
				resultChan <- OAuthResult{Err: errors.New("missing code or state")}
				return
			}

			io.WriteString(w, "Login successful. You may close this window.")
			resultChan <- OAuthResult{Code: code, State: state}

			// Graceful shutdown
			go func() { _ = srv.Shutdown(context.Background()) }()
		})

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		go func() {
			<-ctx.Done()
			resultChan <- OAuthResult{Err: errors.New("timeout waiting for login")}
			_ = srv.Shutdown(context.Background())
		}()

		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			resultChan <- OAuthResult{Err: fmt.Errorf("server error: %w", err)}
		}
	}()

	return resultChan
}

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		// macOS
		cmd = "open"
		args = []string{url}
	case "windows":
		// Windows uses 'start', but must be run through 'cmd'
		cmd = "cmd"
		args = []string{"/c", "start", url}
	default:
		// Linux and other Unix-like systems
		cmd = "xdg-open"
		args = []string{url}
	}

	return exec.Command(cmd, args...).Start()
}
