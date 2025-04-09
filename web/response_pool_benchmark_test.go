package web

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"sync"
	"testing"

	objPool "github.com/fyerfyer/fyer-webframe/web/pool"
)

func BenchmarkBufferAcquireRelease(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := objPool.AcquireBuffer()
		objPool.ReleaseBuffer(buf)
	}
}

func BenchmarkBufferCreateWithoutPool(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := bytes.NewBuffer(make([]byte, 0, 8*1024))
		_ = buf
	}
}

func BenchmarkBufferSizes(b *testing.B) {
	sizes := []int{512, 2 * 1024, 8 * 1024, 32 * 1024}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size-%dK", size/1024), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				buf := objPool.AcquireBufferSize(size)
				objPool.ReleaseBufferSize(buf, size)
			}
		})
	}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("NoPool-Size-%dK", size/1024), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				buf := bytes.NewBuffer(make([]byte, 0, size))
				_ = buf
			}
		})
	}
}

func BenchmarkBufferWriteOperations(b *testing.B) {
	type testData struct {
		Name    string   `json:"name" xml:"name"`
		Age     int      `json:"age" xml:"age"`
		Email   string   `json:"email" xml:"email"`
		Tags    []string `json:"tags" xml:"tags>tag"`
		IsAdmin bool     `json:"is_admin" xml:"is_admin"`
	}

	data := testData{
		Name:    "John Doe",
		Age:     30,
		Email:   "john.doe@example.com",
		Tags:    []string{"developer", "golang", "web"},
		IsAdmin: true,
	}

	b.Run("WriteJSON-WithPool", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf := objPool.AcquireBuffer()
			json.NewEncoder(buf.Buffer).Encode(data)
			objPool.ReleaseBuffer(buf)
		}
	})

	b.Run("WriteJSON-NoPool", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf := bytes.NewBuffer(make([]byte, 0, 8*1024))
			json.NewEncoder(buf).Encode(data)
		}
	})

	b.Run("WriteXML-WithPool", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf := objPool.AcquireBuffer()
			buf.Buffer.WriteString(xml.Header)
			encoder := xml.NewEncoder(buf.Buffer)
			encoder.Encode(data)
			objPool.ReleaseBuffer(buf)
		}
	})

	b.Run("WriteXML-NoPool", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf := bytes.NewBuffer(make([]byte, 0, 8*1024))
			buf.WriteString(xml.Header)
			encoder := xml.NewEncoder(buf)
			encoder.Encode(data)
		}
	})

	b.Run("WriteString-WithPool", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf := objPool.AcquireBuffer()
			fmt.Fprintf(buf.Buffer, "Hello %s, your age is %d and your email is %s", data.Name, data.Age, data.Email)
			objPool.ReleaseBuffer(buf)
		}
	})

	b.Run("WriteString-NoPool", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf := bytes.NewBuffer(make([]byte, 0, 8*1024))
			fmt.Fprintf(buf, "Hello %s, your age is %d and your email is %s", data.Name, data.Age, data.Email)
		}
	})
}

func BenchmarkConcurrentBufferUsage(b *testing.B) {
	b.Run("ConcurrentAcquireRelease", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				buf := objPool.AcquireBuffer()
				buf.Buffer.WriteString("concurrent operation test")
				objPool.ReleaseBuffer(buf)
			}
		})
	})

	b.Run("ConcurrentNoPool", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				buf := bytes.NewBuffer(make([]byte, 0, 8*1024))
				buf.WriteString("concurrent operation test")
			}
		})
	})
}

func BenchmarkPoolVsNewOperations(b *testing.B) {
	data := map[string]interface{}{
		"id":      1234,
		"name":    "Performance Test",
		"status":  "active",
		"details": "This is a test for performance comparison between pooled and non-pooled buffers",
		"tags":    []string{"test", "benchmark", "performance", "pool"},
		"metrics": map[string]float64{
			"cpu":    12.5,
			"memory": 256.75,
			"disk":   1024.5,
		},
	}

	b.Run("WriteAndCopy-WithPool", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf := objPool.AcquireBuffer()
			json.NewEncoder(buf.Buffer).Encode(data)

			result := make([]byte, buf.Buffer.Len())
			copy(result, buf.Buffer.Bytes())

			objPool.ReleaseBuffer(buf)
		}
	})

	b.Run("WriteAndCopy-NoPool", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf := bytes.NewBuffer(make([]byte, 0, 8*1024))
			json.NewEncoder(buf).Encode(data)

			result := make([]byte, buf.Len())
			copy(result, buf.Bytes())
		}
	})
}

func BenchmarkMultiplePoolSizes(b *testing.B) {
	customSizes := []int{1024, 4096, 16384, 65536}

	for _, size := range customSizes {
		b.Run(fmt.Sprintf("CustomPool-%dK", size/1024), func(b *testing.B) {
			pool := objPool.NewResponseBufferPool(size)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				buf := pool.Get()
				buf.Buffer.Grow(size / 2)
				buf.Buffer.WriteString("testing custom sized buffer pool")
				pool.Put(buf)
			}
		})
	}
}

func BenchmarkCustomPoolConcurrent(b *testing.B) {
	pools := []*objPool.ResponseBufferPool{
		objPool.NewResponseBufferPool(2 * 1024),  // Small
		objPool.NewResponseBufferPool(8 * 1024),  // Medium
		objPool.NewResponseBufferPool(32 * 1024), // Large
	}

	//var counter int64
	var wg sync.WaitGroup

	b.Run("MultiPoolConcurrent", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			wg.Add(len(pools))

			for _, pool := range pools {
				go func(idx int, p *objPool.ResponseBufferPool) {
					defer wg.Done()

					buf := p.Get()
					fmt.Fprintf(buf.Buffer, "worker %d writing to buffer from pool %d", i, idx)
					p.Put(buf)
				}(i%10, pool)
			}

			wg.Wait()
		}
	})
}

func BenchmarkBufferLifecycles(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"Small", 2 * 1024},
		{"Medium", 8 * 1024},
		{"Large", 32 * 1024},
	}

	operations := 100

	for _, sz := range sizes {
		b.Run(fmt.Sprintf("PoolLifecycle-%s", sz.name), func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				pool := objPool.NewResponseBufferPool(sz.size)

				for j := 0; j < operations; j++ {
					buf := pool.Get()
					buf.Buffer.WriteString("test data for pool lifecycle")
					pool.Put(buf)
				}
			}
		})

		b.Run(fmt.Sprintf("NoPoolLifecycle-%s", sz.name), func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				for j := 0; j < operations; j++ {
					buf := bytes.NewBuffer(make([]byte, 0, sz.size))
					buf.WriteString("test data for pool lifecycle")
				}
			}
		})
	}
}

func BenchmarkResponseBufferReset(b *testing.B) {
	b.Run("PooledBufferReset", func(b *testing.B) {
		buf := objPool.AcquireBuffer()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf.Buffer.WriteString("data to be reset")
			buf.Reset()
		}

		objPool.ReleaseBuffer(buf)
	})

	b.Run("StandardBufferReset", func(b *testing.B) {
		buf := bytes.NewBuffer(make([]byte, 0, 8*1024))
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf.WriteString("data to be reset")
			buf.Reset()
		}
	})
}