package app

import (
	"encoding/json"
	"net/http"
)

type response struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

func apiResponse(w http.ResponseWriter, code int, msg string, data any) {
	resp, err := json.Marshal(&response{Code: code, Msg: msg, Data: data})
	if err != nil {
		w.Write([]byte("error marshalling response"))
		return
	}
	w.Write(resp)
}
