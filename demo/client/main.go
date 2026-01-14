package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
)

const defaultServerAddr = "demo-server:8443"

func main() {
	ctx := context.Background()
	source := mustFetchSource(ctx)
	defer source.Close()

	svid, err := source.GetX509SVID()
	if err != nil {
		log.Fatalf("failed to fetch client SVID: %v", err)
	}
	log.Printf("Obtained SVID: %s", svid.ID)

	serverID, err := spiffeid.FromString("spiffe://demo.org/workload/demo-server")
	if err != nil {
		log.Fatalf("invalid server SPIFFE ID: %v", err)
	}

	tlsCfg := tlsconfig.MTLSClientConfig(source, source, tlsconfig.AuthorizeID(serverID))
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsCfg,
		},
		Timeout: 10 * time.Second,
	}

	serverAddr := os.Getenv("SERVER_ADDR")
	if serverAddr == "" {
		serverAddr = defaultServerAddr
	}

	url := fmt.Sprintf("https://%s", serverAddr)
	resp, err := client.Get(url)
	if err != nil {
		log.Fatalf("mTLS request failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	log.Printf("mTLS request succeeded (status %s)", resp.Status)
	if len(bytes.TrimSpace(body)) > 0 {
		log.Printf("Response body: %s", bytes.TrimSpace(body))
	}
}

func mustFetchSource(ctx context.Context) *workloadapi.X509Source {
	socketPath := os.Getenv("SPIFFE_ENDPOINT_SOCKET")
	if socketPath == "" {
		log.Fatal("SPIFFE_ENDPOINT_SOCKET is not set")
	}

	var source *workloadapi.X509Source
	var err error
	for i := 0; i < 15; i++ {
		source, err = workloadapi.NewX509Source(ctx,
			workloadapi.WithClientOptions(workloadapi.WithAddr(socketPath)))
		if err == nil {
			return source
		}
		log.Printf("waiting for Workload API (attempt %d): %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	log.Fatalf("giving up connecting to Workload API: %v", err)
	return nil
}
