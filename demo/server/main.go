package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"log"
	"net/http"
	"os"
	"time"
	"net"

	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
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

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
	    log.Fatalf("failed to listen: %v", err)
	}

	log.Printf("demo-server listening on %s", listenAddr)
	server := &http.Server{
	    ReadHeaderTimeout: 5 * time.Second,
	    TLSConfig: tlsconfig.MTLSServerConfig(
	        source, // サーバ証明書/鍵
	        source, // trust bundle
	        tlsconfig.AuthorizeAny(), // 後で AuthorizeID にするの推奨
	    ),
	    Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	        log.Printf("Client SPIFFE ID: %s", clientSPIFFEID(r.TLS))
	        w.WriteHeader(http.StatusOK)
	        _, _ = w.Write([]byte("hello from demo-server\n"))
	    }),
	}

	// certFile/keyFile は TLSConfig 内にあるので空でOK
	if err := server.ServeTLS(listener, "", ""); err != nil && err != http.ErrServerClosed {
	    log.Fatalf("server error: %v", err)
	}
}

func clientSPIFFEID(state *tls.ConnectionState) string {
	if state == nil {
		return "unknown: no TLS state"
	}

	cert := clientCert(state)
	if cert == nil {
		return "unknown: no client certificate"
	}

	for _, uri := range cert.URIs {
		if uri != nil && uri.Scheme == "spiffe" {
			return uri.String()
		}
	}

	return "unknown: no SPIFFE ID in client certificate"
}

func clientCert(state *tls.ConnectionState) *x509.Certificate {
	if len(state.VerifiedChains) > 0 && len(state.VerifiedChains[0]) > 0 {
		return state.VerifiedChains[0][0]
	}
	if len(state.PeerCertificates) > 0 {
		return state.PeerCertificates[0]
	}
	return nil
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
