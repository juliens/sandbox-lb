package sandbox_lb

import "net/http"

type RoundRobinHandler struct {
	*RoundRobin
}

func NewWRRHandler() RoundRobinHandler {
	return RoundRobinHandler{
		RoundRobin: New(),
	}
}

func (r *RoundRobinHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	server, err := r.NextServer()
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

	handler.ServeHTTP(w, req)
}
