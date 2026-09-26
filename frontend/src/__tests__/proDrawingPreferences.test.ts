// @vitest-environment jsdom

import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  CHART_PREFERENCES_STORAGE_KEY,
  getChartPreferences,
  saveChartPreferences,
  type SavedChartDrawing
} from '../services/chartPreferences'
import {
  persistProDrawings,
  restoreProOverlays,
  serializeCoreOverlays,
  type CoreOverlaySnapshot
} from '../services/proDrawingPreferences'

const legacyDrawings: SavedChartDrawing[] = [
  { id: 'trend-1', name: 'segment', points: [{ timestamp: 1000, value: 10 }, { timestamp: 2000, value: 20 }] },
  { id: 'ray-1', name: 'rayLine', points: [{ timestamp: 1500, value: 15 }] }
]

const overlays: CoreOverlaySnapshot[] = legacyDrawings.map(drawing => ({
  ...drawing,
  totalStep: drawing.name === 'segment' ? 3 : 2,
  points: drawing.points
}))

describe('Pro drawing preference adapter', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    localStorage.clear()
  })

  it('round-trips saved Core drawing names, ids, price/time anchors, and styles', () => {
    const withStyles: CoreOverlaySnapshot[] = [{
      ...overlays[0]!,
      styles: { line: { color: '#f00' } },
      visible: false,
      lock: true
    } as CoreOverlaySnapshot]

    const serialized = serializeCoreOverlays(withStyles)
    expect(serialized).toEqual([{ ...legacyDrawings[0]!, styles: { line: { color: '#f00' } }, visible: false, lock: true }])
    expect(restoreProOverlays(serialized)).toEqual(serialized)
    expect(serializeCoreOverlays(overlays)).toEqual(legacyDrawings)
  })

  it('persists drawings from every Pro category by trading pair', () => {
    const proDrawing: CoreOverlaySnapshot = {
      id: 'fib-extension',
      name: 'fibonacciExtension',
      points: [{ timestamp: 1000, value: 10 }, { timestamp: 2000, value: 20 }, { timestamp: 3000, value: 15 }]
    }

    expect(persistProDrawings('ETHUSDT', [proDrawing])).toBe(true)
    expect(getChartPreferences('ETHUSDT').drawings).toEqual([{
      id: 'fib-extension', name: 'fibonacciExtension', points: proDrawing.points
    }])
    expect(getChartPreferences('BTCUSDT').drawings).toEqual([])
  })

  it('preserves names from the line, channel, shape, Fibonacci, and wave menus', () => {
    const names = [
      'horizontalStraightLine', 'horizontalRayLine', 'horizontalSegment', 'verticalStraightLine', 'verticalRayLine',
      'verticalSegment', 'straightLine', 'rayLine', 'segment', 'arrow', 'priceLine', 'priceChannelLine',
      'parallelStraightLine', 'circle', 'rect', 'parallelogram', 'triangle', 'fibonacciLine', 'fibonacciSegment',
      'fibonacciCircle', 'fibonacciSpiral', 'fibonacciSpeedResistanceFan', 'fibonacciExtension', 'gannBox',
      'xabcd', 'abcd', 'threeWaves', 'fiveWaves', 'eightWaves', 'anyWaves'
    ]
    const snapshots: CoreOverlaySnapshot[] = names.map((name, index) => ({
      id: `drawing-${index}`,
      name,
      points: [{ timestamp: 1000 + index, value: 10 + index }]
    }))

    expect(serializeCoreOverlays(snapshots).map(drawing => drawing.name)).toEqual(names)
  })

  it('does not save a Pro drawing before it has all required anchor points', () => {
    expect(serializeCoreOverlays([{
      id: 'unfinished-channel',
      name: 'parallelStraightLine',
      totalStep: 4,
      points: [{ timestamp: 1000, value: 10 }, { timestamp: 2000, value: 20 }]
    }])).toEqual([])
  })

  it('keeps drawings isolated by symbol and makes repeated migration idempotent', () => {
    saveChartPreferences('BTCUSDT', { studies: { MA: { enabled: true, params: [5, 10] } }, drawings: [] })

    expect(persistProDrawings('ETHUSDT', overlays)).toBe(true)
    const first = getChartPreferences('ETHUSDT').drawings
    expect(persistProDrawings('ETHUSDT', overlays)).toBe(true)
    expect(getChartPreferences('ETHUSDT').drawings).toEqual(first)
    expect(getChartPreferences('BTCUSDT')).toEqual({
      studies: { MA: { enabled: true, params: [5, 10] } }, drawings: []
    })
    expect(getChartPreferences('ETHUSDT').studies).toEqual({ MA: { enabled: true, params: [5, 10] } })
  })

  it('retains the previous storage document when migration cannot be saved', () => {
    const prior = JSON.stringify({ version: 2, studies: {}, symbols: { ETHUSDT: { drawings: legacyDrawings } } })
    localStorage.setItem(CHART_PREFERENCES_STORAGE_KEY, prior)
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => { throw new DOMException('quota exceeded', 'QuotaExceededError') })

    expect(persistProDrawings('ETHUSDT', overlays)).toBe(false)
    expect(localStorage.getItem(CHART_PREFERENCES_STORAGE_KEY)).toBe(prior)
  })
})
