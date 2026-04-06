package stats

import "context"

type usageWriter interface {
	SetUserUsage(username string, uploadBytes, downloadBytes int64) error
}

type Collector struct {
	source Source
	writer usageWriter
}

func NewCollector(source Source, writer usageWriter) *Collector {
	return &Collector{
		source: source,
		writer: writer,
	}
}

func (c *Collector) Refresh(ctx context.Context) error {
	records, err := c.source.Collect(ctx)
	if err != nil {
		return err
	}

	for _, record := range records {
		if err := c.writer.SetUserUsage(record.Username, record.UploadBytes, record.DownloadBytes); err != nil {
			return err
		}
	}

	return nil
}
