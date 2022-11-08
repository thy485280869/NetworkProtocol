package tritonhttp

import (
	"bytes"
	"os"
	"testing"
)

func TestWriteStatusLine(t *testing.T) {
	var tests = []struct {
		name string
		res  *Response
		want string
	}{
		{
			"OK",
			&Response{
				StatusCode: 200,
				Proto:      "HTTP/1.1",
			},
			"HTTP/1.1 200 OK\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buffer bytes.Buffer
			if err := tt.res.WriteStatusLine(&buffer); err != nil {
				t.Fatal(err)
			}
			got := buffer.String()
			if got != tt.want {
				t.Fatalf("got: %q, want: %q", got, tt.want)
			}
		})
	}
}

func TestWriteSortedHeaders(t *testing.T) {
	var tests = []struct {
		name string
		res  *Response
		want string
	}{
		{
			"Basic",
			&Response{
				Header: map[string]string{
					"Connection": "close",
					"Date":       "foobar",
					"Misc":       "hello world",
				},
			},
			"Connection: close\r\n" +
				"Date: foobar\r\n" +
				"Misc: hello world\r\n" +
				"\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buffer bytes.Buffer
			if err := tt.res.WriteSortedHeaders(&buffer); err != nil {
				t.Fatal(err)
			}
			got := buffer.String()
			if got != tt.want {
				t.Fatalf("got: %q, want: %q", got, tt.want)
			}
		})
	}
}

func TestWriteBody(t *testing.T) {
	var tests = []struct {
		name string
		path string
	}{
		{
			"Basic",
			"E:/files/index.html",
		},
		{
			"NoBody",
			"E:/files/1.txt", //服务器调用res.WriteBody()之前会验证res.FilePath是否为符合格式的路径，且会把所有相对路径转换为绝对路径，所以这个测试中不能使用不正常的文件路径（例如相对路径和测试样例的""等等）
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := &Response{//调用res.WriteBody()之前res必须要给状态码赋对应的值，且文件路径为转换后的绝对路径，只有状态码200才会去读取文件写入文件，404、400只会写简单的提示语句
				StatusCode: 200,
				FilePath: tt.path,
			}
			var buffer bytes.Buffer
			if err := res.WriteBody(&buffer); err != nil {
				t.Fatal(err)
			}
			bytesGot := buffer.Bytes()

			// No path, no bytes
			var bytesWant []byte
			if tt.path != "" {
				var err error
				if bytesWant, err = os.ReadFile(tt.path); err != nil {
					t.Fatal(err)
				}
			}

			if !bytes.Equal(bytesGot, bytesWant) {
				if len(bytesWant) <= 128 {
					// For small file, show the bytes
					t.Fatalf("\ngot: %q\nwant: %q", bytesGot, bytesWant)
				} else {
					// Otherwise, just show number of bytes
					t.Fatalf(
						"bytes written are different from the file\ngot: %v bytes, want: %v bytes",
						len(bytesGot),
						len(bytesWant),
					)
				}
			}
		})
	}
}
