package requestutils

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
)

type requests struct {
	logger lumber.Logger
	client http.Client
}

func New(logger lumber.Logger) core.Requests {
	return &requests{
		logger: logger,
		client: http.Client{Timeout: 30 * time.Second},
	}
}

func (r *requests) MakeAPIRequest(ctx context.Context, httpMethod, endpoint string, body []byte, token string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, httpMethod, endpoint, bytes.NewBuffer(body))
	if err != nil {
		r.logger.Errorf("error while creating http request %v", err)
		return nil, err
	}

	if token != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	resp, err := r.client.Do(req)
	if err != nil {
		r.logger.Errorf("error while sending http request %v", err)
		return nil, err
	}

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		r.logger.Errorf("error while reading http response body %v", err)
		return respBody, err
	}

	//nolint:gomnd
	if resp.StatusCode >= 300 {
		r.logger.Errorf("non 2xx status code %s", string(respBody))
		return respBody, errors.New("non 2xx status code")
	}

	return respBody, nil
}
