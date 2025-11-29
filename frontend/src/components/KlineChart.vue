<template>
  <div class="kline-chart-container">
    <div ref="chartContainer" class="chart"></div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, onUnmounted, nextTick } from 'vue'
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
let priceLines: any[] = [] // Track price lines for removal

onMounted(async () => {
  await nextTick()
  
  if (!chartContainer.value) {
    console.error('Chart container not found')
    return
  }

  console.log('Initializing chart...')
  console.log('Container dimensions:', chartContainer.value.clientWidth, 'x', chartContainer.value.clientHeight)
  console.log('Candles data:', props.candles.length, 'candles')

  try {
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

    console.log('Chart created successfully')

    // Add candlestick series
    candlestickSeries = chart.addCandlestickSeries({
      upColor: '#26a69a',
      downColor: '#ef5350',
      borderVisible: false,
      wickUpColor: '#26a69a',
      wickDownColor: '#ef5350',
    })

    console.log('Candlestick series added')

    // Update chart with initial data
    if (props.candles && props.candles.length > 0) {
      updateChart()
    }

    // Handle resize
    const resizeObserver = new ResizeObserver(() => {
      if (chart && chartContainer.value) {
        chart.applyOptions({ width: chartContainer.value.clientWidth })
      }
    })
    resizeObserver.observe(chartContainer.value)
  } catch (error) {
    console.error('Error initializing chart:', error)
  }
})

onUnmounted(() => {
  if (chart) {
    chart.remove()
  }
})

watch(() => props.candles, (newCandles) => {
  console.log('Candles updated:', newCandles?.length, 'candles')
  if (newCandles && newCandles.length > 0) {
    updateChart()
  }
}, { deep: true })

watch(() => [props.resistance, props.support], ([newResistance, newSupport]) => {
  console.log('SR levels updated - Resistance:', newResistance?.length, 'Support:', newSupport?.length)
  updateSRLevels()
}, { deep: true })

function updateChart() {
  if (!candlestickSeries || !props.candles || props.candles.length === 0) {
    console.warn('Cannot update chart: series or candles missing')
    return
  }

  console.log('Updating chart with', props.candles.length, 'candles')

  const data = props.candles.map(candle => ({
    time: (candle.timestamp / 1000) as any,
    open: candle.open,
    high: candle.high,
    low: candle.low,
    close: candle.close,
  }))

  console.log('First candle:', data[0])
  console.log('Last candle:', data[data.length - 1])

  candlestickSeries.setData(data)
  console.log('Chart data set successfully')
  
  // Update SR levels after chart data is set
  updateSRLevels()
}

function updateSRLevels() {
  if (!candlestickSeries) {
    console.warn('Cannot update SR levels: series not initialized')
    return
  }

  console.log('Updating SR levels...')

  // Remove old price lines
  priceLines.forEach(line => {
    try {
      candlestickSeries?.removePriceLine(line)
    } catch (e) {
      console.warn('Error removing price line:', e)
    }
  })
  priceLines = []

  // Add resistance levels
  if (props.resistance && props.resistance.length > 0) {
    console.log('Adding', props.resistance.length, 'resistance levels')
    props.resistance.forEach((level, index) => {
      if (index < 5) {
        const strength = Math.max(0.4, level.strength)
        const color = `rgba(239, 83, 80, ${strength})`
        console.log(`Resistance ${index + 1}: $${level.price.toFixed(2)} (strength: ${strength})`)
        
        try {
          const line = candlestickSeries?.createPriceLine({
            price: level.price,
            color: color,
            lineWidth: 2,
            lineStyle: 2, // Dashed
            axisLabelVisible: true,
            title: `R ${level.price.toFixed(2)}`,
          })
          if (line) priceLines.push(line)
        } catch (e) {
          console.error('Error creating resistance line:', e)
        }
      }
    })
  }

  // Add support levels
  if (props.support && props.support.length > 0) {
    console.log('Adding', props.support.length, 'support levels')
    props.support.forEach((level, index) => {
      if (index < 5) {
        const strength = Math.max(0.4, level.strength)
        const color = `rgba(38, 166, 154, ${strength})`
        console.log(`Support ${index + 1}: $${level.price.toFixed(2)} (strength: ${strength})`)
        
        try {
          const line = candlestickSeries?.createPriceLine({
            price: level.price,
            color: color,
            lineWidth: 2,
            lineStyle: 2, // Dashed
            axisLabelVisible: true,
            title: `S ${level.price.toFixed(2)}`,
          })
          if (line) priceLines.push(line)
        } catch (e) {
          console.error('Error creating support line:', e)
        }
      }
    })
  }

  console.log('SR levels updated. Total price lines:', priceLines.length)
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
