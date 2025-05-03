# protect: 指定フィールドを除外してstructをコピーするGoライブラリ

## 概要

protect は、Echo フレームワークの [Binding](https://echo.labstack.com/docs/binding) 機能の利便性を向上させるためのライブラリです。

REST API 開発において、リクエスト内容を `Bind()` で直接モデルに転写できる機能は非常に便利ですが、以下のようなフィールドが上書きされると、システム障害やセキュリティ脆弱性につながる恐れがあります:

* ID
* 作成日時
* セキュリティ関連の設定項目
* ユーザーが更新権限を持たないフィールド

このライブラリを使用すると、`Bind()` 実行時に保護したいフィールドを指定できるようになります。

## 提供機能

### `github.com/ikedam/protect` パッケージ

1. タグ指定に従って特定フィールドを除外した構造体コピー

   ```go
   var src, dst SomeStruct
   err := protect.Copy("create", &src, &dst)
   ```

2. 構造体の完全コピー (クローン) の作成

   ```go
   var src SomeStruct
   dst := protect.Clone(&src)
   ```

    * すべてのフィールドがコピー対象となり、タグは無視されます。

### `github.com/ikedam/protect/protectecho` パッケージ

1. 特定フィールドを保護したEcho Bind()実行

   ```go
   func Handler(c context.Context) error {
       var dst SomeStruct
       err := protectecho.Bind("create", c, &dst)
       // ...
   }
   ```

2. Echo Bind()を複数回実行可能にする機能

   ```go
   func Handler(c context.Context) error {
       c = protectecho.ReBindable(c)
       var dst SomeStruct
       err := protectecho.Bind("create", c, &dst)
       // ...
       var anotherdst SomeStruct
       err = protectecho.Bind("update", c, &dst)
       // ...
   }
   ```

## 構造体の定義方法

フィールドにタグを付けることで、コピー対象から除外するフィールドを指定できます。

```go
type SomeStruct struct {
    ID   string `protectfor:"create,update"` // create、update時に保護
    Code string `protectfor:"update"`        // update時のみ保護
    Name string                              // 常にコピー対象
}
```

* タグにはカンマ区切りで複数の値を指定できます。
* `protect.Copy("タグ値", &src, &dst)` を実行すると、第一引数のタグ値がフィールドのタグに含まれる場合、そのフィールドはコピー対象外になります。
* デフォルトのタグ名は `protectfor` ですが、カスタマイズも可能です。

### 使用例

```go
var src SomeStruct
// ...
// ID はコピーされない、Code と Name はコピーされる。
var dst1 SomeStruct
err := protect.Copy("create", &src, &dst1)

// ID も Code もコピーされない、Name のみコピーされる。
var dst2 SomeStruct
err := protect.Copy("update", &src, &dst2)
```

## コピー処理の詳細仕様

### 基本ルール

* コピー元とコピー先は同じ型である必要があります。
* 構造体の場合、すべてのエクスポートされたフィールドがコピー対象となります。
* ポインタ型の場合:
  * `nil` ならそのまま `nil` をセット。
  * 値がある場合は、再帰的にコピーされた新しいオブジェクトへのポインタをセット。

### スライス型の処理 (`protectopt`タグで制御)

1. `overwrite` (デフォルト)
    * 完全に新しいスライスを作成して上書き。
    * コピー先の既存値は無視される。
    * 各要素は単純にクローンされる (タグによる保護は無視)。

2. `match`
    * 要素ごとにコピー処理を行う。
    * スライスの長さを調整: コピー元の長さに合わせる。
        * コピー元が長い→コピー先を拡張
        * コピー先が長い→コピー先を短縮

3. `longer`
    * コピー先が長い→共通部分だけコピー(余分な要素はそのまま)
    * コピー元が長い→コピー先を拡張
    * 拡張部分は単純クローン (タグ指定は無視)。

4. `shorter`
    * コピー先が長い→コピー先を短縮
    * コピー元が長い→コピー先の長さまでだけコピー(余分な要素は無視)

### マップ型の処理 (`protectopt`タグで制御)

1. `overwrite` (デフォルト)
    * 完全に新しいマップを作成して上書き。
    * コピー先の既存値は無視される。
    * 各要素は単純にクローン (タグによる保護は無視)。

2. `match`
    * 共通キー: コピー先の要素にコピー元の要素をコピー
    * コピー元のみのキー: コピー先に新規追加(クローン)
    * コピー先のみのキー: コピー先から削除

3. `patch`
    * 共通キー: コピー先の要素にコピー元の要素をコピー
    * コピー元のみのキー: コピー先に新規追加(クローン)
    * コピー先のみのキー: 何もしない(保持)

## カスタマイズ

### カスタムProtectorの作成

タグ名をカスタマイズしたい場合は、独自のProtectorを作成できます:

```go
p := protect.NewProtector("protectfor", "protectopt")
p.Copy(&src, &dst)
```

パッケージの `Copy()` 関数を使用する場合は、内部で `protect.DefaultProtector` が使われます。
必要に応じて `protect.DefaultProtector` を上書きすることで、デフォルトで使用されるタグ名を変更できます。

## protectecho.Bind() の動作原理

`protectecho.Bind()` は内部で以下のような処理を行います:

```go
func Bind(tag string, c context.Context, dst any) {
    clone := protect.Clone(dst)     // 対象オブジェクトをクローン
    err := c.Bind(clone)            // クローンにバインド
    if err != nil {
        return err
    }
    return protect.Copy(dst, clone) // タグを考慮しながらコピー
}
```

クローンを作成→クローンにバインド→保護されたフィールドを除いてコピー、という流れで処理が行われます。
