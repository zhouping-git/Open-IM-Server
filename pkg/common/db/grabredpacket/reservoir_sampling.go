package grabredpacket

import (
	"math/rand"
	"time"
)

type ReservoirSampling struct {
	pool []int64
	size int64
}

func BuildReservoirSampling(size int64) *ReservoirSampling {
	var pool = make([]int64, size)
	for i := 0; i < int(size); i++ {
		pool[i] = int64(i)
	}
	return &ReservoirSampling{
		size: size,
		pool: pool,
	}
}

func (o *ReservoirSampling) Sampling(length int) []int32 {
	var result = make([]int32, length)
	for i := 0; i < length; i++ {
		result[i] = int32(o.pool[i])
	}

	rand.NewSource(time.Now().UnixNano())
	for i := length; i < int(o.size); i++ {
		r := rand.Intn(i + 1)
		if r < length {
			result[r] = int32(o.pool[i])
		}
	}

	return result
}
