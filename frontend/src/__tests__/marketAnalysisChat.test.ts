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

function makeChatResponse(analysis: string, symbol: string, interval: string, context_time: number) {
  const plan = {
    direction: 'Long' as const, entry_type: 'market' as const, entry_price: 100,
    take_profit: 110, stop_loss: 90, confidence: 72, leverage: 2, analysis
  }
  return {
    reply: JSON.stringify(plan), plan,
    key_levels: { poc: null, poc_estimated: false, resistance: null, support: null },
    reference_price: 100, symbol, interval, context_time
  }
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
    const btcReply = makeChatResponse('BTC trend summary', 'BTCUSDT', '1d', 1_758_736_800_000)
    mocks.send
      .mockResolvedValueOnce(btcReply)
      .mockResolvedValueOnce(makeChatResponse('ETH trend summary', 'ETHUSDT', '1d', 1_758_736_900_000))
      .mockResolvedValueOnce(makeChatResponse('BTC 3m follow-up', 'BTCUSDT', '3m', 1_758_737_000_000))
    const wrapper = mount(MarketAnalysisChat, { props: { symbol: 'BTCUSDT', interval: '1d' } })

    expect(mocks.send).not.toHaveBeenCalled()
    await wrapper.get('textarea[aria-label="向 AI 行情分析提问"]').setValue('总结 BTC 趋势')
    await wrapper.get('.ai-chat-form').trigger('submit')
    await flushPromises()

    expect(mocks.send).toHaveBeenNthCalledWith(1, {
      symbol: 'BTCUSDT', interval: '1d', message: '总结 BTC 趋势', history: []
    })
    expect(wrapper.text()).toContain('BTC trend summary')
    expect(wrapper.get('.analysis-market-time').attributes('data-context-time')).toBe('1758736800000')
    expect(wrapper.get('.analysis-market-time').text()).toContain('1d')

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
        { role: 'assistant', content: btcReply.reply }
      ]
    })
    expect(wrapper.text()).toContain('BTC 3m follow-up')
    wrapper.unmount()
  })

  it('offers focused questions before the first reply and renders a structured analysis card', async () => {
    mocks.send.mockResolvedValueOnce(makeChatResponse('短线结构偏强，但接近阻力区。', 'BTCUSDT', '1h', 1_758_736_800_000))
    const wrapper = mount(MarketAnalysisChat, { props: { symbol: 'BTCUSDT', interval: '1h' } })

    expect(wrapper.text()).toContain('这段行情，先看什么？')
    const trendQuestion = wrapper.get('button[aria-label="生成方向分析"]')
    await trendQuestion.trigger('click')
    await flushPromises()

    expect(mocks.send).toHaveBeenCalledWith(expect.objectContaining({
      symbol: 'BTCUSDT', interval: '1h', history: [],
      message: expect.stringContaining('生成结构化方向分析')
    }))
    expect(wrapper.get('[data-role="assistant"] .analysis-direction').text()).toContain('Long')
    expect(wrapper.get('[data-role="assistant"] .analysis-narrative').text()).toContain('短线结构偏强')
    expect(wrapper.get('[data-role="assistant"] .analysis-key-level-poc').text()).toContain('未识别')
    wrapper.unmount()
  })

  it('renders the directional plan, derived stop risk, analysis time, and supplied key levels', async () => {
    const plan = {
      direction: 'Long', entry_type: 'market', entry_price: 11.07,
      take_profit: 11.78, stop_loss: 10.74, confidence: 72,
      leverage: 5, analysis: '价格站上短线结构位，接近阻力区域；若跌破支撑则该判断失效。'
    }
    mocks.send.mockResolvedValueOnce({
      reply: JSON.stringify(plan), plan,
      key_levels: { poc: 11.13, resistance: 11.1316, support: 10.7442, poc_estimated: true },
      reference_price: 11.073,
      symbol: 'BTCUSDT', interval: '1h', context_time: 1_758_736_800_000
    })
    const wrapper = mount(MarketAnalysisChat, { props: { symbol: 'BTCUSDT', interval: '1h' } })

    await wrapper.get('textarea[aria-label="向 AI 行情分析提问"]').setValue('分析当前方向和风险位')
    await wrapper.get('.ai-chat-form').trigger('submit')
    await flushPromises()

    const card = wrapper.get('[data-role="assistant"]')
    expect(card.get('.analysis-direction').text()).toContain('Long')
    expect(card.get('.analysis-time').text()).toContain('分析时间')
    expect(card.get('.analysis-entry').text()).toContain('$11.07')
    expect(card.get('.analysis-take-profit').text()).toContain('$11.78')
    expect(card.get('.analysis-stop-loss').text()).toContain('$10.74')
    expect(card.get('.analysis-confidence').text()).toContain('72')
    expect(card.get('.analysis-margin-risk').text()).toContain('15%')
    expect(card.get('.analysis-section-levels').text()).toContain('$11.13')
    expect(card.get('.analysis-market-time').attributes('data-context-time')).toBe('1758736800000')
    expect(card.text()).toContain('不含手续费、滑点和资金费率')
    wrapper.unmount()
  })

  it('does not render an invalid successful payload as a trade-analysis card', async () => {
    mocks.send.mockResolvedValueOnce({
      reply: 'provider text that must not be shown as a successful answer',
      plan: {
        direction: 'Long', entry_type: 'market', entry_price: 100,
        take_profit: 90, stop_loss: 110, confidence: 72,
        leverage: 3, analysis: 'invalid price order'
      },
      key_levels: { poc: null, poc_estimated: false, resistance: null, support: null },
      reference_price: 100,
      symbol: 'BTCUSDT', interval: '1h', context_time: 1
    })
    const wrapper = mount(MarketAnalysisChat, { props: { symbol: 'BTCUSDT', interval: '1h' } })

    await wrapper.get('textarea[aria-label="向 AI 行情分析提问"]').setValue('请分析')
    await wrapper.get('.ai-chat-form').trigger('submit')
    await flushPromises()

    expect(wrapper.get('[role="alert"]').text()).toContain('AI 行情分析暂时失败')
    expect(wrapper.find('[data-role="assistant"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('provider text that must not be shown')
    expect((wrapper.get('textarea[aria-label="向 AI 行情分析提问"]').element as HTMLTextAreaElement).value).toBe('请分析')
    wrapper.unmount()
  })

  it('renders a Short direction with its bearish color and inverse price ordering', async () => {
    const plan = {
      direction: 'Short', entry_type: 'limit', entry_price: 102,
      take_profit: 95, stop_loss: 110, confidence: 64,
      leverage: 2, analysis: '价格跌破结构支撑，反弹无法收回时空头逻辑仍有效。'
    }
    mocks.send.mockResolvedValueOnce({
      reply: JSON.stringify(plan), plan,
      key_levels: { poc: null, poc_estimated: false, resistance: 110, support: 95 },
      reference_price: 100, symbol: 'BTCUSDT', interval: '1h', context_time: 1
    })
    const wrapper = mount(MarketAnalysisChat, { props: { symbol: 'BTCUSDT', interval: '1h' } })

    await wrapper.get('textarea[aria-label="向 AI 行情分析提问"]').setValue('分析空头结构')
    await wrapper.get('.ai-chat-form').trigger('submit')
    await flushPromises()

    expect(wrapper.get('.analysis-direction-short').text()).toContain('Short')
    expect(wrapper.get('.analysis-entry').text()).toContain('$102')
    expect(wrapper.get('.analysis-margin-risk').text()).toContain('16%')
    wrapper.unmount()
  })

  it('archives a late reply to the symbol that started the request', async () => {
    const btcReply = deferred<ReturnType<typeof makeChatResponse>>()
    mocks.send.mockReturnValueOnce(btcReply.promise).mockResolvedValueOnce(makeChatResponse('ETH reply', 'ETHUSDT', '1d', 2))
    const wrapper = mount(MarketAnalysisChat, { props: { symbol: 'BTCUSDT', interval: '1d' } })

    await wrapper.get('textarea[aria-label="向 AI 行情分析提问"]').setValue('BTC question')
    await wrapper.get('.ai-chat-form').trigger('submit')
    await wrapper.setProps({ symbol: 'ETHUSDT' })
    await wrapper.get('textarea[aria-label="向 AI 行情分析提问"]').setValue('ETH question')
    await wrapper.get('.ai-chat-form').trigger('submit')
    await flushPromises()
    expect(wrapper.text()).toContain('ETH reply')
    expect(wrapper.text()).not.toContain('BTC reply')

    btcReply.resolve(makeChatResponse('BTC reply', 'BTCUSDT', '1d', 1))
    await flushPromises()
    expect(wrapper.text()).toContain('ETH reply')
    expect(wrapper.text()).not.toContain('BTC reply')
    await wrapper.setProps({ symbol: 'BTCUSDT' })
    expect(wrapper.text()).toContain('BTC reply')
    wrapper.unmount()
  })

  it('keeps existing messages and the failed question available for retry', async () => {
    mocks.send
      .mockResolvedValueOnce(makeChatResponse('Existing answer', 'BTCUSDT', '1h', 1))
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
