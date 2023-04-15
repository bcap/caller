package handler

import (
	"bytes"
	"log"
	"net/http"

	ptype "github.com/bcap/caller/plan"
	"github.com/bcap/caller/random"
)

func (h *handler) parallel(parallel ptype.Parallel, location string) error {
	return h.processSteps(parallel.Concurrency, 0, parallel.Execution, location)
}

func (h *handler) loop(loop ptype.Loop, location string) error {
	for i := 0; i < loop.Times; i++ {
		if err := h.processSteps(1, 0, loop.Execution, location); err != nil {
			return err
		}
		h.delay(loop.Delay)
	}
	return nil
}

func (h *handler) delay(delay ptype.Delay) error {
	delay.Do(h.Context)
	return nil
}

func (h *handler) call(call ptype.Call, location string) error {
	execute := func() error {
		client := http.Client{}
		var body *bytes.Buffer
		if call.HTTP.RequestBody != "" {
			body = bytes.NewBufferString(call.HTTP.RequestBody)
		} else if call.HTTP.GenRequestBody > 0 {
			str := random.String(call.HTTP.GenRequestBody)
			body = bytes.NewBufferString(str)
		} else {
			body = &bytes.Buffer{}
		}
		req, err := http.NewRequestWithContext(
			h.Context, call.HTTP.Method, call.HTTP.URL.String(), body,
		)
		if err != nil {
			return err
		}
		for key, value := range call.HTTP.RequestHeaders {
			req.Header.Set(key, value)
		}
		if err := WritePlanHeaders(req, h.Plan, location); err != nil {
			return err
		}
		WriteRequestTraceHeader(req, h.RequestID)
		_, err = client.Do(req)
		return err
	}

	if call.Async {
		h.pendingAsyncCalls.Add(1)
		go func() {
			err := execute()
			if err != nil {
				log.Printf("!! async call failed: %v", err)
			}
			h.pendingAsyncCalls.Done()
		}()
		return nil
	} else {
		return execute()
	}
}
