package discovery

import (
	"github.com/Masterminds/cookoo"
	"testing"
)

func TestToken(t *testing.T) {
	orig := TokenFile
	defer func() {
		TokenFile = orig
	}()

	TokenFile = "testdata/secret.txt"

	tok, err := Token()
	if err != nil {
		t.Errorf("Error getting token: %s", err)
	}

	if string(tok) != "c0ff33" {
		t.Errorf("Expected c0ff33, got '%s'", string(tok))
	}

}

func TestFoo(t *testing.T) {
	orig := TokenFile
	defer func() {
		TokenFile = orig
	}()

	TokenFile = "testdata/secret.txt"

	reg, router, cxt := cookoo.Cookoo()

	reg.Route("test", "Test route").
		Does(GetToken, "res")

	if err := router.HandleRequest("test", cxt, true); err != nil {
		t.Error(err)
	}

	v := cxt.Get("res", "").(string)
	if v != "c0ff33" {
		t.Errorf("Expected c0ff33, got '%s'", v)
	}
}
