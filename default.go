// @author Couchbase <info@couchbase.com>
// @copyright 2014 Couchbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cbauth

import (
	"errors"
	"fmt"
	"net/http"
	"net/rpc"
	"time"

	"github.com/couchbase/cbauth/cbauthimpl"
	"github.com/couchbase/cbauth/revrpc"
)

// Default variable holds default authenticator. Default authenticator
// is constructed automatically from environment variables passed by
// ns_server. It is nil if your process was not (correctly) spawned by
// ns_server.
var Default Authenticator

func startDefault(rpcsvc *revrpc.Service) {
	svc := cbauthimpl.NewSVC(5 * time.Second)
	Default = &authImpl{svc}
	go func() {
		defPolicy := revrpc.DefaultBabysitErrorPolicy.New()
		// error restart policy that we're going to use simply
		// resets service before delegating to default restart
		// policy. That way we always mark service as stale
		// right after some error occurred.
		cbauthPolicy := func(err error) error {
			cbauthimpl.ResetSvc(svc)
			return defPolicy(err)
		}
		err := revrpc.BabysitService(func(s *rpc.Server) error {
			return s.RegisterName("AuthCacheSvc", svc)
		}, rpcsvc, revrpc.FnBabysitErrorPolicy(cbauthPolicy))
		panic(err)
	}()
}

func init() {
	rpcsvc, err := revrpc.GetDefaultServiceFromEnv("cbauth")
	if err != nil {
		ErrNotInitialized = fmt.Errorf("Unable to initialize cbauth's revrpc: %s", err)
		return
	}
	startDefault(rpcsvc)
}

// ErrNotInitialized is used to signal that ns_server environment
// variables are not set, and thus Default authenticator is not
// configured for calls that use default authenticator.
var ErrNotInitialized = errors.New("cbauth was not initialized")

// WithDefault calls given body with default authenticator. If default
// authenticator is not configured, it returns ErrNotInitialized.
func WithDefault(body func(a Authenticator) error) error {
	return WithAuthenticator(nil, body)
}

// WithAuthenticator calls given body with either passed authenticator
// or default authenticator if `a' is nil. ErrNotInitialized is
// returned if a is nil and default authenticator is not configured.
func WithAuthenticator(a Authenticator, body func(a Authenticator) error) error {
	if a == nil {
		a = Default
		if a == nil {
			return ErrNotInitialized
		}
	}
	return body(a)
}

// AuthWebCreds method extracts credentials from given http request
// using default authenticator.
func AuthWebCreds(req *http.Request) (creds Creds, err error) {
	if Default == nil {
		return nil, ErrNotInitialized
	}
	return Default.AuthWebCreds(req)
}

// Auth method constructs credentials from given user and password
// pair. Uses default authenticator.
func Auth(user, pwd string) (creds Creds, err error) {
	if Default == nil {
		return nil, ErrNotInitialized
	}
	return Default.Auth(user, pwd)
}

// GetHTTPServiceAuth returns user/password creds giving "admin"
// access to given http service inside couchbase cluster. Uses default
// authenticator.
func GetHTTPServiceAuth(hostport string) (user, pwd string, err error) {
	if Default == nil {
		return "", "", ErrNotInitialized
	}
	return Default.GetHTTPServiceAuth(hostport)
}

// GetMemcachedServiceAuth returns user/password creds given "admin"
// access to given memcached service. Uses default authenticator.
func GetMemcachedServiceAuth(hostport string) (user, pwd string, err error) {
	if Default == nil {
		return "", "", ErrNotInitialized
	}
	return Default.GetMemcachedServiceAuth(hostport)
}
