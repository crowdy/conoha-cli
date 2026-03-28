package api

import "net/http/httptest"

func newTestClient(ts *httptest.Server) *Client {
	return &Client{HTTP: ts.Client(), Token: "test-token", TenantID: "test-tenant"}
}
