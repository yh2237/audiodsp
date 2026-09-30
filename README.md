# audiodsp

float64モノラル波形のピッチ推定と時間伸縮を提供するGoライブラリです。`pitch`と`stretch`を実装しています。標準ライブラリだけを使用し、Go 1.21以上に対応します。

```sh
go get github.com/yh2237/audiodsp@v0.2.0
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

## 検証

```sh
go test ./...
go vet ./...
go test ./pitch -run '^$' -bench . -benchmem
go test ./pitch -run '^$' -fuzz FuzzDetector -fuzztime 10s
go test ./stretch -run '^$' -fuzz FuzzStretch -fuzztime 10s
```

CIにWindows/Linux/macOS、Go 1.21/stable、race・fuzz、Go wasmビルドの検査を設定しています。

## 出典とライセンス

UtauTTSの音声処理から分離したMIT Licenseのコードです。元の著作権表示を保持しています。対応箇所は[ORIGIN.md](ORIGIN.md)に記載しています。
