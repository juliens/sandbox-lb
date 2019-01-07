package sandbox_lb

import (
	"net/http"
)

type StickyLB struct {
	Balancer
	cookieName string
}

// GetBackend returns the backend URL stored in the sticky cookie, iff the backend is still in the valid list of servers.
func (s *StickyLB) GetBackend(req *http.Request) (*server, error) {
	cookie, err := req.Cookie(s.cookieName)
	if err != nil {
		nextServer, err := s.Balancer.NextServer()
		if err != nil {
			return nil, err
		}
		return nextServer, nil
	}

	srv, idx := s.Balancer.FindServerByID(cookie.Value)
	if idx == -1 {
		nextServer, err := s.Balancer.NextServer()
		if err != nil {
			return nil, err
		}
		return nextServer, nil
	}
	return srv, nil
}

// StickBackend creates and sets the cookie
func (s *StickyLB) StickBackend(id string, w http.ResponseWriter) {
	cookie := &http.Cookie{Name: s.cookieName, Value: id, Path: "/"}
	http.SetCookie(w, cookie)
}

func (r *StickyLB) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	server, err := r.GetBackend(req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		w.WriteHeader(statusCode)
		w.Write([]byte(http.StatusText(statusCode) + err.Error()))
		return
	}

	handler, ok := server.server.(http.Handler)
	if !ok {
		statusCode := http.StatusInternalServerError
		w.WriteHeader(statusCode)
		w.Write([]byte(http.StatusText(statusCode)))
		return
	}
	r.StickBackend(server.id, w)
	handler.ServeHTTP(w, req)

}