package tritonhttp

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
)

const (
	CRLF = "\r\n"
	contentTypeHTML1 = "text/html; charset=utf-8"
	contentTypeJPG1  = "image/jpeg"
	contentTypePNG1  = "image/png"
)

type Response struct {
	StatusCode int    // e.g. 200
	Proto      string // e.g. "HTTP/1.1"

	// Header stores all headers to write to the response.
	// Header keys are case-incensitive, and should be stored
	// in the canonical format in this map.
	Header map[string]string

	// Request is the valid request that leads to this response.
	// It could be nil for responses not resulting from a valid request.
	Request *Request

	// FilePath is the local path to the file to serve.
	// It could be "", which means there is no file to serve.
	FilePath string
}

// Write writes the res to the w.
func (res *Response) Write(w io.Writer) error {
	if err := res.WriteStatusLine(w); err != nil {
		return err
	}
	if err := res.WriteSortedHeaders(w); err != nil {
		return err
	}
	if err := res.WriteBody(w); err != nil {
		return err
	}
	return nil
}

// WriteStatusLine writes the status line of res to w, including the ending "\r\n".
// For example, it could write "HTTP/1.1 200 OK\r\n".
func (res *Response) WriteStatusLine(w io.Writer) error {
	res.Proto = "HTTP/1.1"
	var description string
	switch res.StatusCode {
	case 200:
		description = "OK"
	case 400:
		description = "Bad Request"
	case 404:
		description = "Not Found"
	}
	data := res.Proto + " " + strconv.Itoa(res.StatusCode) + " " + description + CRLF
	fmt.Println("WriteStatusLine:")
	fmt.Print(data)
	_,err := w.Write([]byte(data))
	return err
}

// WriteSortedHeaders writes the headers of res to w, including the ending "\r\n".
// For example, it could write "Connection: close\r\nDate: foobar\r\n\r\n".
// For HTTP, there is no need to write headers in any particular order.
// TritonHTTP requires to write in sorted order for the ease of testing.
func (res *Response) WriteSortedHeaders(w io.Writer) error {
	//if res.Header == nil {
	//	res.Header = make(map[string]string)
	//	now := time.Now()
	//	res.Header[CanonicalHeaderKey("date")] = FormatTime(now)
	//	header := CanonicalHeaderKey("date") + ":" + FormatTime(now) + CRLF
	//	if res.StatusCode == 200 {
	//		fileInfo,err := os.Stat(res.FilePath)//get file information
	//		if err != nil {
	//			return err
	//		}
	//		fileExt := path.Ext(path.Base(res.FilePath))
	//		var fileType string
	//		switch fileExt {
	//		case ".html":
	//			fileType = contentTypeHTML1
	//		case "./jpg":
	//			fileType = contentTypeJPG1
	//		case "./png":
	//			fileType = contentTypePNG1
	//		default:
	//			fileType = "text/plain"
	//		}
	//		res.Header[CanonicalHeaderKey("last-modified")] = FormatTime(fileInfo.ModTime())
	//		res.Header[CanonicalHeaderKey("content-type")] = fileType
	//		res.Header[CanonicalHeaderKey("content-length")] = strconv.FormatInt(fileInfo.Size(),10)
	//		header += CanonicalHeaderKey("last-modified") + ":" + FormatTime(fileInfo.ModTime()) + CRLF
	//		header += CanonicalHeaderKey("content-type") + ":" + fileType + CRLF
	//		header += CanonicalHeaderKey("content-length") + ":" + strconv.FormatInt(fileInfo.Size(),10) + CRLF
	//	}
	//	if res.StatusCode == 400 {
	//		res.Header[CanonicalHeaderKey("connection")] = "close"
	//		header += CanonicalHeaderKey("connection") + ":" + "close" + CRLF
	//	} else {
	//		if res.Request != nil && res.Request.Close {
	//			res.Header[CanonicalHeaderKey("connection")] = "close"
	//			header += CanonicalHeaderKey("connection") + ":" + "close" + CRLF
	//		}
	//	}
	//	header += CRLF
	//	fmt.Print(header)
	//	_, err := w.Write([]byte(header))
	//	return err
	//}
	//
	var header string
	for k,v := range res.Header {
		header += CanonicalHeaderKey(k) + ": " + v + CRLF
	}
	header += CRLF
	fmt.Print(header)
	_, err := w.Write([]byte(header))
	return err

}

// WriteBody writes res' file content as the response body to w.
// It doesn't write anything if there is no file to serve.
func (res *Response) WriteBody(w io.Writer) error {
	if res.StatusCode == 200 {
		file, err := os.Open(res.FilePath)
		if err != nil {
			return errors.New("open file error:" + err.Error())
		}
		defer file.Close()
		buf := make([]byte, 128)
		for {
			n, err := file.Read(buf)
			fmt.Print(string(buf))
			w.Write(buf[:n])
			if err == io.EOF {
				break
			} else if err != nil {
				return err
			}
		}
	} else if res.StatusCode == 404 {
		data := []byte("NOT FOUND PAGER")
		w.Write(data)

	} else if res.StatusCode == 400 {
		data := []byte("BAD REQUEST")
		w.Write(data)
	}
	return nil
}
