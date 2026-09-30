# 出典

元のピッチ推定は[UtauTTS](https://github.com/yh2237/UtauTTS)の`d1bba5b`時点の`internal/pitch/yin.go`と`yin_test.go`です。元のMIT Licenseと`Copyright (c) 2026 yh`を保持しています。

この実装には、差分配列・中央値配列を保持するDetectorと、範囲を確定した波形sliceでの差分計算を追加しています。元の平均除去・差分の計算順序、閾値、lag探索、補間、中央値の定義を保持しています。

音源、言語処理、合成エンジン、アプリケーションのキャッシュは含みません。
