package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/spiffe/go-spiffe/v2/spiffetls"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
)

const listenAddr = ":8443"

func main() {
	ctx := context.Background()
	source := mustFetchSource(ctx)
	defer source.Close()

	svid, err := source.GetX509SVID()
	if err != nil {
		log.Fatalf("failed to fetch server SVID: %v", err)
	}
	log.Printf("Server SVID: %s", svid.ID)

	listener, err := spiffetls.Listen("tcp", listenAddr,
		spiffetls.MTLSServerWithSource(source, tlsconfig.AuthorizeAny()))
	if err != nil {
		log.Fatalf("failed to start mTLS listener: %v", err)
	}

	log.Printf("demo-server listening on %s", listenAddr)
	server := &http.Server{
		ReadHeaderTimeout: 5 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientID := "unknown"
			if peer := r.TLS.PeerCertificates; len(peer) > 0 {
				if peerSVID, err := x509svid.Parse(peer[0]); err == nil {
					clientID = peerSVID.ID.String()
				} else {
					log.Printf("could not parse peer SVID: %v", err)
				}
			}
			log.Printf("Client SPIFFE ID: %s", clientID)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("hello from demo-server\n"))
		}),
	}

	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
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
