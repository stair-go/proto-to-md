# .Proto转换成.MD

>将.proto文件转换成.md接口文档

### 规范示例:

 - [简单示例.proto](example/easy.proto)
 - [简单示例-code生成.md](example/easy.md)
 - 
 - [规范示例.proto](example/orm.proto)
 - [规范示例-code生成.md](example/orm.md)

### 使用说明
>go run main.go -imp =.proto文件名  -out = .md文件名

>参数:-imp 输入的.proto文件名

>参数:-out 输出的.md文件名

>以上文件,均无须带后缀名, 且支持绝对路径

1. 将.proto文件注释[规范化](#规范)

2. 打开命令行运行如 以下命令  


``` lsl
   go run main.go -imp=test/orm -out=test
	
```
### <span id=规范> .proto文件 注释规范说明</span>

1. 现有版本暂不支持enum
2. 不要在message中嵌套message,而应该把结构体参数提取出来,然后调用,例如[规范示例.proto](example/orm.proto)中的Uri参数
3. //annotation: 是生成文档注释的关键字,应该注意的是,具体规范可参考[规范示例.proto](example/orm.proto):

> //xxxxxxx //annotation:xxxxxxxxxxxxxx 不行,关键字前方不能再有注释

> //   annotation:xxxxxxxxxxxxxx 不行,关键字和//不应该有空格

> //annotation:xxxxxxxxxxxxxx 正确!请在结构体的参数后方,或者方法体{}上方使用

### 版本说明
   V1.0未经大量测试,可能存在不少Bug和不足,欢迎修改意见或新增需求