package gorest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"reflect"

	goresterr "gorest/error"
	"gorest/resource"
)

func restHandler(ctx *resource.Context) *goresterr.APIError {
	if ctx.Resource.GetAction() != nil {
		return handleAction(ctx)
	}

	switch ctx.Method {
	case http.MethodGet:
		return handleList(ctx)
	case http.MethodPost:
		return handleCreate(ctx)
	case http.MethodPut:
		return handleUpdate(ctx)
	case http.MethodDelete:
		return handleDelete(ctx)
	default:
		return goresterr.NewAPIError(goresterr.NotFound, "no found request handler")
	}
}

func handleCreate(ctx *resource.Context) *goresterr.APIError {
	schema := ctx.Resource.GetSchema()
	handler := schema.GetHandler().GetCreateHandler()
	if handler == nil {
		return goresterr.NewAPIError(goresterr.NotFound, "no handler for create")
	}

	r, err := handler(ctx)
	if err != nil {
		return err
	}

	ctx.Resource.SetID(r.GetID())
	r.SetType(ctx.Resource.GetType())
	httpSchemeAndHost := path.Join(ctx.Request.URL.Scheme, ctx.Request.URL.Host)
	if err := schema.AddLinksToResource(r, httpSchemeAndHost); err != nil {
		return goresterr.NewAPIError(goresterr.ServerError, fmt.Sprintf("generate links failed:%s", err.Error()))
	}
	return WriteResponse(ctx.Response, http.StatusCreated, r)
}

func handleDelete(ctx *resource.Context) *goresterr.APIError {
	handler := ctx.Resource.GetSchema().GetHandler().GetDeleteHandler()
	if handler == nil {
		return goresterr.NewAPIError(goresterr.NotFound, "no handler for delete")
	}

	if err := handler(ctx); err != nil {
		return err
	}

	kind, ok := ctx.Resource.(resource.ResourceKind)
	if !ok {
		panic(fmt.Sprintf("resource %v doesn't implement resource kind", ctx.Resource))
	}
	status := http.StatusNoContent
	if kind.SupportAsyncDelete() {
		status = http.StatusAccepted
	}
	return WriteResponse(ctx.Response, status, nil)
}

func handleUpdate(ctx *resource.Context) *goresterr.APIError {
	schema := ctx.Resource.GetSchema()
	handler := schema.GetHandler().GetUpdateHandler()
	if handler == nil {
		return goresterr.NewAPIError(goresterr.NotFound, "no handler for update")
	}

	r, err := handler(ctx)
	if err != nil {
		return err
	}

	httpSchemeAndHost := path.Join(ctx.Request.URL.Scheme, ctx.Request.URL.Host)
	if err := schema.AddLinksToResource(r, httpSchemeAndHost); err != nil {
		return goresterr.NewAPIError(goresterr.ServerError, fmt.Sprintf("generate links failed:%s", err.Error()))
	}
	r.SetType(ctx.Resource.GetType())
	return WriteResponse(ctx.Response, http.StatusOK, r)
}

func handleList(ctx *resource.Context) *goresterr.APIError {
	var result interface{}
	schema := ctx.Resource.GetSchema()
	if ctx.Resource.GetID() == "" {
		handler := schema.GetHandler().GetListHandler()
		if handler == nil {
			return goresterr.NewAPIError(goresterr.NotFound, "no found for list")
		}

		data, err_ := handler(ctx)
		if err_ != nil {
			return err_
		}
		rc, err := resource.NewResourceCollection(ctx.Resource, data)
		if err != nil {
			return goresterr.NewAPIError(goresterr.ServerError, err.Error())
		}

		httpSchemeAndHost := path.Join(ctx.Request.URL.Scheme, ctx.Request.URL.Host)
		if err := schema.AddLinksToResourceCollection(rc, httpSchemeAndHost); err != nil {
			return goresterr.NewAPIError(goresterr.ServerError, fmt.Sprintf("generate links failed:%s", err.Error()))
		}
		result = rc
	} else {
		handler := schema.GetHandler().GetGetHandler()
		if handler == nil {
			return goresterr.NewAPIError(goresterr.NotFound, "no found for list")
		}
		r, err := handler(ctx)
		if err != nil {
			return err
		}

		if r == nil || (reflect.ValueOf(r).Kind() == reflect.Ptr && reflect.ValueOf(r).IsNil()) {
			return goresterr.NewAPIError(goresterr.NotFound,
				fmt.Sprintf("%s resource with id %s doesn't exist", ctx.Resource.GetType(), ctx.Resource.GetID()))
		} else {
			//the resource handler returns mayn't include schema
			r.SetSchema(ctx.Resource.GetSchema())
			r.SetParent(ctx.Resource.GetParent())
			httpSchemeAndHost := path.Join(ctx.Request.URL.Scheme, ctx.Request.URL.Host)
			if err := schema.AddLinksToResource(r, httpSchemeAndHost); err != nil {
				return goresterr.NewAPIError(goresterr.ServerError, fmt.Sprintf("generate links failed:%s", err.Error()))
			}
			r.SetType(ctx.Resource.GetType())
		}
		result = r
	}

	return WriteResponse(ctx.Response, http.StatusOK, result)
}

func handleAction(ctx *resource.Context) *goresterr.APIError {
	handler := ctx.Resource.GetSchema().GetHandler().GetActionHandler()
	if handler == nil {
		return goresterr.NewAPIError(goresterr.NotFound, "no handler for action")
	}

	result, err := handler(ctx)
	if err != nil {
		return err
	}

	return WriteResponse(ctx.Response, http.StatusOK, result)
}

const ContentTypeKey = "Content-Type"

func WriteResponse(resp http.ResponseWriter, status int, result interface{}) *goresterr.APIError {
	resp.Header().Set(ContentTypeKey, "application/json")
	resp.WriteHeader(status)
	if result == nil {
		return nil
	}

	body, err := json.Marshal(result)
	if err != nil {
		return goresterr.NewAPIError(goresterr.ServerError, fmt.Sprintf("marshal failed:%s", err.Error()))
	}
	resp.Write(body)
	return nil
}
