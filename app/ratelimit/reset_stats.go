package ratelimit

// ResetTrafficForActiveOrGraceDevices обнуляет Rx/Tx (upload/download) у всех deviceEntry,
// которые ещё живут в deviceEntries.m (и активные, и на grace).
func ResetTrafficForDevices() (resetCount int) {
	// 1) Собираем список ConnID под lock
	var ids []ConnID

	deviceEntries.mu.Lock()
	for _, e := range deviceEntries.m {
		if e == nil {
			continue
		}
		ids = append(ids, e.id)
	}
	deviceEntries.mu.Unlock()

	// 2) Обнуляем статистику (вне lock)
	for _, id := range ids {
		ci := Global.Get(id)
		if ci == nil {
			continue
		}

		// Сейчас у тебя: Rx=upload, Tx=download (при wrap на outboundLink)
		ci.RxBytes.Store(0)
		ci.TxBytes.Store(0)

		resetCount++
	}

	return resetCount
}

// ResetTrafficForActiveOrGraceDevicesByUUID — то же самое, но только для uuid.
func ResetTrafficForDevicesByUUID(uuid string) (resetCount int) {
	var ids []ConnID

	deviceEntries.mu.Lock()
	for _, e := range deviceEntries.m {
		if e == nil {
			continue
		}
		if e.uuid == uuid {
			ids = append(ids, e.id)
		}
	}
	deviceEntries.mu.Unlock()

	for _, id := range ids {
		ci := Global.Get(id)
		if ci == nil {
			continue
		}
		ci.RxBytes.Store(0)
		ci.TxBytes.Store(0)
		resetCount++
	}

	return resetCount
}
