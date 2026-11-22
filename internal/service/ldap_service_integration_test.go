//go:build ldap_integration

package service

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/tinyauthapp/tinyauth/internal/model"
)

// Regression test for tinyauthapp/tinyauth#1216: a password check must never leave the
// shared LDAP connection bound as the user being checked, or a concurrent search runs as
// that user and, in a directory where users see only their own entry (LLDAP), finds
// nothing.
//
// It needs a real LLDAP server holding two ordinary users, which CI provides
// (.github/workflows/gp-image.yml). Run with:
//
//	LDAP_TEST_ADDRESS=ldap://localhost:3890 LDAP_TEST_ADMIN_PASSWORD=… \
//	LDAP_TEST_USER_PASSWORD=… go test -tags ldap_integration -race ./internal/service/
func TestLDAPPasswordCheckDoesNotLeakIntoSearches(t *testing.T) {
	address := os.Getenv("LDAP_TEST_ADDRESS")
	if address == "" {
		t.Skip("LDAP_TEST_ADDRESS not set")
	}
	baseDN := envOr("LDAP_TEST_BASE_DN", "dc=example,dc=com")
	userPassword := os.Getenv("LDAP_TEST_USER_PASSWORD")

	config := &model.Config{}
	config.LDAP = model.LDAPConfig{
		Address:      address,
		BindDN:       fmt.Sprintf("uid=admin,ou=people,%s", baseDN),
		BindPassword: os.Getenv("LDAP_TEST_ADMIN_PASSWORD"),
		BaseDN:       baseDN,
		Insecure:     true,
		SearchFilter: "(uid=%s)",
	}
	ldap := &LdapService{
		config: config,
		ctx:    context.Background(),
		bindPw: config.LDAP.BindPassword,
	}
	if _, err := ldap.connect(); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer ldap.conn.Close()

	auth := &AuthService{config: config, runtime: &model.RuntimeConfig{}, ldap: ldap}

	alice, err := auth.SearchUser("alice")
	if err != nil {
		t.Fatalf("search alice before the race: %v", err)
	}

	const rounds = 100
	var wg sync.WaitGroup
	var passwordFailures, searchFailures atomic.Int64
	var firstSearchErr atomic.Value

	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range rounds {
				if err := auth.CheckUserPassword(*alice, userPassword); err != nil {
					passwordFailures.Add(1)
				}
			}
		}()
	}
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range rounds {
				if _, err := auth.SearchUser("bob"); err != nil {
					searchFailures.Add(1)
					firstSearchErr.CompareAndSwap(nil, err.Error())
				}
			}
		}()
	}
	wg.Wait()

	if n := passwordFailures.Load(); n != 0 {
		t.Errorf("%d of %d correct password checks failed", n, 4*rounds)
	}
	if n := searchFailures.Load(); n != 0 {
		t.Errorf("%d of %d searches for an existing user failed, first: %v", n, 4*rounds, firstSearchErr.Load())
	}

	// The password check must still refuse a wrong password.
	if err := auth.CheckUserPassword(*alice, userPassword+"-wrong"); err == nil {
		t.Error("a wrong password was accepted")
	}
}

func envOr(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}
