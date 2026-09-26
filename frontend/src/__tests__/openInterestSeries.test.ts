import { describe, expect, it } from 'vitest'
import { mapOpenInterestToBars } from '../services/openInterestSeries'

describe('open-interest time-series mapping', () => {
  it('maps Binance points to matching bars and leaves missing timestamps empty', () => {
    const candles = [
      { timestamp: 300_000, open: 1, high: 1, low: 1, close: 1, volume: 1 },
      { timestamp: 600_000, open: 1, high: 1, low: 1, close: 1, volume: 1 },
      { timestamp: 900_000, open: 1, high: 1, low: 1, close: 1, volume: 1 }
    ]
    const samples = [
      { timestamp: 300_000, quantity: 10, value: 1000 },
      { timestamp: 900_000, quantity: 30, value: 3000 }
    ]

    expect(mapOpenInterestToBars(candles, samples, '5m')).toEqual([
      { value: 10 }, { value: null }, { value: 30 }
    ])
  })

  it('maps end-of-period sample timestamps into their matching candle bucket without carrying values forward', () => {
    const candles = [
      { timestamp: 300_000, open: 1, high: 1, low: 1, close: 1, volume: 1 },
      { timestamp: 600_000, open: 1, high: 1, low: 1, close: 1, volume: 1 }
    ]
    const samples = [{ timestamp: 599_999, quantity: 10, value: 1000 }]
    expect(mapOpenInterestToBars(candles, samples, '5m')).toEqual([{ value: 10 }, { value: null }])
  })
})
