# audiodsp

モノラル波形のピッチ推定・時間伸縮・対数間隔の振幅計算・正規化相関と、16-bit PCM WAVの読み書きを行うGoライブラリです。波形処理にはfloat64の配列を使います。`pitch`、`stretch`、`spectrum`、`correlation`、`wav`の5パッケージを提供し、標準ライブラリだけに依存しています。Go 1.21以上で利用できます。

```sh
go get github.com/yh2237/audiodsp@v0.5.1
```

## 使用例

```go
import "github.com/yh2237/audiodsp/pitch"

hz := pitch.Estimate(frame, sampleRate)
medianHz := pitch.EstimateMedian(wave, sampleRate)
```

連続するフレームの処理には、ゼロ値の`Detector`を再利用できます。

```go
var detector pitch.Detector
for _, frame := range frames {
    hz := detector.Estimate(frame, sampleRate)
    // hzを使用する
}
```

生成した正弦波を使う例は`go run ./examples/pitch`で実行できます。

## ピッチ推定

- 入力は正規化されたモノラル波形（目安は-1〜1）と、Hz単位のサンプルレートです。入力配列を変更せず、その参照も保持しません。
- 推定した周波数をHz単位で返します。推定不能、無音、DC信号、短すぎる入力、サンプルレートが0以下の場合は0を返します。
- RMSと平均除去後のRMSが0.003未満の入力は推定しません。
- lag探索範囲は`max(2, sampleRate/500)`から`min(len(frame)/2, sampleRate/60)`です。累積平均で正規化した差分と局所補間を用います。
- `EstimateMedian`は40ms窓を10ms刻みで評価し、正の推定値の中央値を返します。40ms未満では入力全体を評価します。
- `Detector`は差分配列と中央値用の配列を保持します。容量が足りる場合は再利用し、大きな入力に使った容量はそのDetector内に残ります。
- 同じDetectorの並行利用と、コピーしたDetectorの同時利用には対応していません。並行処理では別々のDetectorを使えます。パッケージ関数は呼出しごとに独立したDetectorを使います。
- 多重音源の音高分離、録音ファイルの読込、リサンプリングは実装していません。

## 時間伸縮

`github.com/yh2237/audiodsp/stretch`は次の関数を提供します。

```go
output := stretch.WSOLA(source, targetFrames, sampleRate)
mapped := stretch.Anchored(source, targetFrames, sampleRate, sourcePositions, targetPositions)
linear := stretch.Linear(source, targetFrames)
```

- 入力はfloat64の単一チャンネル波形です。フレーム数はサンプル数を表します。入力配列を変更せず、出力配列は呼出しごとに独立しています。
- `WSOLA`は最大40msの窓を使い、出力位置を窓の半分ずつ進めます。相関の探索範囲は最大5msです。窓係数と参照波形のエネルギーは再利用し、重複計算を避けています。
- `Anchored`は厳密に増加する入力位置・出力位置の配列から線形の時間写像を作ります。出力を区間に分割せずに処理し、端の区間では線形に延長します。読込開始位置は入力範囲内に制限します。
- 位置配列の長さが一致しない場合、2点未満の場合、位置が増加しない場合、位置が範囲外の場合は、通常のWSOLAへ戻します。
- 入力が空の場合と、出力フレーム数が0以下の場合はnilを返します。入力・出力が16フレーム未満の場合、サンプルレートが0以下の場合、窓が小さい場合は線形補間を使います。
- `Linear`は両端を含む線形補間です。帯域制限フィルターは使用していません。
- 入力波形には有限値を使い、出力用メモリを確保できるフレーム数を指定します。

生成した波形を使う例は`go run ./examples/stretch`で実行できます。

## スペクトルの振幅

`github.com/yh2237/audiodsp/spectrum`の`Log(values, sampleRate, bands, minimumHz, maximumHz)`は、対数間隔の周波数点で振幅を計算します。周波数点には両端を含め、Hann窓付きの直接DFTを使います。

- 各点の振幅は入力長で割り、`20*log10(max(amplitude, 1e-7))`で返します。無音の下限は-140dBです。
- FFTや周波数帯域のエネルギー平均ではありません。指定した点の振幅を計算します。
- Hann窓の適用は入力ごとに一度だけ行い、周波数点間で再利用します。
- 入力は2サンプル以上、サンプルレートは正、周波数点は2点以上が必要です。周波数範囲の両端には有限の正の値を指定し、最大値を最小値より大きくします。条件に合わない設定や、範囲比・位相計算が無限大になる設定ではnilを返します。
- 入力波形は有限値を前提とします。入力配列を変更せず、出力は独立した配列です。振幅の正規化と平均値の除去は行いません。
- 4096サンプルまでは窓付き波形をローカルの作業領域に置き、それを超える場合は一時配列を確保します。出力配列は呼出しごとに確保します。

使用例は`go run ./examples/spectrum`です。

## WAVの読み書き

`github.com/yh2237/audiodsp/wav`は、RIFF/WAVEの整数PCM（format 1）・16-bitを扱います。

```go
pcm, err := wav.Decode(reader)        // *wav.PCM{SampleRate, Channels, Data []int16}
err = wav.Encode(writer, pcm)          // 44byteのヘッダーとサンプルを書き込む
data, err := wav.Bytes(pcm)            // ファイル全体を1つの配列で返す
```

- `Data`にはチャンネルごとの16-bitサンプルが交互に並びます。`len(Data)`はチャンネル数の倍数です。
- 8/24/32-bit、浮動小数点、WAVE_FORMAT_EXTENSIBLE、RF64は扱いません。対応しない形式はエラーです。
- `fmt `より前にある`data`と、未知のチャンクは読み飛ばします。`fmt `の後に`data`が複数ある場合は最後のものを返します。
- `data`の長さが奇数の場合、16-bitサンプルにならない末尾の1byteを読み飛ばします。長さが奇数のチャンクには埋め草の1byteが必要ですが、最後の`data`チャンクがEOFで終わる場合に限り、省略を許容します。宣言されたデータの途中で終わる入力や、チャンネル数で割り切れないサンプル数はエラーです。
- サンプルは読込バッファから直接復号し、data全体の一時バイト列は作りません。宣言されたdata長が16Mサンプルを超える場合、最初に確保する容量を16Mサンプルまでに抑え、読込に合わせて増やします。
- 書き出しには正のサンプルレートとチャンネル数が必要です。データはRIFFの32-bitサイズに収まる範囲で指定します。`Encode`は固定長のバッファで変換して書き込み、出力全体を先にメモリへ作りません。空のサンプル列も書き出せます。
- 入力配列を変更しません。`Decode`の結果と`Bytes`の戻り値は、呼出しごとに独立した配列です。
- ファイルの作成・置換、リサンプリング、チャンネル変換は実装していません。

使用例は`go run ./examples/wav`です。

## 波形の正規化相関

`correlation`パッケージは`v0.5.0`から利用できます。

```go
signed := correlation.Normalized(left, right, minimumEnergy)
positive := correlation.Max(left, right, maxLag, minOverlap, minimumEnergy)
```

- 平均を除去した波形の積和を、両波形のエネルギーの積の平方根で割ります。ここでのエネルギーは二乗和で、RMSではありません。
- `Normalized`は同じ長さの波形を比較し、負の相関も返します。2サンプル未満と長さの不一致では0を返します。
- `Max`は`-maxLag`から`maxLag`まで位置をずらし、最大の正の相関係数を返します。重なりが`minOverlap`サンプル未満の位置は使いません。正の相関がない場合、`maxLag`が負の場合、`minOverlap`が2未満の場合、入力が`minOverlap`より短い場合は0を返します。
- 正のlagではrightの先頭を、負のlagではleftの先頭を読み飛ばします。各位置の重なり部分で平均とエネルギーを計算します。
- `minimumEnergy`には有限の0以上の値を指定します。不正な値、またはいずれかの波形のエネルギーがこの値以下の場合は0を返します。比較窓の長さと閾値は呼出し側で決めます。
- 入力は有限値で、二乗和とエネルギーの積をfloat64で表現できる範囲の波形を前提とします。丸め誤差を含む相関係数をそのまま返し、`Normalized`を-1〜1へ切り詰める処理は行いません。
- 入力配列を変更せず、参照や作業配列も保持しません。FFT、遅れ位置の返却、窓の選択、音源の評価は行いません。

生成した波形を比較する例は`go run ./examples/correlation`で実行できます。

## 検証

```sh
go test ./...
go vet ./...
go test ./pitch -run '^$' -bench . -benchmem
go test ./pitch -run '^$' -fuzz FuzzDetector -fuzztime 10s
go test ./stretch -run '^$' -fuzz FuzzStretch -fuzztime 10s
go test ./spectrum -run '^$' -fuzz FuzzLog -fuzztime 10s
go test ./wav -run '^$' -fuzz FuzzDecode -fuzztime 10s
go test ./correlation -run '^$' -fuzz FuzzCorrelation -fuzztime 10s
```

CIにはWindows・Linux・macOS、Go 1.21とstable、race・fuzz、Go wasmビルドの検査を設定しています。

## 出典とライセンス

UtauTTSの音声処理から分離したコードを、MIT Licenseで公開しています。元の著作権表示を保持しています。移動元と対応ファイルは[ORIGIN.md](ORIGIN.md)に記載しています。
