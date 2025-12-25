package ratelimit

import (
	"sync"
	"time"
)

// TokenBucket считает "токены" в БАЙТАХ.
type TokenBucket struct {
	mu sync.Mutex

	rateBytesPerSec float64
	burstBytes      float64

	tokens float64
	last   time.Time
}

func NewTokenBucket(rateBytesPerSec float64) *TokenBucket {
	now := time.Now()
	b := &TokenBucket{
		rateBytesPerSec: rateBytesPerSec,
		last:            now,
	}
	b.recalcBurstLocked()
	// стартовый "запас", чтобы не душить мелкие записи
	b.tokens = b.burstBytes
	return b
}

func (b *TokenBucket) recalcBurstLocked() {
	// 200ms burst + минимум 32KB
	b.burstBytes = b.rateBytesPerSec * 0.2
	if b.burstBytes < 32*1024 {
		b.burstBytes = 32 * 1024
	}
	if b.tokens > b.burstBytes {
		b.tokens = b.burstBytes
	}
}

func (b *TokenBucket) SetRate(rateBytesPerSec float64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.rateBytesPerSec = rateBytesPerSec
	b.recalcBurstLocked()
}

// Wait блокирует (ждёт) столько, чтобы можно было пропустить n байт.
func (b *TokenBucket) Wait(n int) {
	if n <= 0 {
		return
	}

	var sleepDur time.Duration

	b.mu.Lock()
	if b.rateBytesPerSec <= 0 {
		b.mu.Unlock()
		return
	}

	now := time.Now()
	// Если виртуальное время уже впереди, не даём now быть меньше last.
	if now.Before(b.last) {
		now = b.last
	}

	// Пополнение токенов за прошедшее время
	elapsed := now.Sub(b.last).Seconds()
	if elapsed > 0 {
		b.tokens += elapsed * b.rateBytesPerSec
		if b.tokens > b.burstBytes {
			b.tokens = b.burstBytes
		}
		b.last = now
	}

	need := float64(n)
	if b.tokens >= need {
		b.tokens -= need
		b.mu.Unlock()
		return
	}

	missing := need - b.tokens
	waitSec := missing / b.rateBytesPerSec
	sleepDur = time.Duration(waitSec * float64(time.Second))

	// “Оплатили” chunk ожиданием: токены в ноль, last вперёд
	b.tokens = 0
	b.last = now.Add(sleepDur)

	b.mu.Unlock()

	if sleepDur > 0 {
		time.Sleep(sleepDur)
	}
}
