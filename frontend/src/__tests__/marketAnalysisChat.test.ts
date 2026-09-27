// @vitest-environment jsdom

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import MarketAnalysisChat from '../components/MarketAnalysisChat.vue'

const mocks = vi.hoisted(() => ({ send: vi.fn(), errorMessage: vi.fn((_error: unknown, fallback: string) => fallback) }))
vi.mock('../services/api', () => ({ api: { sendMarketAnalysisMessage: mocks.send, errorMessage: mocks.errorMessage } }))

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, resolve, reject }
}

describe('MarketAnalysisChat', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.useRealTimers()
    mocks.send.mockReset()
    mocks.errorMessage.mockReset().mockImplementation((_error: unknown, fallback: string) => fallback)
  })

  it('keeps history per symbol and sends the latest interval with follow-up context', async () => {
    mocks.send
      .mockResolvedValueOnce({ reply: 'BTC trend summary', symbol: 'BTCUSDT', interval: '1d', context_time: 1_758_736_800_000 })
      .mockResolvedValueOnce({ reply: 'ETH trend summary', symbol: 'ETHUSDT', interval: '1d', context_time: 1_758_736_900_000 })
      .mockResolvedValueOnce({ reply: 'BTC 3m follow-up', symbol: 'BTCUSDT', interval: '3m', context_time: 1_758_737_000_000 })
    const wrapper = mount(MarketAnalysisChat, { props: { symbol: 'BTCUSDT', interval: '1d' } })

    expect(mocks.send).not.toHaveBeenCalled()
    await wrapper.get('textarea[aria-label="向 AI 行情分析提问"]').setValue('总结 BTC 趋势')
    await wrapper.get('.ai-chat-form').trigger('submit')
    await flushPromises()

    expect(mocks.send).toHaveBeenNthCalledWith(1, {
      symbol: 'BTCUSDT', interval: '1d', message: '总结 BTC 趋势', history: []
    })
    expect(wrapper.text()).toContain('BTC trend summary')
    expect(wrapper.get('.chat-message-assistant time').attributes('data-context-time')).toBe('1758736800000')
    expect(wrapper.get('.chat-message-assistant time').text()).toContain('1d')

    await wrapper.setProps({ symbol: 'ETHUSDT', interval: '1d' })
    expect(wrapper.text()).not.toContain('BTC trend summary')
    await wrapper.get('textarea[aria-label="向 AI 行情分析提问"]').setValue('查看 ETH')
    await wrapper.get('.ai-chat-form').trigger('submit')
    await flushPromises()
    expect(wrapper.text()).toContain('ETH trend summary')

    await wrapper.setProps({ symbol: 'BTCUSDT', interval: '3m' })
    expect(wrapper.text()).toContain('BTC trend summary')
    await wrapper.get('textarea[aria-label="向 AI 行情分析提问"]').setValue('切换周期后怎么看？')
    await wrapper.get('.ai-chat-form').trigger('submit')
    await flushPromises()

    expect(mocks.send).toHaveBeenNthCalledWith(3, {
      symbol: 'BTCUSDT', interval: '3m', message: '切换周期后怎么看？', history: [
        { role: 'user', content: '总结 BTC 趋势' },
        { role: 'assistant', content: 'BTC trend summary' }
      ]
    })
    expect(wrapper.text()).toContain('BTC 3m follow-up')
    wrapper.unmount()
  })

  it('offers focused questions before the first reply and renders a structured analysis card', async () => {
    mocks.send.mockResolvedValueOnce({
      reply: '观点：偏多\n结论：短线结构偏强，但接近阻力区。\n结构依据：价格位于 EMA 上方，MACD 信号偏多。\n关键价位：支撑 $82,000；阻力 $85,270。\n风险提示：若跌破最近支撑，偏多判断失效。',
      symbol: 'BTCUSDT', interval: '1h', context_time: 1_758_736_800_000
    })
    const wrapper = mount(MarketAnalysisChat, { props: { symbol: 'BTCUSDT', interval: '1h' } })

    expect(wrapper.text()).toContain('这段行情，先看什么？')
    const trendQuestion = wrapper.get('button[aria-label="判断当前趋势"]')
    await trendQuestion.trigger('click')
    await flushPromises()

    expect(mocks.send).toHaveBeenCalledWith(expect.objectContaining({
      symbol: 'BTCUSDT', interval: '1h', history: [],
      message: expect.stringContaining('判断趋势偏多、偏空、震荡还是观望')
    }))
    expect(wrapper.get('[data-role="assistant"] .analysis-stance').text()).toBe('偏多')
    expect(wrapper.get('[data-role="assistant"] .analysis-summary').text()).toContain('短线结构偏强')
    expect(wrapper.get('[data-role="assistant"] .analysis-section-levels').text()).toContain('$85,270')
    expect(wrapper.text()).toContain('风险提示')
    wrapper.unmount()
  })

  it('archives a late reply to the symbol that started the request', async () => {
    const btcReply = deferred<{ reply: string; symbol: string; interval: string; context_time: number }>()
    mocks.send.mockReturnValueOnce(btcReply.promise).mockResolvedValueOnce({ reply: 'ETH reply', symbol: 'ETHUSDT', interval: '1d', context_time: 2 })
    const wrapper = mount(MarketAnalysisChat, { props: { symbol: 'BTCUSDT', interval: '1d' } })

    await wrapper.get('textarea[aria-label="向 AI 行情分析提问"]').setValue('BTC question')
    await wrapper.get('.ai-chat-form').trigger('submit')
    await wrapper.setProps({ symbol: 'ETHUSDT' })
    await wrapper.get('textarea[aria-label="向 AI 行情分析提问"]').setValue('ETH question')
    await wrapper.get('.ai-chat-form').trigger('submit')
    await flushPromises()
    expect(wrapper.text()).toContain('ETH reply')
    expect(wrapper.text()).not.toContain('BTC reply')

    btcReply.resolve({ reply: 'BTC reply', symbol: 'BTCUSDT', interval: '1d', context_time: 1 })
    await flushPromises()
    expect(wrapper.text()).toContain('ETH reply')
    expect(wrapper.text()).not.toContain('BTC reply')
    await wrapper.setProps({ symbol: 'BTCUSDT' })
    expect(wrapper.text()).toContain('BTC reply')
    wrapper.unmount()
  })

  it('keeps existing messages and the failed question available for retry', async () => {
    mocks.send
      .mockResolvedValueOnce({ reply: 'Existing answer', symbol: 'BTCUSDT', interval: '1h', context_time: 1 })
      .mockRejectedValueOnce(new Error('provider timeout'))
    const wrapper = mount(MarketAnalysisChat, { props: { symbol: 'BTCUSDT', interval: '1h' } })

    await wrapper.get('textarea[aria-label="向 AI 行情分析提问"]').setValue('First question')
    await wrapper.get('.ai-chat-form').trigger('submit')
    await flushPromises()
    await wrapper.get('textarea[aria-label="向 AI 行情分析提问"]').setValue('Retry me')
    await wrapper.get('.ai-chat-form').trigger('submit')
    await flushPromises()

    expect(wrapper.text()).toContain('Existing answer')
    expect(wrapper.find('.chat-error').exists()).toBe(true)
    expect((wrapper.get('textarea[aria-label="向 AI 行情分析提问"]').element as HTMLTextAreaElement).value).toBe('Retry me')
    expect(mocks.send).toHaveBeenCalledTimes(2)
    wrapper.unmount()
  })
})
