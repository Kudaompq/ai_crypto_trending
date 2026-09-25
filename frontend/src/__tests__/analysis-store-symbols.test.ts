// @vitest-environment jsdom

import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useAnalysisStore } from '../stores/analysis'

const apiMocks = vi.hoisted(() => ({
  validateSymbol: vi.fn(async (raw: string) => raw.trim().toUpperCase())
}))

vi.mock('../services/api', () => ({ api: apiMocks }))

describe('editable preset symbols', () => {
  beforeEach(() => {
    localStorage.clear()
    apiMocks.validateSymbol.mockClear()
    setActivePinia(createPinia())
  })

  it('removes preset symbols persistently and selects a remaining symbol', () => {
    const store = useAnalysisStore()
    store.removeSymbol('ETHUSDT')

    expect(store.availableSymbols.some(item => item.value === 'ETHUSDT')).toBe(false)
    expect(store.symbol).toBe('BTCUSDT')

    setActivePinia(createPinia())
    const restoredStore = useAnalysisStore()
    expect(restoredStore.availableSymbols.some(item => item.value === 'ETHUSDT')).toBe(false)
    expect(restoredStore.symbol).toBe('BTCUSDT')
  })

  it('allows a removed preset to be added back', async () => {
    const store = useAnalysisStore()
    store.removeSymbol('ETHUSDT')

    await expect(store.addCustomSymbol('ETHUSDT')).resolves.toBe('ETHUSDT')
    expect(store.availableSymbols.some(item => item.value === 'ETHUSDT')).toBe(true)
    expect(apiMocks.validateSymbol).not.toHaveBeenCalled()
  })

  it('keeps at least one symbol in the selector', () => {
    const store = useAnalysisStore()
    for (const item of store.availableSymbols.slice(0, -1)) {
      store.removeSymbol(item.value)
    }

    const lastSymbol = store.availableSymbols[0]
    expect(lastSymbol).toBeDefined()
    store.removeSymbol(lastSymbol!.value)
    expect(store.availableSymbols).toHaveLength(1)
  })
})
