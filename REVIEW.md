# Code Review: JmodClassLoader & JDK Class Loading (Milestone 3)

## Summary

JmodClassLoaderによるjmodファイルからの直接クラス読み込み、多数のネイティブメソッドスタブ、30+のオペコード追加により、Fib.java（メモ化フィボナッチ with HashMap/Integer）が正しく動作するようになった。前回レビュー（Milestone 2）で指摘したCritical Issues 3件、Major Issues 4件は全て修正済み。全体的なコード品質は良好。

現在の主な懸念点:
1. JmodClassLoaderがキャッシュミスのたびにjmodファイル全体（~120MB）をメモリに読み込む深刻なパフォーマンス問題
2. `System.arraycopy` の境界チェック不足によるGoランタイムpanic
3. `Float.floatToRawIntBits` ネイティブメソッドの実装が不正（常に0を返す）

---

## 前回レビュー指摘事項の対応状況

| ID | 問題 | 状態 |
|----|------|------|
| C1 | aaload/aastore 安全でない型アサーション | **修正済** - comma-ok + null/境界チェック追加 (instructions.go:276-304) |
| C2 | anewarray 負のサイズでpanic | **修正済** - NegativeArraySizeException チェック追加 (instructions.go:749-751) |
| C3 | executeNativeMethod 安全でない型アサーション | **修正済** - comma-ok パターン使用 (vm.go:119-123) |
| M1 | UserClassLoader キャッシュ確認順序 | **修正済** - キャッシュ確認が最初 (classloader.go:127-129) |
| M2 | スーパークラスのメソッド解決が未実装 | **修正済** - resolveMethod で階層を辿る (vm.go:353-367) |
| M3 | anewarray 要素が null でなく int(0) 初期化 | **修正済** - NullValue() で初期化 (instructions.go:767-770) |
| M4 | 新規オペコードのユニットテスト不足 | **修正済** - iinc, anewarray, aaload/aastore, if_acmpne, instanceof テスト追加 |

全ての Critical/Major 指摘が適切に修正されている。

---

## Critical Issues

### C1: JmodClassLoader がキャッシュミスのたびにjmod全体を読み込む

- **ファイル**: `pkg/vm/classloader.go:33-81`
- **問題**: `LoadClass` でキャッシュミスするたびに、jmodファイル全体をメモリに読み込み、zipをパースし、エントリを線形検索している。java.base.jmod は通常 ~120MB。Fib.java の実行では Object, Integer, HashMap, System, Number 等の多数のJDKクラスをロードするため、合計で ~1GB 以上のメモリ割り当てが発生する。
  ```go
  // classloader.go:50-57 — キャッシュミスのたびに ~120MB を確保
  data := make([]byte, stat.Size())
  if _, err := io.ReadFull(f, data); err != nil { ... }
  zipData := data[4:]
  reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
  ```
- **影響**: 起動が極めて遅く、メモリ使用量が巨大。Fib.java で ~10 クラスをロードする場合、~1.2GB のアロケーション。
- **修正方法**: zip.Reader をキャッシュし、ファイルデータの読み込みを1回に限定する。
  ```go
  type JmodClassLoader struct {
      JmodPath  string
      Cache     map[string]*classfile.ClassFile
      zipReader *zip.Reader    // 追加: 一度だけ作成
      zipData   []byte         // 追加: ファイルデータ保持
  }

  func (cl *JmodClassLoader) ensureOpen() error {
      if cl.zipReader != nil { return nil }
      // ファイル読み込みとzip.Reader作成を1回だけ実行
      ...
  }
  ```

### C2: nativeArraycopy に境界チェックがない

- **ファイル**: `pkg/vm/vm.go:841-843`
- **問題**: `System.arraycopy` のコピーループに境界チェックがない。`srcPos + length > len(srcArr.Elements)` や `destPos + length > len(destArr.Elements)` の場合、Goランタイムの `index out of range` panicが発生する。
  ```go
  // vm.go:841-843 — 境界チェックなし
  for i := 0; i < length; i++ {
      destArr.Elements[destPos+i] = srcArr.Elements[srcPos+i]
  }
  ```
- **影響**: 不正なarraycopy呼び出しでVM全体がクラッシュする。HashMap のリサイズ等で発生しうる。
- **修正方法**:
  ```go
  if srcPos < 0 || destPos < 0 || length < 0 ||
      srcPos+length > len(srcArr.Elements) ||
      destPos+length > len(destArr.Elements) {
      return Value{}, fmt.Errorf("ArrayIndexOutOfBoundsException: arraycopy")
  }
  ```

### C3: Float.floatToRawIntBits が常に0を返す

- **ファイル**: `pkg/vm/vm.go:164`
- **問題**: `return IntValue(args[0].Int), nil` — float値は `Value.Float` フィールドに格納されるが、`Value.Int` フィールド（ゼロ値の0）を返している。任意の非ゼロfloat値に対して0を返す。
  ```go
  // vm.go:164 — args[0].Float を使うべき
  case "java/lang/Float.floatToRawIntBits:(F)I":
      return IntValue(args[0].Int), nil // BUG: Int は常に 0
  ```
- **影響**: Float クラスの静的初期化で floatToRawIntBits が呼ばれた場合、NaN 判定等が正しく動作しない。現在の Fib.java テストでは Float クラスが直接使われないため顕在化していない。
- **修正方法**:
  ```go
  case "java/lang/Float.floatToRawIntBits:(F)I":
      return IntValue(int32(math.Float32bits(args[0].Float))), nil
  ```

---

## Major Issues

### M1: ldc2_w が double を float32 に切り捨てる

- **ファイル**: `pkg/vm/instructions.go:221`
- **問題**: `FloatValue(float32(c.Value))` で double (float64) を float32 に変換しており、精度が失われる。VM に TypeDouble が存在しないため、設計上の制約。
  ```go
  case *classfile.ConstantDouble:
      frame.Push(FloatValue(float32(c.Value))) // 精度損失
  ```
- **影響**: double リテラルを使うプログラムで計算結果が不正確になる。現在の Fib.java では double は使用されていないため影響なし。
- **修正案**: TypeDouble を Value に追加し、double 専用のフィールド (Float64 float64) を持たせる。または Long フィールドに bits を格納して代用する。

### M2: ensureInitialized が LoadClass エラーを無視する

- **ファイル**: `pkg/vm/vm.go:283-285`
- **問題**: `LoadClass` がエラーを返した場合、`return nil` でエラーを握りつぶしているが、その前に `vm.initializedClasses[className] = true` を設定しているため、クラスが後で正常にロード可能になっても初期化が実行されない。
  ```go
  vm.initializedClasses[className] = true // L281: 先に設定

  cf, err := vm.ClassLoader.LoadClass(className)
  if err != nil {
      return nil // L285: エラーを無視、しかし初期化済みフラグは残る
  }
  ```
- **影響**: 現在のクラスローダーでは問題にならないが、クラスパスが動的に変わるシナリオで初期化漏れが起きうる。また、本来のエラー（jmod破損等）が隠蔽される。
- **修正案**: エラー時に `vm.initializedClasses[className] = false` に戻す。

### M3: デフォルトjmodパスがarm64固定

- **ファイル**: `cmd/gojvm/main.go:12`
- **問題**: `defaultJmodPath` が `java-17-openjdk-arm64` にハードコードされている。amd64 環境では動作しない。
  ```go
  const defaultJmodPath = "/usr/lib/jvm/java-17-openjdk-arm64/jmods/java.base.jmod"
  ```
- **修正方法**: `JAVA_HOME` 環境変数から自動検出するか、`/usr/lib/jvm/java-*-openjdk-*/jmods/java.base.jmod` をglobで検索する。環境変数 `JAVA_BASE_JMOD` のフォールバックは既に実装されている (main.go:25-27)。

### M4: invokeinterface の文字列 equals が場当たり的

- **ファイル**: `pkg/vm/vm.go:727-737`
- **問題**: `invokeinterface` で receiver が string の場合に `equals` メソッドだけ特別処理している。`hashCode`, `toString`, `compareTo` 等は処理できない。
  ```go
  if _, isStr := objectRef.Ref.(string); isStr && methodRef.MethodName == "equals" {
      // 特別処理
  }
  ```
- **影響**: 文字列を Map のキーとして使用した場合 (HashMap が hashCode を呼ぶ)、文字列の hashCode が処理できずエラーになる。
- **修正案**: 文字列を JObject でラップし、java/lang/String クラスのメソッドとして統一的に処理する。

---

## Minor Issues

### m1: instanceof がクラス継承を考慮しない

- **ファイル**: `pkg/vm/instructions.go:809`
- **問題**: `obj.ClassName == className` の完全一致のみ。`Integer instanceof Object` が false を返す。
- **現状**: 現在のテストでは同一クラスのチェックのみ使用。将来的に `resolveMethod` のように階層を辿る実装が必要。

### m2: checkcast が型チェックを行わない

- **ファイル**: `pkg/vm/instructions.go:797`
- **問題**: CPインデックスを読み捨てるだけの no-op。ClassCastException を投げない。
- **現状**: instanceof の後でのみ使用されるため安全。

### m3: getfield と getstatic でデフォルト値の扱いが不一致

- **ファイル**: `pkg/vm/vm.go:472-477` vs `vm.go:425-429`
- **問題**: `getstatic` は `defaultValueForDescriptor` を使って型に応じたデフォルト値を返すが、`getfield` は一律 `NullValue()` を返す。int フィールドの未設定時に TypeNull の Value が返る。
  ```go
  // getfield (vm.go:472-477) — 一律 NullValue()
  if !exists {
      frame.Push(NullValue())
  }

  // getstatic (vm.go:425-429) — 型に応じたデフォルト値
  val = defaultValueForDescriptor(fieldRef.Descriptor)
  ```
- **影響**: `.Int` フィールドは 0 なので算術演算は正しく動くが、`ifnull` でチェックすると int フィールドが null と誤判定される。

### m4: テスト用 jmod パスがハードコード

- **ファイル**: `pkg/vm/integration_test.go:10`
- **問題**: `testJmodPath` が arm64 パス固定。CI/CD 環境で失敗する可能性。
  ```go
  const testJmodPath = "/usr/lib/jvm/java-17-openjdk-arm64/jmods/java.base.jmod"
  ```
- **修正案**: `os.Getenv("JAVA_BASE_JMOD")` にフォールバックする。

### m5: CAS 系ネイティブメソッドが常に成功を返す

- **ファイル**: `pkg/vm/vm.go:237-243`
- **問題**: `Unsafe.compareAndSetInt/Long/Reference` が常に `true` を返す。マルチスレッド環境では問題になるが、シングルスレッドの現在の実装では正しい動作。
- **現状**: 許容可能。将来スレッドを実装する際に再実装が必要。

---

## テストレビュー

### カバレッジの評価

| ファイル | テスト | 評価 |
|---------|--------|------|
| `pkg/vm/classloader.go` | Bootstrap/User/Cache/NotFound テスト | **良好** |
| `pkg/vm/object.go` | JArray/JObject ユニットテスト | **良好** |
| `pkg/vm/instructions.go` | 20+ ユニットテスト (iconst, bipush, arithmetic, branch, stack, sipush, overflow, ifnull, areturn, iinc, anewarray, aaload/aastore, if_acmpne, instanceof) | **良好** |
| `pkg/vm/vm.go` | getfield/putfield, invokespecial, checkcast テスト + 統合テスト | **良好** |
| `pkg/vm/integration_test.go` | 6テスト (Hello, Add, Arithmetic, ControlFlow, PrintString, Fib) | **良好** |
| `pkg/vm/frame.go` | 間接的にのみテスト | **可** |
| `pkg/native/system.go` | 統合テスト経由のみ | **可** |

### 前回からの改善

前回レビューで指摘した「新規6オペコードのユニットテスト不足」は全て対応済み:
- `TestIinc`: 正の値、負の値、ゼロ、大きい値 (6ケース)
- `TestAnewarray`: サイズ5、サイズ0 (2ケース)
- `TestAaloadAastore`: store/load、異なるインデックス (2ケース)
- `TestIfAcmpne`: 同一参照、異なる参照、両方null (3ケース)
- `TestInstanceof`: 一致クラス、不一致、null (3ケース)

### 追加推奨テスト

**高優先度:**
1. `nativeArraycopy` のエラーケース（境界外、負のオフセット、null配列）
2. `resolveMethod` のスーパークラス探索テスト

**低優先度:**
3. `tableswitch` / `lookupswitch` のユニットテスト
4. `ldc` / `ldc_w` のユニットテスト（Integer, Float, String, Class 定数）

---

## Good Points

- **前回レビュー指摘の全修正**: Critical 3件 + Major 4件が全て適切に対応されている。特に型アサーションの安全性（comma-ok パターン）、配列境界チェック、NullValue() 初期化は正確に修正されている。
- **resolveMethod の導入**: スーパークラスのメソッド解決が `SuperClassName()` を辿るループとして実装され、`invokevirtual`、`invokespecial`、`invokestatic` で統一的に使用されている (vm.go:353-367)。
- **クラス初期化 (`<clinit>`) の実装**: `ensureInitialized` がスーパークラスを先に初期化し、`<clinit>` を実行する正しい順序で実装されている (vm.go:277-305)。再帰防止フラグも適切。
- **静的フィールドの実装**: `staticFields` マップと `getstatic`/`putstatic` により、クラス変数が正しく機能する。型に応じたデフォルト値 (`defaultValueForDescriptor`) も適切。
- **JmodClassLoader の設計**: jmod ファイルの 4バイトヘッダースキップ、zip として解凍、`classes/` プレフィックス付きパスの正しい構築。キャッシュも実装済み（パフォーマンスは要改善だが機能は正しい）。
- **テストの大幅充実**: classloader_test.go（4テスト）、object_test.go、instructions_test.go の新規テストにより、ユニットテストカバレッジが大幅に向上。
- **ネイティブメソッドディスパッチ**: `className.methodName:descriptor` キーによる switch 文は明快で拡張しやすい。`registerNatives`/`initIDs` パターンマッチも適切。
- **例外ハンドラのパース**: `parseCodeAttribute` が exception_table を正しくパースしている (parser.go:243-261)。athrow も基本的な実装がある。
- **countParams の堅牢性**: 配列型 (`[L...;`, `[I` 等)、プリミティブ型、オブジェクト型を全て正しくパースする (vm.go:779-821)。

---

## アーキテクチャ上の所見

### 設計上の強み
1. **ClassLoader インターフェース**: テストでモック可能。JmodClassLoader/BootstrapClassLoader/UserClassLoader の3層が Parent Delegation Model を正しく表現。
2. **Value struct**: タグ付きユニオンとして Int/Float/Long/Ref を保持する設計はシンプルで効率的。JVM の 2-slot ルール（long/double）を1スロットに簡略化しているが、現在のスコープでは問題ない。
3. **Frame ベースの実行モデル**: 各メソッド呼び出しが独立した Frame を持ち、PC/SP/ローカル変数/オペランドスタックを管理する設計が clean。

### 今後の課題（現スコープ外）
1. **TypeDouble の追加**: double 型を正しくサポートするために Value に Double フィールドを追加
2. **例外処理の完全実装**: athrow + exception_table による catch ブロックへのジャンプ
3. **文字列オブジェクトの統一**: 現在 string は Go の string として Ref に格納されているが、JObject でラップして java/lang/String のメソッドを統一的に処理すべき
4. **スレッドサポート**: synchronized, volatile, CAS の正しい実装
