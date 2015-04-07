package rest

import (
	"fmt"
	"net/http"

	"github.com/clusterit/orca/common"
	"gopkg.in/emicklei/go-restful.v1"
)

// This utility function checks if an error is a "NotFound" and returns a
// HTTP NotFound to the caller. If it is another error it returns in internal
// server error.
func HandleEntity(entity interface{}, err error) restful.RouteFunction {
	return func(rq *restful.Request, response *restful.Response) {
		if err == nil {
			response.WriteEntity(entity)
		} else {
			HandleError(err, response)
		}
	}
}

// Wraps the error in a json document.
func HandleError(err error, response *restful.Response) {
	if common.IsNotFound(err) {
		response.WriteError(http.StatusNotFound, JsonError("entity could not be found"))
	} else {
		response.WriteError(http.StatusInternalServerError, JsonError("%s", err))
	}

}

// Format the message an parametes and return an
// error in json-format.
func JsonError(msg string, pars ...interface{}) error {
	m := fmt.Sprintf(msg, pars...)
	return fmt.Errorf("{\"error\":\"%s\"}", m)
}
