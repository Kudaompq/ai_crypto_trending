<template>
  <section class="ai-chat" aria-label="AI 行情分析" :data-symbol="symbol">
    <header class="chat-heading">
      <div>
        <h2>AI 行情分析</h2>
        <span>{{ symbol }} · {{ interval }}</span>
      </div>
      <span class="chat-readonly">只读分析</span>
    </header>

    <div ref="messageList" class="chat-messages" role="log" aria-live="polite" :aria-label="`${symbol} 对话记录`">
      <div v-if="activeMessages.length === 0" class="chat-empty">
        <span class="welcome-icon" aria-hidden="true">✦</span>
        <span class="welcome-eyebrow">{{ symbol }} · {{ interval }} 行情快照</span>
        <h3>这段行情，先看什么？</h3>
        <p>选择一个问题开始，或在下方输入你关心的方向。</p>
        <div class="prompt-suggestions" aria-label="快捷分析问题">
          <button v-for="suggestion in suggestions" :key="suggestion.label" type="button"
            :aria-label="suggestion.label" :disabled="isSending" @click="askSuggestion(suggestion.prompt)">
            <span aria-hidden="true">{{ suggestion.icon }}</span>{{ suggestion.label }}
          </button>
        </div>
        <div class="scope-note">分析范围：K 线、成交量、OI 与技术指标</div>
      </div>
      <article v-for="(message, index) in activeMessages" :key="`${symbol}-${index}`"
        class="chat-message" :class="`chat-message-${message.role}`" :data-role="message.role">
        <template v-if="message.role === 'assistant' && message.plan">
          <div class="analysis-card-heading">
            <span v-if="message.plan.status === 'actionable'" class="analysis-direction" :class="`analysis-direction-${message.plan.direction?.toLowerCase()}`">
              <span aria-hidden="true">{{ message.plan.direction === 'Long' ? '↗' : '↘' }}</span>
              {{ message.plan.direction }}
            </span>
            <span v-if="message.plan.status === 'actionable'" class="analysis-market-meta">{{ message.contextInterval }} · {{ message.plan.entry_type === 'market' ? '市价' : '限价' }}</span>
            <span class="analysis-time" :class="`analysis-time-${message.plan.timing}`">{{ timingLabel(message.plan.timing) }}</span>
          </div>
          <div v-if="message.plan.status === 'actionable'" class="analysis-prices">
            <div class="analysis-price analysis-entry"><span>入场价</span><strong>{{ formatPrice(message.plan.entry_price!) }}</strong></div>
            <div class="analysis-price analysis-take-profit"><span>止盈</span><strong>{{ formatPrice(message.plan.take_profit!) }}</strong></div>
            <div class="analysis-price analysis-stop-loss"><span>止损</span><strong>{{ formatPrice(message.plan.stop_loss!) }}</strong></div>
          </div>
          <div v-if="message.plan.status === 'actionable'" class="analysis-risk-row">
            <div class="confidence-meter" :style="{ '--confidence': `${message.plan.confidence}%` }"
              role="img" :aria-label="`置信度 ${message.plan.confidence}`">
              <span>{{ message.plan.confidence }}</span>
            </div>
            <div class="analysis-risk-copy">
              <span class="analysis-confidence">置信度 {{ message.plan.confidence }}</span>
              <strong>建议杠杆 ~{{ formatLeverage(message.plan.leverage!) }}x</strong>
              <p class="analysis-margin-risk">打到止损约亏保证金 {{ formatMarginLoss(message.plan) }}%</p>
            </div>
          </div>
          <section class="analysis-section analysis-narrative">
            <h3>行情分析</h3><p>{{ message.plan.analysis }}</p>
          </section>
          <section class="analysis-section analysis-section-levels">
            <h3>关键价位</h3>
            <div class="analysis-levels-grid">
              <div class="analysis-level-row analysis-key-level-resistance">
                <span>阻力</span>
                <strong>{{ formatKeyLevel(message.keyLevels?.resistance) }}</strong>
                <small>{{ formatLevelDistance(message.keyLevels?.resistance, message.referencePrice) }}</small>
              </div>
              <div class="analysis-level-row analysis-key-level-support">
                <span>支撑</span>
                <strong>{{ formatKeyLevel(message.keyLevels?.support) }}</strong>
                <small>{{ formatLevelDistance(message.keyLevels?.support, message.referencePrice) }}</small>
              </div>
            </div>
          </section>
          <p class="analysis-disclaimer">AI 行情推演，仅作研究参考。<template v-if="message.plan.status === 'actionable'">止损亏损估算不含手续费、滑点和资金费率。</template></p>
        </template>
        <div v-else class="chat-message-content">{{ message.content }}</div>
        <time v-if="message.role === 'assistant' && message.contextTime"
          class="analysis-market-time"
          :datetime="new Date(message.contextTime).toISOString()" :data-context-time="message.contextTime">
          数据时间 {{ formatContextTime(message.contextTime) }} · {{ message.contextInterval }}
        </time>
      </article>
      <div v-if="isSending" class="chat-pending" role="status">正在分析 {{ symbol }}…</div>
    </div>

    <div v-if="activeError" class="chat-error" role="alert">{{ activeError }}</div>

    <form class="ai-chat-form" @submit.prevent="sendMessage">
      <textarea v-model="draft" aria-label="向 AI 行情分析提问" rows="3" maxlength="4000"
        :disabled="isSending" placeholder="输入你想了解的问题…"
        @keydown.enter.exact.prevent="sendMessage" />
      <div class="chat-composer-footer">
        <span>基于服务端最新 K 线、成交量与 OI</span>
        <button type="submit" aria-label="发送消息" :disabled="!draft.trim() || isSending">发送</button>
      </div>
    </form>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, reactive, ref, watch } from 'vue'
import { api, type MarketAnalysisChatMessage, type MarketAnalysisChatResponse, type MarketAnalysisKeyLevels, type MarketAnalysisPlan } from '../services/api'

interface DisplayMessage extends MarketAnalysisChatMessage {
  contextTime?: number
  contextInterval?: string
  plan?: MarketAnalysisPlan
  keyLevels?: MarketAnalysisKeyLevels
  referencePrice?: number
}

const suggestions = [
  { label: '生成方向分析', icon: '↗', prompt: '根据当前快照生成结构化方向分析，说明入场、止盈止损和判断依据。' },
  { label: '查看关键价位', icon: '⌖', prompt: '解释当前方向分析引用的支撑和阻力，以及它们与最新收盘价的位置关系。' },
  { label: '结构与风险', icon: '◇', prompt: '总结当前行情结构和方向计划，并说明判断失效条件。' }
]

const props = defineProps<{ symbol: string; interval: string }>()
const conversations = reactive<Record<string, DisplayMessage[]>>({})
const drafts = reactive<Record<string, string>>({})
const errors = reactive<Record<string, string>>({})
const pending = reactive<Record<string, boolean>>({})
const messageList = ref<HTMLElement | null>(null)
const activeMessages = computed(() => conversations[props.symbol] ?? [])
const draft = computed({
  get: () => drafts[props.symbol] ?? '',
  set: (value: string) => { drafts[props.symbol] = value }
})
const activeError = computed(() => errors[props.symbol] ?? '')
const isSending = computed(() => pending[props.symbol] ?? false)

watch(activeMessages, async () => {
  await nextTick()
  if (messageList.value) messageList.value.scrollTop = messageList.value.scrollHeight
}, { deep: true })

async function sendMessage() {
  const symbol = props.symbol
  const interval = props.interval
  const message = (drafts[symbol] ?? '').trim()
  if (!message || pending[symbol]) return

  const history = (conversations[symbol] ?? []).slice(-20).map(({ role, content }) => ({ role, content }))
  errors[symbol] = ''
  pending[symbol] = true
  try {
    const response = await api.sendMarketAnalysisMessage({ symbol, interval, message, history })
    if (response.symbol !== symbol || response.interval !== interval || !Number.isFinite(response.context_time)) {
      throw new Error('分析结果与当前请求的交易对或周期不一致')
    }
    if (!isValidAnalysisResponse(response)) {
      throw new Error('AI 返回的分析格式无效，请重试')
    }
    const thread = conversations[symbol] ?? (conversations[symbol] = [])
    thread.push(
      { role: 'user', content: message },
      {
        role: 'assistant', content: response.reply, contextTime: response.context_time,
        contextInterval: response.interval, plan: response.plan,
        keyLevels: response.key_levels, referencePrice: response.reference_price
      }
    )
    if ((drafts[symbol] ?? '').trim() === message) drafts[symbol] = ''
  } catch (error: unknown) {
    errors[symbol] = api.errorMessage(error, 'AI 行情分析暂时失败，请重试')
  } finally {
    pending[symbol] = false
  }
}

function askSuggestion(message: string) {
  drafts[props.symbol] = message
  void sendMessage()
}

function isValidAnalysisResponse(response: MarketAnalysisChatResponse): boolean {
  const plan = response.plan
  const validPrice = (value: number) => Number.isFinite(value) && value > 0
  const validLevel = (value: number | null) => value === null || validPrice(value)
  if (!plan || !response.key_levels || !validPrice(response.reference_price)) return false
  if (!Number.isInteger(plan.confidence) || plan.confidence < 0 || plan.confidence > 100) return false
  if (!plan.analysis.trim()) return false
  if (plan.status === 'wait') {
    if (plan.timing !== 'undetermined') return false
    if (plan.direction !== undefined || plan.entry_type !== undefined || plan.entry_price !== undefined || plan.take_profit !== undefined || plan.stop_loss !== undefined || plan.leverage !== undefined) return false
    return validLevel(response.key_levels.resistance) && validLevel(response.key_levels.support)
  }
  if (plan.status !== 'actionable' || (plan.timing !== 'left' && plan.timing !== 'right')) return false
  if (!validPrice(plan.entry_price!) || !validPrice(plan.take_profit!) || !validPrice(plan.stop_loss!)) return false
  if (!Number.isFinite(plan.leverage) || plan.leverage! < 1 || plan.leverage! > 5) return false
  if (plan.direction === 'Long' && !(plan.take_profit! > plan.entry_price! && plan.entry_price! > plan.stop_loss!)) return false
  if (plan.direction === 'Short' && !(plan.take_profit! < plan.entry_price! && plan.entry_price! < plan.stop_loss!)) return false
  if (plan.direction !== 'Long' && plan.direction !== 'Short') return false
  if (plan.entry_type !== 'market' && plan.entry_type !== 'limit') return false
  return validLevel(response.key_levels.resistance) && validLevel(response.key_levels.support)
}

function formatContextTime(timestamp: number): string {
  return new Date(timestamp).toLocaleString('zh-CN')
}

function timingLabel(timing: MarketAnalysisPlan['timing']): string {
  if (timing === 'left') return '左侧'
  if (timing === 'right') return '右侧'
  return '暂不可判定 / 等待确认'
}

function formatPrice(value: number): string {
  return `$${new Intl.NumberFormat('en-US', { maximumSignificantDigits: 8 }).format(value)}`
}

function formatKeyLevel(value: number | null | undefined): string {
  return value === null || value === undefined ? '未识别' : formatPrice(value)
}

function formatLevelDistance(value: number | null | undefined, referencePrice: number | undefined): string {
  if (value === null || value === undefined || !referencePrice || referencePrice <= 0) return '—'
  const difference = ((value - referencePrice) / referencePrice) * 100
  return `${difference > 0 ? '+' : ''}${difference.toFixed(1)}%`
}

function formatLeverage(value: number): string {
  return Number.isInteger(value) ? String(value) : value.toFixed(1)
}

function formatMarginLoss(plan: MarketAnalysisPlan): string {
  return String(Math.round(Math.abs(plan.entry_price! - plan.stop_loss!) / plan.entry_price! * plan.leverage! * 100))
}
</script>

<style scoped>
.ai-chat {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  color: #ddd;
}

.chat-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 16px;
  border-bottom: 1px solid #303030;
}

.chat-heading h2 {
  margin: 0 0 4px;
  color: #eee;
  font-size: 15px;
}

.chat-heading span,
.chat-readonly,
.chat-composer-footer span {
  color: #909090;
  font-size: 11px;
}

.chat-readonly {
  padding: 4px 7px;
  border: 1px solid #3c3c3c;
  border-radius: 999px;
}

.chat-messages {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 12px;
  min-height: 180px;
  padding: 14px;
  overflow-y: auto;
}

.chat-empty {
  display: flex;
  width: min(100%, 340px);
  margin: auto;
  align-items: center;
  flex-direction: column;
  text-align: center;
}

.welcome-icon {
  display: grid;
  width: 42px;
  height: 42px;
  margin-bottom: 14px;
  place-items: center;
  border: 1px solid #403b67;
  border-radius: 14px;
  background: linear-gradient(145deg, #302955, #202033);
  color: #b5a5ff;
  font-size: 21px;
}

.welcome-eyebrow {
  color: #a69af4;
  font-size: 11px;
  letter-spacing: .03em;
}

.chat-empty h3 {
  margin: 9px 0 5px;
  color: #f0eff7;
  font-size: 17px;
  font-weight: 600;
}

.chat-empty > p {
  margin: 0;
  color: #96949f;
  font-size: 12px;
  line-height: 1.6;
}

.prompt-suggestions {
  display: flex;
  width: 100%;
  margin-top: 20px;
  flex-direction: column;
  gap: 8px;
}

.prompt-suggestions button {
  display: flex;
  min-height: 38px;
  align-items: center;
  gap: 9px;
  padding: 0 12px;
  border: 1px solid #35343c;
  border-radius: 9px;
  background: #232329;
  color: #d5d3df;
  font: inherit;
  font-size: 12px;
  text-align: left;
  cursor: pointer;
  transition: border-color .16s, background .16s;
}

.prompt-suggestions button:hover:not(:disabled) {
  border-color: #6256a0;
  background: #2b2935;
}

.prompt-suggestions button:disabled { cursor: wait; opacity: .65; }
.prompt-suggestions button span { color: #a99aff; font-size: 15px; }

.scope-note {
  margin-top: 14px;
  color: #73717b;
  font-size: 10px;
}

.chat-message {
  max-width: 94%;
  padding: 10px 12px;
  border-radius: 10px;
  font-size: 13px;
  line-height: 1.55;
  overflow-wrap: anywhere;
}

.chat-message-user {
  align-self: flex-end;
  background: #303a69;
  color: #f1f2ff;
}

.chat-message-assistant {
  width: 100%;
  max-width: 100%;
  box-sizing: border-box;
  align-self: stretch;
  border: 1px solid #35343c;
  background: linear-gradient(145deg, #222228, #1d1d22);
  color: #e2e2e2;
}

.chat-message-content { white-space: pre-wrap; }

.analysis-card-heading {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px 10px;
  margin-bottom: 18px;
}

.analysis-direction {
  display: inline-flex;
  min-height: 34px;
  align-items: center;
  gap: 8px;
  padding: 0 11px;
  border-radius: 10px;
  font-size: 17px;
  font-weight: 650;
}

.analysis-direction span { display: grid; width: 27px; height: 27px; place-items: center; border-radius: 8px; }
.analysis-direction-long { background: #1d2926; color: #08d49b; }
.analysis-direction-long span { background: #20352f; }
.analysis-direction-short { background: #302126; color: #ff5475; }
.analysis-direction-short span { background: #42262e; }
.analysis-market-meta { padding: 5px 8px; border-radius: 8px; background: #303034; color: #a9a8b0; font-size: 11px; }
.analysis-time { margin-left: auto; padding: 5px 8px; border-radius: 8px; font-size: 11px; }
.analysis-time-left { background: #302a42; color: #c5a8ff; }
.analysis-time-right { background: #1d302b; color: #70dfb9; }
.analysis-time-undetermined { background: #303034; color: #b0aeb8; }

.analysis-prices {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  margin-bottom: 18px;
}

.analysis-price { display: flex; min-width: 0; flex-direction: column; gap: 5px; }
.analysis-price span { color: #92919a; font-size: 11px; }
.analysis-price strong { color: #f0eff5; font-size: clamp(15px, 4.4vw, 20px); font-weight: 550; white-space: nowrap; }
.analysis-take-profit strong { color: #08d49b; }
.analysis-stop-loss strong { color: #ff4f70; }

.analysis-risk-row { display: flex; align-items: center; gap: 12px; margin: 2px 0 18px; }
.confidence-meter {
  display: grid;
  width: 56px;
  height: 56px;
  flex: 0 0 56px;
  place-items: center;
  border-radius: 50%;
  background: conic-gradient(#ffc444 var(--confidence), #37383b 0);
  color: #ffce50;
  font-size: 19px;
  font-weight: 650;
  position: relative;
}
.confidence-meter::before { position: absolute; inset: 5px; border-radius: 50%; background: #202024; content: ''; }
.confidence-meter span { z-index: 1; }
.analysis-risk-copy { display: flex; min-width: 0; flex-direction: column; gap: 3px; }
.analysis-confidence { color: #9a98a1; font-size: 11px; }
.analysis-risk-copy strong { color: #e8e6ed; font-size: 14px; font-weight: 550; }
.analysis-margin-risk { margin: 0; color: #aaa8b0; font-size: 12px; }

.analysis-section { margin-top: 12px; }
.analysis-section h3 { margin: 0 0 7px; color: #9996a2; font-size: 11px; font-weight: 500; }
.analysis-section p { margin: 0; color: #d0ced7; font-size: 13px; line-height: 1.65; white-space: pre-wrap; }
.analysis-narrative { margin-top: 14px; }
.analysis-section-levels { padding-top: 12px; border-top: 1px solid #34333a; }
.analysis-levels-grid { display: grid; grid-template-columns: minmax(0, 1fr); gap: 7px; }
.analysis-level-row { display: grid; grid-template-columns: minmax(72px, 1fr) auto auto; align-items: baseline; gap: 9px; min-width: 0; }
.analysis-level-row span { color: #898790; font-size: 12px; }
.analysis-level-row strong { color: #dddbe3; font-size: 13px; font-weight: 550; white-space: nowrap; }
.analysis-level-row small { color: #92909a; font-size: 10px; white-space: nowrap; }
.analysis-key-level-resistance strong { color: #ff7387; }
.analysis-key-level-support strong { color: #08d49b; }
.analysis-disclaimer { margin: 12px 0 0; color: #77757e; font-size: 10px; line-height: 1.5; }
.analysis-market-time { padding-top: 8px; border-top: 1px solid #303034; color: #85838d !important; }

.chat-message time {
  display: block;
  margin-top: 8px;
  color: #969696;
  font-size: 10px;
}

.chat-pending {
  align-self: flex-start;
  color: #aeb8ff;
  font-size: 12px;
}

.chat-error {
  margin: 0 14px 10px;
  padding: 9px 10px;
  border: 1px solid rgba(239, 83, 80, .4);
  border-radius: 8px;
  background: rgba(239, 83, 80, .08);
  color: #ff9b98;
  font-size: 12px;
}

.ai-chat-form {
  flex: 0 0 auto;
  padding: 12px 14px 14px;
  border-top: 1px solid #303030;
}

.ai-chat-form textarea {
  display: block;
  width: 100%;
  min-height: 72px;
  resize: vertical;
  box-sizing: border-box;
  padding: 10px;
  border: 1px solid #3b3b3b;
  border-radius: 8px;
  outline: none;
  background: #111;
  color: #eee;
  font: inherit;
  font-size: 13px;
}

.ai-chat-form textarea:focus { border-color: #7a83d9; }
.ai-chat-form textarea::placeholder { color: #747474; }

.chat-composer-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-top: 8px;
}

.chat-composer-footer button {
  padding: 7px 13px;
  border: 0;
  border-radius: 7px;
  background: #6674da;
  color: #fff;
  cursor: pointer;
}

.chat-composer-footer button:disabled {
  background: #383838;
  color: #858585;
  cursor: not-allowed;
}
</style>
