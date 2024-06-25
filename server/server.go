package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

type Headers struct{
    contentType     string
    contentLength   int
    userAgent       string
    contentEncoding []string
}

type Request struct{
    requestLine     string
    Headers
    body            []byte
}

func getContentType(filename string) string {
    parts := strings.SplitN(filename, ".", 2)
    if len(parts) < 2 {
        return "text/plain"
    }

    ext := parts[1]

    switch ext {
    case "jpeg":
        return "image/jpeg"
    case "png":
        return "image/png"
    case "html":
        return "text/html"
    case "json":
        return "application/json"
    case "xml":
        return "application/xml"
    default:
        return "text/plain"
    }
}

func main() {
	fmt.Println("Logs from your program will appear here!")

    listener, err := net.Listen("tcp", "0.0.0.0:4221")

	if err != nil {
        fmt.Println("Failed to bind to port 4221:", err)
	 	os.Exit(1)
	}
	
    defer listener.Close()
    
    for {
        conn, err := listener.Accept()

        if err != nil {
            fmt.Println("Error accepting connection: ", err.Error())
            os.Exit(1)
        }

        go handleConnection(conn)
    }
}

func responseHeaders(statusCode int, contentType string, contentLength int, contentEncoding []string) string {
	statusText := "OK"
	switch statusCode {
	case 200:
		statusText = "OK"
	case 201:
		statusText = "Created"
	case 404:
		statusText = "Not Found"
	case 405:
		statusText = "Method Not Allowed"
	}

	headers := fmt.Sprintf("HTTP/1.1 %d %s\r\nContent-Type: %s\r\nContent-Length: %d\r\n", statusCode, statusText, contentType, contentLength)
    if len(contentEncoding) > 0 {
        for _,v := range contentEncoding { 
		    headers += fmt.Sprintf("Content-Encoding: %s\r\n", v)
        }
	}
	headers += "\r\n"
	return headers
}

func gzipCompressed(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	_, err := gzipWriter.Write(data)
	if err != nil {
		return nil, err
	}
	if err := gzipWriter.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}


func handleConnection(conn net.Conn){   
    defer conn.Close() 

    reader := bufio.NewReader(conn)
    
    requestLine, err := reader.ReadString('\n')
    if err != nil {
        fmt.Println("Error reading data: ", err)
        return
    }
    request := Request{requestLine: strings.TrimSpace(requestLine)} 
    
    reqLine := strings.Split(request.requestLine, " ") 

    method := reqLine[0]
    url := reqLine[1]


    // read headers

    for {
        line, err := reader.ReadString('\n')
        if err != nil {
            fmt.Println("Error reading data:", err)
            return
        }
        line = strings.TrimSpace(line)
        if line == "" {
            break // headers are terminated by an empty line
        }

        headerParts := strings.SplitN(line, ":", 2)
        if len(headerParts) == 2 {
            headerName := strings.TrimSpace(headerParts[0])
            headerValue := strings.TrimSpace(headerParts[1])
            switch headerName {
			case "User-Agent":
				request.userAgent = headerValue
			case "Content-Length":
				request.contentLength, err = strconv.Atoi(headerValue)
				if err != nil {
					fmt.Println("Cannot convert string to integer")
				}
			case "Accept-Encoding":
                headerValueParts := strings.Split(headerValue, ", ")
                for _, headerVal := range headerValueParts {
                    if headerVal == "gzip" {
                        request.contentEncoding = append(request.contentEncoding, headerVal)
                    }  
                }
            }
        }
    }

    // read body

    if request.contentLength > 0 {
        body := make([]byte, request.contentLength)
        _, err := io.ReadFull(reader, body)

        if err != nil {
            fmt.Println("Cannot read body:", err)
            return
        }
        request.body = body
    }

    urlParts := strings.Split(url, "/")

    filename := ""
    if len(urlParts) > 2 {
        filename = urlParts[2]
    }
    
    switch {
    case url == "/":
        conn.Write([]byte(responseHeaders(200, "text/plain", 0, request.contentEncoding) + ""))
    case len(urlParts) > 1 && urlParts[1] == "echo":
        data := filename
        request.contentLength = len(filename)
        request.contentType = getContentType(filename)
        if len(request.contentEncoding) > 0 && request.contentEncoding[0] == "gzip" {
            compressedData, _ := gzipCompressed([]byte(data))
            request.contentLength = len(compressedData)   
            metadata := responseHeaders(200, request.contentType, request.contentLength, request.contentEncoding)
            conn.Write([]byte(metadata))
            conn.Write(compressedData)
            return
        }
        metadata := responseHeaders(200, request.contentType, request.contentLength, request.contentEncoding)
        conn.Write([]byte(metadata + data))
    case len(urlParts) > 1 && urlParts[1] == "user-agent": 
        data := request.userAgent
        request.contentLength = len(request.userAgent)
        request.contentType = "text/plain"
        if len(request.contentEncoding) > 0 && request.contentEncoding[0] == "gzip" {
            compressedData, _ := gzipCompressed([]byte(data))
            request.contentLength = len(compressedData)   
            metadata := responseHeaders(200, request.contentType, request.contentLength, request.contentEncoding)
            conn.Write([]byte(metadata))
            conn.Write(compressedData)
            return
        }
        metadata := responseHeaders(200, request.contentType, request.contentLength, request.contentEncoding)
        conn.Write([]byte(metadata + data))
    case len(urlParts) > 1 && urlParts[1] == "files":
        dir := "/"
        if len(os.Args) > 2 {
            dir = os.Args[2]
        }
        file := urlParts[2]

        if method == "GET" {
            data, err := os.ReadFile(dir + "/" + file)
            if err!= nil {
                conn.Write([]byte(responseHeaders(404, "text/plain", 0, request.contentEncoding) + "Not Found"))
                return
            }
            if len(request.contentEncoding) > 0 && request.contentEncoding[0] == "gzip" {
                compressedData, _ := gzipCompressed(data)
                request.contentLength = len(compressedData)   
                metadata := responseHeaders(200, request.contentType, request.contentLength, request.contentEncoding)
                conn.Write([]byte(metadata))
                conn.Write(compressedData)
                return
            }
            metadata := responseHeaders(200, "application/octet-stream", len(data), request.contentEncoding)
            conn.Write([]byte(metadata + string(data)))
        } else if method == "POST" {
            newFile, err:= os.Create(dir + "/" + file)
            if err != nil {
                fmt.Println("Failed to create a file")
            }
            
            defer newFile.Close()
            
            newFile.Write(request.body)

            conn.Write([]byte(responseHeaders(201, "text/plain", 0, request.contentEncoding) + "Created"))

        } else {
            conn.Write([]byte(responseHeaders(405, "text/plain", 0, request.contentEncoding) + "Method Not Allowed"))
        }
    default:
        conn.Write([]byte(responseHeaders(404, "text/plain", 0, request.contentEncoding) + "Not Found"))
    }
    
}
