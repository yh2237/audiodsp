# 出典

ピッチ推定は[UtauTTS](https://github.com/yh2237/UtauTTS)の`d1bba5b`時点の`internal/pitch/yin.go`と`yin_test.go`から分離しました。元のMIT Licenseと`Copyright (c) 2026 yh`を保持しています。

分離にあたり、差分配列・中央値用の配列を保持するDetectorを追加し、範囲を確定した波形のスライスで差分を計算するようにしました。平均除去・差分の計算順序、閾値、lag探索、補間、中央値の定義は保っています。

音源、言語処理、合成エンジン、アプリケーションのキャッシュは含みません。

`stretch/wsola.go`のWSOLA・位置写像・線形補間は、UtauTTS `0c6fab4`時点の`internal/render/base/stretch.go`から分離しました。窓係数と参照エネルギーの重複計算を除き、入力範囲外の位置を検査・制限するようにしました。先頭区間の保護、境界での音量混合、デクリック処理はこのパッケージに含みません。

`spectrum/log.go`の対数周波数点での振幅計算は、UtauTTS `b010c7b`時点の`internal/acoustic/features.go`から分離しました。周波数点ごとにHann窓を計算する処理を除き、元と同じ算術順序で計算した窓付きの値を再利用しています。フレームの有声判定、音量・ピッチの指標、接合スコアは含みません。

`wav/wav.go`の16-bit PCM WAVの読み書きは、UtauTTS `481487f`時点の`internal/audio/wav.go`から分離しました。チャンクの扱い・検査・サンプル値を保ち、書き出し時の全サンプル分の一時配列と、宣言されたdata長に基づく一括確保を除きました。ファイルの原子的な置換と原音の切り出しは含みません。

`correlation/correlation.go`は、UtauTTS `6f284c8`時点の`internal/connection/features.go`のmaxCorrelationと、`internal/render/bridge.go`のnormalizedCorrelationから分離しました。平均を除去する正規化相関の計算順序を保ち、最小エネルギーと最小重なり長を呼出し側から指定する形にしました。重なり長が足りないlagは探索範囲から除きます。比較窓の選択、原音評価のスコア、境界補修の採否は含みません。
