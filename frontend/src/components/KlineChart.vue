<template>
  <div class="kline-chart-container">
    <div ref="chartContainer" class="chart"></div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, onUnmounted } from 'vue'
import { createChart, type IChartApi, type ISeriesApi, ColorType } from 'lightweight-charts'
import type { Candle, SRLevel } from '../services/api'

const props = defineProps<{
  candles: Candle[]
  resistance?: SRLevel[]
  support?: SRLevel[]
}>()

const chartContainer = ref<HTMLElement>()
let chart: IChartApi | null = null
let candlestickSeries: ISeriesApi<'Candlestick'> | null = null

onMounted(() => {
  if (!chartContainer.value) return

  // Create chart
  chart = createChart(chartContainer.value, {
    layout: {
      background: { type: ColorType.Solid, color: '#1a1a1a' },
      textColor: '#d1d4dc',
    },
    grid: {
      vertLines: { color: '#2a2a2a' },
      horzLines: { color: '#2a2a2a' },
    },
    width: chartContainer.value.clientWidth,
    height: 500,
    timeScale: {
      timeVisible: true,
      secondsVisible: false,
    },
  })

  // Add candlestick series
  candlestickSeries = chart.addCandlestickSeries({
    upColor: '#26a69a',
    downColor: '#ef5350',
    borderVisible: false,
    wickUpColor: '#26a69a',
    wickDownColor: '#ef5350',
  })

  updateChart()

  // Handle resize
  const resizeObserver = new ResizeObserver(() => {
    if (chart && chartContainer.value) {
      chart.applyOptions({ width: chartContainer.value.clientWidth })
    }
  })
  resizeObserver.observe(chartContainer.value)
})

onUnmounted(() => {
  if (chart) {
    chart.remove()
  }
})

watch(() => props.candles, () => {
  updateChart()
}, { deep: true })

watch(() => [props.resistance, props.support], () => {
  updateSRLevels()
}, { deep: true })

function updateChart() {
  if (!candlestickSeries || !props.candles.length) return

  const data = props.candles.map(candle => ({
    time: (candle.timestamp / 1000) as any,
    open: candle.open,
    high: candle.high,
    low: candle.low,
    close: candle.close,
  }))

  candlestickSeries.setData(data)
  updateSRLevels()
}

function updateSRLevels() {
  if (!chart) return

  // Add resistance levels
  props.resistance?.forEach((level, index) => {
    if (index < 3) { // Show top 3
      const color = `rgba(239, 83, 80, ${level.strength})`
      candlestickSeries?.createPriceLine({
        price: level.price,
        color: color,
        lineWidth: 2,
        lineStyle: 2,
        axisLabelVisible: true,
        title: `R ${level.price.toFixed(2)}`,
      })
    }
  })

  // Add support levels
  props.support?.forEach((level, index) => {
    if (index < 3) { // Show top 3
      const color = `rgba(38, 166, 154, ${level.strength})`
      candlestickSeries?.createPriceLine({
        price: level.price,
        color: color,
        lineWidth: 2,
        lineStyle: 2,
        axisLabelVisible: true,
        title: `S ${level.price.toFixed(2)}`,
      })
    }
  })
}
</script>

<style scoped>
.kline-chart-container {
  width: 100%;
  height: 500px;
  background: #1a1a1a;
  border-radius: 8px;
  overflow: hidden;
}

.chart {
  width: 100%;
  height: 100%;
}
</style>
