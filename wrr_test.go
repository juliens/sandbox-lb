package sandbox_lb

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
)


func TestLoadBalancer(t *testing.T) {
	handler := func (id string) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			rw.Header().Set("server", id)
			rw.WriteHeader(http.StatusOK)
		})
	}

	type server struct {
		id string
		weight int
	}

	testCases := []struct {
	    desc string
	    servers []server
	    expected []string
	}{
	    {
	        desc: "All server with 1 weight",
			servers: []server{
				{
					id: "test1",
					weight: 1,
				},
				{
					id: "test2",
					weight: 1,
				},
				{
					id: "test3",
					weight: 1,
				},
			},
	        expected: []string{"test1", "test2", "test3"},
	    },
	    {
	        desc: "One server with 0 weight",
			servers: []server{
				{
					id: "test1",
					weight: 1,
				},
				{
					id: "test2",
					weight: 0,
				},
				{
					id: "test3",
					weight: 1,
				},
			},
	        expected: []string{"test1", "test3"},
	    },
	}

	for _, test := range testCases {
	    test := test
	    t.Run(test.desc, func(t *testing.T) {
	        t.Parallel()

			wrr := NewWRRHandler()

			for _, server := range test.servers {
				err := wrr.UpsertServer(server.id, handler(server.id), Weight(server.weight))
				require.NoError(t, err)
			}

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "http://foo", nil)

			var result []string
			for range test.expected {
				wrr.ServeHTTP(recorder, request)
				result = append(result, recorder.Header().Get("server"))
			}
			assert.Equal(t, test.expected, result)
		})
	}


}

func TestSticky(t *testing.T) {
	handler := func (id string) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			rw.Header().Set("server", id)
			rw.WriteHeader(http.StatusOK)
		})
	}

	type server struct {
		id string
		weight int
	}

	testCases := []struct {
		desc string
		servers []server
		expected []string
	}{
		{
			desc: "All server with 1 weight",
			servers: []server{
				{
					id: "test1",
					weight: 1,
				},
				{
					id: "test2",
					weight: 1,
				},
				{
					id: "test3",
					weight: 1,
				},
			},
			expected: []string{"test1", "test1", "test1"},
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			wrr := &StickyLB{
				Balancer: New(),
				cookieName: "test",
			}

			for _, server := range test.servers {
				err := wrr.UpsertServer(server.id, handler(server.id), Weight(server.weight))
				require.NoError(t, err)
			}


			srv := httptest.NewServer(wrr)

			request, err := http.NewRequest(http.MethodGet, srv.URL, nil)
			require.NoError(t, err)

			jar, err := cookiejar.New(nil)
			require.NoError(t, err)

			srv.Client().Jar = jar

			var result []string
			for range test.expected {
				response, err := srv.Client().Do(request)
				require.NoError(t, err)

				assert.Equal(t, http.StatusOK, response.StatusCode)
				result = append(result, response.Header.Get("server"))
			}
			assert.Equal(t, test.expected, result)
		})
	}
}