package crudui

import (
	"fmt"
	"net/http"
)

const OpsCreate = 8
const OpsRead = 16
const OpsUpdate = 32
const OpsDelete = 64
const OpsList = 128

func (c *Controller) isStructOperationAllowed(r *http.Request, structName string, op int) bool {
	allowedTypes := r.Context().Value(ContextValue(fmt.Sprintf("AllowedTypes_%d", op)))
	if allowedTypes != nil {
		v, ok := allowedTypes.(map[string]bool)[structName]
		if !ok || !v {
			v2, ok2 := allowedTypes.(map[string]bool)["all"]
			if !ok2 || !v2 {
				return false
			}
		}
	}

	return true
}
