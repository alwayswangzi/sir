package sir

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

type Router struct {
	mux *http.ServeMux
}

func New() *Router {
	return &Router{mux: http.NewServeMux()}
}

type Ctx struct {
	w http.ResponseWriter
	r *http.Request
}

func newCtx(w http.ResponseWriter, r *http.Request) *Ctx {
	return &Ctx{w, r}
}

func (r *Router) ListenAndServe(addr string) {
	if err := http.ListenAndServe(addr, r.mux); err != nil {
		panic(err)
	}
}

func (r *Router) Handler(pattern string, f func(c *Ctx)) {
	r.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		f(newCtx(w, r))
	})
}

func (r *Router) Static(path string) {
	prefix := path[:len(path)-1]
	dir := path[1 : len(path)-1]
	r.mux.Handle(path, http.StripPrefix(prefix, http.FileServer(http.Dir(dir))))
}

func (r *Router) Template(path, file string) {
	t, err := template.ParseFiles(file)
	if err != nil {
		log.Fatal(err)
	}

	r.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if err := t.Execute(w, nil); err != nil {
			log.Fatal(err)
		}
	})
}

func (c *Ctx) Bind(v interface{}) error {
	if c.r.Header.Get("Content-Type") != "application/json" {
		return errors.New("content-type not support")
	}

	if body, err := ioutil.ReadAll(c.r.Body); err != nil {
		return err
	} else if len(body) != 0 {
		if err = json.Unmarshal(body, v); err != nil {
			return err
		}
	}
	return nil
}

var (
	isDigit = regexp.MustCompile(`(?m)\d+`)
	isHex   = regexp.MustCompile(`(?m)[0-9a-fA-F]+`)
)

func (c *Ctx) GetQuery() url.Values {
	return c.r.URL.Query()
}

func (c *Ctx) GetURILastNumber() (int, error) {
	uri := c.r.RequestURI
	if uri == "" {
		return -1, errors.New("uri is empty")
	}
	if strings.Contains(uri, "?") {
		uri = uri[:strings.Index(uri, "?")]
	}
	if strings.HasSuffix(uri, "/") {
		uri = uri[:len(uri)-1]
	}
	p := strings.LastIndex(uri, "/")
	if p == -1 {
		return -1, errors.New("invalid uri")
	}
	number := uri[p+1:]
	if isDigit.FindString(number) == "" {
		return 1, nil
	}
	i, err := strconv.ParseInt(number, 10, 64)
	if err != nil {
		return -1, err
	}
	return int(i), nil
}

func (c *Ctx) GetURILastHex() (string, error) {
	uri := c.r.RequestURI
	if uri == "" {
		return "", errors.New("uri is empty")
	}
	if strings.Contains(uri, "?") {
		uri = uri[strings.Index(uri, "?")+1:]
	}
	if strings.HasSuffix(uri, "/") {
		uri = uri[:len(uri)-1]
	}
	p := strings.LastIndex(uri, "/")
	if p == -1 {
		return "", errors.New("invalid uri")
	}
	hex := uri[p+1:]
	if isHex.FindString(hex) == "" {
		return "", fmt.Errorf("invalid hex:%s", hex)
	}
	return hex, nil
}

func (c *Ctx) Execute(t *template.Template, data interface{}) error {
	return t.Execute(c.w, data)
}

func (c *Ctx) Success(data ...interface{}) {
	c.Response(0, "ok", data...)
}

func (c *Ctx) Fail(err ...error) {
	if len(err) != 0 {
		LogError(err[0], 3)
		c.Response(-1, err[0].Error(), nil)
	} else {
		c.Response(-1, "error", nil)
	}
}

func (c *Ctx) BadRequest(err ...error) {
	c.w.WriteHeader(http.StatusBadRequest)
	if len(err) != 0 {
		_, _ = c.w.Write([]byte(err[0].Error()))
	}
}

func (c *Ctx) NotFound() {
	c.w.WriteHeader(http.StatusNotFound)
}

func (c *Ctx) Response(code int, msg string, data ...interface{}) {
	resp := map[string]interface{}{
		"code": code,
		"msg":  msg,
	}
	if len(data) != 0 {
		resp["data"] = data[0]
	}

	if body, err := json.Marshal(resp); err != nil {
		panic(err)
	} else if _, err = c.w.Write(body); err != nil {
		panic(err)
	}
}

func (c *Ctx) Image(img []byte, fileType string) {
	c.w.Header().Set("Accept-Ranges", "bytes")
	c.w.Header().Set("Content-Type", "image/"+fileType)

	if size, err := c.w.Write(img); err != nil {
		LogError(fmt.Errorf("write image err:%v, size:%d", err, size), 3)
	}
}

func (c *Ctx) Upload(maxSize int64, allowedTypes []string) (string, []byte, error) {
	c.r.Body = http.MaxBytesReader(c.w, c.r.Body, maxSize)
	if err := c.r.ParseMultipartForm(maxSize); err != nil {
		return "", nil, fmt.Errorf("parse form failed! %v, max size: %dKB", err, maxSize/1000)
	}

	file, fileHeader, err := c.r.FormFile("file")
	if err != nil {
		return "", nil, fmt.Errorf("get file form form failed! %v", err)
	}
	defer file.Close()

	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return "", nil, fmt.Errorf("read file failed! %v", err)
	}

	if len(allowedTypes) != 0 {
		filetype := http.DetectContentType(fileBytes)
		ok := false
		for _, t := range allowedTypes {
			if filetype == t {
				ok = true
				break
			}
		}
		if !ok {
			return "", nil, fmt.Errorf("forbidden content type: %s", filetype)
		}
	}

	return fileHeader.Filename, fileBytes, nil
}

func (c *Ctx) Download(filename string, file []byte) error {
	c.w.Header().Add("Content-type", "application/octet-stream")
	c.w.Header().Add("content-disposition", "attachment; filename=\""+filename+"\"")

	if size, err := c.w.Write(file); err != nil {
		LogError(fmt.Errorf("write file data err:%v, size:%d", err, size), 3)
		return err
	}
	return nil
}
