# クラスローダー設計ドキュメント

## 1. 簡易版 stdlib の Java ソース

### stdlib/java/lang/Object.java

```java
package java.lang;

public class Object {
    public Object() {}

    public native int hashCode();

    public boolean equals(Object obj) {
        return this == obj;
    }
}
```

**注意**: `super_class: #0`（親クラスなし）。Object はクラス階層のルート。

### stdlib/java/lang/Integer.java

```java
package java.lang;

public class Integer {
    private int value;

    private Integer(int v) {
        this.value = v;
    }

    public static Integer valueOf(int i) {
        return new Integer(i);
    }

    public int intValue() {
        return this.value;
    }

    public int hashCode() {
        return this.value;
    }

    public boolean equals(Object obj) {
        if (obj instanceof Integer) {
            return this.value == ((Integer) obj).intValue();
        }
        return false;
    }
}
```

### stdlib/java/util/HashMap.java

```java
package java.util;

public class HashMap {
    private Object[] keys;
    private Object[] values;
    private int size;

    public HashMap() {
        this.keys = new Object[64];
        this.values = new Object[64];
        this.size = 0;
    }

    public Object get(Object key) {
        for (int i = 0; i < this.size; i++) {
            if (this.keys[i].equals(key)) {
                return this.values[i];
            }
        }
        return null;
    }

    public Object put(Object key, Object value) {
        for (int i = 0; i < this.size; i++) {
            if (this.keys[i].equals(key)) {
                Object old = this.values[i];
                this.values[i] = value;
                return old;
            }
        }
        this.keys[this.size] = key;
        this.values[this.size] = value;
        this.size = this.size + 1;
        return null;
    }
}
```

## 2. コンパイル方法

```bash
javac --patch-module java.base=stdlib \
    stdlib/java/lang/Object.java \
    stdlib/java/lang/Integer.java \
    stdlib/java/util/HashMap.java
```

`--patch-module java.base=stdlib` により、JDK のシステムクラスを上書きしてコンパイルできる。
Java 17 で動作確認済み。.class ファイルは各 .java と同じディレクトリに生成される。

## 3. 未実装オペコードの網羅的リスト

stdlib の 3 つの .class ファイルと Fib.class の全バイトコードを分析した結果、
以下の **6 つのオペコード** が現在の gojvm で未実装:

| Hex    | ニーモニック  | 使用箇所                     | 動作                                                     |
|--------|-------------|----------------------------|----------------------------------------------------------|
| `0xA5` | `if_acmpne` | Object.equals              | 2つの参照を pop し、異なるなら分岐。オペランド: 16bit 符号付きオフセット |
| `0xC1` | `instanceof`| Integer.equals             | 参照を pop し、指定クラスのインスタンスか判定。true→1, false→0 を push。オペランド: 16bit CP インデックス（CONSTANT_Class） |
| `0xBD` | `anewarray` | HashMap.<init>             | int を pop し、指定型の参照配列を生成して push。オペランド: 16bit CP インデックス（CONSTANT_Class） |
| `0x32` | `aaload`    | HashMap.get, HashMap.put   | 配列参照と int インデックスを pop し、配列要素を push                |
| `0x53` | `aastore`   | HashMap.put                | 配列参照、int インデックス、値を pop し、配列に格納                    |
| `0x84` | `iinc`      | HashMap.get, HashMap.put   | ローカル変数を定数で加算。オペランド: 8bit インデックス + 8bit 符号付き定数  |

### 既に実装済みの全オペコード（37 個）

| カテゴリ              | オペコード                                                     |
|---------------------|---------------------------------------------------------------|
| 定数ロード           | aconst_null(01), iconst_m1(02)〜iconst_5(08), bipush(10), sipush(11), ldc(12) |
| ローカル変数ロード     | iload(15), iload_0(1A)〜iload_3(1D), aload(19), aload_0(2A)〜aload_3(2D) |
| ローカル変数ストア     | istore(36), istore_0(3B)〜istore_3(3E), astore(3A), astore_0(4B)〜astore_3(4E) |
| スタック操作          | pop(57), dup(59), swap(5F)                                     |
| 算術                 | iadd(60), isub(64), imul(68), idiv(6C), irem(70), ineg(74)     |
| 比較・分岐           | ifeq(99)〜ifle(9E), if_icmpeq(9F)〜if_icmple(A4), goto(A7), ifnull(C6) |
| 復帰                 | areturn(B0), ireturn(AC), return(B1)                           |
| メソッド呼び出し      | invokevirtual(B6), invokespecial(B7), invokestatic(B8)         |
| オブジェクト操作      | new(BB), getstatic(B2), getfield(B4), putfield(B5), checkcast(C0) |

## 4. クラスローダーの設計

### 4.1 ClassLoader インターフェース

```go
// ClassLoader loads .class files by class name.
type ClassLoader interface {
    LoadClass(name string) (*classfile.ClassFile, error)
}
```

### 4.2 BootstrapClassLoader

stdlib/ ディレクトリから標準ライブラリクラスを読み込む。

```go
type BootstrapClassLoader struct {
    StdlibDir string                         // "stdlib/"
    Cache     map[string]*classfile.ClassFile // "java/lang/Integer" -> *ClassFile
}

func (cl *BootstrapClassLoader) LoadClass(name string) (*classfile.ClassFile, error) {
    if cf, ok := cl.Cache[name]; ok {
        return cf, nil
    }
    path := filepath.Join(cl.StdlibDir, name+".class")
    cf, err := classfile.ParseFile(path)
    if err != nil {
        return nil, fmt.Errorf("bootstrap: class %s not found: %w", name, err)
    }
    cl.Cache[name] = cf
    return cf, nil
}
```

### 4.3 UserClassLoader

ユーザーの .class ファイルをファイルシステムから読み込む。親ローダーに委譲（parent delegation model）。

```go
type UserClassLoader struct {
    ClassPath string                         // "testdata/"
    Parent    ClassLoader                    // BootstrapClassLoader
    Cache     map[string]*classfile.ClassFile
}

func (cl *UserClassLoader) LoadClass(name string) (*classfile.ClassFile, error) {
    // 1. まず親ローダーに委譲
    if cf, err := cl.Parent.LoadClass(name); err == nil {
        return cf, nil
    }
    // 2. 親にない場合、自分のクラスパスから読む
    if cf, ok := cl.Cache[name]; ok {
        return cf, nil
    }
    path := filepath.Join(cl.ClassPath, name+".class")
    cf, err := classfile.ParseFile(path)
    if err != nil {
        return nil, fmt.Errorf("user: class %s not found: %w", name, err)
    }
    cl.Cache[name] = cf
    return cf, nil
}
```

### 4.4 VM への統合方法

```go
type VM struct {
    ClassLoader ClassLoader
    Stdout      io.Writer
    frameDepth  int
}

func NewVM(classLoader ClassLoader) *VM {
    return &VM{
        ClassLoader: classLoader,
        Stdout:      os.Stdout,
    }
}

func (vm *VM) Execute(mainClassName string) error {
    cf, err := vm.ClassLoader.LoadClass(mainClassName)
    if err != nil {
        return err
    }
    method := cf.FindMethod("main", "([Ljava/lang/String;)V")
    // ...
}
```

### 4.5 Frame の変更

Frame は所属する ClassFile への参照を持つ。これにより、各メソッドのコンスタントプールを
正しく参照できる（各クラスのメソッドは、そのクラスの CP を参照する）。

```go
type Frame struct {
    LocalVars    []Value
    OperandStack []Value
    SP           int
    Code         []byte
    PC           int
    Class        *classfile.ClassFile  // ← このメソッドが属するクラス
}
```

これは現在と同じ構造。クラスローダー導入後も `frame.Class` は
「現在実行中のメソッドが属するクラスの ClassFile」を指す。
別クラスのメソッドを呼ぶ時は、新しい Frame の `Class` がそのクラスの ClassFile になる。

### 4.6 メソッド解決の変更

```go
// resolveMethod resolves a method from its class name.
func (vm *VM) resolveMethod(className, methodName, descriptor string) (*classfile.ClassFile, *classfile.MethodInfo, error) {
    cf, err := vm.ClassLoader.LoadClass(className)
    if err != nil {
        return nil, nil, err
    }
    method := cf.FindMethod(methodName, descriptor)
    if method == nil {
        return nil, nil, fmt.Errorf("method %s.%s:%s not found", className, methodName, descriptor)
    }
    return cf, method, nil
}
```

`invokevirtual`, `invokespecial`, `invokestatic` はすべて、
`methodRef.ClassName` を使ってクラスローダーからクラスを取得し、
そのクラスでメソッドを検索する。

**重要**: JObject の `ClassName` フィールドにより、invokevirtual は
実際のオブジェクトのクラスからメソッドを探す（仮想メソッドディスパッチ）。
ただし、今回のスコープではクラス継承は Object のみなので、
`methodRef.ClassName` での解決で十分。

### 4.7 executeMethod の変更

```go
// executeMethod now takes the class context.
func (vm *VM) executeMethod(cf *classfile.ClassFile, method *classfile.MethodInfo, args []Value) (Value, error) {
    frame := NewFrame(method.Code.MaxLocals, method.Code.MaxStack, method.Code.Code, cf)
    // ... (以下同じ)
}
```

## 5. VM の変更点

### 5.1 新規オペコード実装（6 個）

#### `if_acmpne` (0xA5)

```go
case OpIfAcmpne:
    branchPC := frame.PC - 1
    offset := frame.ReadI16()
    v2 := frame.Pop()
    v1 := frame.Pop()
    // 参照の同一性チェック（identity comparison）
    if v1.Ref != v2.Ref || v1.Type != v2.Type {
        frame.PC = branchPC + int(offset)
    }
```

#### `instanceof` (0xC1)

```go
case OpInstanceof:
    index := frame.ReadU16()
    className, _ := classfile.GetClassName(pool, index)
    ref := frame.Pop()
    if ref.Type == TypeNull {
        frame.Push(IntValue(0))
    } else if obj, ok := ref.Ref.(*JObject); ok && obj.ClassName == className {
        frame.Push(IntValue(1))
    } else {
        frame.Push(IntValue(0))
    }
```

注: `java/lang/Integer` のインスタンスは JObject として生成されるようになるため、
`obj.ClassName` で判定可能。

#### `anewarray` (0xBD)

```go
case OpAnewarray:
    index := frame.ReadU16()  // CP index for element type (unused for now)
    count := frame.Pop().Int
    arr := &JArray{Elements: make([]Value, count)}
    frame.Push(RefValue(arr))
```

新しい型が必要:

```go
// JArray represents a JVM reference array.
type JArray struct {
    Elements []Value
}
```

#### `aaload` (0x32)

```go
case OpAaload:
    index := frame.Pop().Int
    arrRef := frame.Pop()
    arr := arrRef.Ref.(*JArray)
    frame.Push(arr.Elements[index])
```

#### `aastore` (0x53)

```go
case OpAastore:
    value := frame.Pop()
    index := frame.Pop().Int
    arrRef := frame.Pop()
    arr := arrRef.Ref.(*JArray)
    arr.Elements[index] = value
```

#### `iinc` (0x84)

```go
case OpIinc:
    index := frame.ReadU8()
    constVal := frame.ReadI8()
    local := frame.GetLocal(int(index))
    frame.SetLocal(int(index), IntValue(local.Int+int32(constVal)))
```

### 5.2 executeNew の変更

`new` がクラスローダーから対象クラスをロードし、JObject を作成する:

```go
case "java/lang/Object":
    // Object のインスタンスは通常直接 new されないが、念のため
    obj := &JObject{ClassName: className, Fields: make(map[string]Value)}
    frame.Push(RefValue(obj))
default:
    // すべてのクラスで統一的に JObject を作成
    obj := &JObject{ClassName: className, Fields: make(map[string]Value)}
    frame.Push(RefValue(obj))
```

HashMap も Integer も JObject として作成する。ネイティブの特別扱いは不要になる。

### 5.3 invokespecial の変更

コンストラクタ呼び出し時、クラスローダーから対象クラスを取得:

```go
case methodRef.MethodName == "<init>":
    if methodRef.ClassName == "java/lang/Object" {
        // Object.<init> は何もしない（コードは `return` のみ）
        // → stdlib の Object.class をロードして実行してもよい
        return Value{}, false, nil
    }
    // 対象クラスのコンストラクタをクラスローダーから取得して実行
    cf, method, err := vm.resolveMethod(methodRef.ClassName, "<init>", methodRef.Descriptor)
    // cf のコンテキストで executeMethod を呼ぶ
```

### 5.4 invokevirtual の変更

メソッド解決順序:
1. ネイティブメソッド（PrintStream.println, Object.hashCode）を先にチェック
2. JObject の ClassName からクラスローダーでクラスを取得
3. そのクラスでメソッドを検索して実行

```go
// Object.equals は stdlib の Object.class で定義されているが、
// Integer は equals を override しているので、
// JObject.ClassName で正しいクラスからメソッドを探す。
objectClassName := obj.ClassName  // e.g., "java/lang/Integer"
cf, method, err := vm.resolveMethod(objectClassName, methodRef.MethodName, methodRef.Descriptor)
```

### 5.5 invokestatic の変更

```go
// methodRef.ClassName からクラスローダーで静的メソッドを解決
cf, method, err := vm.resolveMethod(methodRef.ClassName, methodRef.MethodName, methodRef.Descriptor)
```

## 6. 削除すべき Native コード

以下のネイティブ実装はクラスローダー導入後に **削除** する:

| ファイル                      | 型/関数                    | 理由                                        |
|------------------------------|---------------------------|---------------------------------------------|
| `pkg/native/hashmap.go`      | `NativeHashMap`           | `stdlib/java/util/HashMap.class` で置き換え  |
| `pkg/native/integer.go`      | `NativeInteger`           | `stdlib/java/lang/Integer.class` で置き換え  |
| `pkg/native/native_test.go`  | `TestNativeHashMap` 等    | ネイティブ型のテストは不要に                    |
| `pkg/vm/vm.go` 内           | HashMap 特殊ハンドリング    | invokevirtual/invokespecial/new のハードコード |
| `pkg/vm/vm.go` 内           | Integer 特殊ハンドリング    | invokestatic/invokevirtual のハードコード      |

## 7. 残すべき Native メソッド

以下は Java ソースでは実装不可能なため、ネイティブ処理を残す:

| クラス                    | メソッド              | 理由                                |
|--------------------------|-----------------------|------------------------------------|
| `java/lang/System`       | `out` (getstatic)     | I/O はネイティブでしか提供できない       |
| `java/io/PrintStream`    | `println` (各種)      | I/O はネイティブでしか提供できない       |
| `java/lang/Object`       | `hashCode` (native)   | Object.java で `native` 宣言済み      |

### Object.hashCode の実装方針

Object.java で `public native int hashCode()` と宣言されている。
VM 側で `ACC_NATIVE` フラグ (0x0100) を検出し、ネイティブハンドラに委譲:

```go
// Method has ACC_NATIVE flag -> dispatch to native handler
if method.AccessFlags & 0x0100 != 0 {
    return vm.executeNativeMethod(className, methodName, descriptor, objectRef, args)
}
```

Object.hashCode の簡易実装: ポインタアドレスからハッシュ値を生成するか、
JObject にインクリメンタルなIDを付与する。

## 8. 実行フロー（Fib.java の例）

```
main()
  → UserClassLoader.LoadClass("Fib") → Fib.class をパース
  → Fib.main を実行
    → new Fib → JObject{ClassName:"Fib"}
    → invokespecial Fib.<init>
      → resolveMethod("Fib", "<init>", "()V") → Fib.class から取得
      → Fib.<init> 実行中:
        → invokespecial Object.<init> → no-op (or load Object.class)
        → new HashMap → JObject{ClassName:"java/util/HashMap"}
        → invokespecial HashMap.<init>
          → resolveMethod("java/util/HashMap", "<init>", "()V")
          → BootstrapClassLoader.LoadClass("java/util/HashMap") → stdlib から読込
          → HashMap.<init> 実行中:
            → invokespecial Object.<init> → no-op
            → anewarray → JArray{Elements: [64]Value}
            → putfield keys
            → anewarray → JArray{Elements: [64]Value}
            → putfield values
    → invokevirtual Fib.fib(10)
      → fib 実行中:
        → invokestatic Integer.valueOf(10)
          → resolveMethod("java/lang/Integer", "valueOf", "(I)Ljava/lang/Integer;")
          → BootstrapClassLoader.LoadClass("java/lang/Integer")
          → Integer.valueOf 実行: new Integer → JObject{ClassName:"java/lang/Integer"}
        → invokevirtual HashMap.get(key)
          → obj.ClassName == "java/util/HashMap" → HashMap.class の get メソッドを実行
          → get 内で: aaload, invokevirtual Object.equals
            → obj.ClassName が "java/lang/Integer" → Integer.class の equals を実行
              → instanceof → JObject.ClassName == "java/lang/Integer" → 1
              → getfield value, intValue で値比較
```

## 9. 実装の優先順位

### Phase 1: 新規オペコード実装（6 個）
1. `iinc` - 最もシンプル
2. `anewarray` + JArray 型
3. `aaload`, `aastore`
4. `if_acmpne`
5. `instanceof`

### Phase 2: クラスローダー基盤
1. ClassLoader インターフェース
2. BootstrapClassLoader（stdlib/ から読む）
3. UserClassLoader（ファイルシステムから読む）
4. classfile.ParseFile ヘルパー関数

### Phase 3: VM 統合
1. VM 構造体変更（ClassFile → ClassLoader）
2. executeMethod の変更（ClassFile 引数追加）
3. resolveMethod の実装
4. executeNew の統一化
5. invokespecial/invokevirtual/invokestatic の変更
6. ACC_NATIVE 対応（Object.hashCode）

### Phase 4: Native コード削除
1. NativeHashMap, NativeInteger 削除
2. ハードコードされた特殊処理を削除
3. PrintStream, System.out は残す

### Phase 5: テスト
1. 既存テスト（Hello, Add, Arithmetic, ControlFlow, PrintString）が引き続き通ること
2. Fib テストがクラスローダー経由で通ること
