package frontend

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/mikerybka/twilio"
)

type Server struct {
	TwilioClient *twilio.Client
	AdminPhone   string
	BackendURL   *url.URL
	error502s    int
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	proxy := httputil.NewSingleHostReverseProxy(s.BackendURL)
	proxy.ModifyResponse = func(r *http.Response) error {
		// Notify the admin if the server goes down
		if r.StatusCode == http.StatusBadGateway {
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
			case 1_000_000:
				s.TwilioClient.SendSMS(s.AdminPhone, "A million 502s")
			case 1_000_000_000:
				s.TwilioClient.SendSMS(s.AdminPhone, "A billion 502s")
			case 1_000_000_000_000:
				s.TwilioClient.SendSMS(s.AdminPhone, "A trillion 502s")
			}
		}
		return nil
	}
	proxy.ServeHTTP(w, r)
}
