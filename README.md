# Schema

本文档用于记录和规范 schema 的使用, 由于实际情况的需要, 相比于使用原始的 jsonschema ，增加了许多扩展的属性。

本文使用 <https://json-schema.org/draft/2020-12> 版本 jsonschema 规范。

通过 schema 渲染表单主要利用到了 jsonschema 中的 core 和 validation 部分。

快速学习 jsonchema 语法： <https://www.learnjsonschema.com/2020-12/>

## 工具

- [chart-built-tool](https://github.com/poxiaoyun/build-tool/tree/main/cmd/chart-build-tool), 用于从 helm chart 的 values.yaml 文件生成 schema 文件。

## 表单渲染

定义了表单渲染的规则，实现时应该尽可能的以用户为中心，最大限度的利用 validation 中的数据。

通用规则：

- integer, string, number 等基本类型，如果没有设置 validation 中的限制，则默认允许为**零值**。
- 若出现了 const 则表示该值不可修改，则展示为只读项目。

### string ✅

```yaml
type: string
title: "标题"
description: "描述"
default: "默认值"
examples:
  - "示例1"
  - "示例2"
maxLength: 10 # 最大长度，若不指定则为无限大
minLength: 1 # 最小长度，若不指定则为 0，即允许空字符串
format: date-time # 在规定了 format 的情况下，使用format对应的输入模式，以及限制规则
pattern: ^[a-zA-Z0-9_\-]+$ # 正则表达式，可以利用正则表达式来限制输入框的输入
maximum: 10 # 最大值，若不指定则为无限大，在format=quantity 时，可以限制值大小。
minimum: 1 # 最小值，若不指定则为 0
enum: # 若指定枚举，则可以更改为选择框
  - "value1"
  - "value2"
readOnly: true # 只读，用户无法修改该值，通常用于仅允许创建时指定且无法修改的属性，例如storagclass。
writeOnly: true # 只写，用户无法读取该值，通常用于密码等敏感信息。
```

常见的可以支持的 format （部分支持）

| format    | 说明                    | 示例｜                  |
| --------- | ----------------------- | ----------------------- |
| cron      | cron 表达式             | "0 0 \* \* \*"          |
| password  | 密码                    | "\*\*\*\*"              |
| email     | 邮箱                    | "<example@example.com>" |
| date-time | 日期时间                | "2020-01-01T12:38:00Z"  |
| date      | 日期，仅选择日期        | "2020-01-01"            |
| time      | 时间，仅选择时间        | "12:38:00"              |
| hostname  | 主机名                  | "example.com"           |
| ipv4      | IPv4 地址               | "192.168.0.1"           |
| ipv6      | IPv6 地址               | "::1"                   |
| uri       | URI (包含 URL)          | "<http://example.com>"  |
| duration  | 时间间隔                | "1m30s"                 |
| quantity  | 数量(使用 k8s quantity) | "100Mi" "1" ,"1Gi"      |

除了标准中规定的 format ，可以自行扩展 format 以支持自定义的选择逻辑。

由于 date 和 time 没有时区信息，不推荐使用，建议优先选择 date-time。

### integer ✅

```yaml
type: integer
default: 1
maximum: 10
minimum: 1
multipleOf: 2 # 只能是 2 的倍数
```

- 在规定了最大值最小值的情况下，可以将数字的输入框自动更改为 slider 等比输入框更友好的交互模式。

### number ✅

number 范围比 integer 更大，一般来说需要使用到小数时才选择 number

```yaml
type: number
default: 0.2
maximum: 1.0
minimum: 0.1
multipleOf: 0.1 # 只能是 0.1 的倍数, 通常用于限制小数位数
x-increasing-only: true # 只能是相同或者增大的数字
```

- 在规定了最大值最小值的情况下，可以将数字的输入框自动更改为 slider 等比输入框更友好的交互模式。

### object ✅

```yaml
type: object
required: # 定义不能为空值/零值的字段
  - name
properties:
  name:
    type: string
  address:
    type: string
```

定义了一个结构体，包含了 name 和 address 两个属性，其中 name 为必填项（即不允许空值和零值）。

> 在实践中，通常不会使用 required 字段。而是在对应字段上施加 validation 中的限制即可。

### array ✅

```yaml
type: array
minItems: 1 # 最少一个元素
uniqueItems: true # 不允许重复
items:
  type: string
enum: # 可以选择的值,此时可以更改为多选框
  - "value1"
  - "value2"
```

对于数组类型，需要增加添加和删除的操作界面。

### 条件渲染 (部分支持)

根据不同的输入，执行不同的条件。

```yaml
type: object
properties:
  mode:
    type: string
    enum:
      - "standalone"
      - "cluster"
if:
  properties:
    mode:
      const: "cluster"
then:
  properties:
    replicas:
      minimum: 3
```

## 通用

### path 语法

path 语法与 jsonpath 语法相同。

规定：

- 以 `.` 开头,则表示当前层级数据。
- 以 `$.` 开头,则表示根层级数据。
- 对于数组查询，`.items[0]` 或者 `.items.0` 均为有效。

例如，我们将 schema 视为整个 context，不同层级的子 schema 之间需要访问数据时，需要使用 path 语法：

```yaml
type: object
properties:
  global:
    type: object
    properties:
      enabled:
        type: boolean
        default: true
  persistence:
    type: object
    x-hidden:
      - path: $.global.enabled
        value: false
    properties:
      enabled:
        type: boolean
        default: true
      storageClass:
        type: string
        x-hidden:
          - path: $.enabled
            value: false
```

其中：

- `.enabled` 为当前层级的 enabled 属性。
- `$.global.enabled` 为根层级的 global.enabled 属性, 使用 $ 允许跨层级访问数据。

### 模板语法

在编写的扩展中，有时需要上下文数据，需要使用模板语法从运行时传递数据。
模板语法使用一对或者两对 `{}` 包裹，内部为 path 语法。

渲染时，根据模板中指名的数据位置，将模板占位符替换。

例如：从环境中填充租户信息

```txt
url: /v1/tenants/{.tenant}
```

等价的

```txt
url: /v1/tenants/{{ .tenant }}
```

### CEL 表达式

部分扩展中，可能需要更复杂的条件判断，支持使用 CEL 表达式。

例如，过滤返回值中的条件: `metadata.spec.foo == 'bar' && metadata.spec.num > 3`

## 扩展

为了与 josnschema 定义中的默认属性作区分，扩展属性均以 `x-` 开头。

### x-enum ✅

相比原始的 enum 增加显示字段，将 value 和展示给用户的属性分开,icon为网络路径

```yaml
# @x-enum 1=选项1 2=选项2
x-enum:
  - label: 选项1
    value: 1
    icon: 可选
  - label: 选项2
    value: 2
    icon: 可选
```

### x-render ✅

表单渲染方式，用于手动指定表单的展示方式

| render   | description    |
| -------- | -------------- |
| textarea | 使用多行文本框 |
| radio    | 使用单选按钮   |

示例：

```yaml
type: string
enum:
  - "value1"
  - "value2"
x-render: radio
```

对 enum 使用 单选按钮模式，而非下拉选择框。

### x-resource-enum ✅

用于从 kubernetes 中加载已经存在的资源。

> x-resource-enum 适用于需要动态获取 Kubernetes 资源列表的场景，与普通 enum 不同，x-resource-enum 的选项来源于集群中的实际资源而非静态枚举值。

对于 namespace scope 资源，需要前端自行增加 namespace 后再查询

```yaml
type: string
x-resource-enum:
  apiVersion: storage.k8s.io/v1
  resource: storageclasses
  label-selector: ismc.xiaishiai.cn/access-mode-readwritemany==true
  field-selector: provisioner=ceph.rook.io/block
  selectable: metadata.labels['ismc.xiaishiai.cn/access-mode-readwritemany'] == 'true'
```

- apiVersion: 资源的 apiVersion
- resource: 资源的名称, 如 pods, deployments, configmaps 等
- label-selector?: 资源的 label 选择器
- field-selector?: 资源的 field 选择器
- selectable?: 该资源是否可选的条件表达式

### x-quantity ❌

适用用作表示数量的 string 类型，将字符串视为 quantity 处理, 通常用于 k8s 的资源大小。

```yaml
type: string
default: 1Gi
x-quantity:
  unit: Gi
  minimum: 1
  maximum: 1024
  step: 1 #默认为1
```

> 相比于在 schema 上设置 maximum 和 minimum（无效的 validation），将 quantity 的验证独立设置更好

对于 x-quantity 的渲染，如果指定了范围，则自动切换为滑动选择条方式。

### x-hidden(depracated)

Depraceted: 使用 if-then-else 替换 x-hidden。

⚠️ hidden 存在的一个问题是 validation 和 hidden 没有互动。即期望被隐藏的项目不需要被验证。

例如，在数据库验证中，当 standalone 模式时，不要求 replicas；在 cluster 模式时，要求 replicas>=3。考虑如下写法：

```yaml
# @x-hidden .mode=standalone
# @schema minimum=3
replicas: 1
```

当 .mode=standalone 时隐藏 replicas，如果 replicas 上设置 minimum=3 则在 standalone 时无法通过验证。
解决此问题的方式应当为使用 if-then 关键字替代该功能。

```yaml
type: object
properties:
  mode:
    type: string
    enum:
      - "standalone"
      - "cluster"
if:
  properties:
    mode:
      const: "cluster"
then:
  properties:
    replicas:
      minimum: 3
```

用在满足某个条件时，隐藏该选项（及下级选项）

```yaml
# @hidden .enabled=false
x-hidden:
  - path: .enabled
    value: false
```

```yaml
# @hidden .enabled=false and .enabled2=true
x-hidden:
  - path: .enabled
    value: false
  - operator: and
    path: .enabled2
    value: true
```

```yaml
# @hidden .enabled=false or .enabled2=true
x-hidden:
  - path: .enabled
    value: false
  - operator: or
    path: .enabled2
    value: true
```

### x-remote-enum ✅

用于从远程接口加载 enum 选项

```yaml
x-remote-enum:
  table:
    columns:
      - jsonPath: $.resources[2].name
        title: GPU
      - jsonPath: $.resources[2].resourceQuantity
        title: GPU数量
      - jsonPath: $.resources[0].resourceQuantity
        title: CPU
      - jsonPath: $.resources[1].resourceQuantity
        title: 内存
      - jsonPath: $.onSale
        title: 是否可用
        type: boolean
    http:
      jsonPath: $.data
      url: /v1/pai/tenants/{.tenant}/regions/{.region}/sku-infos?resource-name=gpu
    key: $.name
```

以如下返回结果举例

```json
{
  "data": [
    {
      "name": "tiny",
      "resources": []
    }
  ]
}
```

其中：

- http.url 为从远程加载的路径，支持模板语法。
- http.jsonPath 为从远程加载的数据中，list 所在的 jsonpath。
- key 为 []data 中每个项目的 jsonpath ，用于最终选择的结果。
- columns 为表格中需要展示的每个项目

> 对于过于复杂的场景，建议增加针对该场景的 扩展。

### x-object-enum ❌

用于从远程加载整个 object

```yaml
type: object
properties:
  name:
    type: string
  age:
    type: string
x-object-enum:
  columns:
    - jsonPath: $.name
      title: 姓名
    - jsonPath: $.currentAge
      title: 年龄
      key: age # 用于 remapping
  http:
    jsonPath: $.items
    url: /v1/users
```

- 注意，没有在 properties 中指定的属性应当不被保留。

### x-s3-enum ✅

用于从已有的 s3 提供者加载 s3 配置。

```yaml
type: object
title: S3 Configuration
properties:
  endpoint:
    type: string
  region:
    type: string
  bucket:
    type: string
  accessKey:
    type: string
  secretKey:
    type: string
  forcePathStyle:
    type: boolean
x-s3-enum: true # 表示该项为 s3 配置
```

实现时，返回的值必须包含如下字段：

```yaml
endpoint: ""
region: "us-east-1"
bucket: ""
accessKey: ""
secretKey: ""
forcePathStyle: true
```

### x-sku-enum

用于在 PAI 中加载 sku 信息。

```yaml
type: string
x-sku-enum:
  type: gpu # sku 类型 筛选, cpu, gpu, npu, vgpu，多个以 ',' 分割
```

### x-resource-enum storageclasses

用于加载存储卷信息及过滤条件

```yaml
type: string
x-resource-enum:
  apiVersion: storage.k8s.io/v1
  resource: storageclasses
  selectable: labels['ismc.xiaishiai.cn/access-mode-readwritemany'] === 'true' // 开启前端多节点读写是否可以勾选
  label-selector: ismc.xiaishiai.cn/access-mode-readwritemany=true // 进行label-selector过滤
  field-selector: key=value // 进行field-selector过滤
```

### x-storageset-enum

用于在 PAI 中加载存储集信息。

```yaml
type: object
properties:
  kind:
    type: string
  name:
    type: string
  source:
    type: string
  targetPath:
    type: string
x-storageset-enum:
  type: dataset # 存储集类型 筛选, dataset, modelset
  label-selector: pai.kubegems.io/tag-deepseek in (true),pai.kubegems.io/tag-train in (true)
```

## 注意

### 关于 required（或 require） 字段 ❌

required 字段用于验证 object 下需要的属性，其值为一个列表：

```yaml
type: object
required:
  - name
properties:
  name:
    type: string
  address:
    type: string
```

下面的用法是错误的

```yaml
type: string
required: true
```

如果需要验证 string 必填，则需要设置 `minLength: 1` 来替代。

```yaml
type: string
minLength: 1
```
