// @vitest-environment jsdom

import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import SimpleChart from '../components/SimpleChart.vue'

describe('SimpleChart', () => {
  it('keeps candles and price information without analysis markers', () => {
    const wrapper = mount(SimpleChart, {
      props: {
        symbol: 'ETHUSDT',
        candles: [
          { timestamp: 1_758_720_000_000, open: 99, high: 102, low: 98, close: 100, volume: 12 },
          { timestamp: 1_758_723_600_000, open: 100, high: 103, low: 99, close: 101, volume: 18 }
        ],
        atr: { value: 1.25, period: 14 }
      }
    })

    expect(wrapper.findAll('.candle-bar')).toHaveLength(2)
    expect(wrapper.text()).toContain('当前价格:')
    expect(wrapper.text()).toContain('ATR(14):')
    expect(wrapper.findAll('.sr-lines .sr-line, .level-badge')).toHaveLength(0)
  })
})
