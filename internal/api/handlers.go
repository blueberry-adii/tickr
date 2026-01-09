package api

import (
	"encoding/json"
	"net/http"
)

type response struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Data    any    `json:"data"`
	Success bool   `json:"sucess"`
}

func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response{
		Status:  http.StatusOK,
		Message: "REST API Up and Working!!!",
		Data:    nil,
		Success: true,
	})
}
