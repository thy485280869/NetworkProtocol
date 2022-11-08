package tritonhttp

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	// Addr specifies the TCP address for the server to listen on,
	// in the form "host:port". It shall be passed to net.Listen()
	// during ListenAndServe().
	Addr string // e.g. ":0"

	// DocRoot specifies the path to the directory to serve static files from.
	DocRoot string
}

// ListenAndServe listens on the TCP network address s.Addr and then
// handles requests on incoming connections.
func (s *Server) ListenAndServe() error {

	server, err := net.Listen("tcp",s.Addr)
	if err != nil {
		return err
	}
	defer server.Close()

	// Hint: call HandleConnection
	for {
		//accept connection from client
		conn, err := server.Accept()
		if err != nil {
			fmt.Println("获取连接出错")
		}
		fmt.Println("client["+conn.RemoteAddr().String()+"]:connecting...")
		//handle the connection by goroutine
		go s.HandleConnection(conn)
	}
}

// HandleConnection reads requests from the accepted conn and handles them.
func (s *Server) HandleConnection(conn net.Conn) {
	if conn == nil {//check the connection
		log.Panic("invalid connection!")
		resp := &Response{}
		resp.HandleBadRequest()
		resp.Write(conn)
		return
	}

	// Hint: use the other methods below
	var req *Request
	reader := bufio.NewReaderSize(conn, 128)
	var bytesReceivied = false

	for {
		// Set timeout
		err := conn.SetDeadline(time.Now().Add(5 * time.Second))//5s
		if err != nil {// Handle timeout
			if !bytesReceivied {//未收到部分请求时 close
				defer conn.Close()
				return
			} else { //400
				resp := &Response{}
				resp.HandleBadRequest()
				resp.Write(conn)
				fmt.Println("connection timeout:" + err.Error())
				return
			}
		}

		// Try to read next request
		req,bytesReceivied,err = ReadRequest(reader)//读取请求 读完全部请求或者出现错误跳出for循环
		if err != nil {
			fmt.Println("read request error.")
			if err.Error() == "EOF" {// Handle EOF
				fmt.Println("Close connection(EOF):" + conn.RemoteAddr().String())
				res := s.HandleGoodRequest(req)
				if res.StatusCode == 200 {
					res.Write(conn)
					defer conn.Close()
					return
				} else if res.StatusCode == 404 {
					res.Write(conn)
					//return
					continue
				} else if res.StatusCode == 400 {
					res.Write(conn)
					defer conn.Close()
					return
				}
				//err = req.HandleUrl(s.DocRoot)//处理url
				//if err !=nil {
				//	if err.Error() == "400" {//返回400 关闭连接
				//		resp := &Response{}
				//		resp.HandleBadRequest()
				//		resp.Write(conn)
				//		defer conn.Close()
				//		return
				//	} else if err.Error() == "404" {//返回404 读取下一个请求
				//		resp := &Response{}
				//		resp.HandleNotFound(req)
				//		resp.Write(conn)
				//		//return
				//		continue
				//	}
				//} else {//200
				//	resp := &Response{}
				//	filePath := s.DocRoot + req.URL
				//	resp.HandleOK(req, filePath)
				//	resp.Write(conn)
				//	defer conn.Close()
				//	return
				//}
			} else if n := strings.Index(err.Error(),"i/o timeout"); n != -1 {// Handle timeout
				fmt.Println("Close connection(i/o timeout):" + conn.RemoteAddr().String())
				if !bytesReceivied {//如果之前未收到部分请求 服务器简单的close
					defer conn.Close()
					return
				} else {//如果之前收到了部分请求 返回400
					resp := &Response{}
					resp.HandleBadRequest()
					resp.Write(conn)
					defer conn.Close()
					return
				}
			} else {//Handle bad request
				fmt.Println("other error:",err.Error())
				resp := &Response{}
				resp.HandleBadRequest()
				resp.Write(conn)
				defer conn.Close()
				return
			}
		} else {// 读取请求没有格式错误时
			fmt.Println("收到Client端发来的请求["+req.Host + "]")
			res := s.HandleGoodRequest(req)
			if res.StatusCode == 200 {
				res.Write(conn)
				// Close conn if requested
				if res.Request.Close { // Close conn if requested
					defer conn.Close()
					return
				}
			} else if res.StatusCode == 404 {
				res.Write(conn)
				//return
				continue
			} else if res.StatusCode == 400 {
				res.Write(conn)
				defer conn.Close()
				return
			}

			//err = req.HandleUrl(s.DocRoot)//处理url
			//if err !=nil {//如果url不符合条件 或者 找不到资源
			//	if err.Error() == "400" {//格式有误
			//		resp := &Response{}
			//		resp.HandleBadRequest()
			//		resp.Write(conn)
			//		defer conn.Close()
			//		return
			//	} else if err.Error() == "404" {//找不到
			//		resp := &Response{}
			//		resp.HandleNotFound(req)
			//		resp.Write(conn)
			//		//return
			//		continue
			//	}
			//} else {
			//	fmt.Println(req.Method, req.Proto, req.URL, req.Close)
			//	// Handle good request
			//	resp := s.HandleGoodRequest(req) //init 200
			//	fmt.Println(resp.Proto, resp.StatusCode, resp.FilePath)
			//	resp.Write(conn)
			//
			//	// Close conn if requested
			//	if resp.Request.Close { // Close conn if requested
			//		defer conn.Close()
			//		return
			//	}
			//}
		}
	}
}

// HandleGoodRequest handles the valid req and generates the corresponding res.
func (s *Server) HandleGoodRequest(req *Request) (res *Response) {//include 200 and 404
	res = &Response{}
	res.Proto = "HTTP/1.1"
	res.Request = req
	res.Header = req.Header
	// Hint: use the other methods below
	//check url format
	err := req.HandleUrl(s.DocRoot)
	if err !=nil {
		if err.Error() == "400" {//返回400 关闭连接
			res.HandleBadRequest()
		} else if err.Error() == "404" {//返回404
			res.HandleNotFound(req)
		}
	} else {//200
		filePath := s.DocRoot + req.URL//拼接绝对路径
		res.HandleOK(req, filePath)
	}
	return
}

// HandleOK prepares res to be a 200 OK response
// ready to be written back to client.
func (res *Response) HandleOK(req *Request, path1 string) {//200
	res.StatusCode = 200
	res.Request = req
	res.FilePath = path1

	res.Header = make(map[string]string)
	now := time.Now()
	res.Header[CanonicalHeaderKey("date")] = FormatTime(now)

	fileInfo,err := os.Stat(res.FilePath)//get file information
	if err != nil {
		fmt.Println(err)
	}
	fileExt := path.Ext(path.Base(res.FilePath))
	var fileType string
	switch fileExt {
	case ".html":
		fileType = contentTypeHTML1
	case "./jpg":
		fileType = contentTypeJPG1
	case "./png":
		fileType = contentTypePNG1
	default:
		fileType = "text/plain"
	}
	res.Header[CanonicalHeaderKey("last-modified")] = FormatTime(fileInfo.ModTime())
	res.Header[CanonicalHeaderKey("content-type")] = fileType
	res.Header[CanonicalHeaderKey("content-length")] = strconv.FormatInt(fileInfo.Size(),10)

	if req != nil && req.Close {
		res.Header[CanonicalHeaderKey("connection")] = "close"
	}
}

// HandleBadRequest prepares res to be a 400 Bad Request response
// ready to be written back to client.
func (res *Response) HandleBadRequest() {//400
	res.StatusCode = 400
	res.Header = make(map[string]string)
	now := time.Now()
	res.Header[CanonicalHeaderKey("date")] = FormatTime(now)
	res.Header[CanonicalHeaderKey("connection")] = "close"
}

// HandleNotFound prepares res to be a 404 Not Found response
// ready to be written back to client.
func (res *Response) HandleNotFound(req *Request) {//404
	res.StatusCode = 404
	res.Request = req
	res.Header = make(map[string]string)
	now := time.Now()
	res.Header[CanonicalHeaderKey("date")] = FormatTime(now)
	if req != nil && req.Close {
		res.Header[CanonicalHeaderKey("connection")] = "close"
	}
}
