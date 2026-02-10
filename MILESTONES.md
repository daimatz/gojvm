# gojvm - JVM Implementation Milestones

GoによるJVM実装のマイルストーン計画。JVM仕様 (The Java Virtual Machine Specification) に基づく。

---

## Milestone 1: 最小限のJVM — 整数演算とHello World

**目標**: `.class`ファイルを読み込み、整数演算を行い、`System.out.println`で結果を出力できるJVMを実装する。

**検証用Javaコード例**:
```java
public class Hello {
    public static int add(int a, int b) {
        return a + b;
    }
    public static void main(String[] args) {
        int result = add(3, 4);
        System.out.println(result); // => 7
    }
}
```

### 1.1 プロジェクト初期化

- `go mod init github.com/daimatz/gojvm`
- ディレクトリ構成:

```
gojvm/
├── cmd/gojvm/
│   └── main.go              # CLIエントリポイント
├── pkg/classfile/
│   ├── parser.go             # classファイルパーサー本体
│   ├── constant_pool.go      # コンスタントプール定義・パース
│   └── types.go              # ClassFile, MethodInfo等の型定義
├── pkg/vm/
│   ├── vm.go                 # VM本体（メソッド探索・実行ループ）
│   ├── frame.go              # スタックフレーム（ローカル変数・オペランドスタック）
│   └── instructions.go       # 命令セット実装
├── pkg/native/
│   └── system.go             # ネイティブメソッド (System.out.println等)
├── go.mod
├── MILESTONES.md
└── testdata/                 # テスト用classファイル
```

### 1.2 classファイルパーサー

JVM仕様 Chapter 4 "The class File Format" に従い、以下のClassFile構造をパースする。

#### ClassFile構造

```
ClassFile {
    u4             magic;                    // 0xCAFEBABE
    u2             minor_version;
    u2             major_version;
    u2             constant_pool_count;
    cp_info        constant_pool[constant_pool_count-1];
    u2             access_flags;
    u2             this_class;
    u2             super_class;
    u2             interfaces_count;
    u2             interfaces[interfaces_count];
    u2             fields_count;
    field_info     fields[fields_count];
    u2             methods_count;
    method_info    methods[methods_count];
    u2             attributes_count;
    attribute_info attributes[attributes_count];
}
```

**注意**: すべてのマルチバイト値はビッグエンディアン (`encoding/binary.BigEndian`)。

#### 1.2.1 マジックナンバー・バージョン

- `magic` (u4): `0xCAFEBABE` を検証。一致しない場合はエラー。
- `minor_version` (u2): マイナーバージョン
- `major_version` (u2): メジャーバージョン (Java 8 = 52, Java 11 = 55, Java 17 = 61)

#### 1.2.2 コンスタントプール

コンスタントプールのインデックスは **1始まり** (0は使わない)。`constant_pool_count - 1` 個のエントリを読む。

各エントリは先頭1バイトの `tag` で種類を判別する:

| Tag | 名前                    | 構造                                                     |
|-----|------------------------|----------------------------------------------------------|
| 1   | CONSTANT_Utf8          | `u2 length; u1 bytes[length]`                            |
| 3   | CONSTANT_Integer       | `u4 bytes` (int値)                                       |
| 7   | CONSTANT_Class         | `u2 name_index` → Utf8                                   |
| 8   | CONSTANT_String        | `u2 string_index` → Utf8                                 |
| 9   | CONSTANT_Fieldref      | `u2 class_index; u2 name_and_type_index`                 |
| 10  | CONSTANT_Methodref     | `u2 class_index; u2 name_and_type_index`                 |
| 12  | CONSTANT_NameAndType   | `u2 name_index; u2 descriptor_index` → 両方Utf8          |

**Milestone 1ではこの7種のtagのみ対応すればよい。**

**注意**: CONSTANT_Long (tag=5) と CONSTANT_Double (tag=6) は2スロット消費する（Milestone 1では不要）。

#### 1.2.3 アクセスフラグ

| フラグ           | 値       | 意味                  |
|-----------------|----------|----------------------|
| ACC_PUBLIC      | 0x0001   | public               |
| ACC_STATIC      | 0x0008   | static               |
| ACC_SUPER       | 0x0020   | invokespecial特殊処理  |

#### 1.2.4 this_class / super_class

- `this_class` (u2): コンスタントプールのCONSTANT_Classへのインデックス
- `super_class` (u2): 同上 (java/lang/Object の場合は0)

#### 1.2.5 interfaces / fields

- `interfaces_count` (u2) + `interfaces[]`: Milestone 1では空（0個）を想定
- `fields_count` (u2) + `fields[]`: Milestone 1では空（0個）を想定

#### 1.2.6 methods

```
method_info {
    u2             access_flags;
    u2             name_index;        // → Utf8 (メソッド名: "main", "add"等)
    u2             descriptor_index;  // → Utf8 (記述子: "([Ljava/lang/String;)V"等)
    u2             attributes_count;
    attribute_info attributes[attributes_count];
}
```

**メソッド記述子の例**:
- `([Ljava/lang/String;)V` — main メソッド (String[] → void)
- `(II)I` — int 2つ受け取り int を返す
- `(I)V` — int 1つ受け取り void を返す

#### 1.2.7 Code属性

メソッドの `attributes` 内にある最重要属性:

```
Code_attribute {
    u2 attribute_name_index;  // → Utf8 "Code"
    u4 attribute_length;
    u2 max_stack;
    u2 max_locals;
    u4 code_length;
    u1 code[code_length];     // バイトコード本体
    u2 exception_table_length;
    {   u2 start_pc;
        u2 end_pc;
        u2 handler_pc;
        u2 catch_type;
    } exception_table[exception_table_length];
    u2 attributes_count;
    attribute_info attributes[attributes_count];  // LineNumberTable等（無視可）
}
```

Milestone 1では `exception_table` は空を想定。Code属性内の子attributesは読み飛ばしてよい。

### 1.3 実行エンジン

#### 1.3.1 VM本体 (`pkg/vm/vm.go`)

```go
type VM struct {
    ClassFiles map[string]*classfile.ClassFile  // クラス名 → ClassFile
    Frames     []*Frame                         // コールスタック
}
```

- エントリポイント: 指定クラスの `main` メソッド (`public static void main(String[])`) を探して実行
- メソッド呼び出し: 新しいFrameをスタックにpush、returnで pop
- 実行ループ: `pc` を進めながら `code[pc]` のopcodeをデコード・実行

#### 1.3.2 スタックフレーム (`pkg/vm/frame.go`)

```go
type Frame struct {
    LocalVars    []int32   // ローカル変数配列 (サイズ: max_locals)
    OperandStack []int32   // オペランドスタック (最大サイズ: max_stack)
    SP           int       // スタックポインタ
    Code         []byte    // バイトコード
    PC           int       // プログラムカウンタ
    ConstantPool []cp_info // コンスタントプール参照
}
```

**注意**: Milestone 1 ではオペランドスタックとローカル変数は `int32` だけでは不十分。オブジェクト参照（`getstatic` で取得する `System.out` 等）も扱う必要がある。`interface{}` または専用のValue型を使用する:

```go
type Value struct {
    Type  ValueType  // Int, Object, Null など
    Int   int32
    Ref   interface{}
}
```

#### 1.3.3 実行ループ

```go
func (vm *VM) executeMethod(method *classfile.MethodInfo, args []Value) Value {
    code := getCodeAttribute(method)
    frame := NewFrame(code.MaxLocals, code.MaxStack, code.Code, class.ConstantPool)
    // 引数をローカル変数にセット
    for i, arg := range args {
        frame.LocalVars[i] = arg
    }
    vm.Frames = append(vm.Frames, frame)
    defer func() { vm.Frames = vm.Frames[:len(vm.Frames)-1] }()

    for frame.PC < len(frame.Code) {
        opcode := frame.Code[frame.PC]
        frame.PC++
        switch opcode {
        case 0x60: // iadd
            // ...
        }
    }
}
```

### 1.4 命令セット（Milestone 1 実装対象）

#### 定数ロード命令

| Opcode | Hex  | ニーモニック  | オペランド    | 動作                                       |
|--------|------|-------------|-------------|-------------------------------------------|
| 1      | 0x01 | aconst_null | なし         | null をスタックにpush                       |
| 2      | 0x02 | iconst_m1   | なし         | int -1 をpush                              |
| 3      | 0x03 | iconst_0    | なし         | int 0 をpush                               |
| 4      | 0x04 | iconst_1    | なし         | int 1 をpush                               |
| 5      | 0x05 | iconst_2    | なし         | int 2 をpush                               |
| 6      | 0x06 | iconst_3    | なし         | int 3 をpush                               |
| 7      | 0x07 | iconst_4    | なし         | int 4 をpush                               |
| 8      | 0x08 | iconst_5    | なし         | int 5 をpush                               |
| 16     | 0x10 | bipush      | byte (i8)   | 符号拡張してint push (PC+1)                 |
| 17     | 0x11 | sipush      | short (i16) | 符号拡張してint push (PC+2、ビッグエンディアン) |
| 18     | 0x12 | ldc         | index (u8)  | コンスタントプール[index]の値をpush           |

#### ローカル変数命令

| Opcode | Hex  | ニーモニック | オペランド   | 動作                                   |
|--------|------|------------|------------|---------------------------------------|
| 21     | 0x15 | iload      | index (u8) | LocalVars[index] を int として push     |
| 26     | 0x1A | iload_0    | なし        | LocalVars[0] を int として push         |
| 27     | 0x1B | iload_1    | なし        | LocalVars[1] を int として push         |
| 28     | 0x1C | iload_2    | なし        | LocalVars[2] を int として push         |
| 29     | 0x1D | iload_3    | なし        | LocalVars[3] を int として push         |
| 25     | 0x19 | aload      | index (u8) | LocalVars[index] を参照として push       |
| 42     | 0x2A | aload_0    | なし        | LocalVars[0] を参照として push           |
| 43     | 0x2B | aload_1    | なし        | LocalVars[1] を参照として push           |
| 44     | 0x2C | aload_2    | なし        | LocalVars[2] を参照として push           |
| 45     | 0x2D | aload_3    | なし        | LocalVars[3] を参照として push           |
| 54     | 0x36 | istore     | index (u8) | スタックからpopしてLocalVars[index]に格納  |
| 59     | 0x3B | istore_0   | なし        | スタックからpopしてLocalVars[0]に格納      |
| 60     | 0x3C | istore_1   | なし        | スタックからpopしてLocalVars[1]に格納      |
| 61     | 0x3D | istore_2   | なし        | スタックからpopしてLocalVars[2]に格納      |
| 62     | 0x3E | istore_3   | なし        | スタックからpopしてLocalVars[3]に格納      |
| 58     | 0x3A | astore     | index (u8) | 参照をpopしてLocalVars[index]に格納       |
| 75     | 0x4B | astore_0   | なし        | 参照をpopしてLocalVars[0]に格納           |
| 76     | 0x4C | astore_1   | なし        | 参照をpopしてLocalVars[1]に格納           |
| 77     | 0x4D | astore_2   | なし        | 参照をpopしてLocalVars[2]に格納           |
| 78     | 0x4E | astore_3   | なし        | 参照をpopしてLocalVars[3]に格納           |

#### 算術命令

| Opcode | Hex  | ニーモニック | 動作                                                    |
|--------|------|------------|--------------------------------------------------------|
| 96     | 0x60 | iadd       | pop 2値、加算してpush                                    |
| 100    | 0x64 | isub       | pop 2値、減算してpush (value1 - value2)                  |
| 104    | 0x68 | imul       | pop 2値、乗算してpush                                    |
| 108    | 0x6C | idiv       | pop 2値、除算してpush (value1 / value2)。0除算はエラー     |
| 112    | 0x70 | irem       | pop 2値、剰余をpush (value1 % value2)                    |
| 116    | 0x74 | ineg       | pop 1値、符号反転してpush                                 |

**注意**: `isub`, `idiv`, `irem` では **value1がスタックの下、value2が上** (先にpushされた方がvalue1)。

#### 比較・分岐命令

すべての分岐命令はオペランドとして **2バイトの符号付きオフセット** (i16, ビッグエンディアン) を取る。
分岐先 = **分岐命令のPC** + オフセット （命令のPCであって、オペランド後のPCではないことに注意）。

| Opcode | Hex  | ニーモニック | 動作                                    |
|--------|------|-----------|----------------------------------------|
| 153    | 0x99 | ifeq      | pop 1値、== 0 なら分岐                   |
| 154    | 0x9A | ifne      | pop 1値、!= 0 なら分岐                   |
| 155    | 0x9B | iflt      | pop 1値、< 0 なら分岐                    |
| 156    | 0x9C | ifge      | pop 1値、>= 0 なら分岐                   |
| 157    | 0x9D | ifgt      | pop 1値、> 0 なら分岐                    |
| 158    | 0x9E | ifle      | pop 1値、<= 0 なら分岐                   |
| 159    | 0x9F | if_icmpeq | pop 2値、value1 == value2 なら分岐       |
| 160    | 0xA0 | if_icmpne | pop 2値、value1 != value2 なら分岐       |
| 161    | 0xA1 | if_icmplt | pop 2値、value1 < value2 なら分岐        |
| 162    | 0xA2 | if_icmpge | pop 2値、value1 >= value2 なら分岐       |
| 163    | 0xA3 | if_icmpgt | pop 2値、value1 > value2 なら分岐        |
| 164    | 0xA4 | if_icmple | pop 2値、value1 <= value2 なら分岐       |
| 167    | 0xA7 | goto      | 無条件分岐                               |

#### スタック操作命令

| Opcode | Hex  | ニーモニック | 動作                               |
|--------|------|-----------|-----------------------------------|
| 87     | 0x57 | pop       | スタックトップの値を破棄              |
| 89     | 0x59 | dup       | スタックトップの値を複製              |
| 95     | 0x5F | swap      | スタックトップ2値を交換               |

#### 戻り値命令

| Opcode | Hex  | ニーモニック | 動作                                   |
|--------|------|-----------|---------------------------------------|
| 172    | 0xAC | ireturn   | int値をpopして呼び出し元に返す           |
| 177    | 0xB1 | return    | void戻り（何も返さない）                 |

#### メソッド呼び出し・フィールドアクセス命令

| Opcode | Hex  | ニーモニック    | オペランド        | 動作                                    |
|--------|------|--------------|-----------------|----------------------------------------|
| 178    | 0xB2 | getstatic    | indexbyte1,2 (u16) | staticフィールドの値をpush              |
| 182    | 0xB6 | invokevirtual| indexbyte1,2 (u16) | インスタンスメソッド呼び出し             |
| 184    | 0xB8 | invokestatic | indexbyte1,2 (u16) | staticメソッド呼び出し                   |
| 187    | 0xBB | new          | indexbyte1,2 (u16) | オブジェクト生成（Milestone 1では最小限） |

**getstatic/invokevirtual/invokestatic のオペランド**: 2バイト (u16) のコンスタントプールインデックス。
- `getstatic`: → CONSTANT_Fieldref → クラス名 + フィールド名 + 記述子
- `invokevirtual`: → CONSTANT_Methodref → クラス名 + メソッド名 + 記述子
- `invokestatic`: → CONSTANT_Methodref → クラス名 + メソッド名 + 記述子

### 1.5 ネイティブメソッド

JVMの標準ライブラリはJavaで書かれているが、Milestone 1では以下のネイティブ実装のみ提供する:

#### System.out.println

`System.out.println` の呼び出しシーケンス:
1. `getstatic java/lang/System.out:Ljava/io/PrintStream;` — System.outオブジェクトをスタックにpush
2. `iload_N` (等) — 引数をpush
3. `invokevirtual java/io/PrintStream.println:(I)V` — メソッド呼び出し

**実装方針**: `invokevirtual` で `java/io/PrintStream.println` を検出したら、Go側の `fmt.Println()` を呼ぶ。

対応するシグネチャ:
- `java/io/PrintStream.println:(I)V` — `println(int)` → `fmt.Println(intValue)`
- `java/io/PrintStream.println:(Ljava/lang/String;)V` — `println(String)` → `fmt.Println(stringValue)`

#### getstatic java/lang/System.out

`getstatic` で `java/lang/System.out` が参照された場合、プレースホルダーのオブジェクト参照をpushする。このオブジェクトは `invokevirtual` 時に `PrintStream` であることを識別するためのマーカーとして使う。

### 1.6 CLIエントリポイント

```
$ gojvm Hello.class
7
```

`cmd/gojvm/main.go`:
1. コマンドライン引数から `.class` ファイルパスを取得
2. ファイルを読み込み、classファイルパーサーでパース
3. `main` メソッド (`public static void main(String[])`) を探す
4. VM実行エンジンで実行

### 1.7 テスト戦略

- **ユニットテスト**: 各パッケージに `_test.go`
  - `pkg/classfile/`: パーサーのテスト（バイト列からClassFile構造体へのパース）
  - `pkg/vm/`: 命令セットの個別テスト（Frame操作）
- **統合テスト**: Javaソースをコンパイルした `.class` ファイルを `testdata/` に配置し、gojvmで実行して出力を検証
- テスト用Javaファイル例:
  - `testdata/Hello.java` — 基本的な整数演算と出力
  - `testdata/Arithmetic.java` — 四則演算・剰余
  - `testdata/ControlFlow.java` — if/else, ループ

### 1.8 実装順序（推奨）

1. `go mod init` + ディレクトリ構成作成
2. `pkg/classfile/types.go` — 型定義
3. `pkg/classfile/constant_pool.go` — コンスタントプールのパース
4. `pkg/classfile/parser.go` — ClassFile全体のパース
5. パーサーのテスト（`javap -v` の出力と比較）
6. `pkg/vm/frame.go` — スタックフレーム
7. `pkg/vm/instructions.go` — 命令セット実装
8. `pkg/vm/vm.go` — 実行ループ
9. `pkg/native/system.go` — ネイティブメソッド
10. `cmd/gojvm/main.go` — CLIエントリポイント
11. 統合テスト

---

## Milestone 2: 配列・文字列操作・例外処理

### 概要
- **配列**: `newarray`, `anewarray`, `iaload`, `iastore`, `arraylength`, `aaload`, `aastore`
- **文字列**: `java/lang/String` の基本サポート（文字列結合 `StringBuilder`、文字列比較）
- **例外処理**: `athrow`, exception_table によるtry-catch、`NullPointerException`, `ArrayIndexOutOfBoundsException`, `ArithmeticException`
- **追加命令**: `long`/`double`/`float` の基本算術、型変換 (`i2l`, `i2f`, `l2i` 等)
- **CONSTANT_Long**, **CONSTANT_Double**, **CONSTANT_Float** のパース対応

## Milestone 3: オブジェクト指向

### 概要
- **クラスの継承**: `invokespecial` (コンストラクタ `<init>`)、`super` 呼び出し
- **フィールド**: `getfield`, `putfield`, `getstatic`, `putstatic` の完全実装
- **ポリモーフィズム**: `invokevirtual` のメソッドディスパッチ（vtable）
- **インターフェース**: `invokeinterface`, `instanceof`, `checkcast`
- **オブジェクトのメモリレイアウト**: ヘッダー（クラスポインタ）+ フィールド値

## Milestone 4: ガベージコレクション

### 概要
- **Mark-and-Sweep GC** の基本実装
- ヒープメモリ管理
- ルート集合（スタックフレーム、staticフィールド）の走査
- GCトリガー条件（ヒープ使用量閾値）
- ファイナライザ (`finalize()`) の基本サポート

## Milestone 5: マルチスレッド

### 概要
- `java/lang/Thread` の基本サポート
- `synchronized` ブロック / メソッド (`monitorenter`, `monitorexit`)
- `wait()`, `notify()`, `notifyAll()`
- Go の goroutine を活用したスレッドモデル
- `volatile` フィールドのメモリモデル

## Milestone 6: クラスローダー・JARサポート

### 概要
- **ブートストラップクラスローダー**: `rt.jar` (Java 8以前) / `jrt-fs.jar` (Java 9+) からの読み込み
- **ユーザー定義クラスローダー**: `java/lang/ClassLoader` のサポート
- **JARファイル**: ZIPフォーマットからの `.class` ファイル読み込み
- **クラスパス** (`-cp` / `-classpath`) オプション
- **リンク**: 検証 (verification), 準備 (preparation), 解決 (resolution)

---

## 参考資料

- [The Java Virtual Machine Specification (SE7)](https://docs.oracle.com/javase/specs/jvms/se7/html/)
- [Chapter 4: The class File Format](https://docs.oracle.com/javase/specs/jvms/se7/html/jvms-4.html)
- [Chapter 6: The JVM Instruction Set](https://docs.oracle.com/javase/specs/jvms/se7/html/jvms-6.html)
- [Java Bytecode Opcodes Reference](https://javaalmanac.io/bytecode/opcodes/)
