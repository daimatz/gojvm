# Code Review - Milestone 1

## Summary

全体的に品質の高い実装。JVM仕様への準拠性は良好で、classファイルパーサー、バイトコード命令セット、実行エンジンの基本構造が正しく実装されている。テストも全てパスしている。主要な懸念点は、コンスタントプール解決時の境界チェック不足（パニックの可能性）と、テストカバレッジの一部不足。

---

## Critical Issues（修正必須）

### [C1] ResolveMethodref/ResolveFieldref の NameAndTypeIndex に対する境界チェック不足

- **ファイル**: pkg/classfile/constant_pool.go:149, :195
- **問題**: `ResolveMethodref` で `pool[mref.NameAndTypeIndex]` にアクセスする際、インデックスの境界チェックが行われていない。`ResolveFieldref` も同様（行195）。不正な classfile を読んだ場合、panic (index out of range) が発生する。
  ```go
  // 行149: 境界チェックなし
  nat, ok := pool[mref.NameAndTypeIndex].(*ConstantNameAndType)
  ```
  一方、同じ関数内の `pool[index]` アクセス（行136-138）では正しく境界チェックが行われている。
- **修正案**: `GetUtf8` や `GetClassName` と同様に、アクセス前にインデックスの範囲と nil チェックを追加する。
  ```go
  if int(mref.NameAndTypeIndex) >= len(pool) || pool[mref.NameAndTypeIndex] == nil {
      return nil, fmt.Errorf("invalid NameAndType index %d", mref.NameAndTypeIndex)
  }
  ```

---

## Major Issues（強く推奨）

### [M1] Frame の Push/Pop に境界チェックがない

- **ファイル**: pkg/vm/frame.go:59-68
- **問題**: `Push` はスタックオーバーフロー、`Pop` はスタックアンダーフロー（SP < 0）のチェックがない。不正なバイトコードや実装バグにより、index out of range panic が発生する可能性がある。
  ```go
  func (f *Frame) Push(v Value) {
      f.OperandStack[f.SP] = v  // SP >= len(OperandStack) で panic
      f.SP++
  }
  func (f *Frame) Pop() Value {
      f.SP--                     // SP < 0 になりうる
      return f.OperandStack[f.SP]
  }
  ```
- **修正案**: 少なくともデバッグ用にパニック前の境界チェックを追加して、意味のあるエラーメッセージを出す。または error を返す設計に変更する（ただし、パフォーマンスとのトレードオフがある）。

### [M2] sipush 命令のテストが存在しない

- **ファイル**: pkg/vm/instructions_test.go
- **問題**: `sipush` (0x11) の実装は存在するが、対応するユニットテストがない。`bipush` は正値・負値・ゼロ・最大・最小の5パターンがテストされているが、`sipush` は1つも存在しない。`sipush` は2バイト符号付き値を扱うため、`bipush` とは異なるコードパスを通る。
- **修正案**: 以下のテストケースを追加する。
  - 正値（例: 1000）
  - 負値（例: -1000）
  - 境界値: 32767 (INT16_MAX), -32768 (INT16_MIN)
  - `bipush` の範囲外の値（例: 200, -200）

### [M3] if_icmpXX 命令群のテストが存在しない

- **ファイル**: pkg/vm/instructions_test.go
- **問題**: 2オペランド比較分岐命令（`if_icmpeq`, `if_icmpne`, `if_icmplt`, `if_icmpge`, `if_icmpgt`, `if_icmple`）のユニットテストが1つも存在しない。統合テスト（ControlFlow.java）で `if_icmple` は間接的にテストされているが、各命令の taken/not taken パスの明示的なテストがない。
- **修正案**: 少なくとも `if_icmpeq`（taken/not taken）と `if_icmplt`（taken/not taken）のユニットテストを追加する。

### [M4] idiv/irem のゼロ除算テストが存在しない

- **ファイル**: pkg/vm/instructions_test.go
- **問題**: `idiv` と `irem` のゼロ除算処理は実装されている（instructions.go:188-189, 196-197）が、これをテストするユニットテストがない。ゼロ除算は JVM 仕様で明確に定義されたエラーケースであり、テストは必須。
- **修正案**: ゼロ除算が `ArithmeticException` エラーを返すことを検証するテストを追加する。

### [M5] 分岐命令の一部テスト不足 (ifge, ifgt, ifle)

- **ファイル**: pkg/vm/instructions_test.go
- **問題**: 単項比較分岐命令のうち `ifeq`, `ifne`, `iflt` はテストされているが、`ifge`, `ifgt`, `ifle` のテストが存在しない。
- **修正案**: 各命令の taken/not taken パスを最低1つずつテストする。

---

## Minor Issues（改善提案）

### [m1] 未使用の SystemOut 構造体

- **ファイル**: pkg/native/system.go:9
- **問題**: `SystemOut struct{}` が定義されているが、コード内のどこからも参照されていない。`PrintStream` が直接使用されている。
- **修正案**: `SystemOut` を削除する。

### [m2] VM が単一クラスのみサポート

- **ファイル**: pkg/vm/vm.go:14-17
- **問題**: `VM` 構造体は `ClassFile *classfile.ClassFile` として単一クラスのみ保持しているが、MILESTONES.md の仕様では `ClassFiles map[string]*classfile.ClassFile` としている。Milestone 1 では同一クラス内の `invokestatic` のみなので動作するが、Milestone 2 以降で複数クラス対応が必要になった際にリファクタリングが必要。
- **修正案**: 今の段階では問題ないが、将来のマイルストーンを見据えた設計メモとして認識しておく。

### [m3] 再帰呼び出しのサイクル検出がない

- **ファイル**: pkg/vm/vm.go:44-72
- **問題**: `executeMethod` は再帰的に呼び出されるが、無限再帰に対する防御がない。Javaプログラムが無限再帰を行った場合、Go のスタックオーバーフローになる。
- **修正案**: フレーム数の上限チェック（例: 1024フレーム）を追加する。Milestone 1 では優先度低。

### [m4] countParams がエラーを返さない

- **ファイル**: pkg/vm/vm.go:228-270
- **問題**: `countParams` は不正なディスクリプタに対して 0 を返すだけで、エラーを報告しない。不正な classfile を読んだ場合にサイレントに失敗する。
- **修正案**: `(int, error)` を返す設計に変更する。

### [m5] GetLocal/SetLocal に境界チェックがない

- **ファイル**: pkg/vm/frame.go:71-78
- **問題**: `GetLocal` と `SetLocal` はインデックスの境界チェックを行っていない。不正なバイトコードにより panic の可能性がある。[M1] と同様の問題。
- **修正案**: [M1] と同様に境界チェックを追加する。

### [m6] ldc 命令のユニットテストが存在しない

- **ファイル**: pkg/vm/instructions_test.go
- **問題**: `ldc` 命令はコンスタントプールへのアクセスを伴うため、統合テストでは間接的にテストされているが、ユニットレベルでのテストが存在しない。`executeAndGetInt` ヘルパーが `nil` ClassFile で Frame を作成するため、ldc のテストが難しい構造になっている。
- **修正案**: ClassFile を持つ Frame を使う ldc 専用のテストヘルパーを追加する。

### [m7] aload/astore 命令のユニットテストが存在しない

- **ファイル**: pkg/vm/instructions_test.go
- **問題**: `aload`, `aload_0`〜`aload_3`, `astore`, `astore_0`〜`astore_3` のユニットテストが存在しない。参照型の値のロード/ストアは整数型とは異なるコードパスを通る。
- **修正案**: RefValue と NullValue を使ったテストケースを追加する。

### [m8] 統合テストに異常系テストがない

- **ファイル**: pkg/vm/integration_test.go
- **問題**: 全ての統合テストが正常系のみ。例えば、main メソッドが存在しない classfile の実行テストや、不正なバイトコードを含む classfile のテストがない。
- **修正案**: 少なくとも「main メソッドが見つからない場合のエラー」テストを追加する。

---

## テストレビュー

### カバレッジの評価

| ファイル | テストカバレッジ | 評価 |
|---------|----------------|------|
| pkg/classfile/parser.go | 正常系2パターン + 不正マジック | **良好** |
| pkg/classfile/constant_pool.go | parser_test 経由で間接テスト | **可** |
| pkg/vm/frame.go | Push/Pop/LocalVars の基本操作 | **良好** |
| pkg/vm/instructions.go | iconst, bipush, 算術, 分岐, スタック操作, ローカル変数 | **可〜良好** |
| pkg/vm/vm.go | 統合テスト経由 | **可** |
| pkg/native/system.go | 統合テスト経由 | **可** |
| cmd/gojvm/main.go | テストなし | **不足** |

### 不足しているテストケース

**高優先度:**
1. `sipush` のユニットテスト（境界値含む）
2. `if_icmpXX` のユニットテスト（各命令の taken/not taken）
3. `idiv`/`irem` のゼロ除算テスト
4. `ifge`/`ifgt`/`ifle` のユニットテスト

**中優先度:**
5. `ldc` 命令のユニットテスト（Integer / String）
6. `aload`/`astore` 系のユニットテスト
7. 統合テストの異常系（main メソッドなし等）
8. `aconst_null` のテスト

**低優先度:**
9. 整数オーバーフロー（INT32_MAX + 1）のテスト
10. `Frame.ReadU8`/`ReadI8`/`ReadU16`/`ReadI16` のユニットテスト
11. コンスタントプールの不正インデックスに対するエラーテスト

---

## Good Points（良い点）

- **JVM仕様への忠実な準拠**: 分岐命令のオフセット計算（opcode PC 基準）、isub/idiv/irem のオペランド順序（value1=下, value2=上）、bipush の符号拡張、コンスタントプールの1始まりインデックスなど、JVM仕様の要注意ポイントが全て正しく実装されている。
- **クリーンなパッケージ構成**: classfile / vm / native / cmd の分離が適切で、パッケージ間の依存関係が一方向。
- **充実したエラーメッセージ**: classfile パーサーのエラーメッセージにインデックス番号やメソッド名が含まれており、デバッグに有用。`fmt.Errorf` の `%w` によるエラーラッピングも適切。
- **Stdout の DI**: `VM.Stdout` を `io.Writer` として注入可能にしている設計は、テスタビリティが高い。統合テストで `bytes.Buffer` にキャプチャできている。
- **テストヘルパーの設計**: `executeAndGetInt` や `runClass` ヘルパーにより、テストが簡潔で読みやすい。
- **メソッドディスクリプタのパース**: `countParams` が配列型（`[I`, `[Ljava/lang/String;`, `[[I`）を正しく処理している。
- **統合テストの網羅性**: Hello（基本出力）、Add（メソッド呼び出し）、Arithmetic（四則演算）、ControlFlow（条件分岐・ループ）、PrintString（文字列出力）と、Milestone 1 の主要機能を網羅している。
