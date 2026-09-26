// @vitest-environment jsdom

import { beforeEach, describe, expect, it } from 'vitest'
import {
  CHART_PREFERENCES_STORAGE_KEY,
  getChartPreferences,
  saveChartPreferences,
  validateChartStudyParameters,
  type SymbolChartPreferences
} from '../services/chartPreferences'

describe('chart preferences', () => {
  beforeEach(() => localStorage.clear())

  it('restores one shared set of enabled studies and parameters for every symbol', () => {
    localStorage.setItem(CHART_PREFERENCES_STORAGE_KEY, JSON.stringify({
      version: 2,
      studies: {
        MA: { enabled: true, params: [7, 14, 28] },
        MACD: { enabled: true, params: [8, 21, 5] }
      },
      symbols: {}
    }))

    const studies = {
      MA: { enabled: true, params: [7, 14, 28] },
      MACD: { enabled: true, params: [8, 21, 5] }
    }
    expect(getChartPreferences('BTCUSDT').studies).toEqual(studies)
    expect(getChartPreferences('ETHUSDT').studies).toEqual(studies)
  })

  it('persists shared settings independently of chart intervals', () => {
    const preferences: SymbolChartPreferences = {
      studies: { EMA: { enabled: true, params: [9, 21, 50] } },
      drawings: []
    }
    saveChartPreferences('ETHUSDT', preferences)

    expect(getChartPreferences('ETHUSDT')).toEqual(preferences)
    expect(getChartPreferences('BTCUSDT').studies).toEqual(preferences.studies)
    expect(getChartPreferences('BTCUSDT').drawings).toEqual([])
  })

  it('persists shared OI enablement without interval-specific settings', () => {
    const preferences: SymbolChartPreferences = {
      studies: { OPEN_INTEREST: { enabled: true, params: [] } },
      drawings: []
    }
    expect(saveChartPreferences('ETHUSDT', preferences)).toBe(true)
    expect(getChartPreferences('ETHUSDT')).toEqual(preferences)
    expect(getChartPreferences('BTCUSDT').studies).toEqual(preferences.studies)
    expect(validateChartStudyParameters('OPEN_INTEREST', []).valid).toBe(true)
    expect(validateChartStudyParameters('OPEN_INTEREST', [5]).valid).toBe(false)
  })

  it('rejects invalid periods and invalid MACD ordering while accepting valid values', () => {
    expect(validateChartStudyParameters('MA', [5, 10, 30])).toMatchObject({ valid: true })
    expect(validateChartStudyParameters('BOLL', [20, 2])).toMatchObject({ valid: true })
    expect(validateChartStudyParameters('MACD', [12, 26, 9])).toMatchObject({ valid: true })
    expect(validateChartStudyParameters('EMA', [9, 0, 50]).valid).toBe(false)
    expect(validateChartStudyParameters('MACD', [26, 12, 9]).valid).toBe(false)
  })

  it('falls back to an empty preference record when browser data is corrupted', () => {
    localStorage.setItem(CHART_PREFERENCES_STORAGE_KEY, '{bad-json')
    expect(getChartPreferences('ETHUSDT')).toEqual({ studies: {}, drawings: [] })
  })

  it('shares studies across symbols while keeping drawings isolated', () => {
    const ethDrawing = {
      id: 'eth-line', name: 'horizontalStraightLine',
      points: [{ timestamp: 1000, value: 100 }]
    }
    const btcDrawing = {
      id: 'btc-line', name: 'horizontalStraightLine',
      points: [{ timestamp: 2000, value: 200 }]
    }
    const ethStudies = { MA: { enabled: true, params: [7, 14, 28] } }
    saveChartPreferences('ETHUSDT', { studies: ethStudies, drawings: [ethDrawing] })

    expect(getChartPreferences('BTCUSDT').studies).toEqual(ethStudies)
    expect(getChartPreferences('BTCUSDT').drawings).toEqual([])

    const updatedStudies = { MACD: { enabled: true, params: [8, 21, 5] } }
    saveChartPreferences('BTCUSDT', { studies: updatedStudies, drawings: [btcDrawing] })
    expect(getChartPreferences('ETHUSDT').studies).toEqual(updatedStudies)
    expect(getChartPreferences('ETHUSDT').drawings).toEqual([ethDrawing])
    expect(getChartPreferences('BTCUSDT').drawings).toEqual([btcDrawing])
  })

  it('migrates version 1 symbol studies to shared settings and preserves per-symbol drawings', () => {
    const ethDrawing = {
      id: 'eth-line', name: 'horizontalStraightLine',
      points: [{ timestamp: 1000, value: 100 }]
    }
    const btcDrawing = {
      id: 'btc-line', name: 'horizontalStraightLine',
      points: [{ timestamp: 2000, value: 200 }]
    }
    localStorage.setItem('crypto-trending-chart-preferences-v1', JSON.stringify({
      version: 1,
      symbols: {
        ETHUSDT: { studies: { MA: { enabled: true, params: [7, 14] } }, drawings: [ethDrawing] },
        BTCUSDT: { studies: { EMA: { enabled: true, params: [9, 21] } }, drawings: [btcDrawing] }
      }
    }))

    expect(getChartPreferences('ETHUSDT').studies).toEqual({ MA: { enabled: true, params: [7, 14] } })
    expect(getChartPreferences('BTCUSDT').studies).toEqual({ MA: { enabled: true, params: [7, 14] } })
    expect(getChartPreferences('ETHUSDT').drawings).toEqual([ethDrawing])
    expect(getChartPreferences('BTCUSDT').drawings).toEqual([btcDrawing])
    expect(JSON.parse(localStorage.getItem(CHART_PREFERENCES_STORAGE_KEY) || '{}').version).toBe(2)
    expect(localStorage.getItem('crypto-trending-chart-preferences-v1')).toBeNull()
  })
})
