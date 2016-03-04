package etcd

import (
	"testing"

	"github.com/Masterminds/cookoo"
	"github.com/coreos/etcd/client"
)

func TestCreateClient(t *testing.T) {

	reg, router, cxt := cookoo.Cookoo()

	reg.Route("test", "Test route").
		Does(CreateClient, "res").Using("url").WithDefault("localhost:4100")

	if err := router.HandleRequest("test", cxt, true); err != nil {
		t.Error(err)
	}

	// All we really want to know is whether we got a valid client back.
	_ = cxt.Get("res", nil).(client.Client)
}

func TestResponse(t *testing.T) {
	// Test that the responses we used to get are still valid.
}

// NOTE MPB: I have removed a bunch of tests because the new Etcd client
// cannot be mocked. The public interfaces import private interfaces which
// we cannot access. The only way it appears that we could mock the Etcd
// client would be to write an HTTP client which would simulate the network
// layer. Perhaps that's an option in the future.
