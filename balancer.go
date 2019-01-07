package sandbox_lb

type Balancer interface {
	FindServerByID(id string) (*server, int)
	NextServer() (*server, error)
	UpsertServer(id string, handler interface{}, options ...ServerOption) error
}
