package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lsdopen/archy/pkg/inspector"
	"github.com/lsdopen/archy/pkg/webhook"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	var (
		port     string
		certFile string
		keyFile  string
	)

	flag.StringVar(&port, "port", "8443", "Port to listen on")
	flag.StringVar(&certFile, "tls-cert", "/etc/webhook/certs/tls.crt", "Path to TLS certificate")
	flag.StringVar(&keyFile, "tls-key", "/etc/webhook/certs/tls.key", "Path to TLS key")
	flag.Parse()

	// Create in-cluster Kubernetes client for accessing secrets
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Failed to create in-cluster config: %v", err)
	}

	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	inspector := inspector.NewRegistryInspector()
	handler := webhook.NewHandler(inspector, k8sClient)

	mux := http.NewServeMux()
	mux.Handle("/mutate", handler)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("ok")); err != nil {
			log.Printf("Failed to write health check response: %v", err)
		}
	})

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		log.Printf("Starting server on port %s...", port)
		if err := server.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on %s: %v", port, err)
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
