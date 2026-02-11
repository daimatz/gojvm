# Code Review: 例外処理・instanceof階層・インターフェース解決 (Milestone 4)

## Summary

instanceof の継承階層チェック、try-catch 例外処理、invokeinterface によるインターフェースメソッド解決を実装。Switch, Sort, Inheritance, TryCatch, Interface を含む全11テストがパス。

前回レビュー（Milestone 3）の Critical 3件は全て修正済み。新たに Critical 2件、Major 3件を検出。特に **ensureInitialized が JavaException を fmt.Errorf でラップして型情報を失う問題** と **NullPointerException の生成方法が不統一で一部が try-catch で捕捉不能な問題** は修正すべき。

---

## 前回レビュー指摘事項の対応状況

| ID | 問題 | 状態 |
|----|------|------|
| C1 | JmodClassLoader がキャッシュミスのたびにjmod全体を読み込む | **修正済** — `ensureZipReader()` で zip.Reader/zipData を1回だけ作成・保持 (classloader.go:35-62) |
| C2 | nativeArraycopy に境界チェックがない | **修正済** — 負値・範囲外チェック追加 (vm.go:929-933) |
| C3 | Float.floatToRawIntBits が常に0を返す | **修正済** — `math.Float32bits(args[0].Float)` を使用 (vm.go:179) |
| M1 | ldc2_w が double を float32 に切り捨てる | **未修正** — TypeDouble がないため設計上の制約 (instructions.go:221) |
| M2 | ensureInitialized が LoadClass エラーを無視する | **修正済** — エラー時に `initializedClasses[className] = false` に戻す (vm.go:300) |
| M3 | デフォルトjmodパスがarm64固定 | **未修正** — 機能的な問題ではない |
| M4 | invokeinterface の文字列 equals が場当たり的 | **未修正** — 既存のまま (vm.go:815-826) |
| m1 | instanceof がクラス継承を考慮しない | **修正済** — `isInstanceOf` で継承階層とインターフェースを再帰的にチェック (vm.go:370-397) |
| m2 | checkcast が型チェックを行わない | **修正済** — `isInstanceOf` を使用して ClassCastException を投げる (instructions.go:803-809) |
| m3 | getfield のデフォルト値が一律 NullValue() | **修正済** — `defaultValueForDescriptor` を使用 (vm.go:560-565) |

---

## Critical Issues

### C1: ensureInitialized が JavaException を fmt.Errorf でラップし型情報を失う

- **ファイル**: `pkg/vm/vm.go:317`
- **問題**: `<clinit>` 実行中に Java 例外が投げられた場合、`fmt.Errorf("error in <clinit> of %s: %w", className, err)` でラップされる。呼び出し元の `executeMethod` (vm.go:104) は `err.(*JavaException)` で直接型アサーションを行うが、`fmt.Errorf` でラップされた `*JavaException` は直接型アサーションでは検出できない（`errors.As()` が必要）。
  ```go
  // vm.go:315-318 — JavaException が fmt.Errorf でラップされる
  _, err := vm.executeMethod(cf, clinit, nil)
  if err != nil {
      return fmt.Errorf("error in <clinit> of %s: %w", className, err)
      // ↑ *JavaException が *fmt.wrapError になり、型情報が失われる
  }

  // vm.go:103-106 — 呼び出し元で直接型アサーション
  javaExc, isJavaExc := err.(*JavaException)
  if !isJavaExc {
      // ↑ ラップされた JavaException はここで false になる！
      return Value{}, fmt.Errorf("in %s.%s:%s at PC=%d: %w", ...)
      // JavaException がさらに別のエラーとしてラップされ、二度と捕捉できない
  }
  ```
- **影響**: `<clinit>` 内で投げられた Java 例外が try-catch で捕捉できなくなる。例えば、static initializer 内での `throw new RuntimeException()` が Java 例外として伝播せず、Go エラーとして扱われて VM がクラッシュする。
- **修正方法**: `ensureInitialized` で `*JavaException` をそのまま伝播するか、`executeMethod` で `errors.As()` を使用する。
  ```go
  // 案1: ensureInitialized で JavaException をそのまま返す
  _, err := vm.executeMethod(cf, clinit, nil)
  if err != nil {
      if _, ok := err.(*JavaException); ok {
          return err  // JavaException はそのまま伝播
      }
      return fmt.Errorf("error in <clinit> of %s: %w", className, err)
  }
  ```

### C2: NullPointerException の生成方法が不統一（一部が try-catch で捕捉不能）

- **ファイル**: `pkg/vm/vm.go:553-554`, `vm.go:581-582`, `vm.go:622-623`, `vm.go:809`
- **問題**: 配列操作（instructions.go:280-281, 295-296 等）では `NewJavaException("java/lang/NullPointerException")` を使って `*JavaException` を返しているが、getfield, putfield, invokevirtual では `fmt.Errorf("getfield: NullPointerException")` で単純な文字列エラーを返している。
  ```go
  // instructions.go:280-281 — 正しい（*JavaException を返す）
  return Value{}, false, NewJavaException("java/lang/NullPointerException")

  // vm.go:553-554 — 問題（fmt.Errorf で返す）
  return Value{}, false, fmt.Errorf("getfield: NullPointerException")
  // ↑ これは *JavaException ではないため try-catch で捕捉できない

  // 同様に:
  // vm.go:581-582 (putfield)
  // vm.go:622-623 (invokevirtual)
  // vm.go:809     (invokeinterface)
  ```
- **影響**: `try { obj.field; } catch (NullPointerException e) { ... }` が期待通りに動作しない。getfield/putfield/invokevirtual 由来の NullPointerException は catch ブロックに到達せず、VM 全体がエラーで停止する。
- **修正方法**: 全ての NullPointerException 生成箇所で `NewJavaException` を使用する。
  ```go
  // vm.go:553-554
  return Value{}, false, NewJavaException("java/lang/NullPointerException")
  ```

---

## Major Issues

### M1: isInstanceOf にサイクル検出がない（malformed classfile で無限再帰）

- **ファイル**: `pkg/vm/vm.go:370-397`
- **問題**: `isInstanceOf` はインターフェースを再帰的にチェックするが、visited セットを持たない。正規の Java プログラムではインターフェースの循環参照は不可能だが、malformed な classfile では可能。
  ```go
  // vm.go:384-388 — 再帰呼び出しにサイクル検出なし
  for _, ifIdx := range cf.Interfaces {
      ifName, err := classfile.GetClassName(cf.ConstantPool, ifIdx)
      if err == nil && (ifName == targetClassName || vm.isInstanceOf(ifName, targetClassName)) {
          //                                         ↑ 循環参照で無限再帰
          return true
      }
  }
  ```
- **影響**: 正常な Java プログラムでは発生しない。ただし、意図的に細工された classfile や JDK 内部クラスの循環的な依存関係（実際には存在しないが）でスタックオーバーフローが発生する可能性がある。`maxFrameDepth` は `executeMethod` にのみ適用され、`isInstanceOf` の再帰には適用されない。
- **修正方法**: visited セットを導入する。
  ```go
  func (vm *VM) isInstanceOf(objectClassName, targetClassName string) bool {
      return vm.isInstanceOfWithVisited(objectClassName, targetClassName, make(map[string]bool))
  }
  ```

### M2: resolveMethod のインターフェース探索にも同様のサイクルリスク

- **ファイル**: `pkg/vm/vm.go:435-453`
- **問題**: `resolveMethod` がインターフェースのメソッドを探索する際に自身を再帰呼び出しするが、visited セットがない。インターフェース A が B を extends し、B が A を extends するような malformed classfile で無限再帰になる。
  ```go
  // vm.go:447-449 — 再帰呼び出しにサイクル検出なし
  ifCf, ifMethod, err := vm.resolveMethod(ifName, methodName, descriptor)
  ```
- **影響**: M1 と同様、正常な Java プログラムでは発生しない。

### M3: ldc2_w が double を float32 に切り捨てる（前回から継続）

- **ファイル**: `pkg/vm/instructions.go:221`
- **問題**: `FloatValue(float32(c.Value))` で double (float64) を float32 に変換しており、精度が失われる。VM に TypeDouble が存在しないため、設計上の制約。
  ```go
  case *classfile.ConstantDouble:
      frame.Push(FloatValue(float32(c.Value))) // 精度損失
  ```
- **影響**: double リテラルを使うプログラムで計算結果が不正確になる。

---

## Minor Issues

### m1: invokeinterface の文字列 equals が場当たり的（前回から継続）

- **ファイル**: `pkg/vm/vm.go:815-826`
- **問題**: `invokeinterface` で receiver が string の場合に `equals` メソッドだけ特別処理している。`hashCode`, `toString`, `compareTo` 等は処理できない。
- **影響**: 文字列を Map のキーとして使用した場合に問題になる可能性。

### m2: テスト用 jmod パスがarm64固定（前回から継続）

- **ファイル**: `pkg/vm/integration_test.go:10`
- **問題**: `testJmodPath` が `java-17-openjdk-arm64` にハードコードされている。
  ```go
  const testJmodPath = "/usr/lib/jvm/java-17-openjdk-arm64/jmods/java.base.jmod"
  ```

### m3: NegativeArraySizeException が JavaException でなく fmt.Errorf

- **ファイル**: `pkg/vm/instructions.go:751`, `instructions.go:765`
- **問題**: C2 と同様のパターン。`NegativeArraySizeException` が `fmt.Errorf` で生成されているため、try-catch で捕捉できない。
  ```go
  return Value{}, false, fmt.Errorf("NegativeArraySizeException: %d", count)
  ```

### m4: arraycopy の各種例外も fmt.Errorf

- **ファイル**: `pkg/vm/vm.go:920-933`
- **問題**: `NullPointerException`, `ArrayStoreException`, `ArrayIndexOutOfBoundsException` が全て `fmt.Errorf` で生成されている。

### m5: arraylength の NullPointerException も fmt.Errorf

- **ファイル**: `pkg/vm/instructions.go:777`
- **問題**: C2 と同じパターン。

---

## テストレビュー

### カバレッジの評価

| ファイル | テスト | 評価 |
|---------|--------|------|
| `pkg/vm/instructions.go` | 20+ ユニットテスト (iconst, bipush, arithmetic, branch, stack, sipush, overflow, ifnull, areturn, iinc, anewarray, aaload/aastore, if_acmpne, instanceof, division by zero) | **良好** |
| `pkg/vm/vm.go` | getfield/putfield, invokespecial, checkcast テスト + 統合テスト | **良好** |
| `pkg/vm/integration_test.go` | 11テスト (Hello, Add, Arithmetic, ControlFlow, PrintString, Fib, Switch, Sort, Inheritance, TryCatch, Interface) | **良好** |
| `pkg/vm/frame.go` | 間接的にのみテスト | **可** |
| `pkg/vm/exception.go` | DivisionByZero テストで JavaException の型チェック (instructions_test.go:311-314) | **可** |

### 新規テストの評価

- **TestSwitch**: tableswitch/lookupswitch の統合テスト。前回レビューで「追加推奨」とした項目がカバーされている。
- **TestSort**: 配列操作（newarray, iaload, iastore）+ ソートアルゴリズムの統合テスト。
- **TestInheritance**: instanceof の継承階層チェックを検証。前回 m1 で指摘した点が統合テストでカバーされている。
- **TestTryCatch**: 例外の throw/catch、catch ブロック内の処理、finally の動作を検証。期待値 "5\n-1\n0\n" は try/catch/finally パターンが正しく動作していることを示す。
- **TestInterface**: invokeinterface によるインターフェースメソッド呼び出しとデフォルトメソッドの解決を検証。

### 追加推奨テスト

**高優先度:**
1. `<clinit>` 内で Java 例外が投げられた場合のテスト（C1 の検証）
2. getfield/putfield に対する null オブジェクトの try-catch テスト（C2 の検証）

**低優先度:**
3. 多階層継承（A extends B extends C implements D, D extends E）での instanceof テスト
4. athrow で null をスローした場合の NullPointerException テスト

---

## Good Points

- **例外処理の基本設計が正しい**: `JavaException` 型を導入し、`executeMethod` 内の実行ループで例外テーブルを検索、ハンドラへのジャンプ、スタックリセット、例外オブジェクトのプッシュが JVM 仕様通りに実装されている (vm.go:103-117)。
- **findExceptionHandler の正確な範囲チェック**: `pc < int(h.StartPC) || pc >= int(h.EndPC)` で半開区間 [startPC, endPC) を正しく処理。CatchType=0 の catch-all（finally）もサポート (vm.go:400-418)。
- **isInstanceOf の階層探索**: スーパークラスチェーン + 各クラスのインターフェースを再帰的にチェックする正しい実装。直接の equal チェック → インターフェースチェック → スーパークラスへ移動の順序が適切 (vm.go:370-397)。
- **resolveMethod のインターフェースデフォルトメソッド解決**: 通常のスーパークラスチェーンでメソッドが見つからない場合、インターフェースのデフォルトメソッドを探索する2パス設計が正しい (vm.go:421-455)。
- **athrow の正しい実装**: null の throw で NullPointerException を生成、JObject の throw で JavaException を伝播、それ以外でエラーを返す3パターンの処理 (instructions.go:785-793)。
- **checkcast の改善**: 前回レビューの m2 指摘を受けて `isInstanceOf` を使った型チェックと ClassCastException の生成を追加 (instructions.go:795-809)。
- **前回 Critical 3件の完全修正**: JmodClassLoader のキャッシュ化、arraycopy の境界チェック、floatToRawIntBits の修正が全て適切に行われている。
- **統合テストの充実**: 5テスト追加（Switch, Sort, Inheritance, TryCatch, Interface）により、新機能の動作が実際の Java プログラムレベルで検証されている。

---

## アーキテクチャ上の所見

### 設計上の強み
1. **JavaException 型の分離**: `exception.go` に独立ファイルとして配置し、`Error()` インターフェースを実装。Go のエラーハンドリングと JVM の例外処理を自然に統合している。
2. **例外テーブルとの統合**: `classfile.ExceptionHandler` 構造体を `findExceptionHandler` で活用し、バイトコードレベルの例外処理を正しく実装。
3. **invokeinterface の分離**: 4バイトオペランド (index, count, reserved) の読み取りと `ResolveInterfaceMethodref` の使用が JVM 仕様準拠。

### 今後の課題（現スコープ外）
1. **TypeDouble の追加**: double 型を正しくサポートするために Value に Double フィールドを追加
2. **文字列オブジェクトの統一**: Go の string を JObject でラップして java/lang/String のメソッドを統一的に処理
3. **全例外を JavaException に統一**: 現在 fmt.Errorf で生成されている各種例外を JavaException に移行
4. **multianewarray**: 多次元配列の生成
