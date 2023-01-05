package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

func (app *application) readJSON(w http.ResponseWriter, r *http.Request, data interface{}) error {
	maxBytes := 1048576 // 1 mb
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	dec := json.NewDecoder(r.Body)
	err := dec.Decode(data)
	if err != nil {
		return err
	}

	err = dec.Decode(&struct{}{}) // make sure that Body only has a single Jason
	// value, in other words, we don't want to separate Jason files in the same body
	if err != io.EOF {
		return errors.New("body must have a single json value ")
	}

	return nil
}

func (app *application) writeJSON(w http.ResponseWriter, status int, data interface{}, headers ...http.Header) error {

	var output []byte
	if app.environment == "development" {
		out, err := json.MarshalIndent(data, "", "\t")
		if err != nil {
			return err
		}
		output = out
	} else {
		out, err := json.Marshal(data)
		if err != nil {
			return err
		}
		output = out

	}

	if len(headers) > 0 {
		for key, value := range headers[0] {
			w.Header()[key] = value
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err := w.Write(output)
	if err != nil {
		return err
	}

	return nil
}

func (app *application) errorJSON(w http.ResponseWriter, err error, status ...int) {
	statusCode := http.StatusBadRequest
	if len(status) > 0 {
		statusCode = status[0]
	}

	var customError error
	switch {
	case strings.Contains(err.Error(), "SQLSTATE 23505"):
		customError = errors.New("dublicate value violates constraints")
		statusCode = http.StatusForbidden

	case strings.Contains(err.Error(), "SQLSTATE 22001"):
		customError = errors.New("the value you are trying to insert is too large!")
		statusCode = http.StatusForbidden

	case strings.Contains(err.Error(), "SQLSTATE 23503"):

		customError = errors.New("foreign key violation")
		statusCode = http.StatusForbidden

	default:
		customError = err
	}

	var payload jsonResponse
	payload.Error = true
	payload.Message = customError.Error()

	app.writeJSON(w, statusCode, payload)
}
