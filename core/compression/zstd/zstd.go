package zstd

import (
	"errors"
	"io"
	"sync"

	"github.com/erashaperavm/BartmossRoar/core/compression"
	"github.com/klauspost/compress/zstd"
)

type ZSTDCompressor struct {
	Encoder  *zstd.Encoder
	Decoder  *zstd.Decoder
	encoders sync.Pool
	decoders sync.Pool
}

func (c *ZSTDCompressor) CompressStream(in io.Reader, out io.Writer) error {
	if in == nil || out == nil {
		return errors.New("input or output stream is nil")
	}

	enc, ok := c.encoders.Get().(*zstd.Encoder)
	if !ok {
		var err error
		enc, err = zstd.NewWriter(nil, zstd.WithEncoderConcurrency(1))
		if err != nil {
			return err
		}
	}
	enc.Reset(out)

	_, err := io.Copy(enc, in)
	if err == nil {
		err = enc.Close()
	}
	enc.Reset(nil)
	c.encoders.Put(enc)
	if err != nil {
		return err
	}
	return nil
}

func (c *ZSTDCompressor) CompressManyStream(in []io.Reader, out []io.Writer) []error {
	if len(in) != len(out) {
		return []error{errors.New("input and output stream slices must have the same length")}
	}

	var wg sync.WaitGroup
	var errs = make([]error, len(in))

	for i := range in {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			if err := c.CompressStream(in[idx], out[idx]); err != nil {
				errs[idx] = err
			}
			return
		}(i)
	}

	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return errs
		}
	}

	return nil
}

func (c *ZSTDCompressor) ExtractStream(in io.Reader, out io.Writer) error {
	if in == nil || out == nil {
		return errors.New("input or output stream is nil")
	}

	dec, ok := c.decoders.Get().(*zstd.Decoder)
	if !ok {
		var err error
		dec, err = zstd.NewReader(nil, zstd.WithDecoderConcurrency(1))
		if err != nil {
			return err
		}
	}
	if err := dec.Reset(in); err != nil {
		c.decoders.Put(dec)
		return err
	}

	_, err := io.Copy(out, dec)
	if err != nil {
		dec.Reset(nil)
		c.decoders.Put(dec)
		return err
	}
	dec.Reset(nil)
	c.decoders.Put(dec)
	return nil
}

func (c *ZSTDCompressor) ExtractManyStream(in []io.Reader, out []io.Writer) error {
	if len(in) != len(out) {
		return errors.New("input and output stream slices must have the same length")
	}

	var wg sync.WaitGroup
	var errs = make([]error, len(in))

	for i := range in {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			if err := c.ExtractStream(in[idx], out[idx]); err != nil {
				errs[idx] = err
			}
			return
		}(i)
	}

	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return errors.New("one or more streams failed to extract")
		}
	}

	return nil
}

func (c *ZSTDCompressor) Compress(origin []byte) ([]byte, error) {
	result := c.Encoder.EncodeAll(origin, make([]byte, 0, len(origin)))

	return result, nil
}

func (c *ZSTDCompressor) CompressMany(origins [][]byte) ([][]byte, []error) {
	results := make([][]byte, len(origins))
	var wg sync.WaitGroup

	for i, origin := range origins {
		wg.Add(1)
		go func(idx int, data []byte) {
			defer wg.Done()
			res, _ := c.Compress(data) // Compress never return error, so we can ignore the error here
			results[idx] = res
		}(i, origin)
	}

	wg.Wait()
	return results, nil
}

func (c *ZSTDCompressor) Extract(compressed []byte) ([]byte, error) {
	return c.Decoder.DecodeAll(compressed, make([]byte, 0))
}

func (c *ZSTDCompressor) ExtractMany(compresseds [][]byte) ([][]byte, []error) {
	results := make([][]byte, len(compresseds))
	var wg sync.WaitGroup

	for i, compressed := range compresseds {
		wg.Add(1)
		go func(idx int, data []byte) {
			defer wg.Done()
			res, _ := c.Extract(data) // Extract never return error, so we can ignore the error here
			results[idx] = res
		}(i, compressed)
	}

	wg.Wait()
	return results, nil
}

func (c *ZSTDCompressor) Algorithm() compression.CompressAlgo {
	return compression.CompressZSTD
}

func NewZSTDEncoder() (*zstd.Encoder, error) {
	return zstd.NewWriter(nil, zstd.WithEncoderConcurrency(1))
}

func NewZSTDDecoder() (*zstd.Decoder, error) {
	return zstd.NewReader(nil, zstd.WithDecoderConcurrency(1))
}
