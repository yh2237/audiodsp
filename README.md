# audiodsp

float64波形のYIN系ピッチ推定を提供するGoライブラリです。実装パッケージは`pitch`です。標準ライブラリだけを使用し、Go 1.21以上に対応します。

```sh
go get github.com/yh2237/audiodsp@v0.1.0
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

## 入出力と処理

- 入力は正規化されたモノラル波形（目安は-1〜1）とサンプルレートHzです。配列を変更せず、参照も保持しません。
- 戻り値は推定周波数Hzです。推定不能、無音、DC信号、短すぎる入力、非正のサンプルレートでは0を返します。
- RMSと平均除去後のRMSが0.003未満の入力は推定しません。
- lag探索範囲は`max(2, sampleRate/500)`から`min(len(frame)/2, sampleRate/60)`です。累積平均で正規化した差分と局所補間を用います。
- `EstimateMedian`は40ms窓を10ms刻みで評価し、正の推定値の中央値を返します。40ms未満では入力全体を評価します。
- `Detector`は差分配列と中央値用配列を保持し、容量が足りる呼び出しでは再利用します。大きい入力に使った容量はそのDetector内に残ります。
- 同じDetectorの並行利用やコピーしたDetectorの同時利用には対応しません。並行処理には個別のDetectorを使えます。パッケージ関数は呼び出しごとに独立したDetectorを使います。
- 多重音源の音高分離、録音ファイルの読込、リサンプリングは実装していません。

## 検証

```sh
go test ./...
go vet ./...
go test ./pitch -run '^$' -bench . -benchmem
go test ./pitch -run '^$' -fuzz FuzzDetector -fuzztime 10s
```

CIにWindows/Linux/macOS、Go 1.21/stable、race・fuzz、Go wasmビルドの検査を設定しています。

## 出典とライセンス

UtauTTSのピッチ推定から分離したMIT Licenseのコードです。元の著作権表示を保持しています。対応箇所は[ORIGIN.md](ORIGIN.md)に記載しています。
