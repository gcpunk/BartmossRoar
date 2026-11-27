package compression_test

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/erashaperavm/BartmossRoar/core/compression"
	"github.com/erashaperavm/BartmossRoar/core/compression/factory"
)

// algoName 将 CompressAlgo 转成可读名称，避免子测试名出现 \x00。
func algoName(algo compression.CompressAlgo) string {
	switch algo {
	case compression.CompressLZMA2:
		return "LZMA2"
	case compression.CompressZSTD:
		return "ZSTD"
	default:
		return fmt.Sprintf("Unknown(%d)", algo)
	}
}

// supportsBlock 报告某个算法是否支持块级 Compress/Extract。
func supportsBlock(algo compression.CompressAlgo) bool {
	switch algo {
	case compression.CompressZSTD:
		return true
	default:
		return false
	}
}

func totalBytes(datas [][]byte) int {
	total := 0
	for _, data := range datas {
		total += len(data)
	}
	return total
}

// ---------- 块级单数据 ----------

func testBlockRoundTrip(t *testing.T, impl compression.Compressor, data []byte) {
	t.Helper()
	compressed, err := impl.Compress(data)
	if err != nil {
		t.Fatalf("Compress: %v", err)
	}
	got, err := impl.Extract(compressed)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("round-trip mismatch:\n  got:  %q\n  want: %q", got, data)
	}
}

func testBlockUnsupported(t *testing.T, impl compression.Compressor, data []byte) {
	t.Helper()
	if out, err := impl.Compress(data); out != nil {
		t.Errorf("Compress should return nil output, got %v", out)
	} else if err == nil {
		t.Errorf("Compress should return non-nil error")
	}
	if out, err := impl.Extract(data); out != nil {
		t.Errorf("Extract should return nil output, got %v", out)
	} else if err == nil {
		t.Errorf("Extract should return non-nil error")
	}
}

func benchmarkBlockCompress(b *testing.B, impl compression.Compressor, data []byte) {
	b.Helper()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

	compressed, err := impl.Compress(data)
	if err != nil {
		b.Fatalf("Compress validation: %v", err)
	}
	ratio := float64(0)
	if len(data) > 0 {
		ratio = float64(len(compressed)) / float64(len(data)) * 100
	}

	b.ResetTimer()
	for b.Loop() {
		if _, err := impl.Compress(data); err != nil {
			b.Fatalf("Compress: %v", err)
		}
	}
	b.ReportMetric(ratio, "compressed-percent")
}

func benchmarkBlockExtract(b *testing.B, impl compression.Compressor, data []byte) {
	b.Helper()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

	compressed, err := impl.Compress(data)
	if err != nil {
		b.Fatalf("Compress validation: %v", err)
	}
	got, err := impl.Extract(compressed)
	if err != nil {
		b.Fatalf("Extract validation: %v", err)
	}
	if !bytes.Equal(got, data) {
		b.Fatal("round-trip mismatch")
	}

	b.ResetTimer()
	for b.Loop() {
		if _, err := impl.Extract(compressed); err != nil {
			b.Fatalf("Extract: %v", err)
		}
	}
}

func benchmarkBlockUnsupported(b *testing.B, impl compression.Compressor, data []byte) {
	b.Helper()

	for b.Loop() {
		if out, err := impl.Compress(data); out != nil {
			b.Errorf("Compress should return nil output, got %v", out)
		} else if err == nil {
			b.Errorf("Compress should return non-nil error")
		}
		if out, err := impl.Extract(data); out != nil {
			b.Errorf("Extract should return nil output, got %v", out)
		} else if err == nil {
			b.Errorf("Extract should return non-nil error")
		}
	}
}

// ---------- 块级多数据 ----------

func testBlockManyRoundTrip(t *testing.T, impl compression.Compressor, datas [][]byte) {
	t.Helper()
	compressed, errs := impl.CompressMany(datas)
	for _, err := range errs {
		if err != nil {
			t.Fatalf("CompressMany: %v", err)
		}
	}
	if len(compressed) != len(datas) {
		t.Fatalf("CompressMany: got %d results, want %d", len(compressed), len(datas))
	}

	got, errs := impl.ExtractMany(compressed)
	for _, err := range errs {
		if err != nil {
			t.Fatalf("ExtractMany: %v", err)
		}
	}
	if len(got) != len(datas) {
		t.Fatalf("ExtractMany: got %d results, want %d", len(got), len(datas))
	}

	for i := range datas {
		if !bytes.Equal(got[i], datas[i]) {
			t.Errorf("index %d round-trip mismatch:\n  got:  %q\n  want: %q", i, got[i], datas[i])
		}
	}
}

func testBlockManyUnsupported(t *testing.T, impl compression.Compressor, datas [][]byte) {
	t.Helper()
	if out, errs := impl.CompressMany(datas); out != nil {
		t.Errorf("CompressMany should return nil outputs, got %v", out)
	} else if errs == nil {
		t.Errorf("CompressMany should return non-nil errors")
	}
	if out, errs := impl.ExtractMany(datas); out != nil {
		t.Errorf("ExtractMany should return nil outputs, got %v", out)
	} else if errs == nil {
		t.Errorf("ExtractMany should return non-nil errors")
	}
}

func benchmarkBlockManyCompress(b *testing.B, impl compression.Compressor, datas [][]byte) {
	b.Helper()
	b.ReportAllocs()
	inputBytes := totalBytes(datas)
	b.SetBytes(int64(inputBytes))

	compressed, errs := impl.CompressMany(datas)
	for _, err := range errs {
		if err != nil {
			b.Fatalf("CompressMany validation: %v", err)
		}
	}
	if len(compressed) != len(datas) {
		b.Fatalf("CompressMany validation: got %d results, want %d", len(compressed), len(datas))
	}
	compressedBytes := totalBytes(compressed)
	ratio := float64(0)
	if inputBytes > 0 {
		ratio = float64(compressedBytes) / float64(inputBytes) * 100
	}

	b.ResetTimer()
	for b.Loop() {
		if _, errs := impl.CompressMany(datas); errs != nil {
			for _, err := range errs {
				if err != nil {
					b.Fatalf("CompressMany: %v", err)
				}
			}
		}
	}
	b.ReportMetric(ratio, "compressed-percent")
}

func benchmarkBlockManyExtract(b *testing.B, impl compression.Compressor, datas [][]byte) {
	b.Helper()
	b.ReportAllocs()
	b.SetBytes(int64(totalBytes(datas)))

	compressed, errs := impl.CompressMany(datas)
	for _, err := range errs {
		if err != nil {
			b.Fatalf("CompressMany validation: %v", err)
		}
	}
	got, errs := impl.ExtractMany(compressed)
	for _, err := range errs {
		if err != nil {
			b.Fatalf("ExtractMany validation: %v", err)
		}
	}
	if len(got) != len(datas) {
		b.Fatalf("ExtractMany validation: got %d results, want %d", len(got), len(datas))
	}
	for i := range datas {
		if !bytes.Equal(got[i], datas[i]) {
			b.Fatal("round-trip mismatch")
		}
	}

	b.ResetTimer()
	for b.Loop() {
		if _, errs := impl.ExtractMany(compressed); errs != nil {
			for _, err := range errs {
				if err != nil {
					b.Fatalf("ExtractMany: %v", err)
				}
			}
		}
	}
}

func benchmarkBlockManyUnsupported(b *testing.B, impl compression.Compressor, datas [][]byte) {
	b.Helper()

	for b.Loop() {
		if out, errs := impl.CompressMany(datas); out != nil {
			b.Errorf("CompressMany should return nil outputs, got %v", out)
		} else if errs == nil {
			b.Errorf("CompressMany should return non-nil errors")
		}
		if out, errs := impl.ExtractMany(datas); out != nil {
			b.Errorf("ExtractMany should return nil outputs, got %v", out)
		} else if errs == nil {
			b.Errorf("ExtractMany should return non-nil errors")
		}
	}
}

// ---------- 流级单数据 ----------

func testStreamRoundTrip(t *testing.T, impl compression.Compressor, data []byte) {
	t.Helper()
	var compressed bytes.Buffer
	if err := impl.CompressStream(bytes.NewReader(data), &compressed); err != nil {
		t.Fatalf("CompressStream: %v", err)
	}

	var extracted bytes.Buffer
	if err := impl.ExtractStream(bytes.NewReader(compressed.Bytes()), &extracted); err != nil {
		t.Fatalf("ExtractStream: %v", err)
	}

	if !bytes.Equal(extracted.Bytes(), data) {
		t.Errorf("round-trip mismatch:\n  got:  %q\n  want: %q", extracted.Bytes(), data)
	}
}

func benchmarkStreamCompress(b *testing.B, impl compression.Compressor, data []byte) {
	b.Helper()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

	var compressed bytes.Buffer
	if err := impl.CompressStream(bytes.NewReader(data), &compressed); err != nil {
		b.Fatalf("CompressStream validation: %v", err)
	}
	ratio := float64(0)
	if len(data) > 0 {
		ratio = float64(compressed.Len()) / float64(len(data)) * 100
	}

	b.ResetTimer()
	for b.Loop() {
		compressed.Reset()
		if err := impl.CompressStream(bytes.NewReader(data), &compressed); err != nil {
			b.Fatalf("CompressStream: %v", err)
		}
	}
	b.ReportMetric(ratio, "compressed-percent")
}

func benchmarkStreamExtract(b *testing.B, impl compression.Compressor, data []byte) {
	b.Helper()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

	var compressed bytes.Buffer
	if err := impl.CompressStream(bytes.NewReader(data), &compressed); err != nil {
		b.Fatalf("CompressStream validation: %v", err)
	}
	var extracted bytes.Buffer
	if err := impl.ExtractStream(bytes.NewReader(compressed.Bytes()), &extracted); err != nil {
		b.Fatalf("ExtractStream validation: %v", err)
	}
	if !bytes.Equal(extracted.Bytes(), data) {
		b.Fatal("round-trip mismatch")
	}

	compressedBytes := compressed.Bytes()
	b.ResetTimer()
	for b.Loop() {
		extracted.Reset()
		if err := impl.ExtractStream(bytes.NewReader(compressedBytes), &extracted); err != nil {
			b.Fatalf("ExtractStream: %v", err)
		}
	}
}

// ---------- 流级多数据 ----------

func testStreamManyRoundTrip(t *testing.T, impl compression.Compressor, datas [][]byte) {
	t.Helper()
	n := len(datas)

	// 1. 多流压缩
	ins := make([]io.Reader, n)
	for i, d := range datas {
		ins[i] = bytes.NewReader(d)
	}
	compressedBufs := make([]bytes.Buffer, n)
	ws := make([]io.Writer, n)
	for i := range compressedBufs {
		ws[i] = &compressedBufs[i]
	}
	if errs := impl.CompressManyStream(ins, ws); errs != nil {
		t.Fatalf("CompressManyStream returned %d errors: %v", len(errs), errs)
	}

	// 2. 多流解压
	compressedReaders := make([]io.Reader, n)
	for i := range compressedBufs {
		compressedReaders[i] = bytes.NewReader(compressedBufs[i].Bytes())
	}
	extractedBufs := make([]bytes.Buffer, n)
	ws2 := make([]io.Writer, n)
	for i := range extractedBufs {
		ws2[i] = &extractedBufs[i]
	}
	if err := impl.ExtractManyStream(compressedReaders, ws2); err != nil {
		t.Fatalf("ExtractManyStream: %v", err)
	}

	// 3. 校验
	for i := range datas {
		if !bytes.Equal(extractedBufs[i].Bytes(), datas[i]) {
			t.Errorf("index %d round-trip mismatch:\n  got:  %q\n  want: %q", i, extractedBufs[i].Bytes(), datas[i])
		}
	}
}

func benchmarkStreamManyCompress(b *testing.B, impl compression.Compressor, datas [][]byte) {
	b.Helper()
	b.ReportAllocs()
	inputBytes := totalBytes(datas)
	b.SetBytes(int64(inputBytes))
	n := len(datas)

	ins := make([]io.Reader, n)
	for i, data := range datas {
		ins[i] = bytes.NewReader(data)
	}
	compressedBufs := make([]bytes.Buffer, n)
	ws := make([]io.Writer, n)
	for i := range compressedBufs {
		ws[i] = &compressedBufs[i]
	}
	if errs := impl.CompressManyStream(ins, ws); errs != nil {
		b.Fatalf("CompressManyStream validation: %v", errs)
	}

	compressedBytes := make([][]byte, n)
	for i := range compressedBufs {
		compressedBytes[i] = compressedBufs[i].Bytes()
	}
	compressedTotal := totalBytes(compressedBytes)
	ratio := float64(0)
	if inputBytes > 0 {
		ratio = float64(compressedTotal) / float64(inputBytes) * 100
	}

	b.ResetTimer()
	for b.Loop() {
		for i := range ins {
			ins[i] = bytes.NewReader(datas[i])
		}
		for i := range compressedBufs {
			compressedBufs[i].Reset()
		}
		if errs := impl.CompressManyStream(ins, ws); errs != nil {
			b.Fatalf("CompressManyStream: %v", errs)
		}
	}
	b.ReportMetric(ratio, "compressed-percent")
}

func benchmarkStreamManyExtract(b *testing.B, impl compression.Compressor, datas [][]byte) {
	b.Helper()
	b.ReportAllocs()
	b.SetBytes(int64(totalBytes(datas)))
	n := len(datas)

	ins := make([]io.Reader, n)
	for i, data := range datas {
		ins[i] = bytes.NewReader(data)
	}
	compressedBufs := make([]bytes.Buffer, n)
	compressWriters := make([]io.Writer, n)
	for i := range compressedBufs {
		compressWriters[i] = &compressedBufs[i]
	}
	if errs := impl.CompressManyStream(ins, compressWriters); errs != nil {
		b.Fatalf("CompressManyStream validation: %v", errs)
	}

	compressedReaders := make([]io.Reader, n)
	for i := range compressedBufs {
		compressedReaders[i] = bytes.NewReader(compressedBufs[i].Bytes())
	}
	extractedBufs := make([]bytes.Buffer, n)
	extractWriters := make([]io.Writer, n)
	for i := range extractedBufs {
		extractWriters[i] = &extractedBufs[i]
	}
	if err := impl.ExtractManyStream(compressedReaders, extractWriters); err != nil {
		b.Fatalf("ExtractManyStream validation: %v", err)
	}
	for i := range datas {
		if !bytes.Equal(extractedBufs[i].Bytes(), datas[i]) {
			b.Fatal("round-trip mismatch")
		}
	}

	b.ResetTimer()
	for b.Loop() {
		for i := range compressedBufs {
			compressedReaders[i] = bytes.NewReader(compressedBufs[i].Bytes())
			extractedBufs[i].Reset()
		}
		if err := impl.ExtractManyStream(compressedReaders, extractWriters); err != nil {
			b.Fatalf("ExtractManyStream: %v", err)
		}
	}
}

// ---------- 主测试 ----------

func TestCompress(t *testing.T) {
	algorithms := []compression.CompressAlgo{
		compression.CompressLZMA2,
		compression.CompressZSTD,
	}

	singleCases := []struct {
		name string
		data []byte
	}{
		{"ordinary", []byte("hello world from TestCompress")},
		{"empty", []byte("")},
		{"fat", bytes.Repeat([]byte("abc"), 1000)},
	}

	multiCases := []struct {
		name string
		data [][]byte
	}{
		{"ordinary", [][]byte{
			[]byte("hello world from TestCompress"),
			[]byte("hej world from TestCompress"),
			[]byte("hi world from TestCompress"),
		}},
		{"empty", [][]byte{
			[]byte(""),
			[]byte(""),
			[]byte(""),
		}},
		{"fat", [][]byte{
			bytes.Repeat([]byte("abc"), 1000),
			bytes.Repeat([]byte("def"), 1000),
			bytes.Repeat([]byte("ghi"), 1000),
		}},
	}

	for _, algo := range algorithms {
		algo := algo // 显式 shadow，兼容老版本 Go 的 range 变量共享
		t.Run(algoName(algo), func(t *testing.T) {
			impl := factory.NewCompressor(algo)
			if impl == nil {
				t.Fatalf("NewCompressor(%v) returned nil", algoName(algo))
			}

			t.Run("Block", func(t *testing.T) {
				for _, tc := range singleCases {
					tc := tc
					t.Run(tc.name, func(t *testing.T) {
						if supportsBlock(algo) {
							testBlockRoundTrip(t, impl, tc.data)
						} else {
							testBlockUnsupported(t, impl, tc.data)
						}
					})
				}
			})

			t.Run("BlockMany", func(t *testing.T) {
				for _, tc := range multiCases {
					tc := tc
					t.Run(tc.name, func(t *testing.T) {
						if supportsBlock(algo) {
							testBlockManyRoundTrip(t, impl, tc.data)
						} else {
							testBlockManyUnsupported(t, impl, tc.data)
						}
					})
				}
			})

			t.Run("Stream", func(t *testing.T) {
				for _, tc := range singleCases {
					tc := tc
					t.Run(tc.name, func(t *testing.T) {
						testStreamRoundTrip(t, impl, tc.data)
					})
				}
			})

			t.Run("StreamMany", func(t *testing.T) {
				for _, tc := range multiCases {
					tc := tc
					t.Run(tc.name, func(t *testing.T) {
						testStreamManyRoundTrip(t, impl, tc.data)
					})
				}
			})
		})
	}
}

func BenchmarkCompress(b *testing.B) {
	algorithms := []compression.CompressAlgo{
		compression.CompressLZMA2,
		compression.CompressZSTD,
	}

	singleCases := []struct {
		name string
		data []byte
	}{
		{"ordinary", []byte("hello world from TestCompress")},
		{"empty", []byte("")},
		{"fat", bytes.Repeat([]byte("abc"), 1000)},
	}

	multiCases := []struct {
		name string
		data [][]byte
	}{
		{"ordinary", [][]byte{
			[]byte("hello world from TestCompress"),
			[]byte("hej world from TestCompress"),
			[]byte("hi world from TestCompress"),
		}},
		{"empty", [][]byte{
			[]byte(""),
			[]byte(""),
			[]byte(""),
		}},
		{"fat", [][]byte{
			bytes.Repeat([]byte("abc"), 1000),
			bytes.Repeat([]byte("def"), 1000),
			bytes.Repeat([]byte("ghi"), 1000),
		}},
	}

	for _, algo := range algorithms {
		algo := algo // 显式 shadow，兼容老版本 Go 的 range 变量共享
		b.Run(algoName(algo), func(b *testing.B) {
			impl := factory.NewCompressor(algo)
			if impl == nil {
				b.Fatalf("NewCompressor(%v) returned nil", algoName(algo))
			}

			b.Run("Block", func(b *testing.B) {
				for _, tc := range singleCases {
					tc := tc
					b.Run(tc.name, func(b *testing.B) {
						if supportsBlock(algo) {
							b.Run("Compress", func(b *testing.B) {
								benchmarkBlockCompress(b, impl, tc.data)
							})
							b.Run("Extract", func(b *testing.B) {
								benchmarkBlockExtract(b, impl, tc.data)
							})
						} else {
							benchmarkBlockUnsupported(b, impl, tc.data)
						}
					})
				}
			})

			b.Run("BlockMany", func(b *testing.B) {
				for _, tc := range multiCases {
					tc := tc
					b.Run(tc.name, func(b *testing.B) {
						if supportsBlock(algo) {
							b.Run("Compress", func(b *testing.B) {
								benchmarkBlockManyCompress(b, impl, tc.data)
							})
							b.Run("Extract", func(b *testing.B) {
								benchmarkBlockManyExtract(b, impl, tc.data)
							})
						} else {
							benchmarkBlockManyUnsupported(b, impl, tc.data)
						}
					})
				}
			})

			b.Run("Stream", func(b *testing.B) {
				for _, tc := range singleCases {
					tc := tc
					b.Run(tc.name, func(b *testing.B) {
						b.Run("Compress", func(b *testing.B) {
							benchmarkStreamCompress(b, impl, tc.data)
						})
						b.Run("Extract", func(b *testing.B) {
							benchmarkStreamExtract(b, impl, tc.data)
						})
					})
				}
			})

			b.Run("StreamMany", func(b *testing.B) {
				for _, tc := range multiCases {
					tc := tc
					b.Run(tc.name, func(b *testing.B) {
						b.Run("Compress", func(b *testing.B) {
							benchmarkStreamManyCompress(b, impl, tc.data)
						})
						b.Run("Extract", func(b *testing.B) {
							benchmarkStreamManyExtract(b, impl, tc.data)
						})
					})
				}
			})
		})
	}
}
