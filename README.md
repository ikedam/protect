# protect: Go library for copying structs excluding specifiec fields

## 概要

protect は Echo の [Bingind](https://echo.labstack.com/docs/binding) の仕組みの利便性を上げることを念頭にした機能を提供します。

REST API の開発で、リクエスト内容を `Bind()` によって直接モデルに転写できる機能は非常に強力ですが、一方で以下のようなフィールドについては `Bind()` で上書きされることでシステムが破損したり、セキュリティの脆弱性に繋がります:

* ID
* データの作成日
* セキュリティ関わる設定項目
* 本来ユーザーが更新権限を持たないフィールド

これを回避するために `Bind()` の実行時に上書きをしないフィールドを設定できるようにします。

## 提供機能

以下の機能を提供します:

* `github.com/ikedam/protect` パッケージ
    * struct をタグの指定に従って特定のフィールドはコピー対象外にして再帰的にコピーする機能

        ```go
        var src, dst SomeStruct
        err := protect.Copy("create", &src, &dst)
        ```

    * struct の新規コピーを作成する機能

        ```go
        var src SomeStruct
        dst := protect.Clone(&src)
        ```

        * タグは参照せず、すべてのフィールドが処理対象になります。

* `github.com/ikedam/protect/protectecho` パッケージ

    * Echo の Bind() を特定フィールドを除外して実行する機能

        ```go
        func Handler(c context.Context) error {
            var dst SomeStruct
            err := protectecho.Bind("create", c, &dst)
            ...
        }
        ```

    * Echo の Bind() を繰り返し実行できるようにする機能

        ```go
        func Handler(c context.Context) error {
            c = protectecho.ReBindable(c)
            var dst SomeStruct
            err := protectecho.Bind("create", c, &dst)
            ...
            var anotherdst SomeStruct
            err = protectecho.Bind("update", c, &dst)
            ...
        }
        ```

## struct の定義

struct のフィールドにタグを付けることでコピー対象外にできます。
タグにはコピー時に指定する値をカンマ区切りで指定します。
`protect.Copy()` の第一引数で指定した値が、タグで指定された値に含まれる場合は該当フィールドはコピー対象外になります。

タグのデフォルト値は `protectfor` ですが、 protect.Protector オブジェクトを作成するか、 protect.DefaultProtector を上書きすることで別のタグを指定できます。

```go
type SomeStruct struct {
    ID string `protectfor:"create,update"`
    Code string `protectfor:"update"`
    Name string
}

var src SomeStruct
...
// ID はコピー対象にならないが、Code, Name はコピー対象になる。
var dst1 SomeStruct
err := protect.Copy("create", &src, &dst1)
...
// ID, Code はコピー対象にならないが、 Name はコピー対象になる。
var dst2 SomeStruct
err := protect.Copy("update", &src, &dst2)
```

## コピー処理の動作

`protectfor` タグで除外されなかった場合の値コピーの仕様は以下のとおりです:

* コピー元とコピー先が同じ型である必要があります。型が異なる場合はエラーになります。
* struct についてはすべての Exported なフィールドがコピー対象になります。
* ポインター型については、 `nil` の場合は `nil` が設定され、 `nil` でない場合は値が再帰的にコピーされた新規のオブジェクトへのポインターが設定されます。
* スライス型については、 `protectopt` タグの指定によって動作が変わります:
    * `protectopt:"overwrite"` (デフォルトの動作)
        * 各要素のクローンから構成するスライスが設定されます。
        * 各要素に設定される `protectfor` や `prptectopt` の指定は無視され、単純なクローンが行われます。
        * コピー先に設定されていた値は無視されます。
    * `protectopt:"match"`
        * スライスの各要素をコピーします。
        * コピー先のスライスの方が長い場合、コピー先のスライスの長さをコピー元のスライスの長さに合わせます。
        * コピー元のスライスの方が長い場合、コピー先のスライスの長さを延長します。
    * `protectopt:"longer"`
        * スライスの各要素をコピーします。
        * コピー先のスライスの方が長い場合、コピー元の要素のぶんだけコピーを行い、残りのコピー先の要素は何もしません。
        * コピー元のスライスの方が長い場合、コピー先のスライスの長さを延長します。
            * 延長したぶんの要素については各要素に設定される `protectfor` や `prptectopt` の指定は無視され、単純なクローンが行われます。
    * `protectopt:"shorter"`
        * スライスの各要素をコピーします。
        * コピー先のスライスの方が長い場合、コピー先のスライスの長さをコピー元のスライスの長さに合わせます。
        * コピー元のスライスの方が長い場合、コピー先のスライスの長さまでコピーを行います。
* マップ型については、 `protectopt` タグの指定によって動作が変わります:
    * `protectopt:"overwrite"` (デフォルトの動作)
        * 各要素のクローンから構成するマップが設定されます。
        * コピー先に設定されていた値は無視されます。
        * 各要素に設定される `protectfor` や `prptectopt` の指定は無視され、単純なクローンが行われます。
    * `protectopt:"match"`
        * コピー元とコピー先の両方に同じキーがある場合、コピー先の要素にコピー元の要素をコピーします。
        * コピー元にあるがコピー先にないキーについては、コピー先にクローンを追加します。
            * 各要素に設定される `protectfor` や `prptectopt` の指定は無視され、単純なクローンが行われます。
        * コピー先にあるがコピー元にないキーについては、コピー先から削除します。
    * `protectopt:"patch"`
        * コピー元とコピー先の両方に同じキーがある場合、コピー先の要素にコピー元の要素をコピーします。
        * コピー元にあるがコピー先にないキーについては、コピー先にクローンを追加します。
            * 各要素に設定される `protectfor` や `prptectopt` の指定は無視され、単純なクローンが行われます。
        * コピー先にあるがコピー元にないキーについては、何も行いません。

## Protector の作成

NewProtector() を使用して、使用するタグを変更することができます。

```go
p := protect.NewProtector("protectfor", "protectopt")
p.Copy(&src, &dst)
```

パッケージの `Copy()` の利用時は、 protect.DefaultProtector が使用されます。
protect.DefaultProtector を上書きすることでパッケージの `Copy()` で参照するタグを変更することができます。

## protectecho.Bind() の動作

protectecho.Bind() の動作は以下のとおりです:

```go
func Bind(tag string, c context.Context, dst any) {
    clone := protect.Clone(dst)
    err := c.Bind(clone)
    if err != nil {
        return err
    }
    return protect.Copy(dst, clone)
}
```
