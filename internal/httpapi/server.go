package httpapi

import "net/http"

type Server struct {
	h *Handlers
}

func New(deps Deps) *Server {
	return &Server{h: NewHandlers(deps)}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("/health", s.h.Health)
	mux.HandleFunc("/plan", s.h.Plan)
	mux.HandleFunc("/apply", s.h.Apply)

	mux.HandleFunc("/ndpi/enable", s.h.NDPIEnable)
	mux.HandleFunc("/ndpi/disable", s.h.NDPIDisable)
}
