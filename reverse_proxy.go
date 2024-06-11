package proxy

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/gonobo/jsonapi"
	"github.com/gonobo/jsonapi/response"
)

// ReverseProxyHandler is a handler that forwards client requests to the JSON:API server at the given base URL.
type ReverseProxyHandler struct {
	jsonapi.URLResolver              // The URL resolver.
	BaseURL             string       // The base URL of the JSON:API server.
	Client              jsonapi.Doer // The HTTP client.
}

// NewReverseProxyHandler creates a new reverse proxy handler. This handler forwards client requests
// to the JSON:API server at the given base URL.
//
// By default, NewReverseProxyHandler generates the following urls (given a base url):
//
//	":base/:type" for search, create
//	":base/:type/:id" for fetch, update, delete
//	":base/:type/:id/relationships/:ref" for fetchRef
//	":base/:type/:id/:ref" for fetchRelated
//
// This behavior can be modified by updating the URLResolver field of a
// ReverseProxyHandler instance.
func NewReverseProxyHandler(baseURL string, options ...func(*ReverseProxyHandler)) ReverseProxyHandler {
	handler := ReverseProxyHandler{
		URLResolver: jsonapi.DefaultURLResolver(),
		BaseURL:     baseURL,
		Client:      http.DefaultClient,
	}

	for _, option := range options {
		option(&handler)
	}

	return handler
}

// ServeJSONAPI forwards the request to the JSON:API server at the given base URL.
func (p ReverseProxyHandler) ServeJSONAPI(r *http.Request) jsonapi.Response {
	ctx, _ := jsonapi.GetContext(r.Context())

	serverURL, err := url.Parse(p.ResolveURL(*ctx, p.BaseURL))
	if err != nil {
		return response.InternalError(err)
	}

	request := r.Clone(context.Background())
	request.URL = serverURL

	res, err := p.Client.Do(request)
	if err != nil {
		return response.InternalError(err)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return response.InternalError(err)
	}

	jsonapiResponse := jsonapi.NewResponse(res.StatusCode)

	for key, value := range res.Header {
		jsonapiResponse.Headers[key] = value[0]
	}

	if len(data) > 0 {
		var doc jsonapi.Document
		err := json.Unmarshal(data, &doc)
		if err != nil {
			return response.InternalError(err)
		}
		jsonapiResponse.Body = &doc
	}

	return jsonapiResponse
}
