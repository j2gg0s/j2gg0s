ElasticSearch 中的 Index, Document 对应 MySQL 中的 Table 和 Row.
但是我们需要注意, index 即时名词也是动词,
在 ES 的语境中 index the document 指将文档插入 ES.

ES 是强类型的, document 的字段类型是确定而且唯一的.
Dynamic Mapping 屏蔽了这一细节, 并让轻度使用者产生了一定程度的错觉.
