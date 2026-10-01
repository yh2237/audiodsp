# audiodsp

float64モノラル波形のピッチ推定・時間伸縮・対数間隔の振幅計算と、16-bit PCM WAVの読み書きを提供するGoライブラリです。`pitch`、`stretch`、`spectrum`、`wav`を実装しています。標準ライブラリだけを使用し、Go 1.21以上に対応します。

```sh
go get github.com/yh2237/audiodsp@v0.4.0
```

## 使用例

```go
import "github.com/yh2237/audiodsp/pitch"

hz := pitch.Estimate(frame, sampleRate)
medianHz := pitch.EstimateMedian(wave, sampleRate)
```

連続するフレームでは、ゼロ値の`Detector`を再利用できます。

```go
var detector pitch.Detector
for _, frame := range frames {
    hz := detector.Estimate(frame, sampleRate)
    // hzを使用する
}
```

実行可能な生成正弦波の例は`go run ./examples/pitch`です。

## ピッチ推定の入出力と処理

- 入力は正規化されたモノラル波形（目安は-1〜1）とサンプルレートHzです。配列を変更せず、参照も保持しません。
- 戻り値は推定周波数Hzです。推定不能、無音、DC信号、短すぎる入力、非正のサンプルレートでは0を返します。
- RMSと平均除去後のRMSが0.003未満の入力は推定しません。
- lag探索範囲は`max(2, sampleRate/500)`から`min(len(frame)/2, sampleRate/60)`です。累積平均で正規化した差分と局所補間を用います。
- `EstimateMedian`は40ms窓を10ms刻みで評価し、正の推定値の中央値を返します。40ms未満では入力全体を評価します。
- `Detector`は差分配列と中央値用配列を保持し、容量が足りる呼び出しでは再利用します。大きい入力に使った容量はそのDetector内に残ります。
- 同じDetectorの並行利用やコピーしたDetectorの同時利用には対応しません。並行処理には個別のDetectorを使えます。パッケージ関数は呼び出しごとに独立したDetectorを使います。
- 多重音源の音高分離、録音ファイルの読込、リサンプリングは実装していません。

## 時間伸縮

`github.com/yh2237/audiodsp/stretch`は次の関数を提供します。

```go
output := stretch.WSOLA(source, targetFrames, sampleRate)
mapped := stretch.Anchored(source, targetFrames, sampleRate, sourcePositions, targetPositions)
linear := stretch.Linear(source, targetFrames)
```

- 入力はfloat64の単一チャンネル波形、フレーム数はサンプル数です。入力を変更せず、出力配列は呼び出しごとに独立しています。
- `WSOLA`は最大40ms窓、窓の半分の出力hop、最大5msの相関探索を使用します。窓係数と参照波形のエネルギーは重複計算せずに再利用します。
- `Anchored`は厳密に増加する入力/出力位置の配列から線形時間写像を作り、出力を区間分割せず処理します。端の区間では線形延長し、読込開始位置を入力範囲内へ制限します。
- 位置配列の長さ不一致、2点未満、非増加、位置の範囲外は通常のWSOLAへ戻します。
- 空の入力や非正の出力フレーム数はnilです。16フレーム未満、非正のサンプルレート、窓が小さい条件では線形補間を使います。
- `Linear`は両端を含む線形補間です。帯域制限フィルターは使用していません。
- 有限値の入力を前提とし、出力用メモリを確保できるフレーム数を指定します。

生成波形の実行例は`go run ./examples/stretch`です。

## スペクトルの振幅

`github.com/yh2237/audiodsp/spectrum`の`Log(values, sampleRate, bands, minimumHz, maximumHz)`は、両端を含む対数間隔の周波数点でHann窓付きの直接DFT振幅を求めます。

- 各点の振幅は入力長で割り、`20*log10(max(amplitude, 1e-7))`で返します。無音の下限は-140dBです。
- FFTや周波数帯域のエネルギー平均ではありません。指定した点の振幅を計算します。
- Hann窓の適用は入力ごとに一度だけ行い、周波数点間で再利用します。
- 入力2サンプル以上、正のレート、2点以上、有限で正かつ増加する周波数範囲を要求します。範囲比や位相計算が無限大になる設定も含め、条件に合わない設定はnilです。
- 入力波形は有限値を前提とします。入力を変更せず、出力は独立した配列です。振幅の正規化や平均値除去は行いません。
- 4096サンプルまでの窓付き波形はローカル作業領域を使い、それを超える入力では一時配列を確保します。出力配列は呼び出しごとに確保します。

使用例は`go run ./examples/spectrum`です。

## WAVの読み書き

`github.com/yh2237/audiodsp/wav`は、RIFF/WAVEの整数PCM（format 1）・16-bitを扱います。

```go
pcm, err := wav.Decode(reader)        // *wav.PCM{SampleRate, Channels, Data []int16}
err = wav.Encode(writer, pcm)          // 44byteのヘッダーとサンプルを書き込む
data, err := wav.Bytes(pcm)            // ファイル全体を1つの配列で返す
```

- `Data`はチャンネルが交互に並ぶ16-bitサンプルです。`len(Data)`はチャンネル数の倍数です。
- 8/24/32-bit、浮動小数点、WAVE_FORMAT_EXTENSIBLE、RF64は扱いません。対応しない形式はエラーです。
- `fmt `より前の`data`、未知のチャンクは読み飛ばします。`fmt `の後に`data`が複数ある場合は最後のものを返します。奇数長チャンクの埋め草1byteを要求し、途中で終わる入力やチャンネル数に揃わないサンプル数はエラーです。
- サンプルは読込バッファから直接復号し、data全体の一時バイト列を作りません。data長の宣言が16Mサンプルを超える場合、最初に確保する容量は16Mサンプルまでとし、読込に合わせて増やします。
- 書き出しは正のサンプルレートとチャンネル数、RIFFの32-bitサイズに収まるデータを要求します。`Encode`は固定長のバッファで変換して書き込み、出力全体を先にメモリへ作りません。空のサンプル列も書き出せます。
- 入力の配列を変更しません。`Decode`の結果と`Bytes`の戻り値は呼び出しごとに独立した配列です。
- ファイルの作成・置換、リサンプリング、チャンネル変換は実装していません。

使用例は`go run ./examples/wav`です。

## 検証

```sh
go test ./...
go vet ./...
go test ./pitch -run '^$' -bench . -benchmem
go test ./pitch -run '^$' -fuzz FuzzDetector -fuzztime 10s
go test ./stretch -run '^$' -fuzz FuzzStretch -fuzztime 10s
go test ./spectrum -run '^$' -fuzz FuzzLog -fuzztime 10s
go test ./wav -run '^$' -fuzz FuzzDecode -fuzztime 10s
```

CIにWindows/Linux/macOS、Go 1.21/stable、race・fuzz、Go wasmビルドの検査を設定しています。

## 出典とライセンス

UtauTTSの音声処理から分離したMIT Licenseのコードです。元の著作権表示を保持しています。対応箇所は[ORIGIN.md](ORIGIN.md)に記載しています。
