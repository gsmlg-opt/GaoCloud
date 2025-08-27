package gorest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	ut "cement/unittest"
	goresterr "gorest/error"
	"gorest/resource"
	"gorest/resource/schema"
)

var (
	schemas = schema.NewSchemaManager()
	version = resource.APIVersion{
		Group:   "testing",
		Version: "v1",
	}
)

type dumbHandler struct{}

func (h *dumbHandler) Create(ctx *resource.Context) (resource.Resource, *goresterr.APIError) {
	return nil, nil
}

func (h *dumbHandler) List(ctx *resource.Context) (interface{}, *goresterr.APIError) {
	return nil, nil
}

func (h *dumbHandler) Delete(ctx *resource.Context) *goresterr.APIError {
	return nil
}

type Foo struct {
	resource.ResourceBase
}

func (f Foo) SupportAsyncDelete() bool {
	return true
}

type Bar struct {
	resource.ResourceBase
}

var gnum int

var dumbHandler1 = func(ctx *resource.Context) *goresterr.APIError {
	ctx.Set("key", &gnum)
	return nil
}

var dumbHandler2 = func(ctx *resource.Context) *goresterr.APIError {
	val_, _ := ctx.Get("key")
	*(val_.(*int)) = 100
	return nil
}

func TestContextPassChain(t *testing.T) {
	schemas.Import(&version, Foo{}, &dumbHandler{})
	schemas.Import(&version, Bar{}, &dumbHandler{})
	req, _ := http.NewRequest("GET", "/apis/testing/v1/foos", nil)
	req.Host = "127.0.0.1:1234"
	w := httptest.NewRecorder()
	s := NewAPIServer(schemas)
	s.Use(dumbHandler1)
	s.Use(dumbHandler2)

	ut.Equal(t, gnum, 0)
	s.ServeHTTP(w, req)
	ut.Equal(t, gnum, 100)

	//test delete return code
	req, _ = http.NewRequest("DELETE", "/apis/testing/v1/foos/1", nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	ut.Equal(t, w.Code, http.StatusAccepted)

	req, _ = http.NewRequest("DELETE", "/apis/testing/v1/bars/1", nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	ut.Equal(t, w.Code, http.StatusNoContent)
}
