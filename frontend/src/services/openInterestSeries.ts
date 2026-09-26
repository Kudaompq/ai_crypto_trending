import type { OpenInterestSample } from './api'

interface TimestampedCandle {
  timestamp: number
}

const intervalMilliseconds: Record<string, number> = {
  '5m': 5 * 60_000,
  '15m': 15 * 60_000,
  '30m': 30 * 60_000,
  '1h': 60 * 60_000,
  '2h': 2 * 60 * 60_000,
  '4h': 4 * 60 * 60_000,
  '6h': 6 * 60 * 60_000,
  '12h': 12 * 60 * 60_000,
  '1d': 24 * 60 * 60_000
}

export function openInterestIntervalMilliseconds(interval: string): number | null {
  return intervalMilliseconds[interval] ?? null
}

export function mapOpenInterestToBars(candles: TimestampedCandle[], samples: OpenInterestSample[], interval: string): Array<{ value: number | null }> {
  const intervalMs = openInterestIntervalMilliseconds(interval)
  if (intervalMs === null) return candles.map(() => ({ value: null }))

  const byBucket = new Map<number, number>()
  for (const sample of samples) {
    if (!Number.isFinite(sample.timestamp) || !Number.isFinite(sample.quantity)) continue
    const bucket = Math.floor(sample.timestamp / intervalMs) * intervalMs
    byBucket.set(bucket, sample.quantity)
  }
  return candles.map(candle => ({ value: byBucket.get(candle.timestamp) ?? null }))
}
