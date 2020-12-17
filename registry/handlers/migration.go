package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	dcontext "github.com/docker/distribution/context"
	"github.com/docker/distribution/registry/api/errcode"
	"github.com/docker/distribution/registry/storage"
)

type migrationHandler struct {
	*Context
	fallback http.Handler
}

func migrationWrapper(ctx *Context, h http.Handler) http.Handler {
	if !ctx.App.Config.Migration.Proxy.Enabled {
		return h
	}

	mh := migrationHandler{Context: ctx, fallback: h}
	return http.HandlerFunc(mh.proxyNewRepositories)
}

func (h migrationHandler) proxyNewRepositories(rw http.ResponseWriter, req *http.Request) {
	// h.Repository is a notifications.repositoryListener and not a storage.repository. We need the latter to be able to
	// use the storage.RepositoryValidator interface and validate if the repository exist, so we have to build one here.
	repo, err := h.registry.Repository(h.Context, h.Repository.Named())
	if err != nil {
		h.Errors = append(h.Errors, fmt.Errorf("unable to build storage.repository from notifications.repositoryListener: %w", err))
		return
	}
	validator, ok := repo.(storage.RepositoryValidator)
	if !ok {
		h.Errors = append(h.Errors, errors.New("repository does not implement RepositoryValidator interface"))
		return
	}

	// check if repository exists in this instance's storage backend, proxy to target registry if not
	exists, err := validator.Exists(h)
	if err != nil {
		h.Errors = append(h.Errors, fmt.Errorf("unable to determine if repository exists: %w", err))
		return
	}
	log := dcontext.GetLogger(h)
	if exists {
		log.Debug("known repository, request will be served")
		h.fallback.ServeHTTP(rw, req)
		return
	}

	// evaluate inclusion filters, if any
	if len(h.App.Config.Migration.Proxy.Include) > 0 {
		var proxy bool
		for _, r := range h.App.Config.Migration.Proxy.Include {
			if r.MatchString(repo.Named().String()) {
				proxy = true
			}
		}
		if !proxy {
			log.Debug("repository name does not match any inclusion filter, request will be served by this registry")
			h.fallback.ServeHTTP(rw, req)
			return
		}
	}

	targetURL := h.App.Config.Migration.Proxy.URL
	log = log.WithField("url", targetURL)
	log.Info("unknown repository, forwarding to target migration registry")

	u, err := url.Parse(targetURL)
	if err != nil {
		h.Errors = append(h.Errors, fmt.Errorf("invalid target registry URL: %w", err))
		return
	}

	// remove any custom headers already added to the response writer, let the target registry add them instead (otherwise
	// we'll end up with duplicated headers in the response)
	resHeaderBkp := rw.Header().Clone()
	rw.Header().Del("Docker-Distribution-API-Version")
	for k, _ := range h.App.Config.HTTP.Headers {
		rw.Header().Del(k)
	}

	proxy := httputil.NewSingleHostReverseProxy(u)

	// modify request before forwarding
	defaultDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		// let default Director update the request URL Scheme, Host, and Path
		defaultDirector(req)
		// proxy.ServeHTTP will set X-Forwarded-For for us, but we should also set X-Forwarded-Host
		req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
		req.Host = req.URL.Host
	}

	// handle errors reaching the target registry
	proxy.ErrorHandler = func(_ http.ResponseWriter, _ *http.Request, err error) {
		// restore response headers
		for k, v := range resHeaderBkp {
			rw.Header().Set(k, strings.Join(v, ", "))
		}
		log.WithError(err).Error("error proxying request to target registry")
		h.Errors = append(h.Errors, errcode.ErrorCodeUnavailable)
		if err := errcode.ServeJSON(rw, h.Errors); err != nil {
			log.WithError(err).Error("error serving error json")
		}
	}

	proxy.ServeHTTP(rw, req)
}
