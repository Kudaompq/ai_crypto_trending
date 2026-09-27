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
        <div class="scope-note">分析范围：当前周期 K 线与技术指标</div>
      </div>
      <article v-for="(message, index) in activeMessages" :key="`${symbol}-${index}`"
        class="chat-message" :class="`chat-message-${message.role}`" :data-role="message.role">
        <template v-if="message.role === 'assistant' && message.analysis">
          <div class="analysis-card-heading">
            <span class="analysis-card-label">行情观点</span>
            <span class="analysis-stance" :class="`analysis-stance-${message.analysis.tone}`">{{ message.analysis.stance }}</span>
          </div>
          <p class="analysis-summary">{{ message.analysis.conclusion }}</p>
          <section v-if="message.analysis.evidence" class="analysis-section">
            <h3>结构依据</h3><p>{{ message.analysis.evidence }}</p>
          </section>
          <section v-if="message.analysis.levels" class="analysis-section analysis-section-levels">
            <h3>关键价位</h3><p>{{ message.analysis.levels }}</p>
          </section>
          <section v-if="message.analysis.risk" class="analysis-section analysis-section-risk">
            <h3>风险提示</h3><p>{{ message.analysis.risk }}</p>
          </section>
        </template>
        <div v-else class="chat-message-content">{{ message.content }}</div>
        <time v-if="message.role === 'assistant' && message.contextTime"
          :datetime="new Date(message.contextTime).toISOString()" :data-context-time="message.contextTime">
          行情截至 {{ formatContextTime(message.contextTime) }} · {{ message.contextInterval }}
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
        <span>基于服务端最新 K 线与指标</span>
        <button type="submit" aria-label="发送消息" :disabled="!draft.trim() || isSending">发送</button>
      </div>
    </form>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, reactive, ref, watch } from 'vue'
import { api, type MarketAnalysisChatMessage } from '../services/api'

interface DisplayMessage extends MarketAnalysisChatMessage {
  contextTime?: number
  contextInterval?: string
  analysis?: StructuredAnalysis
}

interface StructuredAnalysis {
  stance: string
  tone: 'bullish' | 'bearish' | 'neutral' | 'wait'
  conclusion: string
  evidence: string
  levels: string
  risk: string
}

const suggestions = [
  { label: '判断当前趋势', icon: '↗', prompt: '基于当前提供的行情快照和指标，判断趋势偏多、偏空、震荡还是观望，并简要说明依据。' },
  { label: '关键支撑与阻力', icon: '⌖', prompt: '根据当前分析结果，整理最近的关键支撑和阻力位，并说明当前价格所在位置。' },
  { label: '结构与风险', icon: '◇', prompt: '总结当前趋势结构、关键价位与需要关注的风险；只使用本次提供的数据。' }
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
    const thread = conversations[symbol] ?? (conversations[symbol] = [])
    thread.push(
      { role: 'user', content: message },
      {
        role: 'assistant', content: response.reply, contextTime: response.context_time,
        contextInterval: response.interval, analysis: parseAnalysis(response.reply)
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

function parseAnalysis(reply: string): StructuredAnalysis | undefined {
  const labels = new Map<string, string>()
  let currentLabel = ''
  for (const rawLine of reply.split(/\r?\n/)) {
    const line = rawLine.trim()
    const match = /^(观点|结论|结构依据|关键价位|风险提示)\s*[：:]\s*(.*)$/.exec(line)
    if (match) {
      currentLabel = match[1] ?? ''
      if (!currentLabel) continue
      labels.set(currentLabel, match[2] ?? '')
    } else if (currentLabel && line) {
      labels.set(currentLabel, `${labels.get(currentLabel)} ${line}`.trim())
    }
  }

  const stanceLine = labels.get('观点') ?? ''
  const stance = /(偏多|偏空|震荡|观望)/.exec(stanceLine)?.[1]
  const conclusion = labels.get('结论')
  if (!stance || !conclusion) return undefined

  const tone: StructuredAnalysis['tone'] = stance === '偏多' ? 'bullish' : stance === '偏空' ? 'bearish' : stance === '观望' ? 'wait' : 'neutral'
  return {
    stance, tone, conclusion,
    evidence: labels.get('结构依据') ?? '',
    levels: labels.get('关键价位') ?? '',
    risk: labels.get('风险提示') ?? ''
  }
}

function formatContextTime(timestamp: number): string {
  return new Date(timestamp).toLocaleString('zh-CN')
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
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 9px;
}

.analysis-card-label { color: #aaa7b4; font-size: 11px; }

.analysis-stance {
  padding: 4px 9px;
  border: 1px solid currentColor;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 600;
}

.analysis-stance-bullish { border-color: #135b49; background: #12392f; color: #39d3a1; }
.analysis-stance-bearish { border-color: #73394b; background: #40242d; color: #ff718c; }
.analysis-stance-neutral { border-color: #60532b; background: #39321f; color: #e8c76f; }
.analysis-stance-wait { border-color: #484852; background: #303038; color: #c2c0ca; }

.analysis-summary {
  margin: 0 0 14px;
  color: #eeeef3;
  font-size: 13px;
  font-weight: 500;
  line-height: 1.6;
}

.analysis-section { margin-top: 12px; }
.analysis-section h3 { margin: 0 0 4px; color: #9996a2; font-size: 10px; font-weight: 500; }
.analysis-section p { margin: 0; color: #d0ced7; font-size: 12px; line-height: 1.55; white-space: pre-wrap; }
.analysis-section-levels p { color: #bdb3ff; }
.analysis-section-risk { padding-top: 10px; border-top: 1px solid #34333a; }
.analysis-section-risk p { color: #ddaeb4; }

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
