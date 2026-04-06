package stats

import (
	"context"
	"errors"
)

var ErrNotImplemented = errors.New("stats source not implemented")

type Usage struct {
	Username      string
	UploadBytes   int64
	DownloadBytes int64
}

type Source interface {
	Collect(context.Context) ([]Usage, error)
}

type V2RayAPIClient struct{}

func (V2RayAPIClient) Collect(context.Context) ([]Usage, error) {
	return nil, ErrNotImplemented
}
