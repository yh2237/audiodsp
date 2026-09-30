# 出典

元のピッチ推定は[UtauTTS](https://github.com/yh2237/UtauTTS)の`d1bba5b`時点の`internal/pitch/yin.go`と`yin_test.go`です。元のMIT Licenseと`Copyright (c) 2026 yh`を保持しています。

この実装には、差分配列・中央値配列を保持するDetectorと、範囲を確定した波形sliceでの差分計算を追加しています。元の平均除去・差分の計算順序、閾値、lag探索、補間、中央値の定義を保持しています。

音源、言語処理、合成エンジン、アプリケーションのキャッシュは含みません。

`stretch/wsola.go`のWSOLA・位置写像・線形補間はUtauTTS `0c6fab4`時点の`internal/render/base/stretch.go`から分離しています。窓係数と参照エネルギーの重複計算を除き、入力範囲外の位置を検査・制限しています。先頭区間保護・境界の音量混合・デクリック処理はこのパッケージに含めていません。
