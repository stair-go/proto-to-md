package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

//message详细信息
// 请求或响应参数
type message struct {
	name       string //参数名
	goType     string //参数类型
	annotation string //注释
}

//service详细信息
// 接口
type service struct {
	api        string //接口名
	req        string //请求方法
	res        string //响应方法
	annotation string //注释
}

// 使用make函数创建一个serviceMap,保存各个api 信息
var serviceMap = make(map[string]service)

// 使用make函数创建一个serviceMap,保存各个message 信息
var messageMap = make(map[string][]message)

// 使用make函数创建一个serviceMap,保存各个message 的注释信息
var messageAnnoMap = make(map[string]string)

//结构体参数
var structMap = make([]string, 0)

// service集合名
var serviceSetName = ""

// service注释
var serviceAnnotation = ""

//API总数
var ApiCount = 1

//MD文件输出结果
var MdResult = ""

// 示例: go run  main.go -imp=test/orm  -out=oneFile
func main() {
	// 要转换的文件名
	fileName := flag.String("imp", "", "import fileName ")
	//  生成的md文件名
	outFileName := flag.String("out", "", "output fileName ")
	flag.Parse()
	//fmt.Println("import fileName=",*fileName,",output FileName=",*outFileName)
	if *fileName == "" {
		fmt.Println("请输入文件名!")
		return
	}

	//根据绝对路径,通过文件名,查询文件获取数据
	initData(*fileName)

	//扫描结果
	fmt.Println("serviceSetName扫描结果", serviceSetName)
	fmt.Println("serviceAnnotation扫描结果", serviceAnnotation)
	fmt.Println("serviceMap扫描结果", serviceMap)
	fmt.Println("messageMap扫描结果", messageMap)
	fmt.Println("messageMap扫描结果", messageAnnoMap)
	//将map容器里的数据,填充到MdResult中
	setMdResult()
	//填充结果
	//fmt.Println("MdResult : \n",MdResult)

	if *outFileName == "" {
		outFileName = &serviceSetName
		fmt.Println("未输入生成 .MD文件名! 自动用service集合命名:", serviceSetName, ".md")
	}

	//输出dResult到文件中,参数是文件名
	writeMdFile(*outFileName)
}

//从当前目录获取数据,装载到全局变量中
func initData(fileName string) {
	//if fileName == "" {
	//	fmt.Println("请输入文件名!")
	//	return
	//}
	//获取输入流
	file, err := os.OpenFile(fileName+".proto", os.O_RDWR, 0666)
	if err != nil {
		panic("未找到文件!")
	}
	defer file.Close()
	//读取到缓冲区
	buf := bufio.NewReader(file)

	//临时存储 注释
	var annotation = ""
	for {
		// 读取下一行
		line, err := buf.ReadString('\n')
		// 去掉后面的空格
		line = strings.TrimSpace(line)
		//读取注释
		if strings.Contains(line, "//") {
			//清除不带有annotation:的//注释
			if annotation == "\r\n" {
				annotation = ""
			}
			//拼接多行注释
			annotation += fmtAnnotationLine(line) + "\r\n"
			//获取注释
			//fmt.Println("annotation",annotation)

			//获取service即Api接口
		} else if strings.Contains(line, "service") {

			// 获取service集合名,以便后面文件输出
			serviceSetName = getBetweenStr(line, "service", "{")
			//设置该service的Annotation
			serviceAnnotation = annotation

			//fmt.Println("service annotation",annotation)
			//fmt.Println("service",serviceSetName)

			//读取下一行,遍历 API接口,直到读取到 }\r\n
			for line, _ := buf.ReadString('\n'); line != "}\n" && line != "}\r\n" && line != "}"; line, _ = buf.ReadString('\n') {
				//strings.Contains(line,"rpc")||strings.Contains(line,"//")||line=="\r\n"
				//service 中不包含rpc,则认为是注释或空行,跳过
				if !strings.Contains(line, "rpc") {
					continue
				}
				// 去掉annotation之后的注释
				apiAnnotation := fmtAnnotationLine(line)
				//获取APIName
				serviceName := getBetweenStr(line, "rpc", "(")
				//fmt.Println("serviceName",serviceName)

				//获取请求参数
				serviceReq := getBetweenStr(line, "(", ")")
				//fmt.Println("request",serviceReq)

				//截取掉请求参数
				line = string([]rune(line)[strings.Index(line, ")")+1:])

				//获取响应参数
				serviceRes := getBetweenStr(line, "(", ")")
				//fmt.Println("respone",serviceRes)

				// 初始化一个api接口
				serviceType := service{api: serviceName, req: serviceReq, res: serviceRes}

				// 该line 如果有// 则添加接口注释
				if strings.Contains(line, "annotation:") {
					serviceType.annotation = apiAnnotation
					//serviceType.annotation = getBetweenStr(line,"//",")")
				}
				//放到map中
				serviceMap[serviceName] = serviceType
			}
			// 扫描message
		} else if strings.Contains(line, "message") {

			// 获取messageProto第一行,即struct的name
			messageProto := getBetweenStr(line, "message", "{")
			// 设置该message注释
			messageAnnoMap[messageProto] = annotation

			//读取message 中的参数 ,截止为读取到 }
			for line, _ := buf.ReadString('\n'); line != "}\n" && line != "}\r\n" && line != "}"; line, _ = buf.ReadString('\n') {

				//message 中不包含; 则认为是注释或空行,跳过
				if len(line) < 6 || !strings.Contains(line, ";") || !strings.Contains(line, "=") || strings.Contains(line, "enum") || "    //" == line[:6] {
					continue
				}
				// 去掉annotation之后的注释
				messageAnnotation := fmtAnnotationLine(line)
				//格式化一下,去除// = ; 如:
				// string trace = 1; // 这是注释   ==>
				// string trace 1 这是注释
				line = getMessage(line)

				// 通过空格分割为 ==> string[string trace 1 这是注释]
				words := strings.Fields(line)
				if len(words) < 2 {
					continue
				}
				//将之类型转换成GoType
				goType, is_status := checkType(words[0])
				if is_status {
					messageAnnotation = "[(结构体参数)](#" + goType + ")" + messageAnnotation
				}
				//设置参数,后面判断,是否有注释
				message := message{name: words[1], goType: goType}
				//todo  获取注释 ,即从参数index 后,//后面的都视为注释
				// 后期注释可能需要细分
				message.annotation = messageAnnotation
				//if len(words)>3 {
				//	words =words[3:]
				//	for _,v:= range words{
				//		message.annotation += " "+v
				//	}
				//}
				//添加到 messageMap
				messageMap[messageProto] = append(messageMap[messageProto], message)
			}

			//如果下一行不包含注释, 也不包含service,message 则清空
		} else {
			//清空注释
			annotation = ""
		}
		if err != nil {
			if err == io.EOF {
				fmt.Println("完成读取!")
				break
			} else {
				fmt.Println("读取文件错误!", err)
				return
			}
		}
	}
}

func setMdResult() {
	//获取下静态变量map == 无所谓
	services := serviceMap
	messages := messageMap
	messageAnnos := messageAnnoMap
	// 初始化一下,大标题和,总api注释
	writeToMdLine("# " + serviceSetName)
	writeToMdLine("> " + serviceAnnotation)
	newLine()
	//获取当前时间
	formatTimeStr := time.Unix(time.Now().Unix(), 0).Format("2006-01-02")
	writeToMdLine("文档维护|code auto-generation")
	writeToMdLine("| ------- | ------ |")
	writeToMdLine("更新日期|" + formatTimeStr)
	writeToMdLine("文档版本|v1.0")

	for api, ser := range services {
		writeToMdLine("### " + strconv.Itoa(ApiCount) + ".接口 " + api)
		writeToMdLine("> " + ser.annotation)
		newLine()
		writeToMdLine("**请求参数 " + ser.req + "**")
		writeToMdLine("> " + messageAnnos[ser.req])
		newLine()
		writeToMdLine("| 参数名 | 类型 | 备注 |")
		writeToMdLine("| ------- | ------ | -------- |")
		// 请求参数
		for _, v := range messages[ser.req] {
			writeToMdLine("| " + v.name + "| " + v.goType + "| " + v.annotation + "|")
		}
		newLine()
		writeToMdLine("**响应参数 " + ser.res + "**")
		writeToMdLine("> " + messageAnnos[ser.res])
		newLine()
		writeToMdLine("| 参数名 | 类型 | 备注 |")
		writeToMdLine("| ------- | ------ | -------- |")
		// 响应参数
		for _, v := range messages[ser.res] {
			writeToMdLine("| " + v.name + "| " + v.goType + "| " + v.annotation + "|")
		}
		newLine()
		ApiCount++
	}

	if len(structMap) > 0 {
		newLine()
		writeToMdLine("### 结构体参数: ")
		writeToMdLine("> " + "非基本数据类型的参数")
		newLine()
	}
	// 结构体参数,放到md文件最后
	for _, structName := range structMap {
		writeToMdLine("**结构体参数 TypeName: <span id=" + structName + ">" + structName + "</span>**")
		writeToMdLine("> " + messageAnnos[structName])
		newLine()
		writeToMdLine("| 参数名 | 类型 | 备注 |")
		writeToMdLine("| ------- | ------ | -------- |")
		for _, v := range messages[structName] {
			writeToMdLine("| " + v.name + "| " + v.goType + "| " + v.annotation + "|")
		}
	}
}

//输出文件
func writeMdFile(fileName string) {
	var filename = fileName + ".md"
	var f *os.File
	var err1 error
	/***************************** 使用 io.WriteString 写入文件 ***********************************************/
	if checkFileIsExist(filename) { //如果文件存在
		fmt.Println("文件已存在!")
		return
	} else {
		//创建文件
		f, err1 = os.Create(filename)
	}
	defer f.Close()
	//写入文件(字符串)
	n, err1 := io.WriteString(f, MdResult)

	if err1 != nil {
		panic(err1)
	}
	fmt.Printf("写入 %d 个字节", n)
}

// 向输出结果写一行,并换行
func writeToMdLine(s string) {
	MdResult += s
	newLine()
}

// 向输出结果并换行
func newLine() {
	MdResult += "\r\n"
}

//检查类型,将proto类型转换成 go类型
func checkType(s string) (string, bool) {
	switch s {
	case "double":
		return "float64", false
	case "float":
		return "float32", false
	case "int32":
		return s, false
	case "int64":
		return s, false
	case "uint32":
		return s, false
	case "uint64":
		return s, false
	case "sint32":
		return "int32", false
	case "fixed64":
		return "uint64", false
	case "sfixed32":
		return "int32", false
	case "sfixed64":
		return "int64", false
	case "bool":
		return s, false
	case "string":
		return s, false
	case "bytes":
		return "[]byte", false
	default:
		//不是基本类型,则存放到结构体暂时存起来
		for _, v := range structMap {
			if v == s {
				return s, true
			}
		}
		structMap = append(structMap, s)
		return s, true
	}
}

//判断文件是否存在  存在返回 true 不存在返回false
func checkFileIsExist(filename string) bool {
	var exist = true
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		exist = false
	}
	return exist
}

// 获取Message中参数
func getMessage(s string) string {
	//去除所有关键字如//  ; 和=
	s = strings.Replace(s, "required", "", -1)
	s = strings.Replace(s, "repeated", "", -1)
	s = strings.Replace(s, "optional", "", 1)
	s = strings.Replace(s, "option", "", -1)
	s = strings.Replace(s, ":", "", -1)
	s = strings.Replace(s, "//", "", -1)
	s = strings.Replace(s, ";", "", -1)
	s = strings.Replace(s, "=", "", -1)
	return s
}

//带有annotation之后的 不进行解析
func fmtAnnotationLine(s string) string {
	//s = strings.Replace(s,":","",-1)
	s = strings.Replace(s, "\n", "", -1)
	s = strings.Replace(s, "\r\n", "", -1)
	m := strings.Index(s, "//annotation:")
	if m == -1 {
		return ""
	}
	index := strings.Index(s, "//")
	if m == index {
		return string([]byte(s)[m+len("//annotation:"):])
	} else {
		return ""
	}
}

// 获取两个字符串之间的字符,如果end=="",则后置字符不截取
func getBetweenStr(str string, start string, end string) string {
	//str字符的位置
	n := strings.Index(str, start)
	if n == -1 {
		n = 0
	}
	str = string([]byte(str)[n+len(start):])
	if end != "" {
		//截止字符的位置
		m := strings.Index(str, end)
		if m == -1 {
			m = len(str)
		}
		str = string([]byte(str)[:m])
	}
	//去除所有空格
	str = strings.Replace(str, " ", "", -1)
	return str
}
