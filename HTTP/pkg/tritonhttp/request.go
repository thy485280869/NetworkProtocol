package tritonhttp

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

type Request struct {
	Method string // e.g. "GET"
	URL    string // e.g. "/path/to/a/file"
	Proto  string // e.g. "HTTP/1.1"

	// Header stores misc headers excluding "Host" and "Connection",
	// which are stored in special fields below.
	// Header keys are case-incensitive, and should be stored
	// in the canonical format in this map.
	Header map[string]string

	Host  string // determine from the "Host" header
	Close bool   // determine from the "Connection" header
}

// ReadRequest tries to read the next valid request from br.
//
// If it succeeds, it returns the valid request read. In this case,
// bytesReceived should be true, and err should be nil.
//
// If an error occurs during the reading, it returns the error,
// and a nil request. In this case, bytesReceived indicates whether or not
// some bytes are received before the error occurs. This is useful to determine
// the timeout with partial request received condition.
func ReadRequest(br *bufio.Reader) (req *Request, bytesReceived bool, err error) {
	req = &Request{}
	headers := make(map[string]string, 3)
	fmt.Println("开始读取请求头")
	// Read start line
	startLine,err := ReadLine(br)
	if err != nil {
		if err.Error() == "EOF" {//EOF 读完直接跳出循环
			bytesReceived = true
		} else if n := strings.Index(err.Error(),"i/o timeout"); n != -1 {//超时错误
			fmt.Println("i/o timeout:",err.Error())
			return nil, false, errors.New("i/o timeout")
		} else {//其他错误
			fmt.Println("ReadRequest error:",err.Error())
			return nil, false, err
		}
	}
	fmt.Println(startLine)
	// Read headers
	var res string
	for {
		res,err = ReadLine(br) //当读到请求末尾会返回""
		if res == "" {//如果读到当前请求的末尾则直接返回
			break
		}
		fmt.Println("res:"+res)
		if err != nil {
			if err.Error() == "EOF" {//EOF 表示请求全部读完 直接跳出循环
				bytesReceived = true
				break
			} else if n := strings.Index(err.Error(),"i/o timeout"); n != -1 {//超时错误
				fmt.Println("i/o timeout:",err.Error())
				return nil, bytesReceived, errors.New("i/o timeout")
			} else {//其他错误
				fmt.Println("ReadRequest error:",err.Error())
				return nil, bytesReceived, err
			}
		}
		bytesReceived = true
		n := strings.Index(res,":")//分离键和值
		if n==-1 || n==0 {
			fmt.Println("headers format error:"+res)
			return nil,bytesReceived,errors.New("headers format error")
		}
		k1 := res[:n]
		v1 := res[n+1:]
		k := strings.TrimSpace(k1)//去除键的前后多余空格
		v := strings.TrimSpace(v1)//去除值的前后多余空格
		headers[k] = v
	}

	// Check required headers
	host,ok := headers[CanonicalHeaderKey("host")]
	if !ok {//if host not exist
		fmt.Println("key 'host' is not exist")
		return nil,bytesReceived, errors.New("key 'host' is not exist")
	} else {
		req.Host = host
		delete(headers,CanonicalHeaderKey("host"))
	}
	// Handle special headers
	data := strings.Split(startLine," ")
	req.Method = strings.TrimSpace(data[0])
	if req.Method != "GET" {
		return nil,true,errors.New("request method erro")
	}
	req.URL = strings.TrimSpace(data[1])
	req.Proto = strings.TrimSpace(data[2])
	req.Header = headers
	c, ok := headers[CanonicalHeaderKey("connection")]
	if ok {
		if c == "close" {req.Close = true} else {req.Close = false}
		delete(headers,CanonicalHeaderKey("connection"))
	} else {
		req.Close = false
	}

	return
}

//HandleUrl 对请求req的url进行处理，不符合格式和路径不存在的会返回错误 正确的最终都会被转换为相对路径保存至req.URL中
func (req *Request) HandleUrl(rootPath string) error {
	fmt.Println("正在处理请求url:"+req.URL)
	if req.URL == "/" {// 如果url为"/"则重新设置为"/index.html"
		fmt.Println(1)
		filePath := "/index.html"
		req.URL = filePath
		return nil
	} else if n := strings.Index(req.URL,".."); n != -1 {// 如果url包含有 ".." 则返回404
		//404
		return errors.New("404")
	} else if n := strings.Index(req.URL,"/"); n != 0 {// 如果url第一个字符不为 "/"
		if m := strings.Index(req.URL, rootPath); m != 0 {//如果url不是绝对路径 返回404
			//400
			return errors.New("400")
		} else {//如果是绝对路径
			relativePath := strings.Split(req.URL,rootPath)[1] //先去除url前面的根路径得到相对路径
			if n := strings.Index(relativePath,"/"); n != 0 {//如果相对路径不以 ”/“ 开头则返回400
				return errors.New("400")
			} else {//相对路径以 "/" 开头
				if fileInfo,err := os.Stat(rootPath+relativePath); err != nil {
					if os.IsNotExist(err) || fileInfo.IsDir() {//如果路径不存在 或者 是文件夹 返回404
						return errors.New("404")
					} else {//其他错误返回400
						return errors.New("400")
					}
				} else {//将正确的相对路径赋给req.URL
					req.URL = relativePath
					return nil
				}
			}
		}
	} else {//url第一个字符为 "/" 则只需判断文件存不存在就行了 [/sad/as.x]
		filePath := rootPath + req.URL //拼接绝对路径
		if fileInfo,err := os.Stat(filePath); err != nil {
			if os.IsNotExist(err) || fileInfo == nil || fileInfo.IsDir() {//如果路径不存在 或者 文件不存在 或者 是文件夹
				return errors.New("404")
			} else {//其他错误返回400
				return errors.New("400")
			}
		}
		return nil
	}
}