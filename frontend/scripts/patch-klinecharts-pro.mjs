import { readFile, writeFile } from 'node:fs/promises'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { patchLocaleMerge } from './pro-patch-utils.mjs'

const scriptDirectory = dirname(fileURLToPath(import.meta.url))
const packageDirectory = resolve(scriptDirectory, '../node_modules/@klinecharts/pro')
const packageMetadata = JSON.parse(await readFile(resolve(packageDirectory, 'package.json'), 'utf8'))

if (packageMetadata.version !== '0.1.1') {
  throw new Error(`Expected @klinecharts/pro 0.1.1, received ${packageMetadata.version}`)
}

function replaceOnce(source, current, replacement, description) {
  if (source.includes(replacement)) return source
  const occurrences = source.split(current).length - 1
  if (occurrences !== 1) {
    throw new Error(`Expected one ${description} patch point in @klinecharts/pro 0.1.1, found ${occurrences}`)
  }
  return source.replace(current, replacement)
}

const runtimePath = resolve(packageDirectory, 'dist/klinecharts-pro.js')
let runtime = await readFile(runtimePath, 'utf8')
runtime = replaceOnce(runtime,
  '    B1(this, "_chartApi", null);',
  '    B1(this, "_chartApi", null);\n    B1(this, "_disposeSolid", null);',
  'Solid disposer storage')
runtime = replaceOnce(runtime,
  '    H5(() => d(Dl, {',
  '    this._disposeSolid = H5(() => d(Dl, {',
  'Vue lifecycle disposer')
runtime = replaceOnce(runtime,
  '    getPeriod: () => L()\n  });',
  '    getPeriod: () => L(),\n    getChart: () => n\n  });',
  'public Core chart accessor')
runtime = replaceOnce(runtime,
  'n.applyMoreData(X, X.length > 0)',
  'n.applyMoreData(X, typeof e.datafeed.hasMoreHistory === "function" ? e.datafeed.hasMoreHistory(y(), _) : X.length > 0)',
  'history pagination availability')
runtime = replaceOnce(runtime,
  'n.applyNewData(X, X.length > 0)',
  'n.applyNewData(X, typeof e.datafeed.hasMoreHistory === "function" ? e.datafeed.hasMoreHistory(f, v) : X.length > 0)',
  'initial history availability')
runtime = replaceOnce(runtime,
  '"AO"].map((t) => {',
  '"AO", "OPEN_INTEREST"].map((t) => {',
  'custom open-interest indicator menu entry')
runtime = replaceOnce(runtime,
  '  PVT: [],\n  PSY:',
  '  PVT: [],\n  OPEN_INTEREST: [],\n  PSY:',
  'custom open-interest indicator settings')
runtime = patchLocaleMerge(runtime)
runtime = replaceOnce(runtime,
  '  setTheme(t) {\n    var n;',
  '  getChart() {\n    var t;\n    return (t = this._chartApi) == null ? null : t.getChart();\n  }\n  dispose() {\n    var t;\n    (t = this._disposeSolid) == null || t.call(this), this._disposeSolid = null, this._chartApi = null;\n  }\n  setTheme(t) {\n    var n;',
  'public Core chart and lifecycle methods')
await writeFile(runtimePath, runtime)

const typesPath = resolve(packageDirectory, 'dist/index.d.ts')
let types = await readFile(typesPath, 'utf8')
types = replaceOnce(types,
  "import { DeepPartial, KLineData, Styles } from 'klinecharts';",
  "import { Chart, DeepPartial, KLineData, Styles } from 'klinecharts';",
  'Core chart type import')
types = replaceOnce(types,
  `export interface Datafeed {
	searchSymbols(search?: string): Promise<SymbolInfo[]>;
	getHistoryKLineData(symbol: SymbolInfo, period: Period, from: number, to: number): Promise<KLineData[]>;
	subscribe(symbol: SymbolInfo, period: Period, callback: DatafeedSubscribeCallback): void;
	unsubscribe(symbol: SymbolInfo, period: Period): void;
}`,
  `export interface Datafeed {
	searchSymbols(search?: string): Promise<SymbolInfo[]>;
	getHistoryKLineData(symbol: SymbolInfo, period: Period, from: number, to: number): Promise<KLineData[]>;
	subscribe(symbol: SymbolInfo, period: Period, callback: DatafeedSubscribeCallback): void;
	unsubscribe(symbol: SymbolInfo, period: Period): void;
	hasMoreHistory?(symbol: SymbolInfo, period: Period): boolean;
}`,
  'Datafeed pagination state declaration')
types = replaceOnce(types,
  `export interface ChartPro {
	setTheme(theme: string): void;
	getTheme(): string;
	setStyles(styles: DeepPartial<Styles>): void;
	getStyles(): Styles;
	setLocale(locale: string): void;
	getLocale(): string;
	setTimezone(timezone: string): void;
	getTimezone(): string;
	setSymbol(symbol: SymbolInfo): void;
	getSymbol(): SymbolInfo;
	setPeriod(period: Period): void;
	getPeriod(): Period;
}`,
  `export interface ChartPro {
	setTheme(theme: string): void;
	getTheme(): string;
	setStyles(styles: DeepPartial<Styles>): void;
	getStyles(): Styles;
	setLocale(locale: string): void;
	getLocale(): string;
	setTimezone(timezone: string): void;
	getTimezone(): string;
	setSymbol(symbol: SymbolInfo): void;
	getSymbol(): SymbolInfo;
	setPeriod(period: Period): void;
	getPeriod(): Period;
	getChart(): Chart | null;
	dispose(): void;
}`,
  'ChartPro lifecycle and Core API declarations')
types = replaceOnce(types,
  `export declare class KLineChartPro implements ChartPro {
	constructor(options: ChartProOptions);
	private _container;
	private _chartApi;
	setTheme(theme: string): void;
	getTheme(): string;
	setStyles(styles: DeepPartial<Styles>): void;
	getStyles(): Styles;
	setLocale(locale: string): void;
	getLocale(): string;
	setTimezone(timezone: string): void;
	getTimezone(): string;
	setSymbol(symbol: SymbolInfo): void;
	getSymbol(): SymbolInfo;
	setPeriod(period: Period): void;
	getPeriod(): Period;
}`,
  `export declare class KLineChartPro implements ChartPro {
	constructor(options: ChartProOptions);
	private _container;
	private _chartApi;
	private _disposeSolid;
	setTheme(theme: string): void;
	getTheme(): string;
	setStyles(styles: DeepPartial<Styles>): void;
	getStyles(): Styles;
	setLocale(locale: string): void;
	getLocale(): string;
	setTimezone(timezone: string): void;
	getTimezone(): string;
	setSymbol(symbol: SymbolInfo): void;
	getSymbol(): SymbolInfo;
	setPeriod(period: Period): void;
	getPeriod(): Period;
	getChart(): Chart | null;
	dispose(): void;
}`,
  'KLineChartPro lifecycle declarations')
await writeFile(typesPath, types)

console.log('Patched @klinecharts/pro 0.1.1 for lifecycle cleanup, Core access, pagination state, OI menu support, and locale merging.')
