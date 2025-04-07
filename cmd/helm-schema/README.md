# helm-schema

根据 value.yaml 自动生成 value.schema.json 文件

## 安装使用

安装

```bash
go install xiaoshiai.cn/schema/cmd/schema@latest
```

生成

```bash
helm-schema ./charts/mychart
```

## values.yaml 示例

```yaml
# @title "全局配置"
global:
  # @title "存储类"
  # @x-enum local-path="Local Path";ceph-rbd="Ceph RBD"
  storageClass: ""
# @title "服务配置"
# @title.en "Service Contiguration"
# @title.jp "サービス構成"
# @description 对应kubernetes service资源中的配置
# @description.jp kubernetes サービス リソースの構成に対応します
# @description.en Corresponds to the configuration in the kubernetes service resource
service:
  # @title "启用"
  enabled: false
  # @title address
  # @schema minLength=0
  # @if .enabled=true
  address: ""
ingress:
  enabled: false
  # @schema format=hostname
  host: "example.com"
# @title PlacholderList
list:
  - # @title Item Name
    name: "some name"
    # @title Item Value
    values:
      - "127.0.0.1"
# @title EmptyList
# @schema items='{"properties":{"name":{"type":"string"},"vals":{"type":"array","items":{"type":"string"}}}}'
emptyList: []
```
