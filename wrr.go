package sandbox_lb

import (
	"fmt"
	"sync"
)

const defaultWeight = 0

// Weight is an optional functional argument that sets weight of the server
func Weight(w int) ServerOption {
	return func(s *server) error {
		if w < 0 {
			return fmt.Errorf("Weight should be >= 0")
		}
		s.weight = w
		return nil
	}
}

// RoundRobin implements dynamic weighted round robin load balancer http handler
type RoundRobin struct {
	mutex *sync.Mutex
	// Current index (starts from -1)
	index         int
	servers       []*server
	currentWeight int
}

// New created a new RoundRobin
func New() *RoundRobin {
	rr := &RoundRobin{
		index:   -1,
		mutex:   &sync.Mutex{},
		servers: []*server{},
	}

	return rr
}


// NextServer gets the next server
func (r *RoundRobin) NextServer() (*server, error) {
	srv, err := r.nextServer()
	if err != nil {
		return nil, err
	}
	return srv, nil
}

func (r *RoundRobin) nextServer() (*server, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if len(r.servers) == 0 {
		return nil, fmt.Errorf("no servers in the pool")
	}

	// The algo below may look messy, but is actually very simple
	// it calculates the GCD  and subtracts it on every iteration, what interleaves servers
	// and allows us not to build an iterator every time we readjust weights

	// GCD across all enabled servers
	gcd := r.weightGcd()
	// Maximum weight across all enabled servers
	max := r.maxWeight()

	for {
		r.index = (r.index + 1) % len(r.servers)
		if r.index == 0 {
			r.currentWeight = r.currentWeight - gcd
			if r.currentWeight <= 0 {
				r.currentWeight = max
				if r.currentWeight == 0 {
					return nil, fmt.Errorf("all servers have 0 weight")
				}
			}
		}
		srv := r.servers[r.index]
		if srv.weight >= r.currentWeight {
			return srv, nil
		}
	}
}

// RemoveServer remove a server
func (r *RoundRobin) RemoveServer(id string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	e, index := r.FindServerByID(id)
	if e == nil {
		return fmt.Errorf("server not found")
	}
	r.servers = append(r.servers[:index], r.servers[index+1:]...)
	r.resetState()
	return nil
}

// ServerWeight gets the server weight
func (r *RoundRobin) ServerWeight(id string) (int, bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if s, _ := r.FindServerByID(id); s != nil {
		return s.weight, true
	}
	return -1, false
}

// UpsertServer In case if server is already present in the load balancer, returns error
func (r *RoundRobin) UpsertServer(id string, handler interface{}, options ...ServerOption) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if s, _ := r.FindServerByID(id); s != nil {
		for _, o := range options {
			if err := o(s); err != nil {
				return err
			}
		}
		r.resetState()
		return nil
	}

	srv := &server{id: id, server: handler}
	for _, o := range options {
		if err := o(srv); err != nil {
			return err
		}
	}

	if srv.weight == 0 {
		srv.weight = defaultWeight
	}

	r.servers = append(r.servers, srv)
	r.resetState()
	return nil
}

func (r *RoundRobin) resetIterator() {
	r.index = -1
	r.currentWeight = 0
}

func (r *RoundRobin) resetState() {
	r.resetIterator()
}

func (r *RoundRobin) FindServerByID(id string) (*server, int) {
	if len(r.servers) == 0 {
		return nil, -1
	}
	for i, s := range r.servers {
		if s.id == id {
			return s, i
		}
	}
	return nil, -1
}

func (r *RoundRobin) maxWeight() int {
	max := -1
	for _, s := range r.servers {
		if s.weight > max {
			max = s.weight
		}
	}
	return max
}

func (r *RoundRobin) weightGcd() int {
	divisor := -1
	for _, s := range r.servers {
		if divisor == -1 {
			divisor = s.weight
		} else {
			divisor = gcd(divisor, s.weight)
		}
	}
	return divisor
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// ServerOption provides various options for server, e.g. weight
type ServerOption func(*server) error

// Set additional parameters for the server can be supplied when adding server
type server struct {
	id     string
	server interface{}
	// Relative weight for the enpoint to other enpoints in the load balancer
	weight int
}
