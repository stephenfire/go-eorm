## go Excel Object Relational Mapping

### 文件格式

存在两种文件格式，`xls`和`xlsx`。使用不同库进行读取

映射通过对象属性的tag值来完成，tag值定义`title_path`，从而确定当前属性映射的列

### 约定

#### 映射值类型

缺省支持四种值类型的映射：`string`/`int64`/`float64`/`bool`

如果需要其他类型的映射，请使用`setter`方法自主转换

#### 空单元格

一个单元格内容为空时，缺省被处理成零值，即：

* 对于`string`类型，为""
* 对于`int64`和`float64`，为0
* 对于`bool`类型，为`false`

#### 合并的单元格

由于使用的类库中有些并不支持“合并单元格”的功能，所以为了简单，统一不再分辨单元格是被合并还是简单的没有值。
处理方式如下：

* 在**表头**中，遇到空的单元格，被认定为“被横向合并”的单元格，使用同一行前一列的值填充。
* 在**表体**中，遇到空的单元格，则直接返回空。

### TAG

使用name为`eorm`的tag给struct属性绑定列映射，包括`title_path`和`constraint`，格式为

```go
`eorm:"title_path[,constraint]"`
```

#### title_path

* 分隔符：`/`。每一个`/`，就代表表头多一层（也就是excel中的一行）。
* 一个`title_path`是由多级`title`间隔`/`Join出的字符串。
* 一个`title_path`的高度（即层数）即为path中分隔符数量+1。没有分隔符说明表头只有1层（只占excel中的一行）
* 在同一个类型中定义的所有`title_path`层级数必须一致，因为一张表表头所有列高度一致
* 转义：由于`title_path`定义在`StructTag`中，所以需要将一些非法字符转义。使用`'%'+字符HEX码`的方式：`%HH`
* 空`title`：
    * 当`title_path`以分隔符`/`开始，也就是第一个`title`为空，表示跳过表头第一行，第一行表头内容不做匹配，相当于*通配*第一行。
    * 当`title_path`中的任意一层为空字符串（""）时，则优先匹配同一行内最后一个有效值(合并单元格)，或匹配空

与文件系统路径类似，使用层级方式表示一个excel文件的列，如下图title:

![layer titles](layer_titles.png)

| 列 | title_path          |
|---|---------------------|
| A | 序号//                |
| B | 名称//                |
| C | 第一级/反引号%60测试/空%20格  |
| D | 第一级/反引号%60测试/斜杠%2F  |
| E | 第一级/双引号%22测试/反斜杠%5C |
| F | 第一级/双引号%22测试/第三级    |
| G | 第一级/没有第三级/          |

##### 数组映射

由于表头内容有可能出现重复，或由于`title_path`中出现*通配*，此时一个`title_path`可能出现对应多列的情况，值不唯一。

#### constraint

没有`constraint`值是缺省情况，此时类型字段(field)与表列(column)的绑定关系不是必须的，极端情况下，允许所有field都没有与任何column绑定。

##### required

表示当前field所绑定的`title_path`必须存在。如果不存在，则认为表格与对象类型不匹配。可以用来检查类型与表格是否匹配。

##### not_null

* 首先，`not_null`具有`required`的特性，
* 同时，在进行值映射时，当前field对应的column必须有值（非空单元格）

### ORM

#### 缺省类型值映射

对于$kind$为`string` `int64` `float64` `bool`的类型属性，可以由本包的缺省逻辑自动映射

#### 数组值映射

对于值不唯一的`title_path`，分为下列两种情况:

* 如果不使用`setter`映射，则对应的类型属性必须是切片类型，如果切片类型的$Elem$类型又是缺省类型，则缺省逻辑可以自动映射
* 如果使用`setter`映射，则对应`setter`方法参数必须是切片类型

不属于上述两种情况时，运行时会报错

#### setter映射

如果属性类型不是缺省类型，或用户希望自定义解析逻辑，则可以使用`setter`方法进行映射。

通过为类型增加签名为 $Set$_AttributeName_`(s ~string|~int64|~float64|~bool)` 的方法定义`setter`
方法，其中_AttributeName_即为类型属性名，参数为`~string|~int64|~float64|~bool`

通过为类型增加签名为 $Set$_AttributeName_`(ss []~string|[]~int64|[]~float64|[]~bool)` 的方法定义数组`setter`
方法，其中_AttributeName_为类型属性名，参数类型为`~string|~int64|~float64|~bool`的切片
