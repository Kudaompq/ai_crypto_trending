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
      <p v-if="activeMessages.length === 0" class="chat-empty">询问 {{ symbol }} 当前周期的走势、指标或行情结构</p>
      <article v-for="(message, index) in activeMessages" :key="`${symbol}-${index}`"
        class="chat-message" :class="`chat-message-${message.role}`" :data-role="message.role">
        <div class="chat-message-content">{{ message.content }}</div>
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
}

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
      { role: 'assistant', content: response.reply, contextTime: response.context_time, contextInterval: response.interval }
    )
    if ((drafts[symbol] ?? '').trim() === message) drafts[symbol] = ''
  } catch (error: unknown) {
    errors[symbol] = api.errorMessage(error, 'AI 行情分析暂时失败，请重试')
  } finally {
    pending[symbol] = false
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
  margin: auto 0;
  color: #8e8e8e;
  font-size: 13px;
  line-height: 1.6;
  text-align: center;
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
  align-self: flex-start;
  border: 1px solid #333;
  background: #202020;
  color: #e2e2e2;
}

.chat-message-content { white-space: pre-wrap; }

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
