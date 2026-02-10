# Code Review - Milestone 1.5 (Fib.java対応)

## Summary

Milestone 1.5の実装は全体的に良質。JVM仕様に対する準拠性が高く、`Fib.java`（メモ化フィボナッチ）を正しく実行できる。JObject、NativeHashMap、NativeIntegerの設計はシンプルで目的に合っている。主な懸念点は `invokevirtual` 内のネイティブメソッド呼び出しにおける安全でない型アサーション（panicの可能性）と、`invokespecial` でメソッドが見つからない場合のサイレント失敗。テストは統合テスト（TestFib）が正しく89を検証しているが、新規オペコードの個別ユニットテストが不足している。

---

## Critical Issues（修正必須）

なし。現在のFib.javaユースケースで不正な動作を引き起こす問題は確認されなかった。

---

## Major Issues（強く推奨）

### [M1] invokevirtual 内のネイティブメソッド呼び出しで安全でない型アサーション

- **ファイル**: `pkg/vm/vm.go:215, 227, 239`
- **問題**: HashMap.get、HashMap.put、Integer.intValue のネイティブ呼び出しで、comma-okパターンを使わない型アサーションを使用している。レシーバーがnullまたは想定外の型の場合、panicが発生する。
  ```go
  // vm.go:215 — パニックの可能性
  hm := objectRef.Ref.(*native.NativeHashMap)

  // vm.go:227 — 同上
  hm := objectRef.Ref.(*native.NativeHashMap)

  // vm.go:239 — 同上
  ni := objectRef.Ref.(*native.NativeInteger)
  ```
- **対比**: `getfield` / `putfield` では正しくcomma-okパターンが使われている（`vm.go:138, 165`）。
- **修正案**: comma-okパターンに統一する。
  ```go
  hm, ok := objectRef.Ref.(*native.NativeHashMap)
  if !ok {
      return Value{}, false, fmt.Errorf("invokevirtual: HashMap.get receiver is not a NativeHashMap")
  }
  ```

### [M2] invokespecial でユーザークラスメソッドが見つからない場合のサイレント失敗

- **ファイル**: `pkg/vm/vm.go:289-301`
- **問題**: `invokespecial` のdefaultブランチで `frame.Class.FindMethod` がnilを返した場合、何もせずに正常リターンする。不正なクラスファイルや実装バグを隠蔽する可能性がある。
  ```go
  default:
      method := frame.Class.FindMethod(methodRef.MethodName, methodRef.Descriptor)
      if method != nil {
          // ... execute
      }
      // method == nil の場合、エラーなしで処理を続行してしまう
  ```
- **修正案**: method == nil の場合にエラーを返す。
  ```go
  if method == nil {
      return Value{}, false, fmt.Errorf("invokespecial: method %s:%s not found", methodRef.MethodName, methodRef.Descriptor)
  }
  ```

### [M3] getfield / putfield の新規オペコード用ユニットテストが存在しない

- **ファイル**: `pkg/vm/instructions_test.go`
- **問題**: Milestone 1.5で追加された6個のオペコード (`areturn`, `getfield`, `putfield`, `invokespecial`, `checkcast`, `ifnull`) のうち、ユニットレベルでテストされているのは `ifnull` と `areturn` の2つのみ。`getfield`, `putfield`, `invokespecial`, `checkcast` のユニットテストが存在しない。これらはTestFib統合テスト経由でのみ間接的にテストされている。
- **修正案**: 少なくとも `getfield` と `putfield` のユニットテストを追加する。特に以下のケース:
  - 正常なフィールド読み書き
  - 存在しないフィールドの読み取り（デフォルトnull）
  - nullオブジェクトへのアクセス（エラーケース）

---

## Minor Issues（改善提案）

### [m1] NativeHashMap テストに *NativeInteger キーのテストケースがない

- **ファイル**: `pkg/native/native_test.go:53-62`
- **問題**: `TestNativeHashMap` の "integer keys" テストは `int32` を直接キーとして使用しているが、実際のVM動作では `*NativeInteger` がキーとして渡される。`Get`/`Put` メソッドの `*NativeInteger` からの値抽出パスが個別にテストされていない。
  ```go
  // 現在のテスト: int32 を直接使用
  hm.Put(int32(0), int32(1))

  // 実際のVM動作: *NativeInteger を使用
  hm.Put(&NativeInteger{Value: 0}, &NativeInteger{Value: 1})
  ```
- **修正案**: `*NativeInteger` キーでのput/getテストを追加する。特に「同じ値の異なる `*NativeInteger` インスタンスで同じキーとして扱われること」を検証する。

### [m2] HashMap.put の戻り値（前の値）テストが不足

- **ファイル**: `pkg/native/native_test.go`
- **問題**: `Put` メソッドの戻り値（前の値、またはnil）を直接テストするケースがない。Java の `HashMap.put` は前の値を返す仕様で、この戻り値はスタックにpushされる（`vm.go:228-235`）。
- **修正案**: `Put` の戻り値を検証するテストを追加する。

### [m3] ifnonnull (0xC7) オペコードが未実装

- **ファイル**: `pkg/vm/instructions.go`
- **問題**: `ifnull` (0xC6) は実装されているが、対になる `ifnonnull` (0xC7) が未実装。Javaコンパイラは `if (got != null)` を `ifnull`（body外へジャンプ）として生成することが多いが、コンパイラバージョンや最適化レベルによっては `ifnonnull` を使用する可能性がある。
- **現状**: TestFib統合テストが通っていることから、現在のFib.classでは `ifnonnull` は使用されていない。将来的に必要になる可能性がある。

### [m4] getfield / putfield のnullレシーバーエラーメッセージが不正確

- **ファイル**: `pkg/vm/vm.go:139, 165`
- **問題**: nullオブジェクトに対するgetfield/putfieldでは、JVMでは `NullPointerException` が発生するべきだが、現在の実装では "receiver is not a JObject" というエラーメッセージが返される。nullの場合、`objectRef.Ref` は `nil` なので型アサーション `nil.(*JObject)` は `(nil, false)` を返し、`!ok` でエラーになる。動作としては正しくエラーになるが、エラーメッセージがミスリーディング。
- **修正案**: nullチェックを型アサーションの前に行い、`NullPointerException` を返す。
  ```go
  if objectRef.Type == TypeNull || objectRef.Ref == nil {
      return Value{}, false, fmt.Errorf("NullPointerException")
  }
  ```

### [m5] Integer.valueOf のパラメータ数がハードコードされている

- **ファイル**: `pkg/vm/vm.go:316-319`
- **問題**: `invokestatic` の `Integer.valueOf` 処理では、ディスクリプタをパースせずに直接1回のPopでint値を取得している。他のinvokestaticのユーザーメソッド呼び出しでは `countParams` を使ってディスクリプタからパラメータ数を算出している。一貫性がない。
  ```go
  if methodRef.ClassName == "java/lang/Integer" && methodRef.MethodName == "valueOf" {
      intVal := frame.Pop()  // パラメータ数をハードコード
      // ...
  }
  ```
- **現状**: `Integer.valueOf(I)` は1パラメータなので動作に問題はない。将来ネイティブメソッドが増えた際のメンテナンス性の懸念。

### [m6] Milestone 1レビューの指摘事項の一部が修正済み

- **M1 (Push/Pop 境界チェック)**: `frame.go:63-78` にpanicによる境界チェックが追加されている。✓ 修正済み。
- **M2 (sipush テスト)**: `instructions_test.go:259-282` に追加されている。✓ 修正済み。
- **M3 (if_icmpXX テスト)**: `instructions_test.go:384-422` に追加されている。✓ 修正済み。
- **M4 (ゼロ除算テスト)**: `instructions_test.go:284-337` に追加されている。✓ 修正済み。
- **M5 (ifge/ifgt/ifle テスト)**: `instructions_test.go:424-456` に追加されている。✓ 修正済み。
- **m3 (再帰呼び出しのサイクル検出)**: `vm.go:13-14, 53-57` に `maxFrameDepth = 1024` の上限が追加されている。✓ 修正済み。
- **m4 (countParams がエラーを返さない)**: `vm.go:375-417` で `(int, error)` を返すように変更されている。✓ 修正済み。
- **m5 (GetLocal/SetLocal 境界チェック)**: `frame.go:81-93` にpanicによる境界チェックが追加されている。✓ 修正済み。

---

## テストレビュー

### カバレッジの評価

| ファイル | テスト | 評価 |
|---------|--------|------|
| `pkg/vm/object.go` | `object_test.go` — 6テストケース | **良好** |
| `pkg/native/hashmap.go` | `native_test.go` — 5テストケース | **可** (`*NativeInteger` キーのテスト不足) |
| `pkg/native/integer.go` | `native_test.go` — 4テストケース | **良好** |
| `pkg/vm/instructions.go` (新規オペコード) | `ifnull`, `areturn` のみユニットテストあり | **不足** (`getfield`/`putfield`/`checkcast`なし) |
| `pkg/vm/vm.go` (新規メソッド) | TestFib統合テスト経由のみ | **可** |
| `pkg/vm/integration_test.go` | TestFib が89を正しく検証 | **良好** |

### 不足しているテストケース

**高優先度:**
1. `getfield` / `putfield` のユニットテスト（正常系 + nullレシーバー）
2. `NativeHashMap` の `*NativeInteger` キーテスト（値ベース比較の検証）

**中優先度:**
3. `HashMap.put` 戻り値テスト（前の値の返却）
4. `invokespecial` のユニットテスト（コンストラクタ呼び出し）
5. `checkcast` のユニットテスト（no-op動作の検証）

**低優先度:**
6. `invokevirtual` でのnullレシーバーのエラーハンドリングテスト
7. `invokestatic Integer.valueOf` のユニットテスト

---

## Good Points（良い点）

- **JVM仕様への正確な準拠**: ifnull のオフセット計算（opcode PC基準）、putfield のスタック操作順序（value → objectRef の順でpop）、invokespecial/invokevirtual の `this` 引数の処理、すべてJVM仕様に正確に準拠している。
- **NativeInteger の値ベース比較**: `NativeHashMap` が `*NativeInteger` キーから `.Value` (int32) を抽出してGo mapのキーとして使用する設計は正しく、異なるポインタでも同じ値なら同一キーとして扱われる。
- **HashMap の戻り値処理**: `Get` はキーが存在しない場合にnilを返し、`Put` は前の値（なければnil）を返す。Java仕様通り。
- **JObject の簡潔な設計**: `ClassName + map[string]Value` というシンプルな構造で、Milestone 1.5の要件を過不足なく満たしている。
- **new 命令の分岐設計**: クラス名に応じて `NativeHashMap` と `JObject` を切り替える設計は、将来の拡張にも対応しやすい。
- **invokevirtual の汎用化**: PrintStream.println、HashMap.get/put、Integer.intValue、ユーザー定義メソッドの4パターンを段階的にディスパッチする構造が明快。
- **countParams の堅牢性**: 配列型（`[I`, `[Ljava/lang/String;`, `[[I`）やオブジェクト型を正しくパースし、エラーも返すように改善されている。
- **maxFrameDepth による再帰防止**: Fib.javaのような再帰プログラムでの無限再帰を防止する上限（1024フレーム）が設定されている。
- **テストの読みやすさ**: `TestJObjectFields` は6つのサブテストで主要なユースケースを網羅。テスト名も具体的で分かりやすい。
- **TestFib統合テストの正確性**: `Fib(10) = 89` の検証が正しい（fib(0)=1, fib(1)=1 の初期化で、fib(10)=89）。
- **Milestone 1 レビュー指摘事項への対応**: 前回レビューの主要指摘事項（Push/Pop境界チェック、sipushテスト、if_icmpXXテスト、ゼロ除算テスト、countParamsエラー返却、再帰上限）がすべて修正済み。
