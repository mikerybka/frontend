package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/mikerybka/twilio"
	"github.com/mikerybka/util"
	"golang.org/x/crypto/acme/autocert"
)

func main() {
	// Fetch certDir from env var
	certDir := os.Getenv("CERT_DIR")
	if certDir == "" {
		log.Fatal("CERT_DIR environment variable is not set")
	}

	// Fetch BACKEND_URL from environment variable
	backendURL := os.Getenv("BACKEND_URL")
	if backendURL == "" {
		log.Fatal("BACKEND_URL environment variable is not set")
	}

	// Parse the backend URL
	parsedURL, err := url.Parse(backendURL)
	if err != nil {
		log.Fatalf("Invalid BACKEND_URL: %v", err)
	}

	// Create a certificate manager
	certManager := autocert.Manager{
		Cache:  autocert.DirCache(certDir),
		Prompt: autocert.AcceptTOS,
		HostPolicy: func(ctx context.Context, host string) error {
			if len(strings.Split(host, ".")) > 3 {
				return fmt.Errorf("host not allowed: %s", host)
			}
			return nil
		},
	}

	// Create the HTTP handler
	h := &Server{
		TwilioClient: &twilio.Client{
			AccountSID:  util.RequireEnvVar("TWILIO_ACCOUNT_SID"),
			AuthToken:   util.RequireEnvVar("TWILIO_AUTH_TOKEN"),
			PhoneNumber: util.RequireEnvVar("TWILIO_PHONE_NUMBER"),
		},
		AdminPhone: util.RequireEnvVar("ADMIN_PHONE"),
		BackendURL: parsedURL,
	}

	h.TwilioClient.SendSMS(h.AdminPhone, "Frontend started.")

	// Create an HTTPS server using autocert
	httpsServer := &http.Server{
		Addr:      ":443",
		Handler:   h,
		TLSConfig: certManager.TLSConfig(),
	}

	// Redirect HTTP to HTTPS
	go func() {
		http.ListenAndServe(":80", certManager.HTTPHandler(nil))
	}()

	// Start the HTTPS server
	log.Println("Starting server on :443")
	if err := httpsServer.ListenAndServeTLS("", ""); err != nil {
		log.Fatalf("Failed to start HTTPS server: %v", err)
	}
}

type Server struct {
	TwilioClient *twilio.Client
	AdminPhone   string
	BackendURL   *url.URL
	error502s    int
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wr := &StatusCodeRecorder{ResponseWriter: w}

	proxy := httputil.NewSingleHostReverseProxy(s.BackendURL)
	proxy.ServeHTTP(wr, r)

	if wr.StatusCode == http.StatusBadGateway {
		s.error502s++
		switch s.error502s {
		case 1:
			s.TwilioClient.SendSMS(s.AdminPhone, "502 from backend")
		case 10:
			s.TwilioClient.SendSMS(s.AdminPhone, "Ten 502s")
		case 100:
			s.TwilioClient.SendSMS(s.AdminPhone, "A hundred 502s")
		case 1000:
			s.TwilioClient.SendSMS(s.AdminPhone, "A thousand 502s")
		}
	}
}

type StatusCodeRecorder struct {
	http.ResponseWriter
	StatusCode int
}

func (rw *StatusCodeRecorder) WriteHeader(code int) {
	rw.StatusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
